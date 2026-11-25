package application_test

import (
	"context"
	"sync"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

type fakeAppointmentStore struct {
	mu    sync.Mutex
	items map[string]domain.Appointment
}

func newFakeAppointmentStore() *fakeAppointmentStore {
	return &fakeAppointmentStore{items: make(map[string]domain.Appointment)}
}

func (f *fakeAppointmentStore) ListAppointments(_ context.Context, filter application.AppointmentFilter) ([]domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Appointment, 0, len(f.items))
	for _, a := range f.items {
		if (filter.PersonID == "" || a.PersonID == filter.PersonID) && (filter.Status == "" || string(a.Status) == filter.Status) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAppointmentStore) GetAppointment(_ context.Context, id string) (domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[id]
	if !ok {
		return domain.Appointment{}, domain.ErrNotFound
	}
	return a, nil
}

func (f *fakeAppointmentStore) CreateAppointment(_ context.Context, appt domain.Appointment) (domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.items {
		if existing.PersonID == appt.PersonID && existing.Status.IsBlocking() && appt.Status.IsBlocking() &&
			appt.StartsAt.Before(existing.EndsAt) && appt.EndsAt.After(existing.StartsAt) {
			return domain.Appointment{}, domain.ErrTimeConflict
		}
	}
	f.items[appt.ID] = appt
	return appt, nil
}

func (f *fakeAppointmentStore) UpdateAppointment(_ context.Context, appt domain.Appointment) (domain.Appointment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[appt.ID]; !ok {
		return domain.Appointment{}, domain.ErrNotFound
	}
	f.items[appt.ID] = appt
	return appt, nil
}

func (f *fakeAppointmentStore) DeleteAppointment(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.items, id)
	return nil
}
