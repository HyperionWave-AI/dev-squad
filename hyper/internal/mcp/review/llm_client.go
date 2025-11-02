package review

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	aiservice "hyper/internal/ai-service"
)

// ClaudeAPIClient wraps HTTP calls to the Claude API
type ClaudeAPIClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	MaxRetries int
}

// NewClaudeAPIClient creates a new Claude API client from AIConfig
func NewClaudeAPIClient(config *aiservice.AIConfig) (*ClaudeAPIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("AIConfig is required")
	}

	// Determine base URL based on provider
	baseURL := "https://api.anthropic.com"
	if config.Provider == "anthropic" {
		baseURL = "https://api.anthropic.com"
	} else if config.Provider == "openai" || config.Provider == "custom" {
		// For OpenAI-compatible providers, use the provider URL if available
		if config.ProviderURL != "" {
			baseURL = config.ProviderURL
		} else {
			baseURL = "https://api.openai.com"
		}
	}

	return &ClaudeAPIClient{
		APIKey:  config.APIKey,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		MaxRetries: 3,
	}, nil
}

// ClaudeRequest represents a request to the Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Messages    []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	Usage        Usage  `json:"usage"`
	ErrorMessage string `json:"error,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// SendMessage sends a message to Claude API and returns the response
func (c *ClaudeAPIClient) SendMessage(ctx context.Context, prompt string, model string, temperature float64, maxTokens int) (string, error) {
	request := ClaudeRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt < c.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		response, err := c.makeRequest(ctx, request)
		if err != nil {
			lastErr = err
			// Retry on temporary errors
			if isRetryableError(err) {
				continue
			}
			return "", err
		}

		if response.ErrorMessage != "" {
			lastErr = fmt.Errorf("Claude API error: %s", response.ErrorMessage)
			continue
		}

		if len(response.Content) == 0 {
			lastErr = fmt.Errorf("empty response from Claude API")
			continue
		}

		return response.Content[0].Text, nil
	}

	return "", fmt.Errorf("failed after %d retries: %w", c.MaxRetries, lastErr)
}

// makeRequest performs the actual HTTP request to Claude API
func (c *ClaudeAPIClient) makeRequest(ctx context.Context, request ClaudeRequest) (*ClaudeResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &claudeResp, nil
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	// Retry on network errors and 5xx server errors
	return err != nil
}
