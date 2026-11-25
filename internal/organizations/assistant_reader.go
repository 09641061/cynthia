package organizations

import (
	"context"

	"go-service/internal/organizations/domain"
	"go-service/internal/organizations/infrastructure/persistence/postgres"
	sharedpostgres "go-service/internal/shared/postgres"
)

// AssistantConfig exposes the organization settings needed by the WhatsApp assistant.
type AssistantConfig struct {
	AiEnabled       bool
	AiInstructions  string
	Timezone        string
	CurrencyCode    string
	BusinessHours   domain.BusinessHours
	WhatsAppEnabled bool
}

// AssistantConfigReader reads the singleton organization's assistant configuration.
type AssistantConfigReader struct {
	store *postgres.Store
	hasDB bool
}

// NewAssistantConfigReader creates a reader for the singleton organization's assistant settings.
func NewAssistantConfigReader(db *sharedpostgres.Client) *AssistantConfigReader {
	if db == nil {
		return &AssistantConfigReader{}
	}
	return &AssistantConfigReader{store: postgres.NewStore(db), hasDB: true}
}

// GetAssistantConfig returns the organization's assistant configuration.
func (r *AssistantConfigReader) GetAssistantConfig(ctx context.Context) (AssistantConfig, error) {
	if !r.hasDB {
		return AssistantConfig{}, nil
	}
	org, err := r.store.Get(ctx)
	if err != nil {
		return AssistantConfig{}, err
	}
	return AssistantConfig{
		AiEnabled:       org.AiEnabled,
		AiInstructions:  org.AiInstructions,
		Timezone:        org.Timezone,
		CurrencyCode:    org.CurrencyCode,
		BusinessHours:   org.BusinessHours,
		WhatsAppEnabled: org.WhatsAppEnabled,
	}, nil
}
