//go:build integration

package assistant_test

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
		postgres.WithDatabase("assistant_test"),
		postgres.WithUsername("assistant"),
		postgres.WithPassword("assistant"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start PostgreSQL test container: %v\n", err)
		return 1
	}
	testDatabaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get PostgreSQL connection string: %v\n", err)
		_ = container.Terminate(ctx)
		return 1
	}
	code := m.Run()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	_ = container.Terminate(shutdownCtx)
	return code
}
