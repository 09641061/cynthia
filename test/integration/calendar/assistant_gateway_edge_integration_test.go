//go:build integration

package calendar_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar"
	calendaracl "go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/catalog"
	"go-service/internal/iam"
	"go-service/internal/organizations"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestAssistantGatewayEdgeCasesIntegration(t *testing.T) {
	db, refs := newTestDB(t)
	ctx := t.Context()

	otherCustomer := uuid.NewString()
	service60 := uuid.NewString()
	if _, err := db.ExecContext(ctx, `INSERT INTO crm_customers (id, name, phone) VALUES ($1, 'Other', '+573009997766')`, otherCustomer); err != nil {
		t.Fatalf("seed other customer: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO catalog_services (id, category_id, name, duration_minutes, price_minor_units, sort_order)
		SELECT $1, category_id, 'Color', 60, 120000, 2 FROM catalog_services WHERE id = $2`, service60, refs.serviceID); err != nil {
		t.Fatalf("seed 60m service: %v", err)
	}

	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatalf("new breaker: %v", err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	gw := calendar.NewAssistantCalendarGateway(client, calendaracl.NewPersonAdapter(iam.NewPersonReader(client)), catalog.NewServiceReader(client), organizations.NewScheduleReader(client))

	loc, _ := time.LoadLocation("America/Bogota")
	startAt := nextFutureMonday(loc)

	// 1. CreateWhatsAppAppointment persists channel == "whatsapp".
	appt1, err := gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: refs.customerID, ServiceID: refs.serviceID, PersonID: refs.personA, StartAt: startAt,
	})
	if err != nil {
		t.Fatalf("create appt1: %v", err)
	}
	var channel string
	if err := db.QueryRowContext(ctx, `SELECT channel FROM calendar_appointments WHERE id = $1`, appt1.ID).Scan(&channel); err != nil || channel != "whatsapp" {
		t.Fatalf("expected channel=whatsapp, got %q err=%v", channel, err)
	}

	// 2. Reschedule changing ServiceID to 60m service recalculates EndAt = StartAt + 60m.
	resched60, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: appt1.ID, CustomerID: refs.customerID, ServiceID: &service60,
	})
	if err != nil || !resched60.EndAt.Equal(startAt.Add(60*time.Minute).UTC()) {
		t.Fatalf("expected 60m EndAt, got %+v err=%v", resched60, err)
	}

	// 3. Reschedule GiST overlap -> ErrSlotConflict.
	slot2 := startAt.Add(2 * time.Hour)
	appt2, err := gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: otherCustomer, ServiceID: refs.serviceID, PersonID: refs.personA, StartAt: slot2,
	})
	if err != nil {
		t.Fatalf("create appt2: %v", err)
	}
	if _, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: appt1.ID, CustomerID: refs.customerID, StartAt: &slot2,
	}); !errors.Is(err, calendar.ErrSlotConflict) {
		t.Fatalf("expected ErrSlotConflict on reschedule overlap, got %v", err)
	}

	// 4. ListCustomerUpcomingAppointments excludes past, cancelled, and other-customer rows.
	pastStart := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO calendar_appointments (id, person_id, customer_id, service_id, starts_at, ends_at, status, channel)
		 VALUES ($1, $2, $3, $4, $5, $6, 'confirmed', 'whatsapp')`,
		uuid.NewString(), refs.personB, refs.customerID, refs.serviceID, pastStart, pastStart.Add(30*time.Minute),
	); err != nil {
		t.Fatalf("seed past appt: %v", err)
	}
	apptToCancel, err := gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: refs.customerID, ServiceID: refs.serviceID, PersonID: refs.personB, StartAt: startAt.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create apptToCancel: %v", err)
	}
	if _, err := gw.CancelWhatsAppAppointment(ctx, apptToCancel.ID, refs.customerID); err != nil {
		t.Fatalf("cancel apptToCancel: %v", err)
	}

	upcoming, err := gw.ListCustomerUpcomingAppointments(ctx, refs.customerID)
	if err != nil || len(upcoming) != 1 || upcoming[0].ID != appt1.ID || appt2.ID == "" {
		t.Fatalf("expected only appt1 in upcoming, got %+v err=%v", upcoming, err)
	}

	// 5. Reschedule / Cancel on already-cancelled appointment -> ErrAppointmentNotFound.
	newTime := startAt.Add(time.Hour)
	if _, err := gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: apptToCancel.ID, CustomerID: refs.customerID, StartAt: &newTime,
	}); !errors.Is(err, calendar.ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound on reschedule cancelled, got %v", err)
	}
	if _, err := gw.CancelWhatsAppAppointment(ctx, apptToCancel.ID, refs.customerID); !errors.Is(err, calendar.ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound on double cancel, got %v", err)
	}
}
