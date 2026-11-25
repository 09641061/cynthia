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
		&org.ID, &org.Name, &org.BusinessType, &org.ContactEmail, &org.Timezone, &org.CurrencyCode,
		&hoursJSON, &org.AiInstructions,
		&org.WhatsAppPhoneNumber, &org.WhatsAppPhoneNumberID, &org.WhatsAppEnabled,
		&org.CreatedAt, &org.UpdatedAt,
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
	row := s.db.QueryRowContext(ctx, `SELECT id, name, business_type, contact_email, timezone, currency_code,
		business_hours, ai_instructions, whatsapp_phone_number, whatsapp_phone_number_id,
		whatsapp_enabled, created_at, updated_at
		FROM organizations WHERE singleton_key = TRUE`)
	return scanOrganization(row)
}

func (s *Store) update(ctx context.Context, patch domain.UpdatePatch) (result domain.Organization, resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("begin organization transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback organization transaction: %w", err))
		}
	}()
	row := tx.QueryRowContext(ctx, `SELECT id, name, business_type, contact_email, timezone, currency_code,
		business_hours, ai_instructions, whatsapp_phone_number, whatsapp_phone_number_id,
		whatsapp_enabled, created_at, updated_at
		FROM organizations WHERE singleton_key = TRUE FOR UPDATE`)
	current, err := scanOrganization(row)
	if err != nil {
		return domain.Organization{}, err
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
		SET name = $1, business_type = $2, contact_email = $3, timezone = $4, currency_code = $5,
		    business_hours = $6, ai_instructions = $7,
		    whatsapp_phone_number = $8, whatsapp_phone_number_id = $9, whatsapp_enabled = $10,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $11
		RETURNING updated_at`,
		next.Name, next.BusinessType, next.ContactEmail, next.Timezone, next.CurrencyCode,
		hoursJSON, next.AiInstructions,
		next.WhatsAppPhoneNumber, next.WhatsAppPhoneNumberID, next.WhatsAppEnabled,
		next.ID,
	).Scan(&next.UpdatedAt)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("update organization: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Organization{}, fmt.Errorf("commit organization transaction: %w", err)
	}
	return next, nil
}
