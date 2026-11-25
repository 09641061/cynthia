package deepseek

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-service/internal/assistant/domain"
)

func TestClientTextAndToolCallResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-secret-123" {
			t.Errorf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["temperature"].(float64) != 0.2 || req["max_tokens"].(float64) != 600 {
			t.Errorf("unexpected params: %+v", req)
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"content":"Checking","tool_calls":[{"id":"c1","type":"function","function":{"name":"list_services","arguments":"{}"}}]}}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sk-secret-123", "deepseek-chat")
	res, err := c.Complete(context.Background(), []domain.ChatMessage{{Role: "user", Content: "hi"}}, []domain.ToolDefinition{
		{Type: "function", Function: domain.ToolFunction{Name: "list_services", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Content != "Checking" || len(res.ToolCalls) != 1 || res.ToolCalls[0].Function.Name != "list_services" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestClientErrorClassificationAndRedaction(t *testing.T) {
	cases := []struct {
		code      int
		retryable bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte(`{"error":"bad key sk-secret-999"}`))
		}))
		c := NewClient(srv.URL, "sk-secret-999", "deepseek-chat")
		_, err := c.Complete(context.Background(), []domain.ChatMessage{{Role: "user", Content: "hi"}}, nil)
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Retryable != tc.retryable {
			t.Fatalf("code %d: expected retryable=%v, got %v", tc.code, tc.retryable, err)
		}
		if strings.Contains(err.Error(), "sk-secret-999") {
			t.Fatalf("secret leaked in error: %v", err)
		}
	}
}

func TestClientMalformedJSONAndTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sk-1", "deepseek-chat")
	if _, err := c.Complete(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error on malformed JSON")
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer slow.Close()
	c2 := NewClient(slow.URL, "sk-1", "deepseek-chat")
	c2.httpClient.Timeout = 20 * time.Millisecond
	_, err := c2.Complete(context.Background(), nil, nil)
	if !IsRetryable(err) {
		t.Fatalf("expected retryable timeout error, got %v", err)
	}
}
