//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDatabaseURL string

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func runIntegrationTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("crm_test"),
		postgres.WithUsername("crm"),
		postgres.WithPassword("crm"),
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
