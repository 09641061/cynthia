package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

// MessageService orchestrates outbound and inbound WhatsApp messaging.
type MessageService struct {
	customers     CustomerStore
	inboundUsers  InboundCustomerStore
	conversations ConversationStore
	messages      MessageStore
	sender        WhatsAppSender
	configReader  WhatsAppConfigReader
	events        EventPublisher
}

// NewMessageService creates a MessageService with its required ports.
func NewMessageService(
	customers CustomerStore,
	inboundUsers InboundCustomerStore,
	conversations ConversationStore,
	messages MessageStore,
	sender WhatsAppSender,
	configReader WhatsAppConfigReader,
) *MessageService {
	return &MessageService{
		customers:     customers,
		inboundUsers:  inboundUsers,
		conversations: conversations,
		messages:      messages,
		sender:        sender,
		configReader:  configReader,
	}
}

// SendOutbound persists a pending outbound message, delivers it via Meta Cloud API, and records the outcome.
func (s *MessageService) SendOutbound(ctx context.Context, convID, idempotencyKey, rawContent string) (domain.Message, error) {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return domain.Message{}, domain.InvalidFieldError{Field: "idempotency_key", Reason: "must not be blank"}
	}
	content, err := domain.ValidateMessageContent(rawContent)
	if err != nil {
		return domain.Message{}, err
	}
	conv, err := s.conversations.GetConversation(ctx, convID)
	if err != nil {
		return domain.Message{}, err
	}
	if conv.Status == domain.ConversationStatusClosed {
		return domain.Message{}, ErrConflict
	}
	if existing, err := s.messages.GetMessageByIdempotencyKey(ctx, convID, key); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return domain.Message{}, err
	}
	customer, err := s.customers.GetCustomer(ctx, conv.CustomerID)
	if err != nil {
		return domain.Message{}, err
	}
	pending, err := s.messages.InsertOutboundPending(ctx, domain.Message{
		ID:             uuid.NewString(),
		ConversationID: convID,
		IdempotencyKey: &key,
		Direction:      domain.MessageDirectionOutbound,
		Content:        content,
		Status:         domain.MessageStatusPending,
	})
	if err != nil {
		if existing, getErr := s.messages.GetMessageByIdempotencyKey(ctx, convID, key); getErr == nil {
			return existing, nil
		}
		return domain.Message{}, err
	}
	return s.deliverOutbound(ctx, pending, customer.Phone)
}

func (s *MessageService) deliverOutbound(ctx context.Context, pending domain.Message, toPhone string) (domain.Message, error) {
	preview := domain.MessagePreview(pending.Content)
	phoneNumberID, enabled := s.resolveWhatsAppConfig(ctx)
	if !enabled || phoneNumberID == "" || s.sender == nil {
		failed, err := s.messages.UpdateMessageDelivery(ctx, pending.ID, domain.MessageStatusFailed, nil, preview)
		if err != nil {
			return domain.Message{}, err
		}
		return failed, DeliveryError{Message: failed, Reason: "WhatsApp integration is not configured."}
	}
	extID, uncertain, sendErr := s.sender.SendText(ctx, phoneNumberID, toPhone, pending.Content)
	if sendErr != nil {
		slog.ErrorContext(ctx, "whatsapp outbound delivery failed",
			"message_id", pending.ID, "conversation_id", pending.ConversationID, "uncertain", uncertain, "error", sendErr)
		status := domain.MessageStatusFailed
		reason := "WhatsApp delivery failed."
		if uncertain {
			status = domain.MessageStatusUncertain
			reason = "WhatsApp delivery status is uncertain."
		}
		updated, err := s.messages.UpdateMessageDelivery(ctx, pending.ID, status, nil, preview)
		if err != nil {
			return domain.Message{}, err
		}
		return updated, DeliveryError{Message: updated, Reason: reason}
	}
	var extPtr *string
	if extID != "" {
		extPtr = &extID
	}
	updated, err := s.messages.UpdateMessageDelivery(ctx, pending.ID, domain.MessageStatusSent, extPtr, preview)
	if err == nil {
		s.publishMessageCreated(updated.ConversationID)
	}
	return updated, err
}

func (s *MessageService) resolveWhatsAppConfig(ctx context.Context) (string, bool) {
	if s.configReader != nil {
		phoneID, enabled, err := s.configReader.GetWhatsAppConfig(ctx)
		if err == nil && phoneID != "" {
			return phoneID, enabled
		}
		if err == nil && !enabled {
			fallback := strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID"))
			if fallback != "" {
				return fallback, true
			}
			return "", false
		}
	}
	fallback := strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID"))
	return fallback, fallback != ""
}
