package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func baseOrg() Organization {
	openDay := DayHours{IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}
	closedDay := DayHours{IsOpen: false, OpenTime: "09:00", CloseTime: "18:00"}
	return Organization{
		ID:           uuid.NewString(),
		Name:         "Takodu",
		BusinessType: "Salon",
		ContactEmail: "hello@takodu.com",
		Timezone:     "UTC",
		CurrencyCode: "USD",
		BusinessHours: BusinessHours{
			Monday: openDay, Tuesday: openDay, Wednesday: openDay,
			Thursday: openDay, Friday: openDay, Saturday: closedDay, Sunday: closedDay,
		},
		AiInstructions: "Be polite.",
	}
}

func TestApplyPatchValid(t *testing.T) {
	name := "  New Salon  "
	tz := "America/New_York"
	email := ""
	currency := "COP"
	updated, err := ApplyPatch(baseOrg(), UpdatePatch{
		Name:         &name,
		Timezone:     &tz,
		ContactEmail: &email,
		CurrencyCode: &currency,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Salon" {
		t.Fatalf("expected trimmed name, got %q", updated.Name)
	}
	if updated.Timezone != "America/New_York" || updated.ContactEmail != "" || updated.CurrencyCode != "COP" {
		t.Fatalf("unexpected fields: %+v", updated)
	}
}

func TestApplyPatchRejectsInvalidFields(t *testing.T) {
	blank := "   "
	badEmail := "not-an-email"
	badTZ := "Mars/Olympus"
	tooLong := strings.Repeat("a", 4001)
	badHours := baseOrg().BusinessHours
	badHours.Monday = DayHours{IsOpen: true, OpenTime: "18:00", CloseTime: "09:00"}
	emptyCurrency := ""
	lowerCurrency := "usd"
	shortCurrency := "US"
	spacedCurrency := " USD "
	digitCurrency := "U1D"
	unicodeCurrency := "ÜSD"

	cases := []struct {
		name  string
		patch UpdatePatch
		field string
	}{
		{"blank name", UpdatePatch{Name: &blank}, "name"},
		{"invalid email", UpdatePatch{ContactEmail: &badEmail}, "contact_email"},
		{"invalid timezone", UpdatePatch{Timezone: &badTZ}, "timezone"},
		{"empty currency", UpdatePatch{CurrencyCode: &emptyCurrency}, "currency_code"},
		{"lowercase currency", UpdatePatch{CurrencyCode: &lowerCurrency}, "currency_code"},
		{"short currency", UpdatePatch{CurrencyCode: &shortCurrency}, "currency_code"},
		{"spaced currency", UpdatePatch{CurrencyCode: &spacedCurrency}, "currency_code"},
		{"digit currency", UpdatePatch{CurrencyCode: &digitCurrency}, "currency_code"},
		{"unicode currency", UpdatePatch{CurrencyCode: &unicodeCurrency}, "currency_code"},
		{"instructions too long", UpdatePatch{AiInstructions: &tooLong}, "ai_instructions"},
		{"inverted hours", UpdatePatch{BusinessHours: &badHours}, "business_hours.monday"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyPatch(baseOrg(), tc.patch)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
			var fieldErr InvalidFieldError
			if !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
				t.Fatalf("expected field %q, got %+v", tc.field, err)
			}
		})
	}
}
