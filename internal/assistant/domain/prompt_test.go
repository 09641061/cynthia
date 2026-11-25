package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildMessagesSafetyInstructionsAndHistoryCap(t *testing.T) {
	longInstr := strings.Repeat("á", MaxInstructionsRunes+50)
	secretID := uuid.NewString()
	history := make([]HistoryMessage, 0, 25)
	for i := 0; i < 25; i++ {
		dir := "inbound"
		if i%2 == 1 {
			dir = "outbound"
		}
		history = append(history, HistoryMessage{
			Direction: dir,
			Body:      "Call +573001112233 ref " + secretID,
		})
	}
	msgs := BuildMessages(PromptConfig{
		Timezone:       "America/Bogota",
		BusinessHours:  []DayHours{{Day: "Monday", IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}},
		AiInstructions: longInstr,
		Now:            time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC),
	}, history)

	if len(msgs) != 1+MaxHistoryMessages {
		t.Fatalf("expected %d messages, got %d", 1+MaxHistoryMessages, len(msgs))
	}
	sys := msgs[0].Content
	if !strings.Contains(sys, "America/Bogota") || !strings.Contains(sys, "Monday: 09:00-18:00") {
		t.Fatalf("expected timezone and business hours in system prompt: %s", sys)
	}
	if !strings.Contains(sys, "<business_instructions>") {
		t.Fatalf("expected fenced business instructions block")
	}
	if strings.Count(sys, "á") != MaxInstructionsRunes {
		t.Fatalf("expected instructions capped at %d runes, got %d", MaxInstructionsRunes, strings.Count(sys, "á"))
	}
	for _, m := range msgs[1:] {
		if strings.Contains(m.Content, "+573001112233") || strings.Contains(m.Content, secretID) {
			t.Fatalf("expected PII stripped from history message: %q", m.Content)
		}
	}
}

func TestBuildMessagesOmitsBlankInstructions(t *testing.T) {
	msgs := BuildMessages(PromptConfig{Timezone: "UTC", AiInstructions: "   \n\t "}, nil)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if strings.Contains(msgs[0].Content, "<business_instructions>") {
		t.Fatalf("expected blank instructions omitted")
	}
}
