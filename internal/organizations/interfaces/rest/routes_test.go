package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/organizations/application"
	"go-service/internal/organizations/domain"
)

type fakeStore struct {
	org domain.Organization
}

func (f *fakeStore) Get(_ context.Context) (domain.Organization, error) {
	return f.org, nil
}

func (f *fakeStore) Update(_ context.Context, expectedVersion int64, patch domain.UpdatePatch) (domain.Organization, error) {
	if f.org.Version != expectedVersion {
		return domain.Organization{}, application.ErrPreconditionFailed
	}
	next, err := domain.ApplyPatch(f.org, patch)
	if err != nil {
		return domain.Organization{}, err
	}
	next.Version++
	f.org = next
	return next, nil
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	openDay := domain.DayHours{IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}
	closedDay := domain.DayHours{IsOpen: false, OpenTime: "09:00", CloseTime: "18:00"}
	store := &fakeStore{org: domain.Organization{
		ID: uuid.NewString(), Name: "Takodu", BusinessType: "Salon",
		Timezone: "UTC", CurrencyCode: "USD", Version: 1,
		BusinessHours: domain.BusinessHours{
			Monday: openDay, Tuesday: openDay, Wednesday: openDay,
			Thursday: openDay, Friday: openDay, Saturday: closedDay, Sunday: closedDay,
		},
	}}
	RegisterRoutes(r.Group("/v1"), store)
	return r
}

func TestGetAndPatchOrganization(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/organization", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"v1"` {
		t.Fatalf("unexpected GET response: %d %q", rec.Code, rec.Header().Get("ETag"))
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"name":"Studio X"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 without If-Match, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"unknown":1}`))
	req.Header.Set("If-Match", `"v1"`)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on unknown field, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"name":"Studio X","currency_code":"EUR","ai_enabled":true}`))
	req.Header.Set("If-Match", `"v1"`)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `"v2"` {
		t.Fatalf("unexpected PATCH response: %d %q", rec.Code, rec.Header().Get("ETag"))
	}
	var got domain.Organization
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Name != "Studio X" || got.CurrencyCode != "EUR" || !got.AiEnabled {
		t.Fatalf("unexpected body: %v %+v", err, got)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"name":"Stale"}`))
	req.Header.Set("If-Match", `"v1"`)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 on stale ETag, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"timezone":"Invalid/Zone"}`))
	req.Header.Set("If-Match", `"v2"`)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid timezone, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"currency_code":"eur"}`))
	req.Header.Set("If-Match", `"v2"`)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid currency_code, got %d", rec.Code)
	}
}
