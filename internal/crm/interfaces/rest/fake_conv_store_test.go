package rest_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type stubConvStore struct {
	conv  domain.Conversation
	msg   domain.Message
	byExt map[string]domain.Message
}

func (s *stubConvStore) ListConversations(context.Context, application.ConversationFilter) ([]domain.Conversation, int, error) {
	return []domain.Conversation{s.conv}, 1, nil
}
func (s *stubConvStore) GetConversation(_ context.Context, id string) (domain.Conversation, error) {
	if s.conv.ID != "" && id != s.conv.ID {
		return domain.Conversation{}, application.ErrNotFound
	}
	return s.conv, nil
}
func (s *stubConvStore) UpdateConversationStatus(_ context.Context, _ string, expected int64, status string) (domain.Conversation, error) {
	if s.conv.Version != expected {
		return domain.Conversation{}, application.ErrPreconditionFailed
	}
	if s.conv.Status == domain.ConversationStatusClosed && status == domain.ConversationStatusOpen {
		s.conv.AiPaused = false
		s.conv.HandoffReason = ""
	}
	s.conv.Status = status
	s.conv.Version++
	return s.conv, nil
}
func (s *stubConvStore) UpdateConversationAI(_ context.Context, _ string, expected int64, aiPaused bool, reason string) (domain.Conversation, error) {
	if s.conv.Version != expected {
		return domain.Conversation{}, application.ErrPreconditionFailed
	}
	s.conv.AiPaused = aiPaused
	s.conv.HandoffReason = reason
	s.conv.Version++
	return s.conv, nil
}
func (s *stubConvStore) SetConversationAIPaused(_ context.Context, _ string, aiPaused bool, reason string) (domain.Conversation, error) {
	s.conv.AiPaused = aiPaused
	s.conv.HandoffReason = reason
	s.conv.Version++
	return s.conv, nil
}
func (s *stubConvStore) EnsureOpenConversation(context.Context, string) (domain.Conversation, error) {
	return s.conv, nil
}
func (s *stubConvStore) ListMessages(context.Context, string, int, *time.Time) ([]domain.Message, error) {
	return []domain.Message{s.msg}, nil
}
func (s *stubConvStore) GetMessageByIdempotencyKey(context.Context, string, string) (domain.Message, error) {
	return domain.Message{}, application.ErrNotFound
}
func (s *stubConvStore) GetMessageByExternalID(_ context.Context, extID string) (domain.Message, error) {
	if m, ok := s.byExt[extID]; ok {
		return m, nil
	}
	return domain.Message{}, application.ErrNotFound
}
func (s *stubConvStore) InsertOutboundPending(_ context.Context, m domain.Message) (domain.Message, error) {
	s.msg = m
	return m, nil
}
func (s *stubConvStore) UpdateMessageDelivery(_ context.Context, _ string, status string, ext *string, _ string) (domain.Message, error) {
	s.msg.Status = status
	s.msg.ExternalID = ext
	return s.msg, nil
}
func (s *stubConvStore) InsertInboundMessage(_ context.Context, m domain.Message, _ string) (domain.Message, bool, error) {
	s.msg = m
	if s.byExt != nil && m.ExternalID != nil {
		s.byExt[*m.ExternalID] = m
	}
	return m, true, nil
}
func (s *stubConvStore) FindOrCreateCustomerByPhone(_ context.Context, id, name, phone string) (domain.Customer, error) {
	return domain.Customer{ID: id, Name: name, Phone: phone, Version: 1}, nil
}

type stubSender struct {
	err       error
	uncertain bool
}

func (s *stubSender) SendText(context.Context, string, string, string) (string, bool, error) {
	if s.err != nil {
		return "", s.uncertain, s.err
	}
	return "wamid.out." + uuid.NewString(), false, nil
}

type stubConfigReader struct{}

func (stubConfigReader) GetWhatsAppConfig(context.Context) (string, bool, error) {
	return "999000", true, nil
}
