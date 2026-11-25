package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestResendSenderSendsExpectedRequest(t *testing.T) {
	var authHeader string
	var got resendPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewResendSender("re_test_key", "Cynthia <no-reply@example.com>", "https://app.example.com")
	sender.SetBaseURL(srv.URL)

	if err := sender.SendInvitation(t.Context(), "nuevo@example.com", "Sofía", "tok_123"); err != nil {
		t.Fatalf("SendInvitation failed: %v", err)
	}
	if authHeader != "Bearer re_test_key" {
		t.Fatalf("Authorization = %q, want Bearer re_test_key", authHeader)
	}
	if len(got.To) != 1 || got.To[0] != "nuevo@example.com" {
		t.Fatalf("To = %v", got.To)
	}
	if !strings.Contains(got.HTML, "https://app.example.com/invite?token=tok_123") || !strings.Contains(got.HTML, "#10b981") {
		t.Fatalf("HTML missing invite link or emerald accent: %s", got.HTML)
	}
}

func TestResendSenderRetriesTransientFailures(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewResendSender("re_test_key", "Cynthia <no-reply@example.com>", "https://app.example.com")
	sender.SetBaseURL(srv.URL)

	if err := sender.SendInvitation(t.Context(), "nuevo@example.com", "Sofía", "tok_123"); err != nil {
		t.Fatalf("SendInvitation after retries = %v, want nil", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("attempts = %d, want 3", attempts.Load())
	}
}
