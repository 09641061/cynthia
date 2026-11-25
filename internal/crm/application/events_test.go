package application_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
)

func TestEventBrokerPublishAndSubscribe(t *testing.T) {
	broker := application.NewEventBroker()
	sub1, unsub1 := broker.Subscribe()
	defer unsub1()
	sub2, unsub2 := broker.Subscribe()
	defer unsub2()

	convID := uuid.NewString()
	want := application.Event{Type: application.EventMessageCreated, ConversationID: convID}
	broker.Publish(want)

	for i, ch := range []<-chan application.Event{sub1, sub2} {
		select {
		case got := <-ch:
			if got != want {
				t.Fatalf("subscriber %d got %+v, want %+v", i, got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out waiting for event", i)
		}
	}
}

func TestEventBrokerUnsubscribeClosesChannel(t *testing.T) {
	broker := application.NewEventBroker()
	ch, unsub := broker.Subscribe()

	unsub()
	unsub() // idempotent
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("expected channel to be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for channel close")
	}

	// Publishing after unsubscribe must not panic.
	broker.Publish(application.Event{Type: application.EventConversationUpdated, ConversationID: uuid.NewString()})
}

func TestEventBrokerSlowSubscriberDropsWithoutBlocking(t *testing.T) {
	broker := application.NewEventBroker()
	ch, unsub := broker.Subscribe()
	defer unsub()
	convID := uuid.NewString()
	for range 32 {
		broker.Publish(application.Event{Type: application.EventMessageCreated, ConversationID: convID})
	}

	received := 0
	for {
		select {
		case <-ch:
			received++
		default:
			if received != 16 {
				t.Fatalf("got %d buffered events, want 16", received)
			}
			return
		}
	}
}
