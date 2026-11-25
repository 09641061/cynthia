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
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Message, 0)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
			&m.Direction, &m.Content, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// GetMessageByIdempotencyKey fetches an outbound message by (conversation_id, idempotency_key).
func (s *Store) GetMessageByIdempotencyKey(ctx context.Context, convID, key string) (domain.Message, error) {
	var m domain.Message
	err := s.db.QueryRowContext(ctx,
		`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
		 FROM crm_messages WHERE conversation_id = $1 AND idempotency_key = $2`, convID, key,
	).Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
		&m.Direction, &m.Content, &m.Status, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Message{}, application.ErrNotFound
	}
	return m, err
}

// GetMessageByExternalID fetches a message by its global external_id.
func (s *Store) GetMessageByExternalID(ctx context.Context, externalID string) (domain.Message, error) {
	var m domain.Message
	err := s.db.QueryRowContext(ctx,
		`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
		 FROM crm_messages WHERE external_id = $1`, externalID,
	).Scan(&m.ID, &m.ConversationID, &m.ExternalID, &m.IdempotencyKey,
		&m.Direction, &m.Content, &m.Status, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Message{}, application.ErrNotFound
	}
	return m, err
}

// InsertOutboundPending persists an outbound message in pending state before external delivery.
func (s *Store) InsertOutboundPending(ctx context.Context, msg domain.Message) (domain.Message, error) {
	var created domain.Message
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO crm_messages (id, conversation_id, idempotency_key, direction, content, status)
		 VALUES ($1, $2, $3, 'outbound', $4, 'pending')
		 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
		msg.ID, msg.ConversationID, msg.IdempotencyKey, msg.Content,
	).Scan(&created.ID, &created.ConversationID, &created.ExternalID, &created.IdempotencyKey,
		&created.Direction, &created.Content, &created.Status, &created.CreatedAt)
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Message{}, application.ErrConflict
	}
	return created, err
}

// UpdateMessageDelivery updates an outbound message's delivery status and refreshes conversation preview.
func (s *Store) UpdateMessageDelivery(ctx context.Context, id, status string, externalID *string, preview string) (domain.Message, error) {
	var updated domain.Message
	err := s.db.QueryRowContext(ctx,
		`UPDATE crm_messages SET status = $1, external_id = COALESCE($2, external_id)
		 WHERE id = $3
		 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
		status, externalID, id,
	).Scan(&updated.ID, &updated.ConversationID, &updated.ExternalID, &updated.IdempotencyKey,
		&updated.Direction, &updated.Content, &updated.Status, &updated.CreatedAt)
	if err != nil {
		return domain.Message{}, err
	}
	_, _ = s.db.ExecContext(ctx,
		`UPDATE crm_conversations
		 SET last_message_at = $1, last_message_preview = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`, updated.CreatedAt, preview, updated.ConversationID)
	return updated, nil
}

// InsertInboundMessage inserts an inbound message deduplicated globally by external_id.
func (s *Store) InsertInboundMessage(ctx context.Context, msg domain.Message, preview string) (domain.Message, bool, error) {
	var created domain.Message
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO crm_messages (id, conversation_id, external_id, direction, content, status)
		 VALUES ($1, $2, $3, 'inbound', $4, 'received')
		 ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO NOTHING
		 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
		msg.ID, msg.ConversationID, msg.ExternalID, msg.Content,
	).Scan(&created.ID, &created.ConversationID, &created.ExternalID, &created.IdempotencyKey,
		&created.Direction, &created.Content, &created.Status, &created.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) && msg.ExternalID != nil {
		var existing domain.Message
		getErr := s.db.QueryRowContext(ctx,
			`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
			 FROM crm_messages WHERE external_id = $1`, *msg.ExternalID,
		).Scan(&existing.ID, &existing.ConversationID, &existing.ExternalID, &existing.IdempotencyKey,
			&existing.Direction, &existing.Content, &existing.Status, &existing.CreatedAt)
		return existing, false, getErr
	}
	if err != nil {
		return domain.Message{}, false, err
	}
	_, _ = s.db.ExecContext(ctx,
		`UPDATE crm_conversations
		 SET last_message_at = $1, last_message_preview = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`, created.CreatedAt, preview, created.ConversationID)
	return created, true, nil
}
