package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-service/internal/assistant/application"
)

const claimNextQuery = `
SELECT id, conversation_id, trigger_message_id, status, run_after, attempts, max_attempts, reply_text, last_error
FROM assistant_jobs
WHERE (
	(status = 'pending' AND run_after <= NOW() AND NOT EXISTS (
		SELECT 1 FROM assistant_jobs p
		WHERE p.conversation_id = assistant_jobs.conversation_id AND p.status = 'processing'
	))
	OR (status = 'processing' AND locked_until < NOW())
)
ORDER BY run_after ASC, created_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1`

// ClaimNext atomically claims one ready pending job or reclaims one expired processing job.
func (s *JobStore) ClaimNext(ctx context.Context, lease time.Duration) (application.Job, bool, error) {
	if s == nil || s.db == nil {
		return application.Job{}, false, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Job{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var job application.Job
	err = tx.QueryRowContext(ctx, claimNextQuery).Scan(
		&job.ID, &job.ConversationID, &job.TriggerMessageID, &job.Status,
		&job.RunAfter, &job.Attempts, &job.MaxAttempts, &job.ReplyText, &job.LastError,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Job{}, false, nil
	}
	if err != nil {
		return application.Job{}, false, err
	}

	if job.Status == "processing" && job.Attempts >= job.MaxAttempts {
		if _, err := tx.ExecContext(ctx,
			`UPDATE assistant_jobs SET status = 'failed', locked_until = NULL, last_error = 'lease_expired', updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			job.ID,
		); err != nil {
			return application.Job{}, false, err
		}
		return application.Job{}, false, tx.Commit()
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE assistant_jobs SET status = 'processing', locked_until = NOW() + $2 * INTERVAL '1 millisecond', attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		job.ID, lease.Milliseconds(),
	); err != nil {
		return application.Job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return application.Job{}, false, err
	}
	job.Status = "processing"
	job.Attempts++
	return job, true, nil
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
