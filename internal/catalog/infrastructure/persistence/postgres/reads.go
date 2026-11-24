package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
)

func (s *Store) List(ctx context.Context) (categories []domain.Category, resultErr error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, fmt.Errorf("begin catalog read: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback catalog read: %w", err))
		}
	}()
	rows, err := tx.QueryContext(ctx, `SELECT id, name, sort_order, version, created_at, updated_at
		FROM catalog_categories ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("list catalog categories: %w", err)
	}
	categories = make([]domain.Category, 0)
	indexes := make(map[string]int)
	for rows.Next() {
		var category domain.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.SortOrder, &category.Version,
			&category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, errors.Join(fmt.Errorf("scan catalog category: %w", err), rows.Close())
		}
		category.Services = make([]domain.Service, 0)
		indexes[category.ID] = len(categories)
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Join(fmt.Errorf("read catalog categories: %w", err), rows.Close())
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close catalog categories: %w", err)
	}
	if err := listServices(ctx, tx, indexes, categories); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit catalog read: %w", err)
	}
	return categories, nil
}

func listServices(ctx context.Context, tx *sql.Tx, indexes map[string]int, categories []domain.Category) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, category_id, name, duration_minutes,
		price_minor_units, currency_code, sort_order, version, created_at, updated_at
		FROM catalog_services ORDER BY category_id, sort_order`)
	if err != nil {
		return fmt.Errorf("list catalog services: %w", err)
	}
	for rows.Next() {
		var service domain.Service
		if err := rows.Scan(&service.ID, &service.CategoryID, &service.Name,
			&service.DurationMinute, &service.PriceMinor, &service.CurrencyCode, &service.SortOrder,
			&service.Version, &service.CreatedAt, &service.UpdatedAt); err != nil {
			return errors.Join(fmt.Errorf("scan catalog service: %w", err), rows.Close())
		}
		if index, exists := indexes[service.CategoryID]; exists {
			categories[index].Services = append(categories[index].Services, service)
		}
	}
	if err := rows.Err(); err != nil {
		return errors.Join(fmt.Errorf("read catalog services: %w", err), rows.Close())
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close catalog services: %w", err)
	}
	return nil
}

func (s *Store) GetCategory(ctx context.Context, id string) (category domain.Category, resultErr error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return category, fmt.Errorf("begin category read: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback category read: %w", err))
		}
	}()
	if err := tx.QueryRowContext(ctx, `SELECT id, name, sort_order, version, created_at, updated_at
		FROM catalog_categories WHERE id = $1`, id).Scan(&category.ID, &category.Name, &category.SortOrder,
		&category.Version, &category.CreatedAt, &category.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return category, application.ErrNotFound
		}
		return category, fmt.Errorf("read category: %w", err)
	}
	category.Services, err = categoryServices(ctx, tx, id)
	if err != nil {
		return category, err
	}
	if err := tx.Commit(); err != nil {
		return category, fmt.Errorf("commit category read: %w", err)
	}
	return category, nil
}

func (s *Store) GetService(ctx context.Context, id string) (domain.Service, error) {
	var service domain.Service
	err := s.db.QueryRowContext(ctx, `SELECT id, category_id, name, duration_minutes,
		price_minor_units, currency_code, sort_order, version, created_at, updated_at
		FROM catalog_services WHERE id = $1`, id).Scan(&service.ID, &service.CategoryID, &service.Name,
		&service.DurationMinute, &service.PriceMinor, &service.CurrencyCode,
		&service.SortOrder, &service.Version, &service.CreatedAt, &service.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service, application.ErrNotFound
	}
	if err != nil {
		return service, fmt.Errorf("read service: %w", err)
	}
	return service, nil
}
