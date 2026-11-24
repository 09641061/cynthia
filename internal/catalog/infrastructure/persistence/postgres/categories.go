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

func (s *Store) CreateCategory(ctx context.Context, name string) (category domain.Category, err error) {
	category.Services = make([]domain.Service, 0)
	err = s.write(ctx, func(tx *sql.Tx) error {
		var position int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) + 1 FROM catalog_categories`).Scan(&position); err != nil {
			return fmt.Errorf("count catalog categories: %w", err)
		}
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate category id: %w", err)
		}
		return tx.QueryRowContext(ctx, `INSERT INTO catalog_categories (id, name, sort_order)
			VALUES ($1, $2, $3) RETURNING id, name, sort_order, version, created_at, updated_at`, id, name, position).
			Scan(&category.ID, &category.Name, &category.SortOrder, &category.Version, &category.CreatedAt, &category.UpdatedAt)
	})
	return category, persistenceError(err)
}

func (s *Store) UpdateCategory(ctx context.Context, id, name string, version int64) (category domain.Category, err error) {
	err = s.write(ctx, func(tx *sql.Tx) error {
		if err := checkCategoryVersion(ctx, tx, id, version); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `UPDATE catalog_categories SET name = $2, version = version + 1,
			updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND version = $3
			RETURNING id, name, sort_order, version, created_at, updated_at`, id, name, version).
			Scan(&category.ID, &category.Name, &category.SortOrder, &category.Version, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return err
		}
		services, err := categoryServices(ctx, tx, id)
		if err != nil {
			return err
		}
		category.Services = services
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		return category, application.ErrNotFound
	}
	return category, persistenceError(err)
}

func (s *Store) DeleteCategory(ctx context.Context, id string, version int64) error {
	err := s.write(ctx, func(tx *sql.Tx) error {
		if err := checkCategoryVersion(ctx, tx, id, version); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM catalog_categories WHERE id = $1 AND version = $2`, id, version)
		if err != nil {
			return fmt.Errorf("delete catalog category: %w", err)
		}
		if err := requireAffected(result); err != nil {
			return err
		}
		ids, err := categoryIDs(ctx, tx)
		if err != nil {
			return err
		}
		return writeCategoryOrder(ctx, tx, ids)
	})
	return persistenceError(err)
}

func (s *Store) MoveCategory(ctx context.Context, id string, position int, version int64) (category domain.Category, err error) {
	err = s.write(ctx, func(tx *sql.Tx) error {
		if err := checkCategoryVersion(ctx, tx, id, version); err != nil {
			return err
		}
		ids, err := categoryIDs(ctx, tx)
		if err != nil {
			return err
		}
		if position < 1 || position > len(ids) {
			return domain.InvalidFieldError{Field: "sort_order", Reason: "must identify a position in the category list"}
		}
		ids = moveWithin(ids, id, position-1)
		if err := writeCategoryOrder(ctx, tx, ids); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT id, name, sort_order, version, created_at, updated_at
			FROM catalog_categories WHERE id = $1`, id).Scan(&category.ID, &category.Name, &category.SortOrder,
			&category.Version, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return err
		}
		category.Services, err = categoryServices(ctx, tx, id)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return category, application.ErrNotFound
	}
	return category, persistenceError(err)
}

func checkCategoryVersion(ctx context.Context, tx *sql.Tx, id string, expected int64) error {
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT version FROM catalog_categories WHERE id = $1`, id).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return application.ErrNotFound
		}
		return fmt.Errorf("read category version: %w", err)
	}
	if current != expected {
		return application.ErrPreconditionFailed
	}
	return nil
}

func requireAffected(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected row count: %w", err)
	}
	if count == 0 {
		return application.ErrNotFound
	}
	return nil
}
