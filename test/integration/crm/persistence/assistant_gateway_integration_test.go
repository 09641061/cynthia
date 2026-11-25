//go:build integration

package persistence_test

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm"
	"go-service/internal/crm/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestAssistantCRMGatewayIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, db := newTestStore(t)
	ctx := t.Context()

	router := gin.New()
	gw := crm.NewModule(sharedpostgres.NewClient(db, nil), router, nil, nil)

	cust, err := store.FindOrCreateCustomerByPhone(ctx, uuid.NewString(), "Ana", "+573001119988")
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	conv, err := store.EnsureOpenConversation(ctx, cust.ID)
	if err != nil {
		t.Fatalf("ensure open conversation: %v", err)
	}
	now := time.Now().UTC()
	for i, text := range []string{"first", "second"} {
		extID := "wamid.gw." + uuid.NewString()
		_, _, err := store.InsertInboundMessage(ctx, domain.Message{
			ID: uuid.NewString(), ConversationID: conv.ID, Direction: domain.MessageDirectionInbound,
			Content: text, Status: domain.MessageStatusReceived, ExternalID: &extID, CreatedAt: now.Add(time.Duration(i) * time.Second),
		}, text)
		if err != nil {
			t.Fatalf("insert message %d: %v", i, err)
		}
	}

	snap, err := gw.GetConversation(ctx, conv.ID)
	if err != nil || snap.ID != conv.ID || snap.CustomerID != cust.ID || snap.AiPaused {
		t.Fatalf("unexpected GetConversation: (%+v, %v)", snap, err)
	}
	msgs, err := gw.ListRecentMessages(ctx, conv.ID, 10)
	if err != nil || len(msgs) != 2 || msgs[0].Content != "first" || msgs[1].Content != "second" {
		t.Fatalf("expected ASC messages [first, second], got (%+v, %v)", msgs, err)
	}

	srv := httptest.NewServer(router)
	defer srv.Close()
	sseCtx, sseCancel := context.WithTimeout(ctx, 5*time.Second)
	defer sseCancel()
	req, _ := http.NewRequestWithContext(sseCtx, http.MethodGet, srv.URL+"/v1/crm/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}
	defer resp.Body.Close()

	eventCh := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.Contains(sc.Text(), "conversation.updated") {
				eventCh <- sc.Text()
				return
			}
		}
	}()

	if err := gw.PauseForHandoff(ctx, conv.ID, "customer_requested_human"); err != nil {
		t.Fatalf("PauseForHandoff: %v", err)
	}
	snap, err = gw.GetConversation(ctx, conv.ID)
	if err != nil || !snap.AiPaused || snap.HandoffReason != "customer_requested_human" {
		t.Fatalf("expected paused conversation, got (%+v, %v)", snap, err)
	}
	select {
	case <-eventCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected conversation.updated SSE event after PauseForHandoff")
	}

	if _, err := gw.GetConversation(ctx, uuid.NewString()); !errors.Is(err, crm.ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
	if err := gw.PauseForHandoff(ctx, uuid.NewString(), "x"); !errors.Is(err, crm.ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound on PauseForHandoff, got %v", err)
	}
	latest, err := store.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("get latest conversation: %v", err)
	}
	if _, err := store.UpdateConversationStatus(ctx, conv.ID, latest.Version, domain.ConversationStatusClosed); err != nil {
		t.Fatalf("close conversation: %v", err)
	}
	if _, err := gw.SendAssistantReply(ctx, conv.ID, "k-closed", "hi"); !errors.Is(err, crm.ErrConversationClosed) {
		t.Fatalf("expected ErrConversationClosed on closed conversation, got %v", err)
	}
}
