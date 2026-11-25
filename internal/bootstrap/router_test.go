package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm"
	"go-service/internal/iam"
)

func TestRouterProtectsModulesAndKeepsPublicRoutesOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, err := newRouter(nil, nil, iam.ModuleConfig{}, crm.WhatsAppConfig{})
	if err != nil {
		t.Fatalf("newRouter failed: %v", err)
	}
	cases := []struct {
		method string
		path   string
		body   string
		status int
	}{
		// Protected routes fail closed (503 when Redis is nil, never reaching handler unauthenticated).
		{http.MethodPatch, "/v1/organization", `{"name":"Studio"}`, http.StatusServiceUnavailable},
		{http.MethodGet, "/v1/catalog/categories/not-a-uuid", "", http.StatusServiceUnavailable},
		{http.MethodGet, "/v1/appointments", "", http.StatusServiceUnavailable},
		{http.MethodGet, "/v1/customers", "", http.StatusServiceUnavailable},
		// Public auth and WhatsApp webhook routes are reachable without Bearer auth.
		{http.MethodPost, "/v1/auth/login", `{"email":"bad"}`, http.StatusUnauthorized},
		{http.MethodGet, "/v1/crm/webhooks/whatsapp", "", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("%s %s: status = %d, want %d; body = %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}
