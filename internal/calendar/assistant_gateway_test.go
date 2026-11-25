package calendar

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/calendar/domain"
	"go-service/internal/catalog"
)

func TestAssistantGatewayUnavailableGuards(t *testing.T) {
	ctx := context.Background()
	var nilGW *AssistantCalendarGateway
	nilDB := &AssistantCalendarGateway{serviceReader: &catalog.ServiceReader{}}
	nilSvc := &AssistantCalendarGateway{db: &sql.DB{}}

	for _, tc := range []struct {
		name string
		gw   *AssistantCalendarGateway
	}{
		{"nil receiver", nilGW},
		{"nil db", nilDB},
		{"nil serviceReader", nilSvc},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.gw.ListAvailability(ctx, AvailabilityQuery{}); !errors.Is(err, ErrGatewayUnavailable) {
				t.Fatalf("ListAvailability: expected ErrGatewayUnavailable, got %v", err)
			}
			if _, err := tc.gw.ListCustomerUpcomingAppointments(ctx, "c1"); !errors.Is(err, ErrGatewayUnavailable) {
				t.Fatalf("ListCustomerUpcomingAppointments: expected ErrGatewayUnavailable, got %v", err)
			}
			if _, err := tc.gw.CreateWhatsAppAppointment(ctx, CreateWhatsAppAppointmentInput{}); !errors.Is(err, ErrGatewayUnavailable) {
				t.Fatalf("CreateWhatsAppAppointment: expected ErrGatewayUnavailable, got %v", err)
			}
			if _, err := tc.gw.RescheduleWhatsAppAppointment(ctx, RescheduleWhatsAppAppointmentInput{}); !errors.Is(err, ErrGatewayUnavailable) {
				t.Fatalf("RescheduleWhatsAppAppointment: expected ErrGatewayUnavailable, got %v", err)
			}
			if tc.name != "nil serviceReader" {
				if _, err := tc.gw.CancelWhatsAppAppointment(ctx, "a1", "c1"); !errors.Is(err, ErrGatewayUnavailable) {
					t.Fatalf("CancelWhatsAppAppointment: expected ErrGatewayUnavailable, got %v", err)
				}
			}
		})
	}
}

func TestValidateBookingTargetRejectsPastOrZeroSlot(t *testing.T) {
	fixedNow := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	gw := &AssistantCalendarGateway{now: func() time.Time { return fixedNow }}

	for _, startAt := range []time.Time{
		{},
		fixedNow.Add(-time.Minute),
		fixedNow,
	} {
		if _, err := gw.validateBookingTarget(context.Background(), "s1", "p1", startAt); !errors.Is(err, ErrSlotInPast) {
			t.Fatalf("startAt=%v: expected ErrSlotInPast, got %v", startAt, err)
		}
	}
}

func TestMapGatewayError(t *testing.T) {
	if !errors.Is(mapGatewayError(domain.ErrTimeConflict), ErrSlotConflict) {
		t.Fatalf("expected ErrSlotConflict for domain.ErrTimeConflict")
	}
	if !errors.Is(mapGatewayError(&pgconn.PgError{Code: "23P01"}), ErrSlotConflict) {
		t.Fatalf("expected ErrSlotConflict for 23P01")
	}
	other := errors.New("boom")
	if !errors.Is(mapGatewayError(other), other) {
		t.Fatalf("expected passthrough for unrelated error")
	}
}
