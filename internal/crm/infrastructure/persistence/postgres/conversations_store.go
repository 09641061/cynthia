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

const convCols = `id, customer_id, channel, status, last_message_at, last_message_preview, created_at, updated_at`

// FindOrCreateCustomerByPhone atomically inserts a customer if absent and returns the persisted record.
func (s *Store) FindOrCreateCustomerByPhone(ctx context.Context, id, name, phone string) (domain.Customer, error) {
	err := s.db.Do(ctx, func() error {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO crm_customers (id, name, phone) VALUES ($1, $2, $3) ON CONFLICT (phone) DO NOTHING`,
			id, name, phone,
		)
		return err
	})
	if err != nil {
		return domain.Customer{}, err
	}
	return s.GetCustomerByPhone(ctx, phone)
}

// ListConversations returns paginated conversations ordered by most recent activity.
func (s *Store) ListConversations(ctx context.Context, filter application.ConversationFilter) ([]domain.Conversation, int, error) {
	where, args := buildConversationFilter(filter)
	var total int
	var items []domain.Conversation
	err := s.db.Do(ctx, func() error {
		countSQL := `SELECT COUNT(*) FROM crm_conversations c JOIN crm_customers u ON u.id = c.customer_id ` + where
		if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
			return err
		}
		offset := (filter.Page - 1) * filter.PageSize
		qArgs := append(append([]any(nil), args...), filter.PageSize, offset)
		limitIdx, offsetIdx := len(qArgs)-1, len(qArgs)
		querySQL := `SELECT c.id, c.customer_id, c.channel, c.status,
			c.last_message_at, c.last_message_preview, c.created_at, c.updated_at
			FROM crm_conversations c JOIN crm_customers u ON u.id = c.customer_id ` + where +
			` ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC
			 LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx)
		rows, err := s.db.QueryContext(ctx, querySQL, qArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		items = make([]domain.Conversation, 0)
		for rows.Next() {
			var conv domain.Conversation
			if err := rows.Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
				&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
				return err
			}
			items = append(items, conv)
		}
		return rows.Err()
	})
	return items, total, err
}

// GetConversation fetches a conversation by ID.
func (s *Store) GetConversation(ctx context.Context, id string) (domain.Conversation, error) {
	var conv domain.Conversation
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx, `SELECT `+convCols+` FROM crm_conversations WHERE id = $1`, id).
			Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
				&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return conv, err
}

// UpdateConversationStatus updates a conversation's status.
func (s *Store) UpdateConversationStatus(ctx context.Context, id, status string) (domain.Conversation, error) {
	var conv domain.Conversation
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`UPDATE crm_conversations SET status = $1, updated_at = CURRENT_TIMESTAMP
			 WHERE id = $2 RETURNING `+convCols, status, id,
		).Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
			&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt)
	})
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Conversation{}, application.ErrConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return conv, err
}

// EnsureOpenConversation returns an open conversation for the customer, reopening the latest closed one or creating a new one.
func (s *Store) EnsureOpenConversation(ctx context.Context, customerID string) (domain.Conversation, error) {
	for range 3 {
		if err := ctx.Err(); err != nil {
			return domain.Conversation{}, err
		}
		var conv domain.Conversation
		err := s.db.Do(ctx, func() error {
			return s.db.QueryRowContext(ctx,
				`SELECT `+convCols+` FROM crm_conversations WHERE customer_id = $1 AND status = 'open' LIMIT 1`, customerID,
			).Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
				&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt)
		})
		if err == nil {
			return conv, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.Conversation{}, err
		}
		err = s.db.Do(ctx, func() error {
			return s.db.QueryRowContext(ctx,
				`UPDATE crm_conversations SET status = 'open', updated_at = CURRENT_TIMESTAMP
				 WHERE id = (
					SELECT id FROM crm_conversations WHERE customer_id = $1 AND status = 'closed'
					ORDER BY last_message_at DESC NULLS LAST, created_at DESC LIMIT 1
				 ) AND status = 'closed' RETURNING `+convCols, customerID,
			).Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
				&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt)
		})
		if err == nil {
			return conv, nil
		}
		if !errors.Is(err, sql.ErrNoRows) && !isPGCode(err, pgErrUniqueViolation) {
			return domain.Conversation{}, err
		}
		err = s.db.Do(ctx, func() error {
			return s.db.QueryRowContext(ctx,
				`INSERT INTO crm_conversations (id, customer_id, channel, status)
				 VALUES ($1, $2, 'whatsapp', 'open') RETURNING `+convCols,
				uuid.NewString(), customerID,
			).Scan(&conv.ID, &conv.CustomerID, &conv.Channel, &conv.Status,
				&conv.LastMessageAt, &conv.LastMessagePreview, &conv.CreatedAt, &conv.UpdatedAt)
		})
		if isPGCode(err, pgErrUniqueViolation) {
			continue
		}
		return conv, err
	}
	return domain.Conversation{}, application.ErrConflict
}
