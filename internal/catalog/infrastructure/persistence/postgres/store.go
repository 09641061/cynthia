package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
)

// Store persists catalog records in PostgreSQL.
type Store struct{ db *sql.DB }

// NewStore creates a PostgreSQL catalog store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func categoryServices(ctx context.Context, tx *sql.Tx, categoryID string) ([]domain.Service, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, category_id, name, description, duration_minutes,
		price_minor_units, currency_code, sort_order, version, created_at, updated_at
		FROM catalog_services WHERE category_id = $1 ORDER BY sort_order`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list category services: %w", err)
	}
	services := make([]domain.Service, 0)
	for rows.Next() {
		var service domain.Service
		if err := rows.Scan(&service.ID, &service.CategoryID, &service.Name, &service.Description,
			&service.DurationMinute, &service.PriceMinor, &service.CurrencyCode, &service.SortOrder,
			&service.Version, &service.CreatedAt, &service.UpdatedAt); err != nil {
			return nil, errors.Join(fmt.Errorf("scan category service: %w", err), rows.Close())
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Join(fmt.Errorf("read category services: %w", err), rows.Close())
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close category services: %w", err)
	}
	return services, nil
}

func (s *Store) write(ctx context.Context, action func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog transaction: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE catalog_categories, catalog_services IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return errors.Join(fmt.Errorf("lock catalog tables: %w", err), tx.Rollback())
	}
	if err := action(tx); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog transaction: %w", err)
	}
	return nil
}

func persistenceError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23503") {
		return fmt.Errorf("%w: %s", application.ErrConflict, pgErr.ConstraintName)
	}
	return err
}
