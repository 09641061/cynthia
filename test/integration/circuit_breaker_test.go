//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/catalog/application"
	catalogpostgres "go-service/internal/catalog/infrastructure/persistence/postgres"
	orgpostgres "go-service/internal/organizations/infrastructure/persistence/postgres"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestDatabaseCircuitAcrossStoresAndRecovery(t *testing.T) {
	ctx := t.Context()
	parsed, err := url.Parse(testDatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	schema := "circuit_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		if err := admin.Close(); err != nil {
			t.Errorf("close admin: %v", err)
		}
	})
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	migrationDB, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedpostgres.Migrate(ctx, migrationDB, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatal(err)
	}
	proxy := &databaseProxy{address: "127.0.0.1:0", target: parsed.Host}
	proxy.start(t)
	parsed.Host = proxy.address
	db, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxIdleConns(0)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
		proxy.stop(t)
	})
	breaker, err := circuitbreaker.New(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	catalog, organization := catalogpostgres.NewStore(client), orgpostgres.NewStore(client)
	if _, err := catalog.GetCategory(ctx, uuid.NewString()); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("missing category = %v", err)
	}
	_, err = catalog.CreateCategory(ctx, "")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("expected real check constraint violation, got %v", err)
	}
	if org, err := organization.Get(ctx); err != nil || org.Name != "Takodu" {
		t.Fatalf("database did not remain available: %+v, %v", org, err)
	}
	assertInterruptedWriteRollsBack(t, admin, db, schema)
	assertCancelledDatabaseCallKeepsCircuitClosed(t, admin, db, schema)
	proxy.stop(t)
	if _, err := catalog.List(ctx); err == nil || errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected initial connection failure, got %v", err)
	}
	if _, err := organization.Get(ctx); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("organization did not share open circuit: %v", err)
	}
	proxy.start(t)
	if _, err := catalog.CreateCategory(ctx, "Blocked"); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("write was not rejected while open: %v", err)
	}
	proxy.stop(t)

	recoveryBreaker, err := circuitbreaker.New(1, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	recoveryClient := sharedpostgres.NewClient(db, recoveryBreaker)
	catalog, organization = catalogpostgres.NewStore(recoveryClient), orgpostgres.NewStore(recoveryClient)
	if _, err := organization.Get(ctx); err == nil || errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected connection failure before recovery, got %v", err)
	}
	if _, err := organization.Get(ctx); err == nil || errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected a failed recovery probe while disconnected, got %v", err)
	}
	proxy.start(t)
	if categories, err := catalog.List(ctx); err != nil || len(categories) != 0 {
		t.Fatalf("recovery probe failed: %v, %v", categories, err)
	}
	if org, err := organization.Get(ctx); err != nil || org.Name != "Takodu" {
		t.Fatalf("database did not recover: %+v, %v", org, err)
	}
}
