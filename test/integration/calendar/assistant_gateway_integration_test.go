//go:build integration

package calendar_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar"
	"go-service/internal/catalog"
	"go-service/internal/organizations"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func nextFutureMonday(loc *time.Location) time.Time {
	now := time.Now().In(loc).AddDate(0, 0, 7)
	for now.Weekday() != time.Monday {
		now = now.AddDate(0, 0, 1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
}

func TestAssistantGatewayAndCatalogReaderIntegration(t *testing.T) {
	db, refs := newTestDB(t)
	ctx := t.Context()
	otherCustomer := uuid.NewString()
	if _, err := db.ExecContext(ctx, `INSERT INTO crm_customers (id, name, phone) VALUES ($1, 'Other', '+573009998877')`, otherCustomer); err != nil {
		t.Fatalf("seed other customer: %v", err)
	}
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatalf("new breaker: %v", err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	svcReader := catalog.NewServiceReader(client)
	schedReader := organizations.NewScheduleReader(client)
	gw := calendar.NewAssistantCalendarGateway(client, svcReader, schedReader)

	items, err := svcReader.ListCatalogServices(ctx)
	if err != nil || len(items) != 1 || items[0].CategoryName != "Hair" || items[0].PriceMinorUnits != 50000 {
		t.Fatalf("unexpected catalog services: %+v, err=%v", items, err)
	}

	loc, _ := time.LoadLocation("America/Bogota")
	startAt := nextFutureMonday(loc)
	dateStr := startAt.Format("2006-01-02")

	created, err := gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: refs.customerID, ServiceID: refs.serviceID, PersonID: refs.personA, StartAt: startAt, Notes: "wa",
	})
	if err != nil || created.Status != "confirmed" || !created.EndAt.Equal(startAt.Add(30*time.Minute).UTC()) {
		t.Fatalf("unexpected create: %+v, err=%v", created, err)
	}
	if _, err := gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: otherCustomer, ServiceID: refs.serviceID, PersonID: refs.personA, StartAt: startAt,
	}); !errors.Is(err, calendar.ErrSlotConflict) {
		t.Fatalf("expected ErrSlotConflict, got %v", err)
	}

	// Availability excludes created slot for non-owner, includes for owner when ExcludeAppointmentID is set.
	slotsOther, err := gw.ListAvailability(ctx, calendar.AvailabilityQuery{
		Date: dateStr, ServiceID: refs.serviceID, PersonID: refs.personA, ExcludeAppointmentID: created.ID, CustomerID: otherCustomer,
	})
	if err != nil {
		t.Fatalf("ListAvailability other: %v", err)
	}
	slotsOwner, err := gw.ListAvailability(ctx, calendar.AvailabilityQuery{
		Date: dateStr, ServiceID: refs.serviceID, PersonID: refs.personA, ExcludeAppointmentID: created.ID, CustomerID: refs.customerID,
	})
	if err != nil || len(slotsOwner) != len(slotsOther)+1 {
		t.Fatalf("expected owner to see 1 extra excluded slot: owner=%d other=%d err=%v", len(slotsOwner), len(slotsOther), err)
	}

	upcoming, err := gw.ListCustomerUpcomingAppointments(ctx, refs.customerID)
	if err != nil || len(upcoming) != 1 || upcoming[0].ServiceName != "Cut" || upcoming[0].PersonName != "Lucia" {
		t.Fatalf("unexpected upcoming: %+v, err=%v", upcoming, err)
	}

	// Reschedule: no fields -> ErrNoFieldsToUpdate; cross-customer -> ErrAppointmentNotFound; valid -> updated.
	if _, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: created.ID, CustomerID: refs.customerID,
	}); !errors.Is(err, calendar.ErrNoFieldsToUpdate) {
		t.Fatalf("expected ErrNoFieldsToUpdate, got %v", err)
	}
	newStart := startAt.Add(time.Hour)
	if _, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: created.ID, CustomerID: otherCustomer, StartAt: &newStart,
	}); !errors.Is(err, calendar.ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound for cross-customer reschedule, got %v", err)
	}
	resched, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: created.ID, CustomerID: refs.customerID, StartAt: &newStart, PersonID: &refs.personB,
	})
	if err != nil || resched.PersonID != refs.personB || !resched.EndAt.Equal(newStart.Add(30*time.Minute).UTC()) {
		t.Fatalf("unexpected reschedule: %+v, err=%v", resched, err)
	}

	// Cancel: cross-customer -> ErrAppointmentNotFound; owner -> cancelled.
	if _, err := gw.CancelWhatsAppAppointment(ctx, created.ID, otherCustomer); !errors.Is(err, calendar.ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound for cross-customer cancel, got %v", err)
	}
	cancelled, err := gw.CancelWhatsAppAppointment(ctx, created.ID, refs.customerID)
	if err != nil || cancelled.Status != "cancelled" {
		t.Fatalf("unexpected cancel: %+v, err=%v", cancelled, err)
	}
}
