package rest_test

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/interfaces/rest"
)

func TestStreamEventsSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	broker := application.NewEventBroker()
	rest.RegisterRoutes(r.Group("/v1"), nil, nil, nil, nil, broker)

	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/crm/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("X-Accel-Buffering = %q, want no", xab)
	}

	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() || scanner.Text() != ": connected" {
		t.Fatalf("first line = %q, want ': connected'", scanner.Text())
	}
	// Consume blank line after initial comment.
	scanner.Scan()

	convID := uuid.NewString()
	broker.Publish(application.Event{Type: application.EventMessageCreated, ConversationID: convID})

	if !scanner.Scan() || scanner.Text() != "event: message.created" {
		t.Fatalf("event line = %q, want 'event: message.created'", scanner.Text())
	}
	wantData := fmt.Sprintf(`data: {"conversation_id":%q}`, convID)
	if !scanner.Scan() || scanner.Text() != wantData {
		t.Fatalf("data line = %q, want %q", scanner.Text(), wantData)
	}

	cancel()
}
