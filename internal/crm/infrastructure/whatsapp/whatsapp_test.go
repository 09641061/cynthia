package whatsapp_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go-service/internal/crm/infrastructure/whatsapp"
	"go-service/internal/shared/circuitbreaker"
)

func TestClientSendText(t *testing.T) {
	client := whatsapp.NewClient(whatsapp.Config{})
	_, uncertain, err := client.SendText(context.Background(), "12345", "+573001234567", "hi")
	if err == nil || uncertain || strings.Contains(err.Error(), "WHATSAPP_ACCESS_TOKEN") {
		t.Fatalf("expected redacted config error, got uncertain=%v err=%v", uncertain, err)
	}

	var mode string
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		switch mode {
		case "non2xx":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"secret-upstream-detail"}`))
		case "retry5xx":
			if attempts.Load() < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.retried"}]}`))
		case "always5xx":
			w.WriteHeader(http.StatusBadGateway)
		case "badjson":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not-json`))
		default:
			if r.Header.Get("Authorization") != "Bearer test-token" || r.URL.Path != "/v25.0/999000/messages" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.abc123"}]}`))
		}
	}))
	defer srv.Close()

	client = whatsapp.NewClient(whatsapp.Config{AccessToken: "test-token", GraphAPIVersion: "v25.0"})
	client.BaseURL = srv.URL

	extID, uncertain, err := client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err != nil || uncertain || extID != "wamid.abc123" {
		t.Fatalf("expected wamid.abc123, got id=%q uncertain=%v err=%v", extID, uncertain, err)
	}

	mode = "retry5xx"
	attempts.Store(0)
	extID, uncertain, err = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err != nil || uncertain || extID != "wamid.retried" || attempts.Load() != 3 {
		t.Fatalf("expected retry success, got id=%q uncertain=%v err=%v attempts=%d", extID, uncertain, err, attempts.Load())
	}

	mode = "non2xx"
	_, uncertain, err = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err == nil || uncertain || strings.Contains(err.Error(), "secret-upstream-detail") {
		t.Fatalf("expected redacted non-2xx error with uncertain=false, got uncertain=%v err=%v", uncertain, err)
	}

	mode = "badjson"
	_, uncertain, err = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err == nil || !uncertain {
		t.Fatalf("expected uncertain=true for bad JSON, got uncertain=%v err=%v", uncertain, err)
	}

	mode = "always5xx"
	for range 5 {
		_, _, _ = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	}
	_, _, err = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("expected circuitbreaker.ErrOpen after 5 failures, got %v", err)
	}
}

func TestVerifierChallengeAndSignature(t *testing.T) {
	v := whatsapp.NewVerifier(whatsapp.Config{
		VerifyToken: "my-verify-token",
		AppSecret:   "my-app-secret",
	})

	if ch, ok := v.VerifyChallenge("subscribe", "my-verify-token", "12345"); !ok || ch != "12345" {
		t.Fatalf("expected challenge 12345, got %q %v", ch, ok)
	}
	if _, ok := v.VerifyChallenge("subscribe", "wrong", "12345"); ok {
		t.Fatal("expected wrong verify token to fail")
	}

	body := []byte(`{"entry":[]}`)
	mac := hmac.New(sha256.New, []byte("my-app-secret"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !v.VerifySignature(sig, body) {
		t.Fatal("expected valid HMAC signature to pass")
	}
	if v.VerifySignature("sha256=deadbeef", body) {
		t.Fatal("expected invalid HMAC signature to fail")
	}
}
