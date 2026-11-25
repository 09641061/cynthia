package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

// ListMessages returns the latest messages for a conversation ordered chronologically.
func (s *Store) ListMessages(ctx context.Context, convID string, limit int, before *time.Time) ([]domain.Message, error) {
	var items []domain.Message
	err := s.db.Do(ctx, func() error {
		var (
			rows *sql.Rows
			err  error
		)
		if before != nil {
			rows, err = s.db.QueryContext(ctx,
				`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
				 FROM (
					SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
					FROM crm_messages WHERE conversation_id = $1 AND created_at < $2
					ORDER BY created_at DESC, id DESC LIMIT $3
				 ) sub ORDER BY created_at ASC, id ASC`, convID, *before, limit)
		} else {
			rows, err = s.db.QueryContext(ctx,
				`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
				 FROM (
					SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
					FROM crm_messages WHERE conversation_id = $1
					ORDER BY created_at DESC, id DESC LIMIT $2
				 ) sub ORDER BY created_at ASC, id ASC`, convID, limit)
		}
		if err != nil {
			return err
		}
		defer rows.Close()
		items = make([]domain.Message, 0)
		for rows.Next() {
			var m domain.Message
			if err := rows.Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
				&m.Direction, &m.Content, &m.Status, &m.CreatedAt); err != nil {
				return err
			}
			items = append(items, m)
		}
		return rows.Err()
	})
	return items, err
}

// GetMessageByIdempotencyKey fetches an outbound message by (conversation_id, idempotency_key).
func (s *Store) GetMessageByIdempotencyKey(ctx context.Context, convID, key string) (domain.Message, error) {
	var m domain.Message
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
			 FROM crm_messages WHERE conversation_id = $1 AND idempotency_key = $2`, convID, key,
		).Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
			&m.Direction, &m.Content, &m.Status, &m.CreatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Message{}, application.ErrNotFound
	}
	return m, err
}

// GetMessageByExternalID fetches a message by its global external_id.
func (s *Store) GetMessageByExternalID(ctx context.Context, externalID string) (domain.Message, error) {
	var m domain.Message
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
			 FROM crm_messages WHERE external_id = $1`, externalID,
		).Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
			&m.Direction, &m.Content, &m.Status, &m.CreatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Message{}, application.ErrNotFound
	}
	return m, err
}

// InsertOutboundPending persists an outbound message in pending state before external delivery.
func (s *Store) InsertOutboundPending(ctx context.Context, msg domain.Message) (domain.Message, error) {
	var created domain.Message
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`INSERT INTO crm_messages (id, conversation_id, idempotency_key, direction, content, status)
			 VALUES ($1, $2, $3, 'outbound', $4, 'pending')
			 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
			msg.ID, msg.ConversationID, msg.IdempotencyKey, msg.Content,
		).Scan(&created.ID, &created.ConversationID, &created.ExternalID, &created.IdempotencyKey,
			&created.Direction, &created.Content, &created.Status, &created.CreatedAt)
	})
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Message{}, application.ErrConflict
	}
	return created, err
}
