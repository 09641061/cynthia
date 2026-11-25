package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go-service/internal/crm/domain"
)

// UpdateMessageDelivery updates an outbound message's delivery status and refreshes conversation preview atomically.
func (s *Store) UpdateMessageDelivery(ctx context.Context, id, status string, externalID *string, preview string) (domain.Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Message{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var updated domain.Message
	err = tx.QueryRowContext(ctx,
		`UPDATE crm_messages SET status = $1, external_id = COALESCE($2, external_id)
		 WHERE id = $3
		 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
		status, externalID, id,
	).Scan(&updated.ID, &updated.ConversationID, &updated.ExternalID, &updated.IdempotencyKey,
		&updated.Direction, &updated.Content, &updated.Status, &updated.CreatedAt)
	if err != nil {
		return domain.Message{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE crm_conversations
		 SET last_message_at = $1, last_message_preview = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`, updated.CreatedAt, preview, updated.ConversationID,
	); err != nil {
		return domain.Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Message{}, err
	}
	return updated, nil
}

// InsertInboundMessage inserts an inbound message deduplicated globally by external_id and updates conversation metadata atomically.
func (s *Store) InsertInboundMessage(ctx context.Context, msg domain.Message, preview string) (domain.Message, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Message{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var created domain.Message
	err = tx.QueryRowContext(ctx,
		`INSERT INTO crm_messages (id, conversation_id, external_id, direction, content, status)
		 VALUES ($1, $2, $3, 'inbound', $4, 'received')
		 ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO NOTHING
		 RETURNING id, conversation_id, external_id, idempotency_key, direction, content, status, created_at`,
		msg.ID, msg.ConversationID, msg.ExternalID, msg.Content,
	).Scan(&created.ID, &created.ConversationID, &created.ExternalID, &created.IdempotencyKey,
		&created.Direction, &created.Content, &created.Status, &created.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) && msg.ExternalID != nil {
		var existing domain.Message
		getErr := tx.QueryRowContext(ctx,
			`SELECT id, conversation_id, external_id, idempotency_key, direction, content, status, created_at
			 FROM crm_messages WHERE external_id = $1`, *msg.ExternalID,
		).Scan(&existing.ID, &existing.ConversationID, &existing.ExternalID, &existing.IdempotencyKey,
			&existing.Direction, &existing.Content, &existing.Status, &existing.CreatedAt)
		if getErr != nil {
			return domain.Message{}, false, getErr
		}
		if err := tx.Commit(); err != nil {
			return domain.Message{}, false, err
		}
		return existing, false, nil
	}
	if err != nil {
		return domain.Message{}, false, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE crm_conversations
		 SET last_message_at = $1, last_message_preview = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`, created.CreatedAt, preview, created.ConversationID,
	); err != nil {
		return domain.Message{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Message{}, false, err
	}
	return created, true, nil
}
