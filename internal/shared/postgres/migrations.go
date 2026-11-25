package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"os"
	"path/filepath"
)

// Migrate applies pending catalog schema migrations transactionally.
func Migrate(ctx context.Context, db *sql.DB, dir string) (resultErr error) {
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
	for _, migration := range []struct {
		version int
		file    string
	}{
		{1, "0001_20261124175238_create_catalog_tables.sql"},
		{3, "0003_20261124172000_create_organization_tables.sql"},
		{4, "0004_20261124182500_create_crm_tables.sql"},
		{5, "0005_20261125143000_create_calendar_tables.sql"},
		{6, "0006_20261125123000_add_assistant_foundations.sql"},
	} {
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM catalog_schema_migrations WHERE version = $1)`, migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("read catalog migration version: %w", err)
		}
		if applied {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, migration.file))
		if err != nil {
			return fmt.Errorf("read catalog migration: %w", err)
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply catalog migration: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_schema_migrations (version) VALUES ($1)`, migration.version); err != nil {
			return fmt.Errorf("record catalog migration: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog migration: %w", err)
	}
	return nil
}
