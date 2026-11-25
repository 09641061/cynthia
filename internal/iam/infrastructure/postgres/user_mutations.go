package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/iam/domain"
)

// CreateUser inserts a new IAM user.
func (s *UserStore) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	userType := u.Type
	if userType == "" {
		userType = domain.UserTypeUser
	}
	var out domain.User
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx, `
			INSERT INTO iam_users (id, email, first_name, last_name, password_hash, type)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING `+userCols,
			u.ID, u.Email, u.FirstName, u.LastName, u.PasswordHash, userType,
		).Scan(&out.ID, &out.Email, &out.FirstName, &out.LastName, &out.PasswordHash, &out.Type, &out.CreatedAt, &out.UpdatedAt)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.ErrDuplicateEmail
		}
		return domain.User{}, err
	}
	return out, nil
}

// UpdateProfile updates first_name, last_name, and optionally password_hash.
func (s *UserStore) UpdateProfile(ctx context.Context, userID, firstName, lastName, passwordHash string) (domain.User, error) {
	var out domain.User
	err := s.db.Do(ctx, func() error {
		if passwordHash != "" {
			return s.db.QueryRowContext(ctx, `
				UPDATE iam_users SET first_name = $2, last_name = $3, password_hash = $4, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
				RETURNING `+userCols,
				userID, firstName, lastName, passwordHash,
			).Scan(&out.ID, &out.Email, &out.FirstName, &out.LastName, &out.PasswordHash, &out.Type, &out.CreatedAt, &out.UpdatedAt)
		}
		return s.db.QueryRowContext(ctx, `
			UPDATE iam_users SET first_name = $2, last_name = $3, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING `+userCols,
			userID, firstName, lastName,
		).Scan(&out.ID, &out.Email, &out.FirstName, &out.LastName, &out.PasswordHash, &out.Type, &out.CreatedAt, &out.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return out, err
}

// DeleteMember deletes a non-owner user.
func (s *UserStore) DeleteMember(ctx context.Context, userID string) error {
	var rowsAffected int64
	err := s.db.Do(ctx, func() error {
		res, err := s.db.ExecContext(ctx, `DELETE FROM iam_users WHERE id = $1 AND type = 'user'`, userID)
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
		return domain.ErrUserNotFound
	}
	return nil
}

// ReplaceMemberRoles atomically replaces all role memberships for userID.
func (s *UserStore) ReplaceMemberRoles(ctx context.Context, userID string, roleIDs []string) (domain.Actor, error) {
	err := s.db.Do(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()

		res, err := tx.ExecContext(ctx, `UPDATE iam_users SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`, userID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return domain.ErrUserNotFound
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM iam_member_roles WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for _, rid := range roleIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO iam_member_roles (user_id, role_id) VALUES ($1, $2)`, userID, rid); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
	if err != nil {
		return domain.Actor{}, err
	}
	return s.GetActorByID(ctx, userID)
}
