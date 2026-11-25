package deepseek

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

	"go-service/internal/assistant/domain"
)

const (
	defaultTimeout     = 25 * time.Second
	defaultTemperature = 0.2
	defaultMaxTokens   = 600
	maxErrorBodyBytes  = 512
)

// APIError classifies DeepSeek HTTP and transport failures as retryable or permanent.
type APIError struct {
	StatusCode int
	Retryable  bool
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("deepseek error (status=%d retryable=%t): %s", e.StatusCode, e.Retryable, e.Message)
}

// IsRetryable reports whether err is a retryable DeepSeek failure.
func IsRetryable(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Retryable
}

// Client calls DeepSeek's OpenAI-compatible Chat Completions endpoint using stdlib net/http.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient constructs a DeepSeek HTTP client with a 25s timeout.
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

type chatRequest struct {
	Model       string                  `json:"model"`
	Messages    []domain.ChatMessage    `json:"messages"`
	Tools       []domain.ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string                  `json:"tool_choice,omitempty"`
	Temperature float64                 `json:"temperature"`
	MaxTokens   int                     `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string            `json:"content"`
			ToolCalls []domain.ToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete executes a single Chat Completions request against DeepSeek.
func (c *Client) Complete(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition) (domain.CompletionResult, error) {
	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: defaultTemperature,
		MaxTokens:   defaultMaxTokens,
	}
	if len(tools) > 0 {
		payload.ToolChoice = "auto"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.CompletionResult{}, c.redactErr(0, false, err.Error())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return domain.CompletionResult{}, c.redactErr(0, false, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.CompletionResult{}, c.redactErr(0, true, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return domain.CompletionResult{}, c.redactErr(resp.StatusCode, retryable, strings.TrimSpace(string(snippet)))
	}

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return domain.CompletionResult{}, c.redactErr(resp.StatusCode, true, "malformed response JSON")
	}
	if len(decoded.Choices) == 0 {
		return domain.CompletionResult{}, c.redactErr(resp.StatusCode, true, "empty choices")
	}
	ch := decoded.Choices[0]
	return domain.CompletionResult{
		Content:      ch.Message.Content,
		ToolCalls:    ch.Message.ToolCalls,
		FinishReason: ch.FinishReason,
	}, nil
}

func (c *Client) redactErr(status int, retryable bool, msg string) error {
	if c.apiKey != "" {
		msg = strings.ReplaceAll(msg, c.apiKey, "[REDACTED]")
	}
	return &APIError{StatusCode: status, Retryable: retryable, Message: msg}
}
