//go:build integration

package integration_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	catalogpostgres "go-service/internal/catalog/infrastructure/persistence/postgres"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func assertInterruptedWriteRollsBack(t *testing.T, admin, db *sql.DB, schema string) {
	t.Helper()
	ctx := t.Context()
	_, err := admin.ExecContext(ctx, `CREATE FUNCTION `+schema+`.disconnect_on_insert() RETURNS trigger AS $$
		BEGIN PERFORM pg_terminate_backend(pg_backend_pid()); RETURN NEW; END;
		$$ LANGUAGE plpgsql`)
	if err != nil {
		t.Fatalf("create interruption trigger function: %v", err)
	}
	if _, err := admin.ExecContext(ctx, `CREATE TRIGGER disconnect_on_insert AFTER INSERT ON `+schema+`.catalog_categories
		FOR EACH ROW EXECUTE FUNCTION `+schema+`.disconnect_on_insert()`); err != nil {
		t.Fatalf("create interruption trigger: %v", err)
	}
	breaker, err := circuitbreaker.New(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store := catalogpostgres.NewStore(sharedpostgres.NewClient(db, breaker))
	if _, err := store.CreateCategory(ctx, "Interrupted"); err == nil || errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected connection failure during write, got %v", err)
	}
	if _, err := store.List(ctx); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("interrupted write did not open circuit: %v", err)
	}
	var count int
	if err := admin.QueryRowContext(ctx, `SELECT count(*) FROM `+schema+`.catalog_categories`).Scan(&count); err != nil {
		t.Fatalf("read persisted state: %v", err)
	}
	if count != 0 {
		t.Fatalf("interrupted transaction persisted %d categories", count)
	}
	if _, err := admin.ExecContext(ctx, `DROP TRIGGER disconnect_on_insert ON `+schema+`.catalog_categories`); err != nil {
		t.Fatalf("drop interruption trigger: %v", err)
	}
}
