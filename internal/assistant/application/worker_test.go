package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

type workerTestJobStore struct {
	mu      sync.Mutex
	claimed bool
	doneCh  chan struct{}
}

func (s *workerTestJobStore) Enqueue(context.Context, string, string, time.Duration) error {
	return nil
}
func (s *workerTestJobStore) ClaimNext(context.Context, time.Duration) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimed {
		return Job{}, false, nil
	}
	s.claimed = true
	return Job{ID: "j-worker", ConversationID: "c-worker", TriggerMessageID: "m-1", Attempts: 1}, true, nil
}
func (s *workerTestJobStore) HasPendingForConversation(context.Context, string) (bool, error) {
	return false, nil
}
func (s *workerTestJobStore) CountRecentCompleted(context.Context, string, time.Time) (int, error) {
	return 0, nil
}
func (s *workerTestJobStore) SaveReply(context.Context, string, string) error { return nil }
func (s *workerTestJobStore) Complete(context.Context, string) error {
	select {
	case <-s.doneCh:
	default:
		close(s.doneCh)
	}
	return nil
}
func (s *workerTestJobStore) FailOrRetry(context.Context, string, string, string, bool, time.Duration) error {
	return nil
}

type workerTestOrg struct{}

func (workerTestOrg) GetAssistantConfig(context.Context) (OrgConfig, error) {
	return OrgConfig{AiEnabled: false}, nil
}

type workerTestCRM struct{}

func (workerTestCRM) GetConversation(context.Context, string) (Conversation, error) {
	return Conversation{ID: "c-worker", Status: "open"}, nil
}
func (workerTestCRM) ListRecentMessages(context.Context, string, int) ([]Message, error) {
	return nil, nil
}
func (workerTestCRM) SendAssistantReply(context.Context, string, string, string) (Message, error) {
	return Message{}, nil
}
func (workerTestCRM) PauseForHandoff(context.Context, string, string) error { return nil }

func TestNewWorkerDefaultingAndStartCancellation(t *testing.T) {
	if w := NewWorker(nil, nil, 0); w.workers != defaultWorkers {
		t.Fatalf("expected default workers %d for 0, got %d", defaultWorkers, w.workers)
	}
	if w := NewWorker(nil, nil, -2); w.workers != defaultWorkers {
		t.Fatalf("expected default workers %d for -2, got %d", defaultWorkers, w.workers)
	}

	store := &workerTestJobStore{doneCh: make(chan struct{})}
	proc := NewProcessor(store, nil, workerTestOrg{}, workerTestCRM{}, nil, nil)
	w := NewWorker(store, proc, 2)
	w.pollTicker = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		w.Start(ctx)
	}()

	<-store.doneCh
	cancel()
	<-exited
}
