package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-service/internal/shared/circuitbreaker"
)

const (
	defaultBaseURL    = "https://graph.facebook.com"
	defaultAPIVersion = "v25.0"
	defaultTimeout    = 10 * time.Second
	maxSendAttempts   = 3
)

// Client sends outbound WhatsApp messages via the Meta Cloud API.
type Client struct {
	BaseURL         string
	HTTPClient      *http.Client
	Breaker         *circuitbreaker.Breaker
	AccessToken     string
	PhoneNumberID   string
	GraphAPIVersion string
}

// NewClient creates a Meta Cloud API WhatsApp client with a 10s timeout and circuit breaker.
func NewClient(cfg Config) *Client {
	breaker, _ := circuitbreaker.New(5, 30*time.Second)
	return &Client{
		BaseURL:         defaultBaseURL,
		HTTPClient:      &http.Client{Timeout: defaultTimeout},
		Breaker:         breaker,
		AccessToken:     strings.TrimSpace(cfg.AccessToken),
		PhoneNumberID:   strings.TrimSpace(cfg.PhoneNumberID),
		GraphAPIVersion: strings.TrimSpace(cfg.GraphAPIVersion),
	}
}

type transientError struct{ err error }

func (e transientError) Error() string { return e.err.Error() }
func (e transientError) Unwrap() error { return e.err }

// SendText delivers a text message using Meta Graph API with circuit breaker and bounded transient retries.
func (c *Client) SendText(ctx context.Context, phoneNumberID, toPhone, text string) (string, bool, error) {
	token := strings.TrimSpace(c.AccessToken)
	resolvedPhoneID := strings.TrimSpace(phoneNumberID)
	if resolvedPhoneID == "" {
		resolvedPhoneID = strings.TrimSpace(c.PhoneNumberID)
	}
	if token == "" || resolvedPhoneID == "" {
		return "", false, errors.New("whatsapp integration is not configured")
	}
	version := strings.TrimSpace(c.GraphAPIVersion)
	if version == "" {
		version = defaultAPIVersion
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	payload, err := json.Marshal(sendRequest{
		MessagingProduct: "whatsapp",
		To:               strings.TrimPrefix(toPhone, "+"),
		Type:             "text",
		Text:             sendTextBody{Body: text},
	})
	if err != nil {
		return "", false, errors.New("whatsapp delivery failed")
	}
	endpoint := fmt.Sprintf("%s/%s/%s/messages", baseURL, version, resolvedPhoneID)
	var extID string
	var uncertain bool
	exec := func() error {
		extID, uncertain, err = c.sendWithRetry(ctx, endpoint, token, payload)
		return err
	}
	if c.Breaker != nil {
		err = c.Breaker.Do(ctx, exec, func(e error) bool {
			var te transientError
			return errors.As(e, &te)
		})
	} else {
		err = exec()
	}
	return extID, uncertain, err
}

func (c *Client) sendWithRetry(ctx context.Context, endpoint, token string, payload []byte) (string, bool, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	var lastErr error
	for attempt := 1; attempt <= maxSendAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return "", false, errors.New("whatsapp delivery failed")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", true, transientError{err: errors.New("whatsapp delivery status is uncertain")}
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = transientError{err: fmt.Errorf("whatsapp delivery failed (status %d)", resp.StatusCode)}
			if attempt < maxSendAttempts {
				timer := time.NewTimer(time.Duration(attempt) * 50 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return "", false, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			return "", false, lastErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", false, fmt.Errorf("whatsapp delivery failed (status %d)", resp.StatusCode)
		}
		var parsed sendResponse
		if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Messages) == 0 || parsed.Messages[0].ID == "" {
			return "", true, errors.New("whatsapp delivery status is uncertain")
		}
		return parsed.Messages[0].ID, false, nil
	}
	return "", false, lastErr
}
