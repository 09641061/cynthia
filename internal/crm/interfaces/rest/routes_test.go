package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/interfaces/rest"
)

func newTestRouter(store *fakeCustomerStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.RegisterRoutes(r.Group("/v1"), application.NewCustomerService(store), nil, nil, nil)
	return r
}

func doJSON(r *gin.Engine, method, path, body, ifMatch string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCustomerCRUDAndDuplicateConflict(t *testing.T) {
	store := newFakeCustomerStore()
	r := newTestRouter(store)

	rec := doJSON(r, http.MethodPost, "/v1/customers", `{"name":"Ana","phone":"+57 300 123 4567"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") != `"v1"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected headers: ETag=%q Cache-Control=%q", rec.Header().Get("ETag"), rec.Header().Get("Cache-Control"))
	}
	var created struct {
		ID    string `json:"id"`
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if rec.Header().Get("Location") != "/v1/customers/"+created.ID || created.Phone != "+573001234567" {
		t.Fatalf("unexpected created customer or location: %+v %q", created, rec.Header().Get("Location"))
	}

	dup := doJSON(r, http.MethodPost, "/v1/customers", `{"name":"Ana 2","phone":"+573001234567"}`, "")
	if dup.Code != http.StatusConflict || dup.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("duplicate status=%d content-type=%q body=%s", dup.Code, dup.Header().Get("Content-Type"), dup.Body.String())
	}
	var prob struct {
		ExistingCustomer *struct {
			ID string `json:"id"`
		} `json:"existing_customer"`
	}
	if err := json.Unmarshal(dup.Body.Bytes(), &prob); err != nil || prob.ExistingCustomer == nil || prob.ExistingCustomer.ID != created.ID {
		t.Fatalf("expected existing_customer=%s, got %s", created.ID, dup.Body.String())
	}

	list := doJSON(r, http.MethodGet, "/v1/customers?q=Ana&page=1&page_size=10", "", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", list.Code)
	}

	get := doJSON(r, http.MethodGet, "/v1/customers/"+created.ID, "", "")
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"v1"` {
		t.Fatalf("get status=%d etag=%q", get.Code, get.Header().Get("ETag"))
	}

	noMatch := doJSON(r, http.MethodPatch, "/v1/customers/"+created.ID, `{"name":"Ana B","phone":"+573001234567"}`, "")
	if noMatch.Code != http.StatusPreconditionRequired {
		t.Fatalf("patch without If-Match = %d, want 428", noMatch.Code)
	}
	stale := doJSON(r, http.MethodPatch, "/v1/customers/"+created.ID, `{"name":"Ana B","phone":"+573001234567"}`, `"v2"`)
	if stale.Code != http.StatusPreconditionFailed {
		t.Fatalf("patch stale = %d, want 412", stale.Code)
	}
	patched := doJSON(r, http.MethodPatch, "/v1/customers/"+created.ID, `{"name":"Ana B","phone":"+573001234567"}`, `"v1"`)
	if patched.Code != http.StatusOK || patched.Header().Get("ETag") != `"v2"` {
		t.Fatalf("patch status=%d etag=%q", patched.Code, patched.Header().Get("ETag"))
	}

	store.blockedDel[created.ID] = true
	delBlocked := doJSON(r, http.MethodDelete, "/v1/customers/"+created.ID, "", "")
	if delBlocked.Code != http.StatusConflict {
		t.Fatalf("blocked delete = %d, want 409", delBlocked.Code)
	}
	delete(store.blockedDel, created.ID)
	delOK := doJSON(r, http.MethodDelete, "/v1/customers/"+created.ID, "", "")
	if delOK.Code != http.StatusNoContent {
		t.Fatalf("delete = %d, want 204", delOK.Code)
	}

	notFound := doJSON(r, http.MethodGet, "/v1/customers/"+uuid.NewString(), "", "")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing customer = %d, want 404", notFound.Code)
	}
}
