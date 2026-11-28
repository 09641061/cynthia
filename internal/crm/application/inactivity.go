package application

import (
	"context"
	"log/slog"
	"time"
)

// InactiveConversationCloser persists bulk closure of idle open conversations.
type InactiveConversationCloser interface {
	CloseInactiveConversations(ctx context.Context, cutoff time.Time) ([]string, error)
}

// InactivityCloser periodically closes idle open conversations and publishes SSE updates.
type InactivityCloser struct {
	store         InactiveConversationCloser
	events        EventPublisher
	idleTimeout   time.Duration
	sweepInterval time.Duration
	now           func() time.Time
}

// NewInactivityCloser creates an InactivityCloser with the given idle timeout and sweep interval.
func NewInactivityCloser(store InactiveConversationCloser, events EventPublisher, idleTimeout, sweepInterval time.Duration) *InactivityCloser {
	return &InactivityCloser{
		store:         store,
		events:        events,
		idleTimeout:   idleTimeout,
		sweepInterval: sweepInterval,
		now:           time.Now,
	}
}

// Sweep closes open conversations inactive since before now-idleTimeout and publishes conversation.updated events.
func (c *InactivityCloser) Sweep(ctx context.Context) (int, error) {
	if c == nil || c.store == nil || c.idleTimeout <= 0 {
		return 0, nil
	}
	cutoff := c.now().UTC().Add(-c.idleTimeout)
	ids, err := c.store.CloseInactiveConversations(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	if c.events != nil {
		for _, id := range ids {
			c.events.Publish(Event{Type: EventConversationUpdated, ConversationID: id})
		}
	}
	return len(ids), nil
}

// Start runs Sweep on every sweepInterval tick until ctx is cancelled.
func (c *InactivityCloser) Start(ctx context.Context) {
	if c == nil || c.store == nil || c.idleTimeout <= 0 || c.sweepInterval <= 0 {
		return
	}
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := c.Sweep(ctx); err != nil && ctx.Err() == nil {
				slog.Error("crm inactivity sweep failed", "error", err)
			}
		}
	}
}
