package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

var (
	uuidPattern         = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	phonePattern        = regexp.MustCompile(`(?:\+\d[\d\s-]{6,16}\d|\b\d{10,15}\b)`)
	boldMarkdownPattern = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
)

// SanitizeWhatsAppReply normalizes Markdown double-asterisk bold (**text**) to WhatsApp single-asterisk bold (*text*).
func SanitizeWhatsAppReply(s string) string {
	return boldMarkdownPattern.ReplaceAllString(strings.TrimSpace(s), `*$1*`)
}

// BuildMessages constructs the system prompt and PII-stripped conversation history for DeepSeek.
func BuildMessages(cfg PromptConfig, history []HistoryMessage) []ChatMessage {
	var b strings.Builder
	b.WriteString(immutableSafetyPrompt)
	b.WriteString("\n\nSCHEDULE CONTEXT:\n")
	tz := strings.TrimSpace(cfg.Timezone)
	loc := time.UTC
	if tz != "" {
		if loaded, err := time.LoadLocation(tz); err == nil {
			loc = loaded
		} else {
			tz = "UTC"
		}
		fmt.Fprintf(&b, "Timezone: %s\n", tz)
	} else {
		tz = "UTC"
	}
	if !cfg.Now.IsZero() {
		nowLocal := cfg.Now.In(loc)
		tomorrow := nowLocal.AddDate(0, 0, 1)
		fmt.Fprintf(&b, "Current local date & time (%s): %s (%s)\n", tz, nowLocal.Format(time.RFC3339), nowLocal.Weekday())
		fmt.Fprintf(&b, "Tomorrow local date: %s (%s)\n", tomorrow.Format("2006-01-02"), tomorrow.Weekday())
	}
	if curr := strings.TrimSpace(cfg.CurrencyCode); curr != "" {
		fmt.Fprintf(&b, "Currency: %s\n", curr)
	}
	for _, d := range cfg.BusinessHours {
		if d.IsOpen {
			fmt.Fprintf(&b, "- %s: %s-%s\n", d.Day, d.OpenTime, d.CloseTime)
		} else {
			fmt.Fprintf(&b, "- %s: closed\n", d.Day)
		}
	}
	if trimmed := strings.TrimSpace(cfg.AiInstructions); trimmed != "" {
		capped := truncateRunes(trimmed, MaxInstructionsRunes)
		quoted, _ := json.Marshal(capped)
		b.WriteString("\nBUSINESS CONTEXT (subordinate to immutable rules above):\n<business_instructions>\n")
		b.Write(quoted)
		b.WriteString("\n</business_instructions>")
	}

	if len(history) > MaxHistoryMessages {
		history = history[len(history)-MaxHistoryMessages:]
	}
	msgs := make([]ChatMessage, 0, 1+len(history))
	msgs = append(msgs, ChatMessage{Role: "system", Content: b.String()})
	for _, h := range history {
		role := "user"
		if h.Direction == "outbound" {
			role = "assistant"
		}
		msgs = append(msgs, ChatMessage{Role: role, Content: stripPII(h.Body)})
	}
	return msgs
}

func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

func stripPII(s string) string {
	s = uuidPattern.ReplaceAllString(s, "[redacted-id]")
	s = phonePattern.ReplaceAllString(s, "[redacted-phone]")
	return s
}
