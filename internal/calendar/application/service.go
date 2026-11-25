package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar/domain"
)

// Service orchestrates calendar people and appointment use cases.
type Service struct {
	appointments   AppointmentStore
	people         PersonResolver
	customers      CustomerReader
	services       ServiceReader
	schedulePolicy SchedulePolicyReader
}

// NewService creates a calendar application service.
func NewService(
	appointments AppointmentStore,
	people PersonResolver,
	customers CustomerReader,
	services ServiceReader,
	schedulePolicy SchedulePolicyReader,
) *Service {
	return &Service{
		appointments:   appointments,
		people:         people,
		customers:      customers,
		services:       services,
		schedulePolicy: schedulePolicy,
	}
}

// ListPeople returns active bookable people.
func (s *Service) ListPeople(ctx context.Context) ([]domain.Person, error) {
	return s.people.ListActivePeople(ctx)
}

// List returns appointments matching the filter with batch-hydrated references.
func (s *Service) List(ctx context.Context, filter AppointmentFilter) ([]AppointmentView, error) {
	if filter.Status != "" {
		if _, err := domain.ParseStatus(filter.Status); err != nil {
			return nil, err
		}
	}
	items, err := s.appointments.ListAppointments(ctx, filter)
	if err != nil {
		return nil, err
	}
	return s.hydrateAll(ctx, items)
}

// Get returns a single hydrated appointment by ID.
func (s *Service) Get(ctx context.Context, id string) (AppointmentView, error) {
	appt, err := s.appointments.GetAppointment(ctx, id)
	if err != nil {
		return AppointmentView{}, err
	}
	views, err := s.hydrateAll(ctx, []domain.Appointment{appt})
	if err != nil {
		return AppointmentView{}, err
	}
	return views[0], nil
}

// Create validates references, constructs an appointment, and persists it.
func (s *Service) Create(ctx context.Context, in CreateAppointmentInput) (AppointmentView, error) {
	person, err := s.requireActivePerson(ctx, in.PersonID)
	if err != nil {
		return AppointmentView{}, err
	}
	customer, err := s.customers.GetCustomer(ctx, in.CustomerID)
	if err != nil {
		return AppointmentView{}, err
	}
	svc, err := s.services.GetService(ctx, in.ServiceID)
	if err != nil {
		return AppointmentView{}, err
	}
	endsAt := in.EndsAt
	if endsAt.IsZero() && !in.StartsAt.IsZero() && svc.DurationMinutes > 0 {
		endsAt = in.StartsAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)
	}
	appt, err := domain.NewAppointment(domain.CreateInput{
		ID:         uuid.NewString(),
		PersonID:   person.ID,
		CustomerID: customer.ID,
		ServiceID:  svc.ID,
		StartsAt:   in.StartsAt,
		EndsAt:     endsAt,
		Status:     domain.Status(in.Status),
		Channel:    domain.Channel(in.Channel),
		Notes:      in.Notes,
	})
	if err != nil {
		return AppointmentView{}, err
	}
	if err := s.validateBusinessHours(ctx, appt.StartsAt, appt.EndsAt); err != nil {
		return AppointmentView{}, err
	}
	created, err := s.appointments.CreateAppointment(ctx, appt)
	if err != nil {
		return AppointmentView{}, err
	}
	return buildView(created, person, customer, svc), nil
}

// Update validates references and transitions, then persists the appointment.
func (s *Service) Update(ctx context.Context, id string, in UpdateAppointmentInput) (AppointmentView, error) {
	current, err := s.appointments.GetAppointment(ctx, id)
	if err != nil {
		return AppointmentView{}, err
	}
	patch, err := s.validateUpdateReferences(ctx, current, in)
	if err != nil {
		return AppointmentView{}, err
	}
	updated, err := current.ApplyPatch(patch)
	if err != nil {
		return AppointmentView{}, err
	}
	if in.StartsAt != nil || in.EndsAt != nil || in.ServiceID != nil {
		if err := s.validateBusinessHours(ctx, updated.StartsAt, updated.EndsAt); err != nil {
			return AppointmentView{}, err
		}
	}
	saved, err := s.appointments.UpdateAppointment(ctx, updated)
	if err != nil {
		return AppointmentView{}, err
	}
	views, err := s.hydrateAll(ctx, []domain.Appointment{saved})
	if err != nil {
		return AppointmentView{}, err
	}
	return views[0], nil
}

// Delete removes an appointment by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.appointments.DeleteAppointment(ctx, id)
}
