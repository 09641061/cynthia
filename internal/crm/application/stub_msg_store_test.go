package application_test

import (
	"context"
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type stubMsgStore struct {
	customer     domain.Customer
	conv         domain.Conversation
	existing     *domain.Message
	updateCalled bool
	updateCtxErr error
	pausedCalled bool
	pausedReason string
	pauseErr     error
}

func (s *stubMsgStore) ListCustomers(context.Context, string, int, int) ([]domain.Customer, int, error) {
	return nil, 0, nil
}
func (s *stubMsgStore) GetCustomer(context.Context, string) (domain.Customer, error) {
	return s.customer, nil
}
func (s *stubMsgStore) GetCustomerByPhone(context.Context, string) (domain.Customer, error) {
	return s.customer, nil
}
func (s *stubMsgStore) CreateCustomer(context.Context, domain.Customer) (domain.Customer, error) {
	return s.customer, nil
}
func (s *stubMsgStore) UpdateCustomer(context.Context, string, string, string) (domain.Customer, error) {
	return s.customer, nil
}
func (s *stubMsgStore) DeleteCustomer(context.Context, string) error { return nil }
func (s *stubMsgStore) FindOrCreateCustomerByPhone(context.Context, string, string, string) (domain.Customer, error) {
	return s.customer, nil
}
func (s *stubMsgStore) ListConversations(context.Context, application.ConversationFilter) ([]domain.Conversation, int, error) {
	return nil, 0, nil
}
func (s *stubMsgStore) GetConversation(context.Context, string) (domain.Conversation, error) {
	return s.conv, nil
}
func (s *stubMsgStore) UpdateConversationStatus(context.Context, string, string) (domain.Conversation, error) {
	return s.conv, nil
}
func (s *stubMsgStore) UpdateConversationAI(_ context.Context, _ string, aiPaused bool, reason string) (domain.Conversation, error) {
	s.conv.AiPaused = aiPaused
	s.conv.HandoffReason = reason
	return s.conv, nil
}
func (s *stubMsgStore) SetConversationAIPaused(_ context.Context, _ string, aiPaused bool, reason string) (domain.Conversation, error) {
	s.pausedCalled = true
	s.pausedReason = reason
	if s.pauseErr != nil {
		return domain.Conversation{}, s.pauseErr
	}
	s.conv.AiPaused = aiPaused
	s.conv.HandoffReason = reason
	return s.conv, nil
}
func (s *stubMsgStore) EnsureOpenConversation(context.Context, string) (domain.Conversation, error) {
	return s.conv, nil
}
func (s *stubMsgStore) ListMessages(context.Context, string, int, *time.Time) ([]domain.Message, error) {
	return nil, nil
}
func (s *stubMsgStore) GetMessageByIdempotencyKey(context.Context, string, string) (domain.Message, error) {
	if s.existing != nil {
		return *s.existing, nil
	}
	return domain.Message{}, application.ErrNotFound
}
func (s *stubMsgStore) GetMessageByExternalID(context.Context, string) (domain.Message, error) {
	return domain.Message{}, application.ErrNotFound
}
func (s *stubMsgStore) InsertOutboundPending(_ context.Context, msg domain.Message) (domain.Message, error) {
	return msg, nil
}
func (s *stubMsgStore) UpdateMessageDelivery(ctx context.Context, id, status string, extID *string, _ string) (domain.Message, error) {
	s.updateCalled = true
	s.updateCtxErr = ctx.Err()
	return domain.Message{ID: id, ConversationID: s.conv.ID, Status: status, ExternalID: extID}, nil
}
func (s *stubMsgStore) InsertInboundMessage(_ context.Context, msg domain.Message, _ string) (domain.Message, bool, error) {
	return msg, true, nil
}

type stubCfg struct {
	phone   string
	enabled bool
	err     error
}

func (c stubCfg) GetWhatsAppConfig(context.Context) (string, bool, error) {
	return c.phone, c.enabled, c.err
}

type cancelOnSendSender struct{ cancel context.CancelFunc }

func (c cancelOnSendSender) SendText(context.Context, string, string, string) (string, bool, error) {
	c.cancel()
	return "wamid.ok", false, nil
}

type stubInboundHook struct {
	convID string
	msgID  string
	calls  int
}

func (h *stubInboundHook) OnInboundMessage(_ context.Context, convID, msgID string) {
	h.calls++
	h.convID = convID
	h.msgID = msgID
}
