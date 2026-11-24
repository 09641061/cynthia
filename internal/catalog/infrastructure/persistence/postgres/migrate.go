package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-service/migrations"
)

// Migrate applies pending catalog schema migrations transactionally.
func Migrate(ctx context.Context, db *sql.DB) (resultErr error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog migration: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback catalog migration: %w", err))
		}
	}()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(70616, 1)`); err != nil {
		return fmt.Errorf("lock catalog migrations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS catalog_schema_migrations (
		version INTEGER PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create catalog migration table: %w", err)
	}
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM catalog_schema_migrations WHERE version = 1
	)`).Scan(&applied); err != nil {
		return fmt.Errorf("read catalog migration version: %w", err)
	}
	if !applied {
		migration, err := migrations.FS.ReadFile("0001_20261124175238_create_catalog_tables.sql")
		if err != nil {
			return fmt.Errorf("read catalog migration: %w", err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply catalog migration: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_schema_migrations (version) VALUES (1)`); err != nil {
			return fmt.Errorf("record catalog migration: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog migration: %w", err)
	}
	return nil
}
