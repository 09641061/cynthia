package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
	sharedrest "go-service/internal/shared/rest"
)

func routerForTest(store *fakeStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/v1"), application.NewCatalog(store))
	return router
}

func TestCreateCategory(t *testing.T) {
	store := &fakeStore{}
	request := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories", bytes.NewBufferString(`{"name":"Hair"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != "/v1/catalog/categories/"+store.created.ID {
		t.Fatalf("Location = %q", got)
	}
	if got := recorder.Header().Get("ETag"); got != `"v1"` {
		t.Fatalf("ETag = %q, want version 1", got)
	}
	if store.created.Name != "Hair" || store.created.SortOrder != 1 {
		t.Fatalf("created category = %+v", store.created)
	}
}

func TestInvalidCategoryReturnsProblemDetails(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories", bytes.NewBufferString(`{"name":" "}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var result sharedrest.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if result.Status != http.StatusUnprocessableEntity || len(result.Errors) != 1 || result.Errors[0].Field != "name" {
		t.Fatalf("problem = %+v", result)
	}
}

func TestListReturnsNestedCatalog(t *testing.T) {
	categoryID, serviceID := newTestUUID(), newTestUUID()
	service := domain.Service{ID: serviceID, CategoryID: categoryID, Name: "Cut", SortOrder: 1}
	store := &fakeStore{categories: []domain.Category{{ID: categoryID, Name: "Hair", SortOrder: 1, Services: []domain.Service{service}}}}
	request := httptest.NewRequest(http.MethodGet, "/v1/catalog", nil)
	recorder := httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var result []domain.Category
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(result) != 1 || len(result[0].Services) != 1 || result[0].Services[0].ID != service.ID {
		t.Fatalf("catalog response = %+v", result)
	}
}

func TestMalformedPathIDReturns400(t *testing.T) {
	request := httptest.NewRequest(http.MethodDelete, "/v1/catalog/categories/not-a-uuid", nil)
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCreateAndMoveService(t *testing.T) {
	store := &fakeStore{}
	categoryID, destinationID := newTestUUID(), newTestUUID()
	body := `{"name":"Cut","duration_minutes":30,"price_minor_units":2500,"currency_code":"USD","sort_order":1}`
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/catalog/categories/%s/services", categoryID), strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || store.createdService.PriceMinor != 2500 {
		t.Fatalf("create response = %d %s", recorder.Code, recorder.Body.String())
	}
	move := fmt.Sprintf(`{"category_id":%q,"sort_order":2}`, destinationID)
	request = httptest.NewRequest(http.MethodPut, "/v1/catalog/services/"+store.createdService.ID+"/placement", strings.NewReader(move))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"v1"`)
	recorder = httptest.NewRecorder()
	routerForTest(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || store.movedService.CategoryID != destinationID || store.movedService.SortOrder != 2 {
		t.Fatalf("move response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateServiceRejectsRemovedDescription(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories/"+newTestUUID()+"/services", strings.NewReader(`{"name":"Cut","description":"obsolete","duration_minutes":30,"currency_code":"USD"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCreateServiceRejectsInvalidDuration(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories/"+newTestUUID()+"/services", strings.NewReader(`{"name":"Cut","duration_minutes":0,"currency_code":"USD"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	routerForTest(&fakeStore{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
}
