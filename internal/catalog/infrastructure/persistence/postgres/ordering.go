package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func categoryIDs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM catalog_categories ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("read category order: %w", err)
	}
	return readIDs(rows, "category order")
}

func serviceIDs(ctx context.Context, tx *sql.Tx, categoryID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM catalog_services
		WHERE category_id = $1 ORDER BY sort_order`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("read service order: %w", err)
	}
	return readIDs(rows, "service order")
}

func readIDs(rows *sql.Rows, label string) (ids []string, resultErr error) {
	ids = make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			resultErr = fmt.Errorf("scan %s: %w", label, err)
			break
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("read %s: %w", label, err))
	}
	if err := rows.Close(); err != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("close %s: %w", label, err))
	}
	return ids, resultErr
}

func writeCategoryOrder(ctx context.Context, tx *sql.Tx, ids []string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, sort_order FROM catalog_categories`)
	if err != nil {
		return fmt.Errorf("read category positions: %w", err)
	}
	positions, err := readPositions(rows, "category positions")
	if err != nil {
		return err
	}
	for position, id := range ids {
		if positions[id] == position+1 {
			continue
		}
		result, err := tx.ExecContext(ctx, `UPDATE catalog_categories SET sort_order = $2, version = version + 1,
			updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id, position+1)
		if err != nil {
			return fmt.Errorf("write category order: %w", err)
		}
		if err := requireAffected(result); err != nil {
			return err
		}
	}
	return nil
}

func writeServiceOrder(ctx context.Context, tx *sql.Tx, categoryID string, ids []string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, sort_order FROM catalog_services WHERE category_id = $1`, categoryID)
	if err != nil {
		return fmt.Errorf("read service positions: %w", err)
	}
	positions, err := readPositions(rows, "service positions")
	if err != nil {
		return err
	}
	for position, id := range ids {
		if positions[id] == position+1 {
			continue
		}
		result, err := tx.ExecContext(ctx, `UPDATE catalog_services SET category_id = $2, sort_order = $3, version = version + 1,
			updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id, categoryID, position+1)
		if err != nil {
			return fmt.Errorf("write service order: %w", err)
		}
		if err := requireAffected(result); err != nil {
			return err
		}
	}
	return nil
}

func readPositions(rows *sql.Rows, label string) (map[string]int, error) {
	positions := make(map[string]int)
	for rows.Next() {
		var id string
		var position int
		if err := rows.Scan(&id, &position); err != nil {
			return nil, errors.Join(fmt.Errorf("scan %s: %w", label, err), rows.Close())
		}
		positions[id] = position
	}
	if err := rows.Err(); err != nil {
		closeErr := rows.Close()
		return nil, errors.Join(fmt.Errorf("read %s: %w", label, err), closeErr)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close %s: %w", label, err)
	}
	return positions, nil
}

func insertAt(values []string, value string, position int) []string {
	values = append(values, "")
	copy(values[position+1:], values[position:])
	values[position] = value
	return values
}

func removeID(values []string, id string) []string {
	for index, value := range values {
		if value == id {
			return append(values[:index], values[index+1:]...)
		}
	}
	return values
}

func moveWithin(values []string, id string, position int) []string {
	return insertAt(removeID(values, id), id, position)
}
