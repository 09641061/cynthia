package application

import (
	"context"
	"log/slog"
	"strings"

	"go-service/internal/crm/domain"
)

// SendAssistantReply delivers an outbound assistant reply without auto-pausing the conversation.
func (s *MessageService) SendAssistantReply(ctx context.Context, convID, idempotencyKey, rawContent string) (domain.Message, error) {
	msg, _, err := s.sendOutboundCore(ctx, convID, idempotencyKey, rawContent)
	return msg, err
}

func (s *MessageService) pauseAfterStaffReply(ctx context.Context, convID string) {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deliveryPersistTimeout)
	defer cancel()
	if _, err := s.conversations.SetConversationAIPaused(persistCtx, convID, true, "staff_replied"); err != nil {
		slog.WarnContext(ctx, "failed to pause AI after staff reply", "conversation_id", convID, "error", err)
		return
	}
	if s.events != nil {
		s.events.Publish(Event{Type: EventConversationUpdated, ConversationID: convID})
	}
}

func replayOutbound(existing domain.Message) (domain.Message, error) {
	switch existing.Status {
	case domain.MessageStatusFailed:
		return existing, DeliveryError{Message: existing, Reason: "WhatsApp delivery failed."}
	case domain.MessageStatusUncertain:
		return existing, DeliveryError{Message: existing, Reason: "WhatsApp delivery status is uncertain."}
	case domain.MessageStatusPending:
		return domain.Message{}, ErrConflict
	default:
		return existing, nil
	}
}

func (s *MessageService) resolveWhatsAppConfig(ctx context.Context) (string, bool, error) {
	if s.configReader == nil {
		return "", false, nil
	}
	phoneID, enabled, err := s.configReader.GetWhatsAppConfig(ctx)
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(phoneID), enabled, nil
}
