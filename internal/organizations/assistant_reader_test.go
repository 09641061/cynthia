package organizations

import (
	"context"
	"testing"
)

func TestAssistantConfigReaderNilDB(t *testing.T) {
	cfg, err := NewAssistantConfigReader(nil).GetAssistantConfig(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for nil db, got %v", err)
	}
	if cfg.AiEnabled || cfg.AiInstructions != "" || cfg.Timezone != "" || cfg.CurrencyCode != "" || cfg.WhatsAppEnabled {
		t.Fatalf("expected zero AssistantConfig, got %+v", cfg)
	}
}
