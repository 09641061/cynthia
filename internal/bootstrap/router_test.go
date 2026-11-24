package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterMountsOrganizationAndCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(nil)
	cases := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodPatch, "/v1/organization", `{"name":"Studio"}`, http.StatusPreconditionRequired},
		{http.MethodGet, "/v1/catalog/categories/not-a-uuid", "", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("%s %s: status = %d, want %d; body = %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", got)
			}
		})
	}
}
