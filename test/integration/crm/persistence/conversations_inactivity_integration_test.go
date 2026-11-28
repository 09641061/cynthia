//go:build integration

package persistence_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

func TestCloseInactiveConversationsAndReopenReset(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()

	createOpen := func(phone string, paused bool, age time.Duration) (string, string) {
		cust, err := store.FindOrCreateCustomerByPhone(ctx, uuid.NewString(), "User "+phone, phone)
		if err != nil {
			t.Fatalf("create customer %s: %v", phone, err)
		}
		conv, err := store.EnsureOpenConversation(ctx, cust.ID)
		if err != nil {
			t.Fatalf("open conv %s: %v", phone, err)
		}
		reason := ""
		if paused {
			reason = "human_requested"
		}
		ts := time.Now().UTC().Add(-age)
		if _, err := db.ExecContext(ctx,
			`UPDATE crm_conversations SET ai_paused = $1, handoff_reason = $2, created_at = $3, updated_at = $3, last_message_at = $3 WHERE id = $4`,
			paused, reason, ts, conv.ID,
		); err != nil {
			t.Fatalf("backdate conv: %v", err)
		}
		return cust.ID, conv.ID
	}

	_, staleOpenID := createOpen("+51990001001", false, 30*time.Minute)
	stalePausedCustID, stalePausedID := createOpen("+51990001002", true, 30*time.Minute)
	_, activeJobID := createOpen("+51990001003", false, 30*time.Minute)
	_, freshOpenID := createOpen("+51990001004", false, 2*time.Minute)

	// Insert a pending assistant job for activeJobID so it must NOT be auto-closed.
	msgID := uuid.NewString()
	extID := "ext-" + msgID
	if _, _, err := store.InsertInboundMessage(ctx, domain.Message{
		ID: msgID, ConversationID: activeJobID, ExternalID: &extID,
		Direction: domain.MessageDirectionInbound, Content: "Hola", Status: domain.MessageStatusReceived,
	}, "Hola"); err != nil {
		t.Fatalf("insert inbound message: %v", err)
	}
	staleTS := time.Now().UTC().Add(-30 * time.Minute)
	if _, err := db.ExecContext(ctx,
		`UPDATE crm_conversations SET updated_at = $1, last_message_at = $1 WHERE id = $2`,
		staleTS, activeJobID,
	); err != nil {
		t.Fatalf("backdate activeJob conv: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO assistant_jobs (id, conversation_id, trigger_message_id, status, run_after) VALUES ($1, $2, $3, 'pending', CURRENT_TIMESTAMP)`,
		uuid.NewString(), activeJobID, msgID,
	); err != nil {
		t.Fatalf("insert pending assistant job: %v", err)
	}

	cutoff := time.Now().UTC().Add(-15 * time.Minute)
	closedIDs, err := store.CloseInactiveConversations(ctx, cutoff)
	if err != nil {
		t.Fatalf("CloseInactiveConversations: %v", err)
	}
	closedSet := make(map[string]bool, len(closedIDs))
	for _, id := range closedIDs {
		closedSet[id] = true
	}
	if len(closedIDs) != 2 || !closedSet[staleOpenID] || !closedSet[stalePausedID] {
		t.Fatalf("expected only staleOpen and stalePaused closed, got %v", closedIDs)
	}
	if closedSet[activeJobID] || closedSet[freshOpenID] {
		t.Fatalf("expected activeJob and freshOpen to remain open, got %v", closedIDs)
	}

	// Reopening the auto-closed paused conversation via EnsureOpenConversation resets ai_paused = false.
	reopened, err := store.EnsureOpenConversation(ctx, stalePausedCustID)
	if err != nil {
		t.Fatalf("reopen conv: %v", err)
	}
	if reopened.ID != stalePausedID || reopened.Status != domain.ConversationStatusOpen || reopened.AiPaused || reopened.HandoffReason != "" {
		t.Fatalf("expected reopened conv with ai_paused=false, got %+v", reopened)
	}
}
