package rest

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-service/internal/catalog/application"
)

func TestUpdateCategoryNotFoundReturns404Problem(t *testing.T) {
	store := &fakeStore{updateErr: application.ErrNotFound}
	request := httptest.NewRequest(http.MethodPut, "/v1/catalog/categories/"+newTestUUID(), bytes.NewBufferString(`{"name":"Hair"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"v1"`)
	recorder := httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound || recorder.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
}

func TestConditionalUpdateRequiresFreshETag(t *testing.T) {
	path := fmt.Sprintf("/v1/catalog/categories/%s", newTestUUID())
	request := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(`{"name":"Hair"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match status = %d, want 428", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(`{"name":"Hair"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"v1"`)
	recorder = httptest.NewRecorder()
	routerForTest(&fakeStore{updateErr: application.ErrPreconditionFailed}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale If-Match status = %d, want 412", recorder.Code)
	}
}

func TestGetServiceReturnsStrongETag(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/catalog/services/"+newTestUUID(), nil)
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("ETag") != `"v1"` {
		t.Fatalf("response = %d ETag %q", recorder.Code, recorder.Header().Get("ETag"))
	}
}

func TestRejectsUnknownFieldsAndHidesInternalErrors(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/v1/catalog/services/"+newTestUUID(), bytes.NewBufferString(`{"sort_order":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"v1"`)
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d", recorder.Code)
	}
	store := &fakeStore{listErr: errors.New("database password leaked")}
	request = httptest.NewRequest(http.MethodGet, "/v1/catalog", nil)
	recorder = httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "password") {
		t.Fatalf("unsafe server error response = %d %s", recorder.Code, recorder.Body.String())
	}
}
