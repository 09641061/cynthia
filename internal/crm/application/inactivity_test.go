package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeInactiveStore struct {
	mu      sync.Mutex
	cutoffs []time.Time
	ids     []string
}

func (f *fakeInactiveStore) CloseInactiveConversations(_ context.Context, cutoff time.Time) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cutoffs = append(f.cutoffs, cutoff)
	return f.ids, nil
}

type fakeEventPub struct {
	mu     sync.Mutex
	events []Event
}

func (f *fakeEventPub) Publish(ev Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func TestInactivityCloserSweepPublishesUpdates(t *testing.T) {
	id1, id2 := uuid.NewString(), uuid.NewString()
	store := &fakeInactiveStore{ids: []string{id1, id2}}
	pub := &fakeEventPub{}
	closer := NewInactivityCloser(store, pub, 15*time.Minute, time.Minute)
	fixedNow := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	closer.now = func() time.Time { return fixedNow }

	n, err := closer.Sweep(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("expected 2 closed, got n=%d err=%v", n, err)
	}
	wantCutoff := fixedNow.Add(-15 * time.Minute)
	if len(store.cutoffs) != 1 || !store.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("unexpected cutoff: got=%v want=%v", store.cutoffs, wantCutoff)
	}
	if len(pub.events) != 2 || pub.events[0].Type != EventConversationUpdated || pub.events[0].ConversationID != id1 || pub.events[1].ConversationID != id2 {
		t.Fatalf("unexpected published events: %+v", pub.events)
	}
}

func TestInactivityCloserStartGuardAndCancellation(t *testing.T) {
	store := &fakeInactiveStore{}
	// Disabled when idleTimeout <= 0 or sweepInterval <= 0
	NewInactivityCloser(store, nil, 0, time.Millisecond).Start(context.Background())
	NewInactivityCloser(store, nil, time.Minute, 0).Start(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		NewInactivityCloser(store, &fakeEventPub{}, time.Minute, 5*time.Millisecond).Start(ctx)
	}()
	time.Sleep(15 * time.Millisecond)
	cancel()
	<-done
}
