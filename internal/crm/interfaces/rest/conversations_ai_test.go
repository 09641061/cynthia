package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
	"go-service/internal/crm/interfaces/rest"
)

func TestPatchConversationAIRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	custStore := newFakeCustomerStore()
	custID, convID := uuid.NewString(), uuid.NewString()
	_, _ = custStore.CreateCustomer(context.Background(), domain.Customer{ID: custID, Name: "Ana", Phone: "+573001112233", Version: 1})
	cs := &stubConvStore{
		conv: domain.Conversation{ID: convID, CustomerID: custID, Channel: "whatsapp", Status: "open", Version: 1},
	}
	r := gin.New()
	convSvc := application.NewConversationService(cs, cs)
	rest.RegisterRoutes(r.Group("/v1"), application.NewCustomerService(custStore), convSvc, nil, nil)

	patchAI := func(etag, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/v1/conversations/"+convID+"/ai", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if etag != "" {
			req.Header.Set("If-Match", etag)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := patchAI("", `{"ai_paused":true}`); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 without If-Match, got %d", rec.Code)
	}
	if rec := patchAI(`"v99"`, `{"ai_paused":true}`); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 on stale ETag, got %d", rec.Code)
	}
	if rec := patchAI(`"v1"`, `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on missing ai_paused, got %d", rec.Code)
	}
	notFoundReq := httptest.NewRequest(http.MethodPatch, "/v1/conversations/"+uuid.NewString()+"/ai", bytes.NewBufferString(`{"ai_paused":true}`))
	notFoundReq.Header.Set("Content-Type", "application/json")
	notFoundReq.Header.Set("If-Match", `"v1"`)
	notFoundRec := httptest.NewRecorder()
	r.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on unknown conversation ID, got %d", notFoundRec.Code)
	}

	rec := patchAI(`"v1"`, `{"ai_paused":true}`)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"v2"` {
		t.Fatalf("expected 200 ETag v2, got %d %q", rec.Code, rec.Header().Get("ETag"))
	}
	var got struct {
		AiPaused      bool   `json:"ai_paused"`
		HandoffReason string `json:"handoff_reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !got.AiPaused || got.HandoffReason != "manual_pause" {
		t.Fatalf("unexpected pause body: %v %+v", err, got)
	}

	cs.conv.HandoffReason = "staff_replied"
	rec = patchAI(`"v2"`, `{"ai_paused":true}`)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"v3"` {
		t.Fatalf("expected 200 ETag v3, got %d %q", rec.Code, rec.Header().Get("ETag"))
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !got.AiPaused || got.HandoffReason != "staff_replied" {
		t.Fatalf("expected existing handoff_reason preserved, got %v %+v", err, got)
	}

	rec = patchAI(`"v3"`, `{"ai_paused":false}`)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"v4"` {
		t.Fatalf("expected 200 ETag v4, got %d %q", rec.Code, rec.Header().Get("ETag"))
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.AiPaused || got.HandoffReason != "" {
		t.Fatalf("unexpected resume body: %v %+v", err, got)
	}

	cs.conv.Status = domain.ConversationStatusClosed
	if rec := patchAI(`"v4"`, `{"ai_paused":true}`); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on closed conversation, got %d", rec.Code)
	}
}
