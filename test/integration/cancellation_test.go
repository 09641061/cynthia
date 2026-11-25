//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	catalogpostgres "go-service/internal/catalog/infrastructure/persistence/postgres"
	orgpostgres "go-service/internal/organizations/infrastructure/persistence/postgres"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func assertCancelledDatabaseCallKeepsCircuitClosed(t *testing.T, admin, db *sql.DB, schema string) {
	t.Helper()
	lock, err := admin.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("release test lock: %v", err)
		}
	}()
	if _, err := lock.ExecContext(t.Context(), `LOCK TABLE `+schema+`.catalog_categories IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	breaker, err := circuitbreaker.New(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	catalog, organization := catalogpostgres.NewStore(client), orgpostgres.NewStore(client)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := catalog.List(ctx)
		done <- err
	}()
	deadline, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	for {
		var waiting bool
		err := admin.QueryRowContext(deadline, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity WHERE wait_event_type = 'Lock'
			AND query LIKE '%FROM catalog_categories ORDER BY sort_order%')`).Scan(&waiting)
		if err != nil {
			t.Fatalf("wait for blocked query: %v", err)
		}
		if waiting {
			break
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled PostgreSQL read = %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatalf("release test lock: %v", err)
	}
	if org, err := organization.Get(t.Context()); err != nil || org.Name != "Takodu" {
		t.Fatalf("cancellation opened circuit: %+v, %v", org, err)
	}
}
