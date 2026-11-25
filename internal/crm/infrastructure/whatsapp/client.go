package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://graph.facebook.com"
	defaultAPIVersion = "v25.0"
	defaultTimeout    = 10 * time.Second
)

// Client sends outbound WhatsApp messages via the Meta Cloud API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Meta Cloud API WhatsApp client with a 10s timeout.
func NewClient() *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
	}
}

type sendRequest struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             sendTextBody `json:"text"`
}

type sendTextBody struct {
	Body string `json:"body"`
}

type sendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// SendText delivers a text message using Meta Graph API.
func (c *Client) SendText(ctx context.Context, phoneNumberID, toPhone, text string) (string, bool, error) {
	token := strings.TrimSpace(os.Getenv("WHATSAPP_ACCESS_TOKEN"))
	if token == "" {
		return "", false, errors.New("whatsapp integration is not configured")
	}
	resolvedPhoneID := strings.TrimSpace(phoneNumberID)
	if resolvedPhoneID == "" {
		resolvedPhoneID = strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID"))
	}
	if resolvedPhoneID == "" {
		return "", false, errors.New("whatsapp integration is not configured")
	}
	version := strings.TrimSpace(os.Getenv("WHATSAPP_GRAPH_API_VERSION"))
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", false, errors.New("whatsapp delivery failed")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", true, errors.New("whatsapp delivery status is uncertain")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("whatsapp delivery failed (status %d)", resp.StatusCode)
	}
	var parsed sendResponse
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Messages) == 0 || parsed.Messages[0].ID == "" {
		return "", true, errors.New("whatsapp delivery status is uncertain")
	}
	return parsed.Messages[0].ID, false, nil
}
