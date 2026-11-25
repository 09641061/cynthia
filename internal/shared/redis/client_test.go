package redis_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go-service/internal/shared/circuitbreaker"
	sharedredis "go-service/internal/shared/redis"
)

func TestClientDoCircuitBreaker(t *testing.T) {
	breaker, err := circuitbreaker.New(2, time.Minute)
	if err != nil {
		t.Fatalf("new breaker: %v", err)
	}
	client := sharedredis.NewClient(nil, breaker)
	ctx := context.Background()

	// goredis.Nil is a cache miss, not a dependency failure.
	for range 3 {
		err := client.Do(ctx, func() error { return goredis.Nil })
		if !errors.Is(err, goredis.Nil) {
			t.Fatalf("expected goredis.Nil, got %v", err)
		}
	}

	// Network/EOF errors trip the breaker after threshold.
	netErr := &net.OpError{Op: "dial", Err: io.EOF}
	for range 2 {
		if err := client.Do(ctx, func() error { return netErr }); !errors.Is(err, netErr) {
			t.Fatalf("expected netErr, got %v", err)
		}
	}
	if err := client.Do(ctx, func() error { return nil }); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
}
