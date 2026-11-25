package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"go-service/internal/organizations/application"
	"go-service/internal/organizations/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

// Store persists the singleton organization record in PostgreSQL.
type Store struct{ db *sharedpostgres.Client }

// NewStore creates a PostgreSQL organization store.
func NewStore(db *sharedpostgres.Client) *Store { return &Store{db: db} }

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrganization(row rowScanner) (domain.Organization, error) {
	var org domain.Organization
	var hoursJSON []byte
	err := row.Scan(
		&org.ID, &org.Name, &org.BusinessType, &org.ContactEmail, &org.Timezone,
		&hoursJSON, &org.AiInstructions, &org.Version, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Organization{}, application.ErrNotFound
		}
		return domain.Organization{}, fmt.Errorf("scan organization: %w", err)
	}
	if err := json.Unmarshal(hoursJSON, &org.BusinessHours); err != nil {
		return domain.Organization{}, fmt.Errorf("unmarshal business hours: %w", err)
	}
	return org, nil
}

func (s *Store) get(ctx context.Context) (domain.Organization, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, business_type, contact_email, timezone,
		business_hours, ai_instructions, version, created_at, updated_at
		FROM organizations WHERE singleton_key = TRUE`)
	return scanOrganization(row)
}

func (s *Store) update(ctx context.Context, expectedVersion int64, patch domain.UpdatePatch) (result domain.Organization, resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("begin organization transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback organization transaction: %w", err))
		}
	}()
	row := tx.QueryRowContext(ctx, `SELECT id, name, business_type, contact_email, timezone,
		business_hours, ai_instructions, version, created_at, updated_at
		FROM organizations WHERE singleton_key = TRUE FOR UPDATE`)
	current, err := scanOrganization(row)
	if err != nil {
		return domain.Organization{}, err
	}
	if current.Version != expectedVersion {
		return domain.Organization{}, application.ErrPreconditionFailed
	}
	next, err := domain.ApplyPatch(current, patch)
	if err != nil {
		return domain.Organization{}, err
	}
	hoursJSON, err := json.Marshal(next.BusinessHours)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("marshal business hours: %w", err)
	}
	err = tx.QueryRowContext(ctx, `UPDATE organizations
		SET name = $1, business_type = $2, contact_email = $3, timezone = $4,
		    business_hours = $5, ai_instructions = $6, version = version + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
		RETURNING version, updated_at`,
		next.Name, next.BusinessType, next.ContactEmail, next.Timezone,
		hoursJSON, next.AiInstructions, next.ID,
	).Scan(&next.Version, &next.UpdatedAt)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("update organization: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Organization{}, fmt.Errorf("commit organization transaction: %w", err)
	}
	return next, nil
}
