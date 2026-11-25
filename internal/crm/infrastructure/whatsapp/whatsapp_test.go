package whatsapp_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-service/internal/crm/infrastructure/whatsapp"
)

func TestClientSendText(t *testing.T) {
	t.Setenv("WHATSAPP_ACCESS_TOKEN", "")
	client := whatsapp.NewClient()
	_, uncertain, err := client.SendText(context.Background(), "12345", "+573001234567", "hi")
	if err == nil || uncertain || strings.Contains(err.Error(), "WHATSAPP_ACCESS_TOKEN") {
		t.Fatalf("expected redacted config error, got uncertain=%v err=%v", uncertain, err)
	}

	var mode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "non2xx":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"secret-upstream-detail"}`))
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

	t.Setenv("WHATSAPP_ACCESS_TOKEN", "test-token")
	t.Setenv("WHATSAPP_GRAPH_API_VERSION", "v25.0")
	client.BaseURL = srv.URL

	extID, uncertain, err := client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err != nil || uncertain || extID != "wamid.abc123" {
		t.Fatalf("expected wamid.abc123, got id=%q uncertain=%v err=%v", extID, uncertain, err)
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

	// Network failure -> uncertain=true, no raw URL leaked
	srv.Close()
	_, uncertain, err = client.SendText(context.Background(), "999000", "+573001234567", "hello")
	if err == nil || !uncertain || strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("expected redacted network error with uncertain=true, got uncertain=%v err=%v", uncertain, err)
	}
}

func TestVerifierChallengeAndSignature(t *testing.T) {
	t.Setenv("WHATSAPP_VERIFY_TOKEN", "my-verify-token")
	t.Setenv("WHATSAPP_APP_SECRET", "my-app-secret")
	v := whatsapp.NewVerifier()

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
