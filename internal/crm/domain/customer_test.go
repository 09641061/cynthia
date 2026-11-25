package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

func TestNormalizePhone(t *testing.T) {
	validCases := []struct {
		input string
		want  string
	}{
		{"+57 300 123 4567", "+573001234567"},
		{"  +1 (415) 555-2671  ", "+14155552671"},
		{"+1234567", "+1234567"},
		{"+123456789012345", "+123456789012345"},
	}
	for _, tc := range validCases {
		got, err := domain.NormalizePhone(tc.input)
		if err != nil {
			t.Fatalf("NormalizePhone(%q) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizePhone(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	invalidCases := []string{
		"",
		"   ",
		"573001234567",
		"+012345678",
		"+123456",
		"+1234567890123456",
		"+57-abc-1234",
	}
	for _, input := range invalidCases {
		_, err := domain.NormalizePhone(input)
		if !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("NormalizePhone(%q) error = %v, want ErrInvalid", input, err)
		}
		var fieldErr domain.InvalidFieldError
		if !errors.As(err, &fieldErr) || fieldErr.Field != "phone" {
			t.Fatalf("NormalizePhone(%q) error = %v, want InvalidFieldError(phone)", input, err)
		}
	}
}

func TestNewCustomer(t *testing.T) {
	id := uuid.NewString()
	customer, err := domain.NewCustomer(id, "  Ana Perez  ", "+57 (300) 123-4567")
	if err != nil {
		t.Fatalf("NewCustomer unexpected error: %v", err)
	}
	if customer.ID != id || customer.Name != "Ana Perez" || customer.Phone != "+573001234567" {
		t.Fatalf("unexpected customer: %+v", customer)
	}

	_, err = domain.NewCustomer(uuid.NewString(), "   ", "+573001234567")
	var fieldErr domain.InvalidFieldError
	if !errors.As(err, &fieldErr) || fieldErr.Field != "name" {
		t.Fatalf("expected InvalidFieldError(name), got %v", err)
	}
}
