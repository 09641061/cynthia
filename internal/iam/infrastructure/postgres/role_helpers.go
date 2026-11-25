package postgres

import (
	"context"
	"encoding/json"

	"go-service/internal/iam/domain"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRole(scanner rowScanner) (domain.Role, error) {
	var r domain.Role
	var permsJSON string
	if err := scanner.Scan(&r.ID, &r.Name, &r.Position, &permsJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return domain.Role{}, err
	}
	if err := json.Unmarshal([]byte(permsJSON), &r.Permissions); err != nil {
		return domain.Role{}, err
	}
	return r, nil
}

// ReorderRoles atomically updates role positions to 1..N according to orderedIDs.
func (s *RoleStore) ReorderRoles(ctx context.Context, orderedIDs []string) ([]domain.Role, error) {
	err := s.db.Do(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		for idx, id := range orderedIDs {
			if _, err := tx.ExecContext(ctx, `UPDATE iam_roles SET position = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id, idx+1); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
	if err != nil {
		return nil, err
	}
	return s.ListRoles(ctx)
}
