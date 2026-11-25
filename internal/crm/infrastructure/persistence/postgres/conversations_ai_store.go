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

const conversationColumns = `id, customer_id, channel, status, ai_paused, handoff_reason, last_message_at, last_message_preview, created_at, updated_at`

func scanConversation(row rowScanner) (domain.Conversation, error) {
	var conv domain.Conversation
	err := row.Scan(
		&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
		&conv.AiPaused, &conv.HandoffReason,
		&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt,
	)
	return conv, err
}

func (s *Store) queryConversation(ctx context.Context, query string, args ...any) (domain.Conversation, error) {
	var conv domain.Conversation
	err := s.db.Do(ctx, func() error {
		var err error
		conv, err = scanConversation(s.db.QueryRowContext(ctx, query, args...))
		return err
	})
	return conv, err
}

// UpdateConversationAI updates AI pause state for a conversation.
func (s *Store) UpdateConversationAI(ctx context.Context, id string, aiPaused bool, handoffReason string) (domain.Conversation, error) {
	conv, err := s.queryConversation(ctx,
		`UPDATE crm_conversations
		 SET ai_paused = $1, handoff_reason = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3
		 RETURNING `+conversationColumns,
		aiPaused, handoffReason, id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return conv, err
}

// SetConversationAIPaused updates AI pause state for a conversation.
func (s *Store) SetConversationAIPaused(ctx context.Context, id string, aiPaused bool, handoffReason string) (domain.Conversation, error) {
	return s.UpdateConversationAI(ctx, id, aiPaused, handoffReason)
}
