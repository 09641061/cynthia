package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/shared/circuitbreaker"
)

// Client shares one circuit breaker between stores using the same database pool.
type Client struct {
	*sql.DB
	breaker *circuitbreaker.Breaker
}

// NewClient attaches a breaker to a database pool without changing pool ownership.
func NewClient(db *sql.DB, breaker *circuitbreaker.Breaker) *Client {
	return &Client{DB: db, breaker: breaker}
}

// Do protects a complete database operation, including its transaction if any.
func (c *Client) Do(ctx context.Context, fn func() error) error {
	return c.breaker.Do(ctx, fn, connectionFailure)
}

func connectionFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	switch e := err.(type) {
	case *pgconn.PgError:
		return strings.HasPrefix(e.Code, "08") || e.Code == "57P01" || e.Code == "57P02" || e.Code == "57P03" || e.Code == "53300"
	case *pgconn.ConnectError:
		var pgErr *pgconn.PgError
		if !errors.As(e, &pgErr) {
			return true
		}
	case net.Error:
		return true
	}
	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		for _, cause := range e.Unwrap() {
			if connectionFailure(cause) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return connectionFailure(e.Unwrap())
	}
	return false
}
