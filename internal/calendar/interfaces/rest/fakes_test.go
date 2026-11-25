package rest_test

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	"go-service/internal/calendar/interfaces/rest"
)

type restFixture struct {
	router     *gin.Engine
	personID   string
	customerID string
	serviceID  string
}

func newRESTFixture() restFixture {
	gin.SetMode(gin.TestMode)
	personID := uuid.NewString()
	customerID := uuid.NewString()
	serviceID := uuid.NewString()
	svc := application.NewService(
		&memStore{items: make(map[string]domain.Appointment)},
		memPeople{people: map[string]domain.Person{
			personID: {ID: personID, Name: "Lucia", Active: true},
		}},
		memCustomers{customers: map[string]application.CustomerSummary{
			customerID: {ID: customerID, Name: "Ana", Phone: "+573001112233"},
		}},
		memServices{services: map[string]application.ServiceSummary{
			serviceID: {ID: serviceID, Name: "Haircut", DurationMinutes: 30},
		}},
		nil,
	)
	router := gin.New()
	rest.RegisterRoutes(router.Group("/v1"), svc)
	return restFixture{router: router, personID: personID, customerID: customerID, serviceID: serviceID}
}

type memStore struct {
	mu    sync.Mutex
	items map[string]domain.Appointment
}

func (m *memStore) ListAppointments(_ context.Context, _ application.AppointmentFilter) ([]domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Appointment, 0, len(m.items))
	for _, a := range m.items {
		out = append(out, a)
	}
	return out, nil
}

func (m *memStore) GetAppointment(_ context.Context, id string) (domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.items[id]
	if !ok {
		return domain.Appointment{}, domain.ErrNotFound
	}
	return a, nil
}

func (m *memStore) CreateAppointment(_ context.Context, appt domain.Appointment) (domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.items {
		if existing.PersonID == appt.PersonID && existing.Status.IsBlocking() && appt.Status.IsBlocking() &&
			appt.StartsAt.Before(existing.EndsAt) && appt.EndsAt.After(existing.StartsAt) {
			return domain.Appointment{}, domain.ErrTimeConflict
		}
	}
	m.items[appt.ID] = appt
	return appt, nil
}

func (m *memStore) UpdateAppointment(_ context.Context, expectedVersion int64, appt domain.Appointment) (domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.items[appt.ID]
	if !ok {
		return domain.Appointment{}, domain.ErrNotFound
	}
	if cur.Version != expectedVersion {
		return domain.Appointment{}, domain.ErrVersionConflict
	}
	appt.Version = cur.Version + 1
	m.items[appt.ID] = appt
	return appt, nil
}

func (m *memStore) DeleteAppointment(_ context.Context, id string, expectedVersion int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.items[id]
	if !ok {
		return domain.ErrNotFound
	}
	if cur.Version != expectedVersion {
		return domain.ErrVersionConflict
	}
	delete(m.items, id)
	return nil
}
