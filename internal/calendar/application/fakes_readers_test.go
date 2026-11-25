package application_test

import (
	"context"
	"time"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

type fakePersonResolver struct{ people map[string]domain.Person }

func (f fakePersonResolver) ListActivePeople(_ context.Context) ([]domain.Person, error) {
	out := make([]domain.Person, 0, len(f.people))
	for _, p := range f.people {
		if p.Active {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f fakePersonResolver) GetPerson(_ context.Context, id string) (domain.Person, error) {
	p, ok := f.people[id]
	if !ok {
		return domain.Person{}, domain.ErrPersonNotFound
	}
	return p, nil
}

func (f fakePersonResolver) GetPeopleByIDs(_ context.Context, ids []string) (map[string]domain.Person, error) {
	out := make(map[string]domain.Person, len(ids))
	for _, id := range ids {
		if p, ok := f.people[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

type fakeCustomerReader struct {
	customers map[string]application.CustomerSummary
}

func (f fakeCustomerReader) GetCustomer(_ context.Context, id string) (application.CustomerSummary, error) {
	c, ok := f.customers[id]
	if !ok {
		return application.CustomerSummary{}, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (f fakeCustomerReader) GetCustomersByIDs(_ context.Context, ids []string) (map[string]application.CustomerSummary, error) {
	out := make(map[string]application.CustomerSummary, len(ids))
	for _, id := range ids {
		if c, ok := f.customers[id]; ok {
			out[id] = c
		}
	}
	return out, nil
}

type fakeServiceReader struct {
	services map[string]application.ServiceSummary
}

func (f fakeServiceReader) GetService(_ context.Context, id string) (application.ServiceSummary, error) {
	s, ok := f.services[id]
	if !ok {
		return application.ServiceSummary{}, domain.ErrServiceNotFound
	}
	return s, nil
}

func (f fakeServiceReader) GetServicesByIDs(_ context.Context, ids []string) (map[string]application.ServiceSummary, error) {
	out := make(map[string]application.ServiceSummary, len(ids))
	for _, id := range ids {
		if s, ok := f.services[id]; ok {
			out[id] = s
		}
	}
	return out, nil
}

type fakeSchedulePolicyReader struct {
	policy domain.SchedulePolicy
}

func (f fakeSchedulePolicyReader) GetSchedulePolicy(_ context.Context) (domain.SchedulePolicy, error) {
	return f.policy, nil
}

func defaultTestSchedulePolicy() domain.SchedulePolicy {
	open := domain.DaySchedule{IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}
	closed := domain.DaySchedule{IsOpen: false, OpenTime: "09:00", CloseTime: "18:00"}
	return domain.SchedulePolicy{
		Timezone: "UTC",
		Days: map[time.Weekday]domain.DaySchedule{
			time.Monday: open, time.Tuesday: open, time.Wednesday: open,
			time.Thursday: open, time.Friday: open, time.Saturday: closed, time.Sunday: closed,
		},
	}
}
