package crm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
	"go-service/internal/crm/infrastructure/persistence/postgres"
)

// Root sentinel errors returned by AssistantCRMGateway.
var (
	ErrConversationNotFound = errors.New("crm conversation not found")
	ErrConversationClosed   = errors.New("crm conversation closed or conflicted")
	ErrDeliveryFailed       = errors.New("crm message delivery failed")
	ErrDeliveryUncertain    = errors.New("crm message delivery uncertain")
	ErrGatewayUnavailable   = errors.New("crm assistant gateway unavailable")
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
	hasDB         bool
}

func (g *AssistantCRMGateway) available() bool { return g != nil && g.hasDB }

func toMessageSnapshot(m domain.Message) MessageSnapshot {
	return MessageSnapshot{
		ID: m.ID, Direction: m.Direction, Content: m.Content, Status: m.Status, CreatedAt: m.CreatedAt,
	}
}

func translateGatewayErr(err error) error {
	if err == nil {
		return nil
	}
	var dErr application.DeliveryError
	if errors.As(err, &dErr) {
		if dErr.Message.Status == domain.MessageStatusUncertain {
			return fmt.Errorf("%w: %w", ErrDeliveryUncertain, err)
		}
		return fmt.Errorf("%w: %w", ErrDeliveryFailed, err)
	}
	if errors.Is(err, application.ErrNotFound) {
		return fmt.Errorf("%w: %w", ErrConversationNotFound, err)
	}
	if errors.Is(err, application.ErrConflict) {
		return fmt.Errorf("%w: %w", ErrConversationClosed, err)
	}
	return err
}

// GetConversation returns the current snapshot of a conversation.
func (g *AssistantCRMGateway) GetConversation(ctx context.Context, convID string) (ConversationSnapshot, error) {
	if !g.available() {
		return ConversationSnapshot{}, ErrGatewayUnavailable
	}
	conv, err := g.store.GetConversation(ctx, convID)
	if err != nil {
		return ConversationSnapshot{}, translateGatewayErr(err)
	}
	return ConversationSnapshot{
		ID: conv.ID, CustomerID: conv.CustomerID, Status: conv.Status,
		AiPaused: conv.AiPaused, HandoffReason: conv.HandoffReason,
	}, nil
}

// ListRecentMessages returns up to limit recent messages ordered oldest-to-newest (ASC).
func (g *AssistantCRMGateway) ListRecentMessages(ctx context.Context, convID string, limit int) ([]MessageSnapshot, error) {
	if !g.available() {
		return nil, ErrGatewayUnavailable
	}
	items, err := g.conversations.ListMessages(ctx, convID, limit, nil)
	if err != nil {
		return nil, translateGatewayErr(err)
	}
	out := make([]MessageSnapshot, 0, len(items))
	for _, m := range items {
		out = append(out, toMessageSnapshot(m))
	}
	return out, nil
}

// SendAssistantReply delivers an outbound assistant reply without auto-pausing AI.
func (g *AssistantCRMGateway) SendAssistantReply(ctx context.Context, convID, idempotencyKey, content string) (MessageSnapshot, error) {
	if !g.available() {
		return MessageSnapshot{}, ErrGatewayUnavailable
	}
	msg, err := g.messages.SendAssistantReply(ctx, convID, idempotencyKey, content)
	if err != nil {
		var dErr application.DeliveryError
		if errors.As(err, &dErr) {
			return toMessageSnapshot(dErr.Message), translateGatewayErr(err)
		}
		return MessageSnapshot{}, translateGatewayErr(err)
	}
	return toMessageSnapshot(msg), nil
}

// PauseForHandoff pauses AI on a conversation for human handoff and publishes a conversation.updated event.
func (g *AssistantCRMGateway) PauseForHandoff(ctx context.Context, convID, reason string) error {
	if !g.available() {
		return ErrGatewayUnavailable
	}
	if _, err := g.store.SetConversationAIPaused(ctx, convID, true, reason); err != nil {
		return translateGatewayErr(err)
	}
	if g.broker != nil {
		g.broker.Publish(application.Event{Type: application.EventConversationUpdated, ConversationID: convID})
	}
	return nil
}
