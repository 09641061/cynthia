package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JobStore persists assistant background jobs in PostgreSQL.
type JobStore struct {
	db *sql.DB
}

// NewJobStore constructs a PostgreSQL-backed assistant JobStore.
func NewJobStore(db *sql.DB) *JobStore {
	return &JobStore{db: db}
}

// Enqueue atomically inserts or coalesces a pending assistant job for a conversation.
func (s *JobStore) Enqueue(ctx context.Context, convID, triggerMsgID string, debounce time.Duration) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO assistant_jobs (id, conversation_id, trigger_message_id, status, run_after)
		VALUES ($1, $2, $3, 'pending', NOW() + $4 * INTERVAL '1 millisecond')
		ON CONFLICT (conversation_id) WHERE status = 'pending'
		DO UPDATE SET
			trigger_message_id = EXCLUDED.trigger_message_id,
			run_after = EXCLUDED.run_after,
			updated_at = CURRENT_TIMESTAMP`,
		uuid.NewString(), convID, triggerMsgID, debounce.Milliseconds(),
	)
	return err
}

// HasPendingForConversation reports whether another pending job exists for convID.
func (s *JobStore) HasPendingForConversation(ctx context.Context, convID string) (bool, error) {
	if s == nil || s.db == nil {
		return false, nil
	}
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM assistant_jobs WHERE conversation_id = $1 AND status = 'pending')`,
		convID,
	).Scan(&exists)
	return exists, err
}

// CountRecentCompleted counts completed jobs for convID since the given timestamp.
func (s *JobStore) CountRecentCompleted(ctx context.Context, convID string, since time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM assistant_jobs WHERE conversation_id = $1 AND status = 'done' AND updated_at >= $2`,
		convID, since.UTC(),
	).Scan(&count)
	return count, err
}

// SaveReply persists the generated reply text before outbound delivery.
func (s *JobStore) SaveReply(ctx context.Context, jobID, replyText string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE assistant_jobs SET reply_text = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		jobID, replyText,
	)
	return err
}

// Complete marks a job as done and releases its lease.
func (s *JobStore) Complete(ctx context.Context, jobID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE assistant_jobs SET status = 'done', locked_until = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		jobID,
	)
	return err
}

// FailOrRetry either marks the job failed, supersedes it if a newer pending job exists, or reschedules it.
func (s *JobStore) FailOrRetry(ctx context.Context, jobID, convID, sanitizedErr string, retryable bool, backoff time.Duration) error {
	if s == nil || s.db == nil {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var attempts, maxAttempts int
	err = tx.QueryRowContext(ctx,
		`SELECT attempts, max_attempts FROM assistant_jobs WHERE id = $1 FOR UPDATE`, jobID,
	).Scan(&attempts, &maxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !retryable || attempts >= maxAttempts {
		_, err = tx.ExecContext(ctx,
			`UPDATE assistant_jobs SET status = 'failed', locked_until = NULL, last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			jobID, sanitizedErr,
		)
	} else {
		err = requeueOrSupersede(ctx, tx, jobID, convID, sanitizedErr, backoff)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func requeueOrSupersede(ctx context.Context, tx *sql.Tx, jobID, convID, sanitizedErr string, backoff time.Duration) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, convID); err != nil {
		return err
	}
	var hasPending bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM assistant_jobs WHERE conversation_id = $1 AND status = 'pending')`, convID,
	).Scan(&hasPending); err != nil {
		return err
	}
	if hasPending {
		_, err := tx.ExecContext(ctx,
			`UPDATE assistant_jobs SET status = 'done', locked_until = NULL, last_error = 'superseded', updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			jobID,
		)
		return err
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE assistant_jobs SET status = 'pending', locked_until = NULL, run_after = NOW() + $3 * INTERVAL '1 millisecond', last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		jobID, sanitizedErr, backoff.Milliseconds(),
	)
	return err
}
