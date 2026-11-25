//go:build integration

package assistant_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/assistant/infrastructure/postgres"
)

func TestJobStoreCoalescingAndSingleActivePerConversation(t *testing.T) {
	client, seed := newTestClient(t)
	store := postgres.NewJobStore(client.DB)
	ctx := t.Context()

	msg1, msg2 := uuid.NewString(), uuid.NewString()
	if err := store.Enqueue(ctx, seed.convID, msg1, 0); err != nil {
		t.Fatalf("enqueue 1: %v", err)
	}
	if err := store.Enqueue(ctx, seed.convID, msg2, 0); err != nil {
		t.Fatalf("enqueue 2: %v", err)
	}
	var count int
	_ = client.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_jobs WHERE conversation_id = $1`, seed.convID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected coalesced single pending job, got %d", count)
	}

	job, ok, err := store.ClaimNext(ctx, 2*time.Minute)
	if err != nil || !ok || job.TriggerMessageID != msg2 || job.Attempts != 1 {
		t.Fatalf("expected claimed job with msg2 and attempts=1, got ok=%v job=%+v err=%v", ok, job, err)
	}

	// Enqueue another pending job while first is processing; ClaimNext must NOT claim it.
	if err := store.Enqueue(ctx, seed.convID, uuid.NewString(), 0); err != nil {
		t.Fatalf("enqueue 3: %v", err)
	}
	if _, ok2, err := store.ClaimNext(ctx, 2*time.Minute); err != nil || ok2 {
		t.Fatalf("expected single-active-per-conv to block second claim, got ok=%v err=%v", ok2, err)
	}

	// FailOrRetry on first job while a pending job exists must mark first job done/superseded.
	if err := store.FailOrRetry(ctx, job.ID, seed.convID, "transient", true, time.Second); err != nil {
		t.Fatalf("fail or retry superseded: %v", err)
	}
	var status, lastErr string
	_ = client.DB.QueryRowContext(ctx, `SELECT status, last_error FROM assistant_jobs WHERE id = $1`, job.ID).Scan(&status, &lastErr)
	if status != "done" || lastErr != "superseded" {
		t.Fatalf("expected done/superseded, got %s/%s", status, lastErr)
	}
}

func TestJobStoreSkipLockedAndStaleLeaseRecovery(t *testing.T) {
	client, seed := newTestClient(t)
	store := postgres.NewJobStore(client.DB)
	ctx := t.Context()

	conv2 := uuid.NewString()
	_, err := client.DB.ExecContext(ctx, `INSERT INTO crm_conversations (id, customer_id, status) VALUES ($1, $2, 'open')`, conv2, seed.otherCust)
	if err != nil {
		t.Fatalf("seed conv2: %v", err)
	}
	_ = store.Enqueue(ctx, seed.convID, uuid.NewString(), 0)
	_ = store.Enqueue(ctx, conv2, uuid.NewString(), 0)

	j1, ok1, err := store.ClaimNext(ctx, 2*time.Minute)
	j2, ok2, err2 := store.ClaimNext(ctx, 2*time.Minute)
	if err != nil || err2 != nil || !ok1 || !ok2 || j1.ID == j2.ID {
		t.Fatalf("expected two distinct claimed jobs, got j1=%+v j2=%+v", j1, j2)
	}

	// Simulate expired lease on j1 with attempts < max_attempts -> reclaimed with attempts=2.
	_, _ = client.DB.ExecContext(ctx, `UPDATE assistant_jobs SET locked_until = NOW() - INTERVAL '5 second' WHERE id = $1`, j1.ID)
	reclaimed, ok, err := store.ClaimNext(ctx, 2*time.Minute)
	if err != nil || !ok || reclaimed.ID != j1.ID || reclaimed.Attempts != 2 {
		t.Fatalf("expected reclaimed j1 with attempts=2, got ok=%v job=%+v err=%v", ok, reclaimed, err)
	}

	// Simulate expired lease at max_attempts -> marked failed with lease_expired.
	_, _ = client.DB.ExecContext(ctx, `UPDATE assistant_jobs SET attempts = max_attempts, locked_until = NOW() - INTERVAL '5 second' WHERE id = $1`, j1.ID)
	if _, ok, err := store.ClaimNext(ctx, 2*time.Minute); err != nil || ok {
		t.Fatalf("expected exhausted stale job to not be returned, got ok=%v err=%v", ok, err)
	}
	var status, lastErr string
	_ = client.DB.QueryRowContext(ctx, `SELECT status, last_error FROM assistant_jobs WHERE id = $1`, j1.ID).Scan(&status, &lastErr)
	if status != "failed" || lastErr != "lease_expired" {
		t.Fatalf("expected failed/lease_expired, got %s/%s", status, lastErr)
	}
}
