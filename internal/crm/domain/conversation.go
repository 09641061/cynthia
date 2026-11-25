package domain

import (
	"strings"
	"time"
)

const (
	ConversationChannelWhatsApp = "whatsapp"
	ConversationStatusOpen      = "open"
	ConversationStatusClosed    = "closed"
)

// Conversation represents a messaging thread with a CRM customer.
type Conversation struct {
	ID                 string
	CustomerID         string
	Channel            string
	Status             string
	LastMessageAt      *time.Time
	LastMessagePreview *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ValidateConversationStatus trims and validates a conversation status.
func ValidateConversationStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	if status != ConversationStatusOpen && status != ConversationStatusClosed {
		return "", InvalidFieldError{
			Field:  "status",
			Reason: "must be open or closed",
		}
	}
	return status, nil
}

// MessagePreview returns a truncated single-line preview of message content.
func MessagePreview(content string) string {
	trimmed := strings.TrimSpace(content)
	runes := []rune(trimmed)
	if len(runes) > 160 {
		return string(runes[:160])
	}
	return trimmed
}
