package domain_test

import (
	"errors"
	"testing"

	"go-service/internal/organizations/domain"
)

func validBaseOrg() domain.Organization {
	day := domain.DayHours{IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}
	return domain.Organization{
		Name:     "Acme Clinic",
		Timezone: "UTC",
		BusinessHours: domain.BusinessHours{
			Monday: day, Tuesday: day, Wednesday: day, Thursday: day,
			Friday: day, Saturday: day, Sunday: day,
		},
	}
}

func TestApplyPatchWhatsAppValidation(t *testing.T) {
	phone := " +57 (300) 123-4567 "
	phoneID := " 109876543210 "
	enabled := true
	org, err := domain.ApplyPatch(validBaseOrg(), domain.UpdatePatch{
		WhatsAppPhoneNumber:   &phone,
		WhatsAppPhoneNumberID: &phoneID,
		WhatsAppEnabled:       &enabled,
	})
	if err != nil {
		t.Fatalf("expected valid whatsapp config, got %v", err)
	}
	if org.WhatsAppPhoneNumber != "+573001234567" {
		t.Fatalf("expected stripped E.164 phone, got %q", org.WhatsAppPhoneNumber)
	}
	if org.WhatsAppPhoneNumberID != "109876543210" || !org.WhatsAppEnabled {
		t.Fatalf("unexpected whatsapp fields: %+v", org)
	}

	badPhone := "3001234567"
	if _, err := domain.ApplyPatch(validBaseOrg(), domain.UpdatePatch{WhatsAppPhoneNumber: &badPhone}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for non-E.164 phone, got %v", err)
	}

	badID := "abc123"
	if _, err := domain.ApplyPatch(validBaseOrg(), domain.UpdatePatch{WhatsAppPhoneNumberID: &badID}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for non-numeric phone_number_id, got %v", err)
	}

	if _, err := domain.ApplyPatch(validBaseOrg(), domain.UpdatePatch{WhatsAppEnabled: &enabled}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when enabling without phone and phone_number_id, got %v", err)
	}
}
