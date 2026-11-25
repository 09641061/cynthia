//go:build integration

package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type fakeSender struct {
	calls int
}

func (f *fakeSender) SendText(_ context.Context, _, _, _ string) (string, bool, error) {
	f.calls++
	return "wamid.out." + uuid.NewString(), false, nil
}

type fakeConfigReader struct{}

func (fakeConfigReader) GetWhatsAppConfig(context.Context) (string, bool, error) {
	return "109876543210", true, nil
}

func TestConversationsInboundReopenDedupeAndOutboundIdempotency(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()
	sender := &fakeSender{}
	convSvc := application.NewConversationService(store, store)
	msgSvc := application.NewMessageService(store, store, store, store, sender, fakeConfigReader{})

	// 1. Inbound auto-registers customer and opens conversation
	extID := "wamid.in." + uuid.NewString()
	m1, err := msgSvc.ReceiveInbound(ctx, extID, "573001234567", "Maria", "Hola!")
	if err != nil {
		t.Fatalf("receive inbound: %v", err)
	}
	// Global external_id deduplication
	m1Dup, err := msgSvc.ReceiveInbound(ctx, extID, "573001234567", "Maria", "Hola!")
	if err != nil || m1Dup.ID != m1.ID {
		t.Fatalf("expected deduplicated message %s, got %s (err=%v)", m1.ID, m1Dup.ID, err)
	}

	// 2. Outbound with Idempotency-Key + replay
	out1, err := msgSvc.SendOutbound(ctx, m1.ConversationID, "idem-1", "Respuesta")
	if err != nil || out1.Status != domain.MessageStatusSent {
		t.Fatalf("expected sent outbound message, got %+v err=%v", out1, err)
	}
	outReplay, err := msgSvc.SendOutbound(ctx, m1.ConversationID, "idem-1", "Respuesta")
	if err != nil || outReplay.ID != out1.ID || sender.calls != 1 {
		t.Fatalf("expected idempotent replay with 1 send call, got id=%s calls=%d err=%v", outReplay.ID, sender.calls, err)
	}

	// 3. Close conversation
	closed, err := convSvc.UpdateStatus(ctx, m1.ConversationID, "closed")
	if err != nil || closed.Status != "closed" {
		t.Fatalf("expected closed, got %+v err=%v", closed, err)
	}

	// 4. Inbound reopens the most recent closed conversation
	m2, err := msgSvc.ReceiveInbound(ctx, "wamid.in."+uuid.NewString(), "573001234567", "", "Vuelvo")
	if err != nil || m2.ConversationID != m1.ConversationID {
		t.Fatalf("expected reopened conversation %s, got %s (err=%v)", m1.ConversationID, m2.ConversationID, err)
	}
	reopened, err := store.GetConversation(ctx, m1.ConversationID)
	if err != nil || reopened.Status != "open" {
		t.Fatalf("expected open conversation, got %+v err=%v", reopened, err)
	}

	// 5. Max-1-open partial unique index prevents a second open conversation for same customer
	_, err = db.ExecContext(ctx,
		`INSERT INTO crm_conversations (id, customer_id, channel, status) VALUES ($1, $2, 'whatsapp', 'open')`,
		uuid.NewString(), reopened.CustomerID)
	if err == nil {
		t.Fatal("expected partial unique index violation for second open conversation")
	}
}
