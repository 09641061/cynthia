package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
)

func (s *Store) CreateService(ctx context.Context, categoryID string, service domain.Service, position int) (created domain.Service, err error) {
	err = s.write(ctx, func(tx *sql.Tx) error {
		if err := requireCategory(ctx, tx, categoryID); err != nil {
			return err
		}
		ids, err := serviceIDs(ctx, tx, categoryID)
		if err != nil {
			return err
		}
		if position == 0 {
			position = len(ids) + 1
		}
		if position < 1 || position > len(ids)+1 {
			return domain.InvalidFieldError{Field: "sort_order", Reason: "must be within the category list"}
		}
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate service id: %w", err)
		}
		created = service
		created.ID, created.CategoryID, created.SortOrder = id.String(), categoryID, position
		if err := tx.QueryRowContext(ctx, `INSERT INTO catalog_services
			(id, category_id, name, duration_minutes, price_minor_units, sort_order)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING version, created_at, updated_at`, created.ID, categoryID, service.Name,
			service.DurationMinute, service.PriceMinor,
			created.SortOrder).Scan(&created.Version, &created.CreatedAt, &created.UpdatedAt); err != nil {
			return fmt.Errorf("create catalog service: %w", err)
		}
		ids = insertAt(ids, created.ID, position-1)
		return writeServiceOrder(ctx, tx, categoryID, ids)
	})
	return created, persistenceError(err)
}

func (s *Store) UpdateService(ctx context.Context, id string, service domain.Service, version int64) (updated domain.Service, err error) {
	err = s.write(ctx, func(tx *sql.Tx) error {
		if err := checkServiceVersion(ctx, tx, id, version); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, `UPDATE catalog_services SET name = $2,
			duration_minutes = $3, price_minor_units = $4, version = version + 1,
			updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND version = $5
			RETURNING id, category_id, name, duration_minutes, price_minor_units,
			sort_order, version, created_at, updated_at`, id, service.Name,
			service.DurationMinute, service.PriceMinor, version).
			Scan(&updated.ID, &updated.CategoryID, &updated.Name, &updated.DurationMinute,
				&updated.PriceMinor, &updated.SortOrder, &updated.Version,
				&updated.CreatedAt, &updated.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return updated, application.ErrNotFound
	}
	return updated, persistenceError(err)
}

func (s *Store) DeleteService(ctx context.Context, id string, version int64) error {
	return s.write(ctx, func(tx *sql.Tx) error {
		if err := checkServiceVersion(ctx, tx, id, version); err != nil {
			return err
		}
		var categoryID string
		if err := tx.QueryRowContext(ctx, `SELECT category_id FROM catalog_services WHERE id = $1`, id).Scan(&categoryID); err != nil {
			return fmt.Errorf("find catalog service: %w", err)
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM catalog_services WHERE id = $1 AND version = $2`, id, version)
		if err != nil {
			return fmt.Errorf("delete catalog service: %w", err)
		}
		if err := requireAffected(result); err != nil {
			return err
		}
		ids, err := serviceIDs(ctx, tx, categoryID)
		if err != nil {
			return err
		}
		return writeServiceOrder(ctx, tx, categoryID, ids)
	})
}

func checkServiceVersion(ctx context.Context, tx *sql.Tx, id string, expected int64) error {
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT version FROM catalog_services WHERE id = $1`, id).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return application.ErrNotFound
		}
		return fmt.Errorf("read service version: %w", err)
	}
	if current != expected {
		return application.ErrPreconditionFailed
	}
	return nil
}

func requireCategory(ctx context.Context, tx *sql.Tx, id string) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM catalog_categories WHERE id = $1)`, id).Scan(&exists); err != nil {
		return fmt.Errorf("check catalog category: %w", err)
	}
	if !exists {
		return application.ErrNotFound
	}
	return nil
}
