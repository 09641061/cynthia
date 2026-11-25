package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
)

func TestEventBrokerPublishAndSubscribe(t *testing.T) {
	broker := application.NewEventBroker()
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	sub1 := broker.Subscribe(ctx1)
	sub2 := broker.Subscribe(ctx2)

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
	ctx, cancel := context.WithCancel(context.Background())
	ch := broker.Subscribe(ctx)

	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("expected channel to be closed after context cancel")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for channel close")
	}

	// Publishing after unsubscribe must not panic.
	broker.Publish(application.Event{Type: application.EventConversationUpdated, ConversationID: uuid.NewString()})
}

func TestEventBrokerSlowSubscriberDropsWithoutBlocking(t *testing.T) {
	broker := application.NewEventBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := broker.Subscribe(ctx)
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
