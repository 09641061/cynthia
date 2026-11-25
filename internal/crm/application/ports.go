package application

import (
	"context"
	"time"

	"go-service/internal/crm/domain"
)

// ConversationFilter constrains conversation listing queries.
type ConversationFilter struct {
	Query      string
	Status     string
	CustomerID string
	Page       int
	PageSize   int
}

// ConversationPage represents a paginated list of conversations.
type ConversationPage struct {
	Items    []domain.Conversation
	Total    int
	Page     int
	PageSize int
}

// ConversationStore defines persistence operations for CRM conversations.
type ConversationStore interface {
	ListConversations(ctx context.Context, filter ConversationFilter) ([]domain.Conversation, int, error)
	GetConversation(ctx context.Context, id string) (domain.Conversation, error)
	UpdateConversationStatus(ctx context.Context, id, status string) (domain.Conversation, error)
	EnsureOpenConversation(ctx context.Context, customerID string) (domain.Conversation, error)
}

// MessageStore defines persistence operations for CRM messages.
type MessageStore interface {
	ListMessages(ctx context.Context, convID string, limit int, before *time.Time) ([]domain.Message, error)
	GetMessageByIdempotencyKey(ctx context.Context, convID, key string) (domain.Message, error)
	GetMessageByExternalID(ctx context.Context, externalID string) (domain.Message, error)
	InsertOutboundPending(ctx context.Context, msg domain.Message) (domain.Message, error)
	UpdateMessageDelivery(ctx context.Context, id, status string, externalID *string, preview string) (domain.Message, error)
	InsertInboundMessage(ctx context.Context, msg domain.Message, preview string) (domain.Message, bool, error)
}

// InboundCustomerStore atomically resolves or creates a customer by phone.
type InboundCustomerStore interface {
	FindOrCreateCustomerByPhone(ctx context.Context, id, name, phone string) (domain.Customer, error)
}

// WhatsAppSender delivers outbound text messages via Meta WhatsApp Cloud API.
type WhatsAppSender interface {
	SendText(ctx context.Context, phoneNumberID, toPhone, text string) (externalID string, uncertain bool, err error)
}

// WebhookVerifier validates Meta WhatsApp webhook challenges and request signatures.
type WebhookVerifier interface {
	VerifyChallenge(mode, token, challenge string) (string, bool)
	VerifySignature(signatureHeader string, body []byte) bool
}

// WhatsAppConfigReader reads the organization's WhatsApp configuration.
type WhatsAppConfigReader interface {
	GetWhatsAppConfig(ctx context.Context) (phoneNumberID string, enabled bool, err error)
}
