//go:build integration

package iam_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDatabaseURL string
	testRedisAddr   string
)

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func runIntegrationTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("iam_test"),
		postgres.WithUsername("iam"),
		postgres.WithPassword("iam"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start PostgreSQL container: %v\n", err)
		return 1
	}
	defer func() { _ = pgContainer.Terminate(context.Background()) }()

	testDatabaseURL, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL connection string: %v\n", err)
		return 1
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start Redis container: %v\n", err)
		return 1
	}
	defer func() { _ = redisContainer.Terminate(context.Background()) }()

	host, err := redisContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Redis host: %v\n", err)
		return 1
	}
	port, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Redis port: %v\n", err)
		return 1
	}
	testRedisAddr = fmt.Sprintf("%s:%s", host, port.Port())
	return m.Run()
}
