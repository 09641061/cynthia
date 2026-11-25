package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-service/internal/shared/circuitbreaker"
)

const (
	defaultResendBaseURL = "https://api.resend.com"
	maxSendAttempts      = 3
	retryBaseDelay       = 100 * time.Millisecond
)

// ResendSender sends invitation emails via the Resend HTTP API.
type ResendSender struct {
	apiKey  string
	from    string
	appURL  string
	baseURL string
	client  *http.Client
	breaker *circuitbreaker.Breaker
}

// NewResendSender creates a ResendSender.
func NewResendSender(apiKey, from, appURL string) *ResendSender {
	breaker, _ := circuitbreaker.New(5, 30*time.Second)
	return &ResendSender{
		apiKey:  strings.TrimSpace(apiKey),
		from:    strings.TrimSpace(from),
		appURL:  strings.TrimRight(strings.TrimSpace(appURL), "/"),
		baseURL: defaultResendBaseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		breaker: breaker,
	}
}

// SetBaseURL overrides the Resend API base URL (used in tests).
func (s *ResendSender) SetBaseURL(baseURL string) {
	if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
		s.baseURL = trimmed
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type transientError struct{ err error }

func (e transientError) Error() string { return e.err.Error() }
func (e transientError) Unwrap() error { return e.err }

// SendInvitation sends an emerald-accented invitation email with bounded retries and circuit breaker.
func (s *ResendSender) SendInvitation(ctx context.Context, toEmail, firstName, rawToken string) error {
	if s.apiKey == "" || s.from == "" || s.appURL == "" {
		return fmt.Errorf("resend email sender is not configured")
	}
	inviteURL := fmt.Sprintf("%s/invite?token=%s", s.appURL, url.QueryEscape(rawToken))
	bodyBytes, err := json.Marshal(resendPayload{
		From:    s.from,
		To:      []string{toEmail},
		Subject: "Invitación a Cynthia",
		HTML:    renderInviteHTML(firstName, inviteURL),
	})
	if err != nil {
		return err
	}
	return s.breaker.Do(ctx, func() error {
		return s.sendWithRetry(ctx, bodyBytes)
	}, func(err error) bool {
		var t transientError
		return errors.As(err, &t)
	})
}

func (s *ResendSender) sendWithRetry(ctx context.Context, bodyBytes []byte) error {
	var lastErr error
	for attempt := range maxSendAttempts {
		if attempt > 0 {
			timer := time.NewTimer(retryBaseDelay * time.Duration(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		retryable, err := s.sendOnce(ctx, bodyBytes)
		if err == nil {
			return nil
		}
		if !retryable {
			return err
		}
		lastErr = transientError{err: err}
	}
	return lastErr
}

func (s *ResendSender) sendOnce(ctx context.Context, bodyBytes []byte) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return true, fmt.Errorf("send resend email: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	return retryable, fmt.Errorf("resend API returned status %d", resp.StatusCode)
}

func renderInviteHTML(firstName, inviteURL string) string {
	safeName := html.EscapeString(strings.TrimSpace(firstName))
	safeURL := html.EscapeString(inviteURL)
	return fmt.Sprintf(`<!doctype html><html><body style="font-family:sans-serif;background:#f8fafc;padding:24px;color:#0f172a;">
<div style="max-width:520px;margin:0 auto;background:#ffffff;border-radius:12px;padding:32px;border-top:4px solid #10b981;">
<h2 style="margin:0 0 12px;color:#065f46;">Hola, %s</h2>
<p style="margin:0 0 20px;line-height:1.5;">Has sido invitado a unirte al equipo en Cynthia. Haz clic en el botón para activar tu cuenta.</p>
<a href="%s" style="display:inline-block;background:#10b981;color:#ffffff;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:600;">Aceptar invitación</a>
<p style="margin:20px 0 0;font-size:12px;color:#64748b;">Si el botón no funciona, copia este enlace: %s</p>
</div></body></html>`, safeName, safeURL, safeURL)
}
