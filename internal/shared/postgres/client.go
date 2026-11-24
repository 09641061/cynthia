package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ConfigFromEnv builds a PostgreSQL URL from the process environment.
func ConfigFromEnv() (string, error) {
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return "", errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	username, password := os.Getenv("DATABASE_USER"), os.Getenv("DATABASE_PASSWORD")
	if username != "" {
		parsed.User = url.UserPassword(username, password)
	}
	return parsed.String(), nil
}

// Open creates and verifies a PostgreSQL connection pool.
func Open(ctx context.Context, connectionString string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("connect to PostgreSQL: %w", err), db.Close())
	}
	return db, nil
}
