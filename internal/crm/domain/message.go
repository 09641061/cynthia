package domain

import (
	"strings"
	"time"
)

const (
	MessageDirectionInbound  = "inbound"
	MessageDirectionOutbound = "outbound"

	MessageStatusReceived  = "received"
	MessageStatusPending   = "pending"
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusFailed    = "failed"
	MessageStatusUncertain = "uncertain"
)

// Message represents a single inbound or outbound message in a conversation.
type Message struct {
	ID             string
	ConversationID string
	ExternalID     *string
	IdempotencyKey *string
	Direction      string
	Content        string
	Status         string
	CreatedAt      time.Time
}

// ValidateMessageContent trims and validates message body text.
func ValidateMessageContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", InvalidFieldError{
			Field:  "content",
			Reason: "must not be blank",
		}
	}
	if len([]rune(content)) > 4096 {
		return "", InvalidFieldError{
			Field:  "content",
			Reason: "must not exceed 4096 characters",
		}
	}
	return content, nil
}
