package application

import (
	"context"
	"time"

	"go-service/internal/calendar/domain"
)

func (s *Service) validateBusinessHours(ctx context.Context, startsAt, endsAt time.Time) error {
	if s.schedulePolicy == nil {
		return nil
	}
	policy, err := s.schedulePolicy.GetSchedulePolicy(ctx)
	if err != nil {
		return err
	}
	if policy.Timezone == "" && len(policy.Days) == 0 {
		return nil
	}
	return domain.ValidateWithinBusinessHours(startsAt, endsAt, policy)
}

func (s *Service) requireActivePerson(ctx context.Context, personID string) (domain.Person, error) {
	person, err := s.people.GetPerson(ctx, personID)
	if err != nil {
		return domain.Person{}, err
	}
	if !person.Active {
		return domain.Person{}, domain.InvalidFieldError{Field: "person_id", Reason: "person is not active"}
	}
	return person, nil
}

func (s *Service) validateUpdateReferences(ctx context.Context, current domain.Appointment, in UpdateAppointmentInput) (domain.Patch, error) {
	patch := domain.Patch{
		PersonID:   in.PersonID,
		CustomerID: in.CustomerID,
		ServiceID:  in.ServiceID,
		StartsAt:   in.StartsAt,
		EndsAt:     in.EndsAt,
		Notes:      in.Notes,
	}
	if in.PersonID != nil && *in.PersonID != current.PersonID {
		if _, err := s.requireActivePerson(ctx, *in.PersonID); err != nil {
			return domain.Patch{}, err
		}
	}
	if in.CustomerID != nil && *in.CustomerID != current.CustomerID {
		if _, err := s.customers.GetCustomer(ctx, *in.CustomerID); err != nil {
			return domain.Patch{}, err
		}
	}
	if in.ServiceID != nil && *in.ServiceID != current.ServiceID {
		if _, err := s.services.GetService(ctx, *in.ServiceID); err != nil {
			return domain.Patch{}, err
		}
	}
	if in.Status != nil {
		parsed, err := domain.ParseStatus(*in.Status)
		if err != nil {
			return domain.Patch{}, err
		}
		patch.Status = &parsed
	}
	return patch, nil
}

func (s *Service) hydrateAll(ctx context.Context, items []domain.Appointment) ([]AppointmentView, error) {
	if len(items) == 0 {
		return []AppointmentView{}, nil
	}
	personIDs, customerIDs, serviceIDs := collectUniqueIDs(items)
	people, err := s.people.GetPeopleByIDs(ctx, personIDs)
	if err != nil {
		return nil, err
	}
	customers, err := s.customers.GetCustomersByIDs(ctx, customerIDs)
	if err != nil {
		return nil, err
	}
	services, err := s.services.GetServicesByIDs(ctx, serviceIDs)
	if err != nil {
		return nil, err
	}
	views := make([]AppointmentView, len(items))
	for i, item := range items {
		views[i] = buildView(item, people[item.PersonID], customers[item.CustomerID], services[item.ServiceID])
	}
	return views, nil
}

func collectUniqueIDs(items []domain.Appointment) ([]string, []string, []string) {
	pSeen := make(map[string]struct{}, len(items))
	cSeen := make(map[string]struct{}, len(items))
	sSeen := make(map[string]struct{}, len(items))
	pIDs := make([]string, 0, len(items))
	cIDs := make([]string, 0, len(items))
	sIDs := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := pSeen[item.PersonID]; !ok {
			pSeen[item.PersonID] = struct{}{}
			pIDs = append(pIDs, item.PersonID)
		}
		if _, ok := cSeen[item.CustomerID]; !ok {
			cSeen[item.CustomerID] = struct{}{}
			cIDs = append(cIDs, item.CustomerID)
		}
		if _, ok := sSeen[item.ServiceID]; !ok {
			sSeen[item.ServiceID] = struct{}{}
			sIDs = append(sIDs, item.ServiceID)
		}
	}
	return pIDs, cIDs, sIDs
}

func buildView(appt domain.Appointment, person domain.Person, customer CustomerSummary, svc ServiceSummary) AppointmentView {
	return AppointmentView{
		Appointment:            appt,
		PersonName:             person.Name,
		CustomerName:           customer.Name,
		CustomerPhone:          customer.Phone,
		ServiceName:            svc.Name,
		ServiceDurationMinutes: svc.DurationMinutes,
	}
}
