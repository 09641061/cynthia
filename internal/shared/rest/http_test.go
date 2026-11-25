package rest_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/shared/rest"
)

func TestParseUUID(t *testing.T) {
	raw := uuid.NewString()
	id, ok := rest.ParseUUID(strings.ToUpper(raw))
	if !ok || id != raw {
		t.Fatalf("unexpected result: id=%q ok=%v", id, ok)
	}
	if _, ok := rest.ParseUUID("not-a-uuid"); ok {
		t.Fatal("expected invalid UUID to fail")
	}
}

func TestPathUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/items/:id", func(c *gin.Context) {
		id, ok := rest.PathUUID(c, "id")
		if !ok {
			return
		}
		c.String(http.StatusOK, id)
	})

	want := uuid.NewString()
	req := httptest.NewRequest(http.MethodGet, "/items/"+want, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Fatalf("unexpected valid response: code=%d body=%q", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/items/bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestParseUUIDSlice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/roles", nil)

	valid := uuid.NewString()
	if !rest.ParseUUIDSlice(c, []string{valid}) {
		t.Fatal("expected valid slice to pass")
	}
	if rest.ParseUUIDSlice(c, []string{valid, "bad"}) {
		t.Fatal("expected invalid slice to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestQueryInt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query string
		want  int
	}{
		{"", 20},
		{"?limit=abc", 20},
		{"?limit=0", 1},
		{"?limit=500", 100},
		{"?limit=42", 42},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/items"+tc.query, nil)
		if got := rest.QueryInt(c, "limit", 20, 1, 100); got != tc.want {
			t.Fatalf("query %q: got %d, want %d", tc.query, got, tc.want)
		}
	}
}
