//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"go-service/internal/organizations/application"
	"go-service/internal/organizations/domain"
)

func TestOrganizationStoreGetUpdateAndPrecondition(t *testing.T) {
	store := newTestOrgStore(t)
	ctx := t.Context()

	org, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get seeded organization: %v", err)
	}
	if org.Name != "Takodu" || org.CurrencyCode != "USD" || org.Version != 1 || !org.BusinessHours.Monday.IsOpen {
		t.Fatalf("unexpected seeded organization: %+v", org)
	}

	name := "Studio Nova"
	email := "hello@studionova.com"
	tz := "America/New_York"
	currency := "COP"
	instructions := "Confirm appointments 24h in advance."
	updated, err := store.Update(ctx, org.Version, domain.UpdatePatch{
		Name: &name, ContactEmail: &email, Timezone: &tz, CurrencyCode: &currency, AiInstructions: &instructions,
	})
	if err != nil {
		t.Fatalf("update organization: %v", err)
	}
	if updated.Version != 2 || updated.Name != "Studio Nova" || updated.ContactEmail != email || updated.Timezone != tz || updated.CurrencyCode != "COP" {
		t.Fatalf("unexpected updated organization: %+v", updated)
	}

	reloaded, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("reload organization: %v", err)
	}
	if reloaded.Version != 2 || reloaded.Name != "Studio Nova" || reloaded.CurrencyCode != "COP" || reloaded.AiInstructions != instructions {
		t.Fatalf("persisted state mismatch: %+v", reloaded)
	}

	staleName := "Stale"
	if _, err := store.Update(ctx, 1, domain.UpdatePatch{Name: &staleName}); !errors.Is(err, application.ErrPreconditionFailed) {
		t.Fatalf("expected ErrPreconditionFailed on stale version, got %v", err)
	}
	blank := "   "
	if _, err := store.Update(ctx, 2, domain.UpdatePatch{Name: &blank}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on blank name, got %v", err)
	}
	badCurrency := "usd"
	if _, err := store.Update(ctx, 2, domain.UpdatePatch{CurrencyCode: &badCurrency}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on invalid currency_code, got %v", err)
	}

	afterReject, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get after rejected updates: %v", err)
	}
	if afterReject.Version != 2 || afterReject.Name != "Studio Nova" || afterReject.CurrencyCode != "COP" {
		t.Fatalf("rejected updates mutated persisted state: %+v", afterReject)
	}
}
