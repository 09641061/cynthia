package domain_test

import (
	"errors"
	"strings"
	"testing"

	"go-service/internal/crm/domain"
)

func TestValidateConversationStatus(t *testing.T) {
	for _, valid := range []string{"open", " closed "} {
		got, err := domain.ValidateConversationStatus(valid)
		if err != nil {
			t.Fatalf("expected %q valid, got %v", valid, err)
		}
		if got != strings.TrimSpace(valid) {
			t.Fatalf("expected trimmed status, got %q", got)
		}
	}
	if _, err := domain.ValidateConversationStatus("archived"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestValidateMessageContentAndPreview(t *testing.T) {
	got, err := domain.ValidateMessageContent("  Hola mundo  ")
	if err != nil || got != "Hola mundo" {
		t.Fatalf("unexpected result: %q, %v", got, err)
	}
	if _, err := domain.ValidateMessageContent("   "); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for blank content, got %v", err)
	}
	long := strings.Repeat("a", 200)
	if preview := domain.MessagePreview(long); len([]rune(preview)) != 160 {
		t.Fatalf("expected 160 rune preview, got %d", len([]rune(preview)))
	}
}
