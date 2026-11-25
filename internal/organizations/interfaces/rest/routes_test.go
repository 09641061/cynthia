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
	"go-service/internal/organizations/domain"
)

type fakeStore struct {
	org domain.Organization
}

func (f *fakeStore) Get(_ context.Context) (domain.Organization, error) {
	return f.org, nil
}

func (f *fakeStore) Update(_ context.Context, patch domain.UpdatePatch) (domain.Organization, error) {
	next, err := domain.ApplyPatch(f.org, patch)
	if err != nil {
		return domain.Organization{}, err
	}
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
		Timezone: "UTC", CurrencyCode: "USD",
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
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected GET response: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"unknown":1}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on unknown field, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"name":"Studio X","currency_code":"EUR"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH response: %d", rec.Code)
	}
	var got domain.Organization
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Name != "Studio X" || got.CurrencyCode != "EUR" {
		t.Fatalf("unexpected body: %v %+v", err, got)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"timezone":"Invalid/Zone"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid timezone, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/organization", bytes.NewBufferString(`{"currency_code":"eur"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid currency_code, got %d", rec.Code)
	}
}
