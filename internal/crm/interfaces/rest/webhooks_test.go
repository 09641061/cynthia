package rest_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
	"go-service/internal/crm/infrastructure/whatsapp"
	"go-service/internal/crm/interfaces/rest"
)

func TestWhatsAppWebhookRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	custStore := newFakeCustomerStore()
	convID, custID := uuid.NewString(), uuid.NewString()
	cs := &stubConvStore{
		conv:  domain.Conversation{ID: convID, CustomerID: custID, Channel: "whatsapp", Status: "open"},
		byExt: map[string]domain.Message{},
	}
	r := gin.New()
	convSvc := application.NewConversationService(cs, cs)
	msgSvc := application.NewMessageService(custStore, cs, cs, cs, nil, nil)
	rest.RegisterRoutes(r.Group("/v1"), application.NewCustomerService(custStore), convSvc, msgSvc, whatsapp.NewVerifier(whatsapp.Config{
		VerifyToken: "verify-secret",
		AppSecret:   "app-secret",
	}))

	// GET challenge: valid & invalid
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/crm/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=verify-secret&hub.challenge=xyz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "xyz" {
		t.Fatalf("expected 200 xyz, got %d %q", rec.Code, rec.Body.String())
	}

	body := []byte(`{"entry":[{"changes":[{"value":{"contacts":[{"wa_id":"573001112233","profile":{"name":"Ana"}}],"messages":[{"id":"wamid.in.1","from":"573001112233","type":"text","text":{"body":"Hola"}}]}}]}]}`)

	// POST with bad signature -> 401
	req := httptest.NewRequest(http.MethodPost, "/v1/crm/webhooks/whatsapp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", rec.Code)
	}

	// POST with valid HMAC-SHA256 -> 200 and message persisted
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write(body)
	req = httptest.NewRequest(http.MethodPost, "/v1/crm/webhooks/whatsapp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid signature, got %d", rec.Code)
	}
	if _, ok := cs.byExt["wamid.in.1"]; !ok {
		t.Fatal("expected inbound message wamid.in.1 to be stored")
	}
}
