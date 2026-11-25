//go:build integration

package persistence_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

func TestConversationAIAndReopenReset(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()

	var aiEnabled bool
	if err := db.QueryRowContext(ctx, `SELECT ai_enabled FROM organizations WHERE singleton_key = TRUE`).Scan(&aiEnabled); err != nil {
		t.Fatalf("select organizations.ai_enabled: %v", err)
	}
	if aiEnabled {
		t.Fatalf("expected organizations.ai_enabled default false")
	}

	cust, err := store.FindOrCreateCustomerByPhone(ctx, uuid.NewString(), "Ana", "+573001112233")
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	conv, err := store.EnsureOpenConversation(ctx, cust.ID)
	if err != nil {
		t.Fatalf("ensure open conversation: %v", err)
	}
	if conv.AiPaused || conv.HandoffReason != "" {
		t.Fatalf("expected fresh conversation unpaused, got %+v", conv)
	}

	if _, err := store.UpdateConversationAI(ctx, uuid.NewString(), true, "manual_pause"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	conv, err = store.UpdateConversationAI(ctx, conv.ID, true, "manual_pause")
	if err != nil || !conv.AiPaused || conv.HandoffReason != "manual_pause" {
		t.Fatalf("expected paused conversation, got (%+v, %v)", conv, err)
	}

	// open -> open preserves ai_paused and handoff_reason.
	conv, err = store.UpdateConversationStatus(ctx, conv.ID, domain.ConversationStatusOpen)
	if err != nil || !conv.AiPaused || conv.HandoffReason != "manual_pause" {
		t.Fatalf("expected open -> open to preserve AI state, got (%+v, %v)", conv, err)
	}

	// Close then reopen via UpdateConversationStatus -> resets ai_paused and handoff_reason.
	conv, err = store.UpdateConversationStatus(ctx, conv.ID, domain.ConversationStatusClosed)
	if err != nil || !conv.AiPaused {
		t.Fatalf("expected close to preserve ai_paused, got (%+v, %v)", conv, err)
	}
	conv, err = store.UpdateConversationStatus(ctx, conv.ID, domain.ConversationStatusOpen)
	if err != nil || conv.AiPaused || conv.HandoffReason != "" {
		t.Fatalf("expected reopen via UpdateConversationStatus to reset AI state, got (%+v, %v)", conv, err)
	}

	// SetConversationAIPaused + close + reopen via EnsureOpenConversation -> resets AI state.
	conv, err = store.SetConversationAIPaused(ctx, conv.ID, true, "staff_replied")
	if err != nil || !conv.AiPaused || conv.HandoffReason != "staff_replied" {
		t.Fatalf("unexpected SetConversationAIPaused result: (%+v, %v)", conv, err)
	}
	conv, err = store.UpdateConversationStatus(ctx, conv.ID, domain.ConversationStatusClosed)
	if err != nil {
		t.Fatalf("close conversation: %v", err)
	}
	conv, err = store.EnsureOpenConversation(ctx, cust.ID)
	if err != nil || conv.AiPaused || conv.HandoffReason != "" {
		t.Fatalf("expected EnsureOpenConversation reopen to reset AI state, got (%+v, %v)", conv, err)
	}
}

func TestAssistantJobsPartialUniqueIndexes(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()

	cust, err := store.FindOrCreateCustomerByPhone(ctx, uuid.NewString(), "Luis", "+573004445566")
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	conv, err := store.EnsureOpenConversation(ctx, cust.ID)
	if err != nil {
		t.Fatalf("ensure open conversation: %v", err)
	}
	insertJob := func(status string) error {
		_, err := db.ExecContext(ctx,
			`INSERT INTO assistant_jobs (id, conversation_id, trigger_message_id, status, run_after)
			 VALUES ($1, $2, $3, $4, $5)`,
			uuid.NewString(), conv.ID, uuid.NewString(), status, time.Now().UTC(),
		)
		return err
	}

	if err := insertJob("pending"); err != nil {
		t.Fatalf("insert first pending job: %v", err)
	}
	if err := insertJob("pending"); err == nil {
		t.Fatalf("expected second pending job for same conversation to violate unique index")
	}
	if err := insertJob("processing"); err != nil {
		t.Fatalf("insert first processing job alongside pending: %v", err)
	}
	if err := insertJob("processing"); err == nil {
		t.Fatalf("expected second processing job for same conversation to violate unique index")
	}
	if err := insertJob("done"); err != nil {
		t.Fatalf("insert first done job: %v", err)
	}
	if err := insertJob("done"); err != nil {
		t.Fatalf("insert second done job: %v", err)
	}
}
