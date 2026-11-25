//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"go-service/internal/organizations/domain"
)

func TestOrganizationStoreGetAndUpdate(t *testing.T) {
	store := newTestOrgStore(t)
	ctx := t.Context()

	org, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get seeded organization: %v", err)
	}
	if org.Name != "Takodu" || org.CurrencyCode != "USD" || !org.BusinessHours.Monday.IsOpen {
		t.Fatalf("unexpected seeded organization: %+v", org)
	}

	name := "Studio Nova"
	email := "hello@studionova.com"
	tz := "America/New_York"
	currency := "COP"
	instructions := "Confirm appointments 24h in advance."
	updated, err := store.Update(ctx, domain.UpdatePatch{
		Name: &name, ContactEmail: &email, Timezone: &tz, CurrencyCode: &currency, AiInstructions: &instructions,
	})
	if err != nil {
		t.Fatalf("update organization: %v", err)
	}
	if updated.Name != "Studio Nova" || updated.ContactEmail != email || updated.Timezone != tz || updated.CurrencyCode != "COP" {
		t.Fatalf("unexpected updated organization: %+v", updated)
	}

	reloaded, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("reload organization: %v", err)
	}
	if reloaded.Name != "Studio Nova" || reloaded.CurrencyCode != "COP" || reloaded.AiInstructions != instructions {
		t.Fatalf("persisted state mismatch: %+v", reloaded)
	}

	blank := "   "
	if _, err := store.Update(ctx, domain.UpdatePatch{Name: &blank}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on blank name, got %v", err)
	}
	badCurrency := "usd"
	if _, err := store.Update(ctx, domain.UpdatePatch{CurrencyCode: &badCurrency}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on invalid currency_code, got %v", err)
	}

	afterReject, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get after rejected updates: %v", err)
	}
	if afterReject.Name != "Studio Nova" || afterReject.CurrencyCode != "COP" {
		t.Fatalf("rejected updates mutated persisted state: %+v", afterReject)
	}
}
