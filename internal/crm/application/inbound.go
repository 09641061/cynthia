package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

// ReceiveInbound processes an inbound WhatsApp message: auto-registers customer, opens/reopens conversation, and deduplicates by external_id.
func (s *MessageService) ReceiveInbound(ctx context.Context, externalID, rawPhone, profileName, rawContent string) (domain.Message, error) {
	extID := strings.TrimSpace(externalID)
	if extID == "" {
		return domain.Message{}, domain.InvalidFieldError{Field: "external_id", Reason: "must not be blank"}
	}
	if existing, err := s.messages.GetMessageByExternalID(ctx, extID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return domain.Message{}, err
	}
	phoneInput := strings.TrimSpace(rawPhone)
	if phoneInput != "" && !strings.HasPrefix(phoneInput, "+") {
		phoneInput = "+" + phoneInput
	}
	phone, err := domain.NormalizePhone(phoneInput)
	if err != nil {
		return domain.Message{}, err
	}
	content, err := domain.ValidateMessageContent(rawContent)
	if err != nil {
		return domain.Message{}, err
	}
	name := strings.TrimSpace(profileName)
	if name == "" {
		name = phone
	}
	customer, err := s.inboundUsers.FindOrCreateCustomerByPhone(ctx, uuid.NewString(), name, phone)
	if err != nil {
		return domain.Message{}, err
	}
	conv, err := s.conversations.EnsureOpenConversation(ctx, customer.ID)
	if err != nil {
		return domain.Message{}, err
	}
	msg, _, err := s.messages.InsertInboundMessage(ctx, domain.Message{
		ID:             uuid.NewString(),
		ConversationID: conv.ID,
		ExternalID:     &extID,
		Direction:      domain.MessageDirectionInbound,
		Content:        content,
		Status:         domain.MessageStatusReceived,
	}, domain.MessagePreview(content))
	return msg, err
}
