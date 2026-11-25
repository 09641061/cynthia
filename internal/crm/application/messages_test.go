package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type stubMsgStore struct {
	customer     domain.Customer
	conv         domain.Conversation
	existing     *domain.Message
	updateCalled bool
	updateCtxErr error
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

func TestSendOutboundIdempotencyReplayStatuses(t *testing.T) {
	convID, custID := uuid.NewString(), uuid.NewString()
	base := func(status string) (*application.MessageService, *stubMsgStore) {
		st := &stubMsgStore{
			customer: domain.Customer{ID: custID, Phone: "+573001112233"},
			conv:     domain.Conversation{ID: convID, CustomerID: custID, Status: domain.ConversationStatusOpen},
			existing: &domain.Message{ID: uuid.NewString(), ConversationID: convID, Status: status},
		}
		return application.NewMessageService(st, st, st, st, nil, stubCfg{phone: "p1", enabled: true}), st
	}
	for _, status := range []string{domain.MessageStatusFailed, domain.MessageStatusUncertain} {
		svc, _ := base(status)
		var dErr application.DeliveryError
		if _, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); !errors.As(err, &dErr) {
			t.Fatalf("status=%s: expected DeliveryError, got %v", status, err)
		}
	}
	svc, _ := base(domain.MessageStatusPending)
	if _, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("status=pending: expected ErrConflict, got %v", err)
	}
	svc, _ = base(domain.MessageStatusSent)
	if msg, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); err != nil || msg.Status != domain.MessageStatusSent {
		t.Fatalf("status=sent: expected (sent, nil), got (%+v, %v)", msg, err)
	}
}

func TestSendOutboundConfigErrorsAndWithoutCancelPersist(t *testing.T) {
	convID, custID := uuid.NewString(), uuid.NewString()
	st := &stubMsgStore{
		customer: domain.Customer{ID: custID, Phone: "+573001112233"},
		conv:     domain.Conversation{ID: convID, CustomerID: custID, Status: domain.ConversationStatusOpen},
	}
	cfgErr := errors.New("db down")
	svc := application.NewMessageService(st, st, st, st, nil, stubCfg{err: cfgErr})
	if _, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); !errors.Is(err, cfgErr) {
		t.Fatalf("expected configReader error to propagate, got %v", err)
	}
	t.Setenv("WHATSAPP_PHONE_NUMBER_ID", "env-phone")
	svc = application.NewMessageService(st, st, st, st, nil, stubCfg{phone: "p1", enabled: false})
	var dErr application.DeliveryError
	if _, err := svc.SendOutbound(context.Background(), convID, "k2", "hi"); !errors.As(err, &dErr) {
		t.Fatalf("expected disabled config to stay disabled, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc = application.NewMessageService(st, st, st, st, cancelOnSendSender{cancel: cancel}, stubCfg{phone: "p1", enabled: true})
	if _, err := svc.SendOutbound(ctx, convID, "k3", "hi"); err != nil {
		t.Fatalf("expected sent message despite canceled ctx, got %v", err)
	}
	if !st.updateCalled || st.updateCtxErr != nil {
		t.Fatalf("expected UpdateMessageDelivery to receive uncanceled context, called=%v err=%v", st.updateCalled, st.updateCtxErr)
	}
}
