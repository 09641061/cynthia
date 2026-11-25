package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/iam/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

// RoleStore persists IAM roles and hierarchy positions in PostgreSQL.
type RoleStore struct{ db *sharedpostgres.Client }

// NewRoleStore creates a PostgreSQL-backed RoleStore.
func NewRoleStore(db *sharedpostgres.Client) *RoleStore { return &RoleStore{db: db} }

const roleCols = `id, name, position, COALESCE(array_to_json(permissions)::text, '[]'), created_at, updated_at`

// ListRoles returns all roles ordered by position ascending.
func (s *RoleStore) ListRoles(ctx context.Context) ([]domain.Role, error) {
	var roles []domain.Role
	err := s.db.Do(ctx, func() error {
		rows, err := s.db.QueryContext(ctx, `SELECT `+roleCols+` FROM iam_roles ORDER BY position ASC, id ASC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		roles = make([]domain.Role, 0)
		for rows.Next() {
			r, err := scanRole(rows)
			if err != nil {
				return err
			}
			roles = append(roles, r)
		}
		return rows.Err()
	})
	return roles, err
}

// GetRole fetches a single role by ID.
func (s *RoleStore) GetRole(ctx context.Context, roleID string) (domain.Role, error) {
	var r domain.Role
	err := s.db.Do(ctx, func() error {
		var err error
		r, err = scanRole(s.db.QueryRowContext(ctx, `SELECT `+roleCols+` FROM iam_roles WHERE id = $1`, roleID))
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	return r, err
}

// CountRoles returns the total number of active roles.
func (s *RoleStore) CountRoles(ctx context.Context) (int, error) {
	var count int
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM iam_roles`).Scan(&count)
	})
	return count, err
}

// CreateRole inserts a new role at the bottom of the hierarchy (MAX(position) + 1).
func (s *RoleStore) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	var out domain.Role
	err := s.db.Do(ctx, func() error {
		var err error
		out, err = scanRole(s.db.QueryRowContext(ctx, `
			INSERT INTO iam_roles (id, name, position, permissions)
			VALUES ($1, $2, COALESCE((SELECT MAX(position) FROM iam_roles), 0) + 1, $3::text[])
			RETURNING `+roleCols,
			role.ID, role.Name, role.Permissions,
		))
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Role{}, domain.ErrDuplicateRoleName
		}
		return domain.Role{}, err
	}
	return out, nil
}

// UpdateRole updates a role's name and permissions.
func (s *RoleStore) UpdateRole(ctx context.Context, roleID, name string, perms []string) (domain.Role, error) {
	var out domain.Role
	err := s.db.Do(ctx, func() error {
		var err error
		out, err = scanRole(s.db.QueryRowContext(ctx, `
			UPDATE iam_roles SET name = $2, permissions = $3::text[], updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING `+roleCols,
			roleID, name, perms,
		))
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Role{}, domain.ErrDuplicateRoleName
		}
		return domain.Role{}, err
	}
	return out, nil
}

// DeleteRole deletes a role by ID.
func (s *RoleStore) DeleteRole(ctx context.Context, roleID string) error {
	var rowsAffected int64
	err := s.db.Do(ctx, func() error {
		res, err := s.db.ExecContext(ctx, `DELETE FROM iam_roles WHERE id = $1`, roleID)
		if err != nil {
			return err
		}
		rowsAffected, _ = res.RowsAffected()
		return nil
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}
