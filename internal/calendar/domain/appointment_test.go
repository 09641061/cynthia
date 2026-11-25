package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar/domain"
)

func TestNewAppointmentDefaultsAndValidation(t *testing.T) {
	start := time.Date(2026, 11, 25, 10, 0, 0, 0, time.FixedZone("COT", -5*3600))
	end := start.Add(45 * time.Minute)
	appt, err := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: uuid.NewString(), CustomerID: uuid.NewString(), ServiceID: uuid.NewString(),
		StartsAt: start, EndsAt: end,
	})
	if err != nil {
		t.Fatalf("NewAppointment failed: %v", err)
	}
	if appt.Status != domain.StatusPending || appt.Channel != domain.ChannelManual || appt.Version != 1 {
		t.Fatalf("unexpected defaults: %+v", appt)
	}
	if appt.StartsAt.Location() != time.UTC || appt.EndsAt.Location() != time.UTC {
		t.Fatal("expected UTC timestamps")
	}

	cases := []struct {
		name  string
		input domain.CreateInput
		field string
	}{
		{"blank id", domain.CreateInput{PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: end}, "id"},
		{"blank person", domain.CreateInput{ID: "i", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: end}, "person_id"},
		{"end before start", domain.CreateInput{ID: "i", PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: end, EndsAt: start}, "ends_at"},
		{"equal times", domain.CreateInput{ID: "i", PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: start}, "ends_at"},
		{"terminal initial status", domain.CreateInput{ID: "i", PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: end, Status: domain.StatusCancelled}, "status"},
		{"invalid channel", domain.CreateInput{ID: "i", PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: end, Channel: "sms"}, "channel"},
		{"notes too long", domain.CreateInput{ID: "i", PersonID: "p", CustomerID: "c", ServiceID: "s", StartsAt: start, EndsAt: end, Notes: strings.Repeat("a", 1001)}, "notes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewAppointment(tc.input)
			var fieldErr domain.InvalidFieldError
			if !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
				t.Fatalf("got err %v, want InvalidFieldError(%s)", err, tc.field)
			}
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("expected ErrInvalid unwrap, got %v", err)
			}
		})
	}
}

func TestCanTransitionAndApplyPatchAtomicity(t *testing.T) {
	if !domain.CanTransition(domain.StatusPending, domain.StatusConfirmed) {
		t.Fatal("pending -> confirmed must be allowed")
	}
	if domain.CanTransition(domain.StatusConfirmed, domain.StatusPending) {
		t.Fatal("confirmed -> pending must be rejected")
	}
	if domain.CanTransition(domain.StatusCancelled, domain.StatusConfirmed) {
		t.Fatal("cancelled -> confirmed must be rejected")
	}
	if !domain.StatusPending.IsBlocking() || domain.StatusCancelled.IsBlocking() {
		t.Fatal("unexpected IsBlocking result")
	}

	start := time.Date(2026, 11, 25, 14, 0, 0, 0, time.UTC)
	orig, err := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: uuid.NewString(), CustomerID: uuid.NewString(), ServiceID: uuid.NewString(),
		StartsAt: start, EndsAt: start.Add(time.Hour), Status: domain.StatusConfirmed, Channel: domain.ChannelWhatsApp, Notes: "initial",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	badStatus := domain.StatusPending
	badNotes := "mutated"
	_, err = orig.ApplyPatch(domain.Patch{Status: &badStatus, Notes: &badNotes})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if orig.Notes != "initial" || orig.Status != domain.StatusConfirmed {
		t.Fatalf("original appointment mutated on failed patch: %+v", orig)
	}

	tooLongNotes := strings.Repeat("a", 1001)
	_, err = orig.ApplyPatch(domain.Patch{Notes: &tooLongNotes})
	var fieldErr domain.InvalidFieldError
	if !errors.As(err, &fieldErr) || fieldErr.Field != "notes" || !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("got err %v, want InvalidFieldError(notes)", err)
	}
	if orig.Notes != "initial" {
		t.Fatalf("original notes mutated on failed patch: %q", orig.Notes)
	}

	goodStatus := domain.StatusCompleted
	updated, err := orig.ApplyPatch(domain.Patch{Status: &goodStatus})
	if err != nil {
		t.Fatalf("valid ApplyPatch failed: %v", err)
	}
	if updated.Status != domain.StatusCompleted {
		t.Fatalf("got status %s, want completed", updated.Status)
	}
}
