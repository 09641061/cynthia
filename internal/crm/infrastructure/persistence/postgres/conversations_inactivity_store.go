package postgres

import (
	"context"
	"time"
)

const closeInactiveConversationsSQL = `UPDATE crm_conversations c
SET status = 'closed', updated_at = CURRENT_TIMESTAMP
WHERE c.status = 'open'
  AND GREATEST(COALESCE(c.last_message_at, c.created_at), c.updated_at) < $1
  AND NOT EXISTS (
    SELECT 1 FROM assistant_jobs j
    WHERE j.conversation_id = c.id AND j.status IN ('pending', 'processing')
  )
RETURNING c.id`

// CloseInactiveConversations closes open conversations whose latest activity is before cutoff
// and that have no pending or processing assistant jobs, returning the closed conversation IDs.
func (s *Store) CloseInactiveConversations(ctx context.Context, cutoff time.Time) ([]string, error) {
	var closed []string
	err := s.db.Do(ctx, func() error {
		rows, err := s.db.QueryContext(ctx, closeInactiveConversationsSQL, cutoff)
		if err != nil {
			return err
		}
		defer rows.Close()
		closed = make([]string, 0)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			closed = append(closed, id)
		}
		return rows.Err()
	})
	return closed, err
}
