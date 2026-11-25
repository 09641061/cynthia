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
	if !strings.HasPrefix(createdRec.Header().Get("Location"), "/v1/appointments/") {
		t.Fatalf("unexpected Location=%q", createdRec.Header().Get("Location"))
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

	okPatch := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"confirmed"}`, nil)
	if okPatch.Code != http.StatusOK {
		t.Fatalf("PATCH = %d, body=%s", okPatch.Code, okPatch.Body.String())
	}

	badTransition := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"status":"pending"}`, nil)
	if badTransition.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid transition = %d, want 422", badTransition.Code)
	}

	if rec := doReq(fx.router, http.MethodPatch, "/v1/appointments/"+created.ID, `{"notes":"`+longNotes+`"}`, nil); rec.Code != http.StatusUnprocessableEntity {
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

	delRec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", nil)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", delRec.Code)
	}
	if rec := doReq(fx.router, http.MethodDelete, "/v1/appointments/"+created.ID, "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing = %d, want 404", rec.Code)
	}
}
