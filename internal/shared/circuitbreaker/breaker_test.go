package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

var dependencyFailure = errors.New("dependency failure")

func testBreaker(t *testing.T, threshold int) (*Breaker, *time.Time) {
	t.Helper()
	b, err := New(threshold, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	b.now = func() time.Time { return now }
	return b, &now
}

func call(b *Breaker, err error) error {
	return b.Do(context.Background(), func() error { return err }, func(err error) bool {
		return errors.Is(err, dependencyFailure)
	})
}

func TestInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		threshold int
		cooldown  time.Duration
	}{{0, time.Second}, {-1, time.Second}, {1, 0}, {1, -time.Second}} {
		if _, err := New(tc.threshold, tc.cooldown); err == nil {
			t.Fatalf("New(%d, %s) accepted invalid configuration", tc.threshold, tc.cooldown)
		}
	}
}

func TestFailuresMustBeConsecutive(t *testing.T) {
	b, now := testBreaker(t, 2)
	businessErr := errors.New("not found")
	for _, err := range []error{dependencyFailure, nil, dependencyFailure, businessErr, dependencyFailure} {
		if got := call(b, err); !errors.Is(got, err) {
			t.Fatalf("call returned %v, want %v", got, err)
		}
	}
	if err := call(b, fmt.Errorf("wrapped: %w", dependencyFailure)); !errors.Is(err, dependencyFailure) {
		t.Fatalf("last failure lost: %v", err)
	}
	if err := call(b, nil); !errors.Is(err, ErrOpen) {
		t.Fatalf("expected open, got %v", err)
	}
	*now = now.Add(time.Second)
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatalf("probe failure = %v", err)
	}
	if err := call(b, nil); !errors.Is(err, ErrOpen) {
		t.Fatalf("failed probe did not reopen: %v", err)
	}
	*now = now.Add(time.Second)
	if err := call(b, nil); err != nil {
		t.Fatalf("successful probe: %v", err)
	}
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatalf("breaker did not reset: %v", err)
	}
	if err := call(b, nil); err != nil {
		t.Fatalf("closed circuit refused call: %v", err)
	}
}

func TestCancelledCallsDoNotChangeState(t *testing.T) {
	b, now := testBreaker(t, 2)
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	executed := false
	if err := b.Do(ctx, func() error { executed = true; return nil }, func(error) bool { return true }); !errors.Is(err, context.Canceled) || executed {
		t.Fatalf("cancelled call = %v, executed = %v", err, executed)
	}
	if err := call(b, context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatal(err)
	}
	if err := call(b, nil); !errors.Is(err, ErrOpen) {
		t.Fatalf("cancelled operation changed failure count: %v", err)
	}
	*now = now.Add(time.Second)
	if err := call(b, context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := call(b, dependencyFailure); !errors.Is(err, dependencyFailure) {
		t.Fatalf("cancelled probe was not released: %v", err)
	}
	if err := call(b, nil); !errors.Is(err, ErrOpen) {
		t.Fatalf("failed probe did not reopen: %v", err)
	}
}
