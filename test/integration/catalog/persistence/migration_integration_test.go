//go:build integration

package persistence_test

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestMigrationRemovesDescriptionFromExistingSchema(t *testing.T) {
	db, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	db.SetMaxOpenConns(1)
	ctx := t.Context()
	schema := "catalog_upgrade_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	if _, err := db.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		t.Fatal(err)
	}
	initial, err := os.ReadFile("../../../../migrations/0001_20261124175238_create_catalog_tables.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(initial)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE catalog_schema_migrations (version INTEGER PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP); INSERT INTO catalog_schema_migrations(version) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if err := sharedpostgres.Migrate(ctx, db, "../../../../migrations"); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'catalog_services' AND column_name = 'description')`, schema).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("description column remains after migration")
	}
}
