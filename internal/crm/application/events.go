package application

import (
	"sync"
)

const (
	EventMessageCreated      = "message.created"
	EventConversationUpdated = "conversation.updated"
	eventSubscriberBuffer    = 16
)

// Event represents a real-time CRM domain notification.
type Event struct {
	Type           string
	ConversationID string
}

// EventPublisher broadcasts CRM events to active subscribers.
type EventPublisher interface {
	Publish(event Event)
}

// EventSubscriber streams CRM events until the returned unsubscribe function is called.
type EventSubscriber interface {
	Subscribe() (<-chan Event, func())
}

// EventBroker is a thread-safe in-memory pub/sub broker for CRM events.
type EventBroker struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

// NewEventBroker creates an empty EventBroker.
func NewEventBroker() *EventBroker {
	return &EventBroker{subs: make(map[chan Event]struct{})}
}

// Publish sends an event to all subscribers without blocking slow consumers.
func (b *EventBroker) Publish(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe registers a subscriber channel and returns an idempotent unsubscribe function.
func (b *EventBroker) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, eventSubscriberBuffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, ch)
			close(ch)
			b.mu.Unlock()
		})
	}
	return ch, unsubscribe
}
