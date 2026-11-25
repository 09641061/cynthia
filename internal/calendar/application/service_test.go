package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

func newTestFixture() (*application.Service, string, string, string, string) {
	activePersonID := uuid.NewString()
	inactivePersonID := uuid.NewString()
	customerID := uuid.NewString()
	serviceID := uuid.NewString()
	svc := application.NewService(
		newFakeAppointmentStore(),
		fakePersonResolver{people: map[string]domain.Person{
			activePersonID:   {ID: activePersonID, Name: "Lucia", Active: true},
			inactivePersonID: {ID: inactivePersonID, Name: "Inactive", Active: false},
		}},
		fakeCustomerReader{customers: map[string]application.CustomerSummary{
			customerID: {ID: customerID, Name: "Ana", Phone: "+573001112233"},
		}},
		fakeServiceReader{services: map[string]application.ServiceSummary{
			serviceID: {ID: serviceID, Name: "Haircut", DurationMinutes: 30},
		}},
		fakeSchedulePolicyReader{policy: defaultTestSchedulePolicy()},
	)
	return svc, activePersonID, inactivePersonID, customerID, serviceID
}

func TestServiceCreateHydratesAndComputesDuration(t *testing.T) {
	svc, personID, _, customerID, serviceID := newTestFixture()
	start := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

	created, err := svc.Create(t.Context(), application.CreateAppointmentInput{
		PersonID: personID, CustomerID: customerID, ServiceID: serviceID,
		StartsAt: start, Channel: "whatsapp",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !created.Appointment.EndsAt.Equal(start.Add(30 * time.Minute)) {
		t.Fatalf("EndsAt = %v, want +30m", created.Appointment.EndsAt)
	}
	if created.PersonName != "Lucia" || created.CustomerName != "Ana" || created.ServiceName != "Haircut" {
		t.Fatalf("expected hydrated names, got %+v", created)
	}

	list, err := svc.List(t.Context(), application.AppointmentFilter{PersonID: personID})
	if err != nil || len(list) != 1 || list[0].CustomerPhone != "+573001112233" {
		t.Fatalf("List failed: %v, %+v", err, list)
	}
}

func TestServiceCreateRejectsInvalidReferences(t *testing.T) {
	svc, personID, inactiveID, customerID, serviceID := newTestFixture()
	start := time.Date(2026, 11, 25, 11, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		in   application.CreateAppointmentInput
		want error
	}{
		{"missing person", application.CreateAppointmentInput{PersonID: uuid.NewString(), CustomerID: customerID, ServiceID: serviceID, StartsAt: start}, domain.ErrPersonNotFound},
		{"inactive person", application.CreateAppointmentInput{PersonID: inactiveID, CustomerID: customerID, ServiceID: serviceID, StartsAt: start}, domain.ErrInvalid},
		{"missing customer", application.CreateAppointmentInput{PersonID: personID, CustomerID: uuid.NewString(), ServiceID: serviceID, StartsAt: start}, domain.ErrCustomerNotFound},
		{"missing service", application.CreateAppointmentInput{PersonID: personID, CustomerID: customerID, ServiceID: uuid.NewString(), StartsAt: start}, domain.ErrServiceNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(t.Context(), tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got err %v, want %v", err, tc.want)
			}
		})
	}
}

func TestServiceUpdateAndDelete(t *testing.T) {
	svc, personID, _, customerID, serviceID := newTestFixture()
	start := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)
	created, err := svc.Create(t.Context(), application.CreateAppointmentInput{
		PersonID: personID, CustomerID: customerID, ServiceID: serviceID, StartsAt: start,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	confirmed := "confirmed"
	updated, err := svc.Update(t.Context(), created.Appointment.ID, application.UpdateAppointmentInput{Status: &confirmed})
	if err != nil || updated.Appointment.Status != domain.StatusConfirmed {
		t.Fatalf("Update failed: err=%v, updated=%+v", err, updated)
	}
	if err := svc.Delete(t.Context(), created.Appointment.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if err := svc.Delete(t.Context(), created.Appointment.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestServiceEnforcesBusinessHours(t *testing.T) {
	svc, personID, _, customerID, serviceID := newTestFixture()

	_, err := svc.Create(t.Context(), application.CreateAppointmentInput{
		PersonID: personID, CustomerID: customerID, ServiceID: serviceID,
		StartsAt: time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC),
	})
	var fieldErr domain.InvalidFieldError
	if !errors.As(err, &fieldErr) || fieldErr.Field != "starts_at" {
		t.Fatalf("expected starts_at error on Create before open, got %v", err)
	}

	created, err := svc.Create(t.Context(), application.CreateAppointmentInput{
		PersonID: personID, CustomerID: customerID, ServiceID: serviceID,
		StartsAt: time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("valid Create failed: %v", err)
	}

	closedDay := time.Date(2026, 11, 28, 10, 0, 0, 0, time.UTC)
	end := closedDay.Add(30 * time.Minute)
	_, err = svc.Update(t.Context(), created.Appointment.ID, application.UpdateAppointmentInput{
		StartsAt: &closedDay, EndsAt: &end,
	})
	if !errors.As(err, &fieldErr) || fieldErr.Field != "starts_at" {
		t.Fatalf("expected starts_at error on Update to closed day, got %v", err)
	}
}
