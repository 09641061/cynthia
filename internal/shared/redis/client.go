package redis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go-service/internal/shared/circuitbreaker"
)

// Client wraps a Redis client with a shared circuit breaker.
type Client struct {
	*goredis.Client
	Breaker *circuitbreaker.Breaker
}

// NewClient attaches a circuit breaker to an existing Redis client.
func NewClient(client *goredis.Client, breaker *circuitbreaker.Breaker) *Client {
	return &Client{Client: client, Breaker: breaker}
}

// Do executes fn protected by the client's circuit breaker.
func (c *Client) Do(ctx context.Context, fn func() error) error {
	return c.Breaker.Do(ctx, fn, connectionFailure)
}

// Open parses redisURL, creates a Redis client with a circuit breaker, and verifies connectivity.
func Open(ctx context.Context, redisURL string) (*Client, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	opts.DialTimeout = 3 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	raw := goredis.NewClient(opts)
	if err := raw.Ping(ctx).Err(); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("configure Redis circuit breaker: %w", err)
	}
	return NewClient(raw, breaker), nil
}

func connectionFailure(err error) bool {
	if err == nil || errors.Is(err, goredis.Nil) {
		return false
	}
	if errors.Is(err, goredis.ErrClosed) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
