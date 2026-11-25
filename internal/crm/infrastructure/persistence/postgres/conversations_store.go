package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

// FindOrCreateCustomerByPhone atomically inserts a customer if absent and returns the persisted record.
func (s *Store) FindOrCreateCustomerByPhone(ctx context.Context, id, name, phone string) (domain.Customer, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO crm_customers (id, name, phone, version) VALUES ($1, $2, $3, 1)
		 ON CONFLICT (phone) DO NOTHING`,
		id, name, phone,
	)
	if err != nil {
		return domain.Customer{}, err
	}
	return s.GetCustomerByPhone(ctx, phone)
}

// ListConversations returns paginated conversations ordered by most recent activity.
func (s *Store) ListConversations(ctx context.Context, filter application.ConversationFilter) ([]domain.Conversation, int, error) {
	where, args := buildConversationFilter(filter)
	var total int
	countSQL := `SELECT COUNT(*) FROM crm_conversations c JOIN crm_customers u ON u.id = c.customer_id ` + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)
	limitIdx, offsetIdx := len(args)-1, len(args)
	querySQL := `SELECT c.id, c.customer_id, c.channel, c.status, c.ai_paused, c.handoff_reason, c.version,
		c.last_message_at, c.last_message_preview, c.created_at, c.updated_at
		FROM crm_conversations c JOIN crm_customers u ON u.id = c.customer_id ` + where +
		` ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC
		 LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx)
	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]domain.Conversation, 0)
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, conv)
	}
	return items, total, rows.Err()
}

// GetConversation fetches a conversation by ID.
func (s *Store) GetConversation(ctx context.Context, id string) (domain.Conversation, error) {
	conv, err := scanConversation(s.db.QueryRowContext(ctx,
		`SELECT `+conversationColumns+` FROM crm_conversations WHERE id = $1`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return conv, err
}

// UpdateConversationStatus updates a conversation's status using optimistic concurrency.
func (s *Store) UpdateConversationStatus(ctx context.Context, id string, expectedVersion int64, status string) (domain.Conversation, error) {
	conv, err := scanConversation(s.db.QueryRowContext(ctx,
		`UPDATE crm_conversations
		 SET status = $1,
		     ai_paused = CASE WHEN status = 'closed' AND $1 = 'open' THEN FALSE ELSE ai_paused END,
		     handoff_reason = CASE WHEN status = 'closed' AND $1 = 'open' THEN '' ELSE handoff_reason END,
		     version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2 AND version = $3
		 RETURNING `+conversationColumns,
		status, id, expectedVersion,
	))
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Conversation{}, application.ErrConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetConversation(ctx, id); getErr != nil {
			return domain.Conversation{}, getErr
		}
		return domain.Conversation{}, application.ErrPreconditionFailed
	}
	return conv, err
}

// EnsureOpenConversation returns an open conversation for the customer, reopening the latest closed one or creating a new one.
func (s *Store) EnsureOpenConversation(ctx context.Context, customerID string) (domain.Conversation, error) {
	const maxAttempts = 3
	for range maxAttempts {
		if err := ctx.Err(); err != nil {
			return domain.Conversation{}, err
		}
		conv, err := scanConversation(s.db.QueryRowContext(ctx,
			`SELECT `+conversationColumns+` FROM crm_conversations WHERE customer_id = $1 AND status = 'open' LIMIT 1`, customerID,
		))
		if err == nil {
			return conv, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.Conversation{}, err
		}
		conv, err = scanConversation(s.db.QueryRowContext(ctx,
			`UPDATE crm_conversations
			 SET status = 'open', ai_paused = FALSE, handoff_reason = '', version = version + 1, updated_at = CURRENT_TIMESTAMP
			 WHERE id = (
				SELECT id FROM crm_conversations WHERE customer_id = $1 AND status = 'closed'
				ORDER BY last_message_at DESC NULLS LAST, created_at DESC LIMIT 1
			 ) AND status = 'closed'
			 RETURNING `+conversationColumns,
			customerID,
		))
		if err == nil {
			return conv, nil
		}
		if !errors.Is(err, sql.ErrNoRows) && !isPGCode(err, pgErrUniqueViolation) {
			return domain.Conversation{}, err
		}
		conv, err = scanConversation(s.db.QueryRowContext(ctx,
			`INSERT INTO crm_conversations (id, customer_id, channel, status, version)
			 VALUES ($1, $2, 'whatsapp', 'open', 1)
			 RETURNING `+conversationColumns,
			uuid.NewString(), customerID,
		))
		if isPGCode(err, pgErrUniqueViolation) {
			continue
		}
		return conv, err
	}
	return domain.Conversation{}, application.ErrConflict
}
