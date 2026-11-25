package acl

import (
	"context"
	"errors"
	"testing"

	"go-service/internal/assistant/application"
	"go-service/internal/calendar"
	"go-service/internal/crm"
)

func TestMapCRMErr(t *testing.T) {
	if mapCRMErr(nil) != nil {
		t.Fatal("expected nil for nil error")
	}
	other := errors.New("other")
	if !errors.Is(mapCRMErr(other), other) {
		t.Fatal("expected passthrough for unknown error")
	}
	cases := []struct {
		in   error
		want error
	}{
		{crm.ErrConversationClosed, application.ErrConversationClosed},
		{crm.ErrDeliveryFailed, application.ErrDeliveryFailed},
		{crm.ErrDeliveryUncertain, application.ErrDeliveryUncertain},
	}
	for _, tc := range cases {
		if got := mapCRMErr(tc.in); !errors.Is(got, tc.want) {
			t.Fatalf("mapCRMErr(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestMapCalendarErr(t *testing.T) {
	if mapCalendarErr(nil) != nil {
		t.Fatal("expected nil for nil error")
	}
	other := errors.New("other")
	if !errors.Is(mapCalendarErr(other), other) {
		t.Fatal("expected passthrough for unknown error")
	}
	cases := []struct {
		in   error
		want error
	}{
		{calendar.ErrSlotConflict, application.ErrSlotConflict},
		{calendar.ErrOutsideBusinessHours, application.ErrOutsideBusinessHours},
		{calendar.ErrSlotInPast, application.ErrSlotInPast},
		{calendar.ErrServiceNotFound, application.ErrServiceNotFound},
		{calendar.ErrPersonNotFound, application.ErrPersonNotFound},
		{calendar.ErrAppointmentNotFound, application.ErrAppointmentNotFound},
		{calendar.ErrNoFieldsToUpdate, application.ErrNoFieldsToUpdate},
	}
	for _, tc := range cases {
		if got := mapCalendarErr(tc.in); !errors.Is(got, tc.want) {
			t.Fatalf("mapCalendarErr(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNilReaderGuards(t *testing.T) {
	if cfg, err := NewOrgAdapter(nil).GetAssistantConfig(context.Background()); err != nil || cfg.AiEnabled {
		t.Fatalf("expected zero OrgConfig for nil reader, got cfg=%+v err=%v", cfg, err)
	}
	if items, err := NewCatalogAdapter(nil).ListCatalogServices(context.Background()); err != nil || len(items) != 0 {
		t.Fatalf("expected empty services for nil reader, got items=%+v err=%v", items, err)
	}
}
