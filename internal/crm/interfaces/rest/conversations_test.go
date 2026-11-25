package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
	"go-service/internal/crm/interfaces/rest"
)

func TestConversationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	custStore := newFakeCustomerStore()
	custID, convID := uuid.NewString(), uuid.NewString()
	_, _ = custStore.CreateCustomer(context.Background(), domain.Customer{ID: custID, Name: "Ana", Phone: "+573001112233"})
	cs := &stubConvStore{
		conv:  domain.Conversation{ID: convID, CustomerID: custID, Channel: "whatsapp", Status: "open"},
		byExt: map[string]domain.Message{},
	}
	sender := &stubSender{}
	r := gin.New()
	convSvc := application.NewConversationService(cs, cs)
	msgSvc := application.NewMessageService(custStore, cs, cs, cs, sender, stubConfigReader{})
	rest.RegisterRoutes(r.Group("/v1"), application.NewCustomerService(custStore), convSvc, msgSvc, nil)

	// GET /v1/conversations & GET /v1/conversations/:id/messages
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/conversations", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list conversations, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/conversations/"+convID+"/messages", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list messages, got %d", rec.Code)
	}

	// POST message: 400 missing key, 201 sent, 502 redacted
	req := httptest.NewRequest(http.MethodPost, "/v1/conversations/"+convID+"/messages", bytes.NewBufferString(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/conversations/"+convID+"/messages", bytes.NewBufferString(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	sender.err = errors.New("http://secret.internal/raw upstream body leak")
	req = httptest.NewRequest(http.MethodPost, "/v1/conversations/"+convID+"/messages", bytes.NewBufferString(`{"content":"hi2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k2")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "secret.internal") {
		t.Fatalf("expected redacted 502, got %d: %s", rec.Code, rec.Body.String())
	}

	// PATCH status: 200, then 409 on closed
	req = httptest.NewRequest(http.MethodPatch, "/v1/conversations/"+convID+"/status", bytes.NewBufferString(`{"status":"closed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/conversations/"+convID+"/messages", bytes.NewBufferString(`{"content":"closed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k3")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var prob struct{ Status int }
	if rec.Code != http.StatusConflict || json.Unmarshal(rec.Body.Bytes(), &prob) != nil {
		t.Fatalf("expected 409 on closed conversation, got %d", rec.Code)
	}
}
