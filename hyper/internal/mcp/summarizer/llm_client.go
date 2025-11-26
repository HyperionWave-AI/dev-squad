package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// LLMClient is the interface for LLM-based code summarization
type LLMClient interface {
	// Summarize generates a summary for the given code
	Summarize(ctx context.Context, code string, metadata CodeMetadata) (string, error)
	// Close cleans up resources
	Close() error
}

// ClaudeClient implements LLMClient for Anthropic's Claude API
type ClaudeClient struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
	logger     *zap.Logger
	config     SummarizerConfig
}

// OpenAIClient implements LLMClient for OpenAI's API
type OpenAIClient struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
	logger     *zap.Logger
	config     SummarizerConfig
}

// NewClaudeClient creates a new Claude client for code summarization
func NewClaudeClient(apiKey string, config SummarizerConfig, logger *zap.Logger) (*ClaudeClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &ClaudeClient{
		apiKey:    apiKey,
		model:     config.Model,
		maxTokens: config.MaxTokens,
		httpClient: &http.Client{
			Timeout: config.LLMTimeout,
		},
		logger: logger,
		config: config,
	}, nil
}

// NewOpenAIClient creates a new OpenAI client for code summarization
func NewOpenAIClient(apiKey string, config SummarizerConfig, logger *zap.Logger) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &OpenAIClient{
		apiKey:    apiKey,
		model:     config.Model,
		maxTokens: config.MaxTokens,
		httpClient: &http.Client{
			Timeout: config.LLMTimeout,
		},
		logger: logger,
		config: config,
	}, nil
}

// Summarize generates a summary for the given code using Claude
func (c *ClaudeClient) Summarize(ctx context.Context, code string, metadata CodeMetadata) (string, error) {
	prompt := buildSummarizationPrompt(code, metadata)

	c.logger.Debug("Sending summarization request to Claude",
		zap.String("model", c.model),
		zap.String("file", metadata.FilePath),
		zap.String("node_type", metadata.NodeType),
	)

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		summary, err := c.summarizeWithRetry(ctx, prompt, attempt)
		if err == nil {
			return summary, nil
		}

		lastErr = err
		if attempt < 2 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			// Add jitter to prevent thundering herd
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			waitTime := backoff + jitter

			c.logger.Warn("Claude API request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", waitTime),
				zap.Error(err),
			)

			select {
			case <-time.After(waitTime):
				// Continue to next attempt
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	c.logger.Error("Failed to get summary from Claude after retries",
		zap.String("model", c.model),
		zap.Error(lastErr),
	)

	return "", fmt.Errorf("failed to summarize with Claude: %w", lastErr)
}

// summarizeWithRetry performs a single summarization request to Claude
func (c *ClaudeClient) summarizeWithRetry(ctx context.Context, prompt string, attempt int) (string, error) {
	reqBody := map[string]interface{}{
		"model":     c.model,
		"max_tokens": c.maxTokens,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited by Claude API")
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Claude API error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return "", fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(body))
	}

	var respData struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if respData.Error != nil {
		return "", fmt.Errorf("Claude API error: %s", respData.Error.Message)
	}

	if len(respData.Content) == 0 {
		return "", fmt.Errorf("no content in Claude response")
	}

	summary := respData.Content[0].Text
	c.logger.Debug("Successfully generated summary with Claude",
		zap.Int("summary_length", len(summary)),
	)

	return summary, nil
}

// Close cleans up resources
func (c *ClaudeClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// Summarize generates a summary for the given code using OpenAI
func (o *OpenAIClient) Summarize(ctx context.Context, code string, metadata CodeMetadata) (string, error) {
	prompt := buildSummarizationPrompt(code, metadata)

	o.logger.Debug("Sending summarization request to OpenAI",
		zap.String("model", o.model),
		zap.String("file", metadata.FilePath),
		zap.String("node_type", metadata.NodeType),
	)

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		summary, err := o.summarizeWithRetry(ctx, prompt, attempt)
		if err == nil {
			return summary, nil
		}

		lastErr = err
		if attempt < 2 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			// Add jitter to prevent thundering herd
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			waitTime := backoff + jitter

			o.logger.Warn("OpenAI API request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", waitTime),
				zap.Error(err),
			)

			select {
			case <-time.After(waitTime):
				// Continue to next attempt
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	o.logger.Error("Failed to get summary from OpenAI after retries",
		zap.String("model", o.model),
		zap.Error(lastErr),
	)

	return "", fmt.Errorf("failed to summarize with OpenAI: %w", lastErr)
}

// summarizeWithRetry performs a single summarization request to OpenAI
func (o *OpenAIClient) summarizeWithRetry(ctx context.Context, prompt string, attempt int) (string, error) {
	reqBody := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": o.maxTokens,
		"temperature": 0.3, // Lower temperature for more consistent summaries
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited by OpenAI API")
	}

	if resp.StatusCode != http.StatusOK {
		o.logger.Error("OpenAI API error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var respData struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if respData.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", respData.Error.Message)
	}

	if len(respData.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	summary := respData.Choices[0].Message.Content
	o.logger.Debug("Successfully generated summary with OpenAI",
		zap.Int("summary_length", len(summary)),
	)

	return summary, nil
}

// Close cleans up resources
func (o *OpenAIClient) Close() error {
	o.httpClient.CloseIdleConnections()
	return nil
}

// buildSummarizationPrompt constructs the prompt for code summarization
func buildSummarizationPrompt(code string, metadata CodeMetadata) string {
	prompt := fmt.Sprintf(`Provide a concise, technical summary of the following %s code.

File: %s
Node Type: %s
Node Name: %s
Language: %s

Code:
%s

Requirements:
1. Summary should be 1-3 sentences maximum
2. Focus on what the code does, not how it does it
3. Include important parameters or return values if relevant
4. Be technical but clear
5. Do not include code examples in the summary

Summary:`, metadata.Language, metadata.FilePath, metadata.NodeType, metadata.NodeName, metadata.Language, code)

	return prompt
}

// NewLLMClientFromConfig creates an appropriate LLM client based on configuration
func NewLLMClientFromConfig(config SummarizerConfig, logger *zap.Logger) (LLMClient, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("summarizer is disabled")
	}

	if config.LLMAPIKey == "" {
		return nil, fmt.Errorf("LLM API key is required")
	}

	// Determine provider from model name
	// Check if it's a Claude/Anthropic model
	isClaude := config.Model == "claude" ||
		config.Model == "claude-3-opus" ||
		config.Model == "claude-3-sonnet" ||
		config.Model == "claude-3-haiku" ||
		config.Model == "claude-haiku-4-5-20251001" ||
		len(config.Model) > 6 && config.Model[:6] == "claude"

	if isClaude {
		// Use claude-haiku-4-5 for summarization (fast and cost-effective)
		// Override model to a valid Claude model name if just "claude" is specified
		if config.Model == "claude" {
			config.Model = "claude-haiku-4-5-20251001"
		}
		logger.Info("Creating Claude client for summarization",
			zap.String("model", config.Model),
		)
		return NewClaudeClient(config.LLMAPIKey, config, logger)
	}

	// Default to OpenAI for gpt models or unknown
	logger.Info("Creating OpenAI client for summarization",
		zap.String("model", config.Model),
	)
	return NewOpenAIClient(config.LLMAPIKey, config, logger)
}
