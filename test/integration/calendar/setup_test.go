//go:build integration

package calendar_test

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	sharedpostgres "go-service/internal/shared/postgres"
)

type seededRefs struct {
	personA    string
	personB    string
	customerID string
	serviceID  string
}

func newTestDB(t *testing.T) (*sql.DB, seededRefs) {
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
	schema := "cal_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	db, err = sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatalf("open isolated PostgreSQL connection: %v", err)
	}
	db.SetMaxOpenConns(8)
	if err := sharedpostgres.Migrate(ctx, db, "../../../migrations"); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	if err := sharedpostgres.Migrate(ctx, db, "../../../migrations"); err != nil {
		t.Fatalf("repeat migration idempotency: %v", err)
	}
	return db, seedRefs(t, ctx, db)
}

func seedRefs(t *testing.T, ctx context.Context, db *sql.DB) seededRefs {
	t.Helper()
	refs := seededRefs{
		personA:    uuid.NewString(),
		personB:    uuid.NewString(),
		customerID: uuid.NewString(),
		serviceID:  uuid.NewString(),
	}
	catID := uuid.NewString()
	queries := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO calendar_people (id, name, active) VALUES ($1, 'Lucia', true), ($2, 'Elena', true)`, []any{refs.personA, refs.personB}},
		{`INSERT INTO crm_customers (id, name, phone) VALUES ($1, 'Ana', '+573001112233')`, []any{refs.customerID}},
		{`INSERT INTO catalog_categories (id, name, sort_order) VALUES ($1, 'Hair', 1)`, []any{catID}},
		{`INSERT INTO catalog_services (id, category_id, name, duration_minutes, price_minor_units, sort_order) VALUES ($1, $2, 'Cut', 30, 50000, 1)`, []any{refs.serviceID, catID}},
	}
	for _, q := range queries {
		if _, err := db.ExecContext(ctx, q.sql, q.args...); err != nil {
			t.Fatalf("seed reference data: %v", err)
		}
	}
	return refs
}
