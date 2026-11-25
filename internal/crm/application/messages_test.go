package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

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
		svc, st := base(status)
		var dErr application.DeliveryError
		if _, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); !errors.As(err, &dErr) {
			t.Fatalf("status=%s: expected DeliveryError, got %v", status, err)
		}
		if st.pausedCalled {
			t.Fatalf("status=%s: expected failed replay not to pause AI", status)
		}
	}
	svc, st := base(domain.MessageStatusPending)
	if _, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("status=pending: expected ErrConflict, got %v", err)
	}
	if st.pausedCalled {
		t.Fatalf("status=pending: expected conflict not to pause AI")
	}
	svc, st = base(domain.MessageStatusSent)
	if msg, err := svc.SendOutbound(context.Background(), convID, "k1", "hi"); err != nil || msg.Status != domain.MessageStatusSent {
		t.Fatalf("status=sent: expected (sent, nil), got (%+v, %v)", msg, err)
	}
	if st.pausedCalled {
		t.Fatalf("expected idempotent replay not to re-pause AI")
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
	if st.pausedCalled {
		t.Fatalf("expected configReader error not to pause AI")
	}
	t.Setenv("WHATSAPP_PHONE_NUMBER_ID", "env-phone")
	svc = application.NewMessageService(st, st, st, st, nil, stubCfg{phone: "p1", enabled: false})
	var dErr application.DeliveryError
	if _, err := svc.SendOutbound(context.Background(), convID, "k2", "hi"); !errors.As(err, &dErr) {
		t.Fatalf("expected disabled config to stay disabled, got %v", err)
	}
	if st.pausedCalled {
		t.Fatalf("expected failed delivery not to pause AI")
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc = application.NewMessageService(st, st, st, st, cancelOnSendSender{cancel: cancel}, stubCfg{phone: "p1", enabled: true})
	if _, err := svc.SendOutbound(ctx, convID, "k3", "hi"); err != nil {
		t.Fatalf("expected sent message despite canceled ctx, got %v", err)
	}
	if !st.updateCalled || st.updateCtxErr != nil {
		t.Fatalf("expected UpdateMessageDelivery to receive uncanceled context, called=%v err=%v", st.updateCalled, st.updateCtxErr)
	}
	if !st.pausedCalled || st.pausedReason != "staff_replied" {
		t.Fatalf("expected staff reply to pause AI with staff_replied, got called=%v reason=%q", st.pausedCalled, st.pausedReason)
	}
}

func TestSendAssistantReplyDoesNotAutoPauseAndInboundHookFires(t *testing.T) {
	convID, custID := uuid.NewString(), uuid.NewString()
	st := &stubMsgStore{
		customer: domain.Customer{ID: custID, Phone: "+573001112233"},
		conv:     domain.Conversation{ID: convID, CustomerID: custID, Status: domain.ConversationStatusOpen},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := application.NewMessageService(st, st, st, st, cancelOnSendSender{cancel: cancel}, stubCfg{phone: "p1", enabled: true})
	if _, err := svc.SendAssistantReply(ctx, convID, "k-ai", "hello from ai"); err != nil {
		t.Fatalf("expected SendAssistantReply to succeed, got %v", err)
	}
	if st.pausedCalled {
		t.Fatalf("expected SendAssistantReply not to pause AI")
	}

	hook := &stubInboundHook{}
	svc.SetInboundHook(hook)
	msg, err := svc.ReceiveInbound(context.Background(), "wamid.in.1", "+573001112233", "Ana", "hola")
	if err != nil {
		t.Fatalf("expected ReceiveInbound to succeed, got %v", err)
	}
	if hook.calls != 1 || hook.convID != convID || hook.msgID != msg.ID {
		t.Fatalf("expected inbound hook called once with (%s,%s), got %+v", convID, msg.ID, hook)
	}
}
