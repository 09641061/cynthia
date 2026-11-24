package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrOpen indicates that a dependency call was rejected without being executed.
var ErrOpen = errors.New("circuit breaker is open")

// Breaker counts consecutive failures and permits one recovery probe at a time.
type Breaker struct {
	mu         sync.Mutex
	threshold  int
	cooldown   time.Duration
	now        func() time.Time
	failures   int
	opened     time.Time
	generation uint64
	probing    bool
}

// New creates a breaker with a failure threshold and an open-state duration.
func New(threshold int, cooldown time.Duration) (*Breaker, error) {
	if threshold < 1 || cooldown <= 0 {
		return nil, errors.New("circuit breaker requires a positive threshold and cooldown")
	}
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now}, nil
}

// Do executes fn if permitted. failed identifies dependency availability failures;
// other errors indicate a reachable dependency, except cancellations and deadlines.
func (b *Breaker) Do(ctx context.Context, fn func() error, failed func(error) bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	probe := false
	if !b.opened.IsZero() {
		if b.probing || b.now().Before(b.opened.Add(b.cooldown)) {
			b.mu.Unlock()
			return ErrOpen
		}
		b.probing = true
		probe = true
	}
	generation := b.generation
	b.mu.Unlock()
	finished := false
	defer func() {
		if !finished && probe {
			b.mu.Lock()
			if generation == b.generation {
				b.probing = false
			}
			b.mu.Unlock()
		}
	}()
	err := fn()
	inconclusive := err != nil && (ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
	failure := err != nil && !inconclusive && failed(err)
	b.mu.Lock()
	defer b.mu.Unlock()
	finished = true
	if generation != b.generation {
		return err
	}
	if inconclusive {
		if probe {
			b.probing = false
		}
		return err
	}
	if probe {
		b.probing = false
		b.generation++
		if failure {
			b.opened = b.now()
		} else {
			b.opened = time.Time{}
			b.failures = 0
		}
	} else if failure {
		b.failures++
		if b.failures >= b.threshold {
			b.opened = b.now()
			b.generation++
		}
	} else {
		b.failures = 0
	}
	return err
}
