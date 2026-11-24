//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	orgpostgres "go-service/internal/organizations/infrastructure/persistence/postgres"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

var testDatabaseURL string

func TestMain(m *testing.M) { os.Exit(runIntegrationTests(m)) }

func runIntegrationTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("organizations_test"),
		postgres.WithUsername("organizations"),
		postgres.WithPassword("organizations"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start PostgreSQL test container: %v\n", err)
		return 1
	}
	testDatabaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get PostgreSQL test connection string: %v\n", err)
		_ = container.Terminate(ctx)
		return 1
	}
	code := m.Run()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	if err := container.Terminate(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "terminate PostgreSQL test container: %v\n", err)
		return 1
	}
	return code
}

func newTestOrgStore(t *testing.T) *orgpostgres.Store {
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
	schema := "org_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	if db, err = sql.Open("pgx", parsed.String()); err != nil {
		t.Fatalf("open isolated PostgreSQL connection: %v", err)
	}
	if err := sharedpostgres.Migrate(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatalf("configure circuit breaker: %v", err)
	}
	return orgpostgres.NewStore(sharedpostgres.NewClient(db, breaker))
}
