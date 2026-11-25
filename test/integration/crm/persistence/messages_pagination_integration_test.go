//go:build integration

package persistence_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/organizations"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestListMessagesPaginationAndDuplicateReplayOnClosedConversation(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()
	convSvc := application.NewConversationService(store, store)
	msgSvc := application.NewMessageService(store, store, store, store, &fakeSender{}, fakeConfigReader{})

	ext1 := "wamid.p1." + uuid.NewString()
	m1, err := msgSvc.ReceiveInbound(ctx, ext1, "573009998877", "Luis", "msg-1")
	if err != nil {
		t.Fatalf("m1: %v", err)
	}
	m2, err := msgSvc.ReceiveInbound(ctx, "wamid.p2."+uuid.NewString(), "573009998877", "Luis", "msg-2")
	if err != nil {
		t.Fatalf("m2: %v", err)
	}
	m3, err := msgSvc.ReceiveInbound(ctx, "wamid.p3."+uuid.NewString(), "573009998877", "Luis", "msg-3")
	if err != nil {
		t.Fatalf("m3: %v", err)
	}
	m4, err := msgSvc.ReceiveInbound(ctx, "wamid.p4."+uuid.NewString(), "573009998877", "Luis", "msg-4")
	if err != nil {
		t.Fatalf("m4: %v", err)
	}

	base := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{m1.ID, m2.ID, m3.ID, m4.ID} {
		if _, err := db.ExecContext(ctx, `UPDATE crm_messages SET created_at = $1 WHERE id = $2`, base.Add(time.Duration(i)*time.Second), id); err != nil {
			t.Fatalf("set deterministic created_at for m%d: %v", i+1, err)
		}
	}

	page1, err := convSvc.ListMessages(ctx, m1.ConversationID, 2, nil)
	if err != nil || len(page1) != 2 || page1[0].ID != m3.ID || page1[1].ID != m4.ID {
		t.Fatalf("expected latest 2 [m3,m4], got %+v err=%v", page1, err)
	}
	page2, err := convSvc.ListMessages(ctx, m1.ConversationID, 2, &page1[0].CreatedAt)
	if err != nil || len(page2) != 2 || page2[0].ID != m1.ID || page2[1].ID != m2.ID {
		t.Fatalf("expected older 2 [m1,m2], got %+v err=%v", page2, err)
	}

	if _, err := convSvc.UpdateStatus(ctx, m1.ConversationID, 1, "closed"); err != nil {
		t.Fatalf("close conversation: %v", err)
	}
	replay, err := msgSvc.ReceiveInbound(ctx, ext1, "573009998877", "Luis", "msg-1")
	if err != nil || replay.ID != m1.ID {
		t.Fatalf("expected deduplicated replay %s, got %s err=%v", m1.ID, replay.ID, err)
	}
	conv, err := store.GetConversation(ctx, m1.ConversationID)
	if err != nil || conv.Status != "closed" {
		t.Fatalf("expected conversation to remain closed after duplicate replay, got %+v err=%v", conv, err)
	}

	_, err = db.ExecContext(ctx,
		`UPDATE organizations
		 SET whatsapp_phone_number = '+573001112233', whatsapp_phone_number_id = '109876543210', whatsapp_enabled = TRUE
		 WHERE singleton_key = TRUE`)
	if err != nil {
		t.Fatalf("update organization whatsapp config: %v", err)
	}
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatalf("new breaker: %v", err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	reader := organizations.NewWhatsAppConfigReader(client)
	phoneID, enabled, err := reader.GetWhatsAppConfig(ctx)
	if err != nil || phoneID != "109876543210" || !enabled {
		t.Fatalf("expected whatsapp config (109876543210, true), got (%q, %v, err=%v)", phoneID, enabled, err)
	}
}
