package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type rowScanner interface {
	Scan(dest ...any) error
}

const conversationColumns = `id, customer_id, channel, status, ai_paused, handoff_reason, version, last_message_at, last_message_preview, created_at, updated_at`

func scanConversation(row rowScanner) (domain.Conversation, error) {
	var conv domain.Conversation
	err := row.Scan(
		&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
		&conv.AiPaused, &conv.HandoffReason, &conv.Version,
		&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt,
	)
	return conv, err
}

// UpdateConversationAI updates AI pause state with optimistic concurrency control.
func (s *Store) UpdateConversationAI(ctx context.Context, id string, expectedVersion int64, aiPaused bool, handoffReason string) (domain.Conversation, error) {
	conv, err := scanConversation(s.db.QueryRowContext(ctx,
		`UPDATE crm_conversations
		 SET ai_paused = $1, handoff_reason = $2, version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3 AND version = $4
		 RETURNING `+conversationColumns,
		aiPaused, handoffReason, id, expectedVersion,
	))
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetConversation(ctx, id); getErr != nil {
			return domain.Conversation{}, getErr
		}
		return domain.Conversation{}, application.ErrPreconditionFailed
	}
	return conv, err
}

// SetConversationAIPaused updates AI pause state without optimistic concurrency checks.
func (s *Store) SetConversationAIPaused(ctx context.Context, id string, aiPaused bool, handoffReason string) (domain.Conversation, error) {
	conv, err := scanConversation(s.db.QueryRowContext(ctx,
		`UPDATE crm_conversations
		 SET ai_paused = $2, handoff_reason = $3, version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1
		 RETURNING `+conversationColumns,
		id, aiPaused, handoffReason,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return conv, err
}
