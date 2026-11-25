//go:build integration

package integration_test

import (
	"testing"

	"go-service/internal/organizations"
	"go-service/internal/organizations/domain"
	orgpostgres "go-service/internal/organizations/infrastructure/persistence/postgres"
)

func TestAssistantConfigReaderIntegration(t *testing.T) {
	client := newTestOrgClient(t)
	store := orgpostgres.NewStore(client)
	reader := organizations.NewAssistantConfigReader(client)
	ctx := t.Context()

	cfg, err := reader.GetAssistantConfig(ctx)
	if err != nil {
		t.Fatalf("read default assistant config: %v", err)
	}
	if cfg.AiEnabled {
		t.Fatalf("expected default ai_enabled=false")
	}

	org, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get organization: %v", err)
	}
	enabled := true
	instructions := "Always greet politely."
	if _, err := store.Update(ctx, org.Version, domain.UpdatePatch{
		AiEnabled:      &enabled,
		AiInstructions: &instructions,
	}); err != nil {
		t.Fatalf("update organization ai_enabled: %v", err)
	}

	updated, err := reader.GetAssistantConfig(ctx)
	if err != nil {
		t.Fatalf("read updated assistant config: %v", err)
	}
	if !updated.AiEnabled || updated.AiInstructions != instructions {
		t.Fatalf("expected ai_enabled=true and instructions=%q, got %+v", instructions, updated)
	}
}
