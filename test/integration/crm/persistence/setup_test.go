//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go-service/internal/crm/infrastructure/persistence/postgres"
	sharedpostgres "go-service/internal/shared/postgres"
)

func newTestStore(t *testing.T) (*postgres.Store, *sql.DB) {
	t.Helper()
	parsed, err := url.Parse(testDatabaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		t.Fatal("Testcontainers did not provide a PostgreSQL connection URL")
	}
	admin, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatalf("open PostgreSQL connection: %v", err)
	}
	ctx := t.Context()
	if err := admin.PingContext(ctx); err != nil {
		_ = admin.Close()
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	schema := "crm_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		_ = admin.Close()
		t.Fatalf("create test schema: %v", err)
	}
	var db *sql.DB
	t.Cleanup(func() {
		if db != nil {
			_ = db.Close()
		}
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err = sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatalf("open isolated PostgreSQL connection: %v", err)
	}
	db.SetMaxOpenConns(8)
	if err := sharedpostgres.Migrate(ctx, db, "../../../../migrations"); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	if err := sharedpostgres.Migrate(ctx, db, "../../../../migrations"); err != nil {
		t.Fatalf("repeat migration idempotency: %v", err)
	}
	return postgres.NewStore(db), db
}
