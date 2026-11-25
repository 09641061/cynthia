package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"go-service/internal/iam/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

const userCols = `id, email, first_name, last_name, password_hash, type, created_at, updated_at`

// UserStore persists IAM users and role memberships in PostgreSQL.
type UserStore struct{ db *sharedpostgres.Client }

// NewUserStore creates a PostgreSQL-backed UserStore.
func NewUserStore(db *sharedpostgres.Client) *UserStore { return &UserStore{db: db} }

// GetUserByEmail fetches a user by normalized email.
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM iam_users WHERE email = $1`, email).
			Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &u.Type, &u.CreatedAt, &u.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, err
}

// GetActorByID fetches a user and all assigned roles ordered by role position.
func (s *UserStore) GetActorByID(ctx context.Context, userID string) (domain.Actor, error) {
	var u domain.User
	var roles []domain.Role
	err := s.db.Do(ctx, func() error {
		if err := s.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM iam_users WHERE id = $1`, userID).
			Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &u.Type, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		var err error
		roles, err = s.loadUserRoles(ctx, userID)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Actor{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.Actor{}, err
	}
	return domain.Actor{User: u, Roles: roles}, nil
}

// ListActors returns all users (Owner first, then by name) with their assigned roles.
func (s *UserStore) ListActors(ctx context.Context) ([]domain.Actor, error) {
	var actors []domain.Actor
	err := s.db.Do(ctx, func() error {
		rows, err := s.db.QueryContext(ctx, `
			SELECT `+userCols+` FROM iam_users ORDER BY (type = 'owner') DESC, first_name ASC, last_name ASC, id ASC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		actors = make([]domain.Actor, 0)
		byID := make(map[string]int)
		for rows.Next() {
			var u domain.User
			if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &u.Type, &u.CreatedAt, &u.UpdatedAt); err != nil {
				return err
			}
			byID[u.ID] = len(actors)
			actors = append(actors, domain.Actor{User: u, Roles: []domain.Role{}})
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return s.attachActorRoles(ctx, actors, byID)
	})
	return actors, err
}

func (s *UserStore) attachActorRoles(ctx context.Context, actors []domain.Actor, byID map[string]int) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT mr.user_id, r.id, r.name, r.position, COALESCE(array_to_json(r.permissions)::text, '[]'), r.created_at, r.updated_at
		FROM iam_member_roles mr
		JOIN iam_roles r ON r.id = mr.role_id
		ORDER BY r.position ASC, r.id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, permsJSON string
		var r domain.Role
		if err := rows.Scan(&uid, &r.ID, &r.Name, &r.Position, &permsJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(permsJSON), &r.Permissions); err != nil {
			return err
		}
		if idx, ok := byID[uid]; ok {
			actors[idx].Roles = append(actors[idx].Roles, r)
		}
	}
	return rows.Err()
}

func (s *UserStore) loadUserRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.name, r.position, COALESCE(array_to_json(r.permissions)::text, '[]'), r.created_at, r.updated_at
		FROM iam_member_roles mr
		JOIN iam_roles r ON r.id = mr.role_id
		WHERE mr.user_id = $1
		ORDER BY r.position ASC, r.id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := make([]domain.Role, 0)
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}
