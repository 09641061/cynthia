package crm

import (
	"context"
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/infrastructure/persistence/postgres"
)

// ConversationSnapshot exposes conversation state to the assistant module without leaking CRM domain types.
type ConversationSnapshot struct {
	ID            string
	CustomerID    string
	Status        string
	AiPaused      bool
	HandoffReason string
}

// MessageSnapshot exposes a conversation message to the assistant module without leaking CRM domain types.
type MessageSnapshot struct {
	ID        string
	Direction string
	Content   string
	Status    string
	CreatedAt time.Time
}

// AssistantCRMGateway provides the assistant module with controlled access to CRM conversations and messaging.
type AssistantCRMGateway struct {
	store         *postgres.Store
	broker        *application.EventBroker
	conversations *application.ConversationService
	messages      *application.MessageService
}

// GetConversation returns the current snapshot of a conversation.
func (g *AssistantCRMGateway) GetConversation(ctx context.Context, convID string) (ConversationSnapshot, error) {
	conv, err := g.store.GetConversation(ctx, convID)
	if err != nil {
		return ConversationSnapshot{}, err
	}
	return ConversationSnapshot{
		ID:            conv.ID,
		CustomerID:    conv.CustomerID,
		Status:        conv.Status,
		AiPaused:      conv.AiPaused,
		HandoffReason: conv.HandoffReason,
	}, nil
}

// ListRecentMessages returns up to limit recent messages ordered oldest-to-newest (ASC).
func (g *AssistantCRMGateway) ListRecentMessages(ctx context.Context, convID string, limit int) ([]MessageSnapshot, error) {
	items, err := g.conversations.ListMessages(ctx, convID, limit, nil)
	if err != nil {
		return nil, err
	}
	out := make([]MessageSnapshot, 0, len(items))
	for _, m := range items {
		out = append(out, MessageSnapshot{
			ID:        m.ID,
			Direction: m.Direction,
			Content:   m.Content,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

// SendAssistantReply delivers an outbound assistant reply without auto-pausing AI.
func (g *AssistantCRMGateway) SendAssistantReply(ctx context.Context, convID, idempotencyKey, content string) (MessageSnapshot, error) {
	msg, err := g.messages.SendAssistantReply(ctx, convID, idempotencyKey, content)
	if err != nil {
		return MessageSnapshot{}, err
	}
	return MessageSnapshot{
		ID:        msg.ID,
		Direction: msg.Direction,
		Content:   msg.Content,
		Status:    msg.Status,
		CreatedAt: msg.CreatedAt,
	}, nil
}

// PauseForHandoff pauses AI on a conversation for human handoff and publishes a conversation.updated event.
func (g *AssistantCRMGateway) PauseForHandoff(ctx context.Context, convID, reason string) error {
	if _, err := g.store.SetConversationAIPaused(ctx, convID, true, reason); err != nil {
		return err
	}
	if g.broker != nil {
		g.broker.Publish(application.Event{Type: application.EventConversationUpdated, ConversationID: convID})
	}
	return nil
}
