//go:build integration

package calendar_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-service/internal/calendar"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func doHTTP(r http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCalendarHTTPPostgreSQLLifecycle(t *testing.T) {
	db, refs := newTestDB(t)
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client := sharedpostgres.NewClient(db, breaker)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	calendar.NewModule(client, router, nil,
		crm.NewCustomerReader(client),
		catalog.NewServiceReader(client),
		organizations.NewScheduleReader(client),
	)

	peopleRec := doHTTP(router, http.MethodGet, "/v1/calendar/people", "", nil)
	if peopleRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/calendar/people = %d", peopleRec.Code)
	}

	body := `{"person_id":"` + refs.personA + `","customer_id":"` + refs.customerID + `","service_id":"` + refs.serviceID + `","starts_at":"2026-11-25T10:00:00Z","channel":"whatsapp"}`
	createRec := doHTTP(router, http.MethodPost, "/v1/appointments", body, nil)
	if createRec.Code != http.StatusCreated || createRec.Header().Get("ETag") != `"v1"` {
		t.Fatalf("POST /v1/appointments = %d; body=%s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	if rec := doHTTP(router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("PATCH without If-Match = %d, want 428", rec.Code)
	}
	if rec := doHTTP(router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, map[string]string{"If-Match": `"v9"`}); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("PATCH with stale If-Match = %d, want 412", rec.Code)
	}
	okPatch := doHTTP(router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, map[string]string{"If-Match": `"v1"`})
	if okPatch.Code != http.StatusOK || okPatch.Header().Get("ETag") != `"v2"` {
		t.Fatalf("PATCH confirm = %d, ETag=%q", okPatch.Code, okPatch.Header().Get("ETag"))
	}
	if rec := doHTTP(router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"pending"}`, map[string]string{"If-Match": `"v2"`}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid transition = %d, want 422", rec.Code)
	}
	if rec := doHTTP(router, http.MethodDelete, "/v1/appointments/"+created.ID, "", nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("DELETE without If-Match = %d, want 428", rec.Code)
	}
	if rec := doHTTP(router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v1"`}); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("DELETE with stale If-Match = %d, want 412", rec.Code)
	}
	if rec := doHTTP(router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v2"`}); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", rec.Code)
	}
	if rec := doHTTP(router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v2"`}); rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing = %d, want 404", rec.Code)
	}

	outOfHours := `{"person_id":"` + refs.personA + `","customer_id":"` + refs.customerID + `","service_id":"` + refs.serviceID + `","starts_at":"2026-11-25T08:00:00Z","channel":"whatsapp"}`
	if rec := doHTTP(router, http.MethodPost, "/v1/appointments", outOfHours, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST outside business hours = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}

	closedDay := `{"person_id":"` + refs.personA + `","customer_id":"` + refs.customerID + `","service_id":"` + refs.serviceID + `","starts_at":"2026-11-28T10:00:00Z","channel":"whatsapp"}`
	if rec := doHTTP(router, http.MethodPost, "/v1/appointments", closedDay, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST on closed day = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
}
