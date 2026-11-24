package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
)

func (s *Store) MoveService(ctx context.Context, id, categoryID string, position int, version int64) (moved domain.Service, err error) {
	err = s.write(ctx, func(tx *sql.Tx) error {
		if err := checkServiceVersion(ctx, tx, id, version); err != nil {
			return err
		}
		var sourceID string
		if err := tx.QueryRowContext(ctx, `SELECT category_id FROM catalog_services WHERE id = $1`, id).Scan(&sourceID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return application.ErrNotFound
			}
			return fmt.Errorf("find service to move: %w", err)
		}
		if err := requireCategory(ctx, tx, categoryID); err != nil {
			return err
		}
		source, err := serviceIDs(ctx, tx, sourceID)
		if err != nil {
			return err
		}
		destination, err := serviceIDs(ctx, tx, categoryID)
		if err != nil {
			return err
		}
		if sourceID == categoryID {
			if position < 1 || position > len(source) {
				return domain.InvalidFieldError{Field: "sort_order", Reason: "must identify a position in the category list"}
			}
			source = moveWithin(source, id, position-1)
			if err := writeServiceOrder(ctx, tx, sourceID, source); err != nil {
				return err
			}
		} else {
			if position < 1 || position > len(destination)+1 {
				return domain.InvalidFieldError{Field: "sort_order", Reason: "must be within the destination category list"}
			}
			destination = insertAt(destination, id, position-1)
			source = removeID(source, id)
			if err := writeServiceOrder(ctx, tx, sourceID, source); err != nil {
				return err
			}
			if err := writeServiceOrder(ctx, tx, categoryID, destination); err != nil {
				return err
			}
		}
		return tx.QueryRowContext(ctx, `SELECT id, category_id, name, description, duration_minutes,
			price_minor_units, currency_code, sort_order, version, created_at, updated_at FROM catalog_services
			WHERE id = $1`, id).Scan(&moved.ID, &moved.CategoryID, &moved.Name, &moved.Description,
			&moved.DurationMinute, &moved.PriceMinor, &moved.CurrencyCode, &moved.SortOrder, &moved.Version,
			&moved.CreatedAt, &moved.UpdatedAt)
	})
	return moved, persistenceError(err)
}
