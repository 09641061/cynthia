package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func doReq(router http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCalendarRESTLifecycleAndErrors(t *testing.T) {
	fx := newRESTFixture()

	rec := doReq(fx.router, http.MethodGet, "/v1/calendar/people", "", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET /v1/calendar/people = %d, cache=%q", rec.Code, rec.Header().Get("Cache-Control"))
	}

	payload := `{"person_id":"` + fx.personID + `","customer_id":"` + fx.customerID + `","service_id":"` + fx.serviceID + `","starts_at":"2026-11-25T10:00:00Z"}`
	createdRec := doReq(fx.router, http.MethodPost, "/v1/appointments", payload, nil)
	if createdRec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/appointments = %d; body=%s", createdRec.Code, createdRec.Body.String())
	}
	if createdRec.Header().Get("ETag") != `"v1"` || !strings.HasPrefix(createdRec.Header().Get("Location"), "/v1/appointments/") {
		t.Fatalf("unexpected headers: ETag=%q Location=%q", createdRec.Header().Get("ETag"), createdRec.Header().Get("Location"))
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	conflictRec := doReq(fx.router, http.MethodPost, "/v1/appointments", payload, nil)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("overlap POST = %d, want 409", conflictRec.Code)
	}

	longNotes := strings.Repeat("a", 1001)
	badNotesPayload := `{"person_id":"` + fx.personID + `","customer_id":"` + fx.customerID + `","service_id":"` + fx.serviceID + `","starts_at":"2026-11-25T12:00:00Z","notes":"` + longNotes + `"}`
	if rec := doReq(fx.router, http.MethodPost, "/v1/appointments", badNotesPayload, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST notes > 1000 = %d, want 422", rec.Code)
	}

	missingIfMatch := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, nil)
	if missingIfMatch.Code != http.StatusPreconditionRequired {
		t.Fatalf("PATCH without If-Match = %d, want 428", missingIfMatch.Code)
	}

	staleIfMatch := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, map[string]string{"If-Match": `"v9"`})
	if staleIfMatch.Code != http.StatusPreconditionFailed {
		t.Fatalf("PATCH with stale If-Match = %d, want 412", staleIfMatch.Code)
	}

	okPatch := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, map[string]string{"If-Match": `"v1"`})
	if okPatch.Code != http.StatusOK || okPatch.Header().Get("ETag") != `"v2"` {
		t.Fatalf("PATCH = %d, ETag=%q", okPatch.Code, okPatch.Header().Get("ETag"))
	}

	badTransition := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"pending"}`, map[string]string{"If-Match": `"v2"`})
	if badTransition.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid transition = %d, want 422", badTransition.Code)
	}

	if rec := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"notes":"`+longNotes+`"}`, map[string]string{"If-Match": `"v2"`}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("PATCH notes > 1000 = %d, want 422", rec.Code)
	}

	badID := doReq(fx.router, http.MethodGet, "/v1/appointments/not-a-uuid", "", nil)
	if badID.Code != http.StatusBadRequest {
		t.Fatalf("bad uuid = %d, want 400", badID.Code)
	}

	notFound := doReq(fx.router, http.MethodGet, "/v1/appointments/"+uuid.NewString(), "", nil)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing appointment = %d, want 404", notFound.Code)
	}

	if rec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("DELETE without If-Match = %d, want 428", rec.Code)
	}
	if rec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v1"`}); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("DELETE with stale If-Match = %d, want 412", rec.Code)
	}
	delRec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v2"`})
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", delRec.Code)
	}
	if rec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", map[string]string{"If-Match": `"v2"`}); rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing = %d, want 404", rec.Code)
	}
}
