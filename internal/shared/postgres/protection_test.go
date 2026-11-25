package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/shared/circuitbreaker"
)

func TestConnectionFailureClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection refused", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}, true},
		{"bad connection", driver.ErrBadConn, true},
		{"end of stream", io.EOF, true},
		{"connection exception", &pgconn.PgError{Code: "08006"}, true},
		{"server shutdown", &pgconn.PgError{Code: "57P01"}, true},
		{"connection capacity", &pgconn.PgError{Code: "53300"}, true},
		{"constraint violation", &pgconn.PgError{Code: "23505"}, false},
		{"constraint and interrupted rollback", errors.Join(&pgconn.PgError{Code: "23505"}, io.EOF), true},
		{"constraint and server shutdown", errors.Join(&pgconn.PgError{Code: "23505"}, &pgconn.PgError{Code: "57P01"}), true},
		{"invalid credentials", &pgconn.PgError{Code: "28P01"}, false},
		{"missing row", sql.ErrNoRows, false},
		{"cancelled", context.Canceled, false},
		{"business failure", errors.New("invalid operation"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := connectionFailure(fmt.Errorf("wrapped: %w", tc.err)); got != tc.want {
				t.Fatalf("connectionFailure(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestClientStopsCallingFailedDependency(t *testing.T) {
	breaker, err := circuitbreaker.New(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(nil, breaker)
	executed := 0
	call := func(err error) error {
		return client.Do(t.Context(), func() error { executed++; return err })
	}
	if err := call(sql.ErrNoRows); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing row = %v", err)
	}
	if err := call(io.EOF); !errors.Is(err, io.EOF) {
		t.Fatalf("connection failure = %v", err)
	}
	if err := call(nil); !errors.Is(err, circuitbreaker.ErrOpen) || executed != 2 {
		t.Fatalf("open circuit = %v, executed = %d", err, executed)
	}
}
