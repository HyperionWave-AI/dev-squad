package aiservice

// Provider implementations for different AI services (OpenAI, Anthropic, etc.)
//
// Token Usage Tracking:
// Each provider tracks token usage from API responses and includes it in ToolResponse.
// Token usage is automatically logged and aggregated via TokenUsageLogger.
//
// - OpenAI: Extracts prompt_tokens and completion_tokens from response
// - Anthropic: Extracts input_tokens and output_tokens from response body
// - Groq: Uses same format as OpenAI (prompt_tokens, completion_tokens)
//
// See TOKEN_USAGE.md for detailed documentation on token tracking.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Message represents a chat message with role and content
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system", "tool_call", or "tool_result"
	Content string `json:"content"` // Message content

	// Tool-related fields (optional, only for tool_call and tool_result roles)
	ToolCall   *ToolCall   `json:"toolCall,omitempty"`
	ToolResult *ToolResult `json:"toolResult,omitempty"`

	// Provider tracks which AI provider generated this message (for cross-model fallback handling)
	// Values: "openai", "anthropic", etc. Empty for user/system messages.
	Provider string `json:"provider,omitempty"`
}

// Tool represents a function/tool that can be called by the AI
// This is a native type that replaces the langchaingo llms.Tool dependency
type Tool struct {
	Type     string              `json:"type"` // "function"
	Function *FunctionDefinition `json:"function,omitempty"`
}

// FunctionDefinition describes a function tool that can be called by the AI
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatProvider defines the interface for AI chat providers
type ChatProvider interface {
	// StreamChat sends messages and returns a channel that streams response tokens
	StreamChat(ctx context.Context, messages []Message) (<-chan string, error)
}

// ToolCapableProvider extends ChatProvider with tool calling support
type ToolCapableProvider interface {
	ChatProvider
	// SupportsTools returns true if the provider/model supports tool calling
	SupportsTools() bool
	// StreamChatWithTools sends messages with tools and returns a response with tool calls
	StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error)
}

// ToolResponse contains the streaming response and any tool calls made by the AI
// 
// Token Usage: The TokenUsage field contains token consumption metrics from the API response.
// This includes prompt_tokens (input), completion_tokens (output), and total_tokens.
// Token usage is automatically extracted from provider responses and logged via TokenUsageLogger.
type ToolResponse struct {
	TextChannel <-chan string // Channel for streaming text tokens
	ToolCalls   []ToolCall    // Tool calls requested by the AI
	StopReason  string        // Why the model stopped: "end_turn", "tool_use", "max_tokens", etc.
	TokenUsage  *TokenUsage   // Token usage metrics from the API response (prompt, completion, total tokens)
}

// NewChatProvider creates a ChatProvider based on the configuration
// metricsStore is optional - pass nil to disable metrics recording
func NewChatProvider(config *AIConfig, metricsStore MetricsStore) (ChatProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	switch config.Provider {
	case "openai":
		return newOpenAIProvider(config, metricsStore)
	case "anthropic":
		return newAnthropicProvider(config, metricsStore)
	case "custom":
		return newCustomProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// Global HTTP logger instance for AI requests/responses
var httpLogger *HTTPLogger

func init() {
	// Initialize HTTP logger to ./logs/ directory
	httpLogger = NewHTTPLogger("./logs")
}

// openAIStreamChunk represents a streaming response chunk from OpenAI
// Kept here for compatibility with HTTP logger
type openAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string `json:"role,omitempty"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// openAICompletionResponse represents a non-streaming response from OpenAI
// Kept here for compatibility with HTTP logger
type openAICompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CacheBreakpointConfig configures cache breakpoint placement strategy.
// Anthropic allows up to 4 cache breakpoints per request:
// - BP1: Tools (handled separately when building tools array)
// - BP2: System message (handled when building system prompt)
// - BP3 & BP4: Conversation history (configured here)
//
// Strategy: Place breakpoints at FIXED positions to ensure cache hits.
// A moving breakpoint (like "2nd-to-last message") invalidates cache every request
// because the prefix changes. Fixed positions maintain stable cache prefixes.
type CacheBreakpointConfig struct {
	// Enabled controls whether cache breakpoints are added.
	// When false, no cache breakpoints will be added to requests.
	Enabled bool

	// StandardInterval defines the message interval for standard breakpoints.
	// A breakpoint is placed at the last message where (index % StandardInterval == 0).
	// Example: StandardInterval=10 places breakpoint at message 10, 20, 30, etc.
	// Default: 10
	StandardInterval int

	// MinMessagesForCaching is the minimum number of messages required before
	// adding any conversation history breakpoints. This prevents overhead
	// on short conversations where caching provides minimal benefit.
	// Default: 5
	MinMessagesForCaching int
}

// DefaultCacheBreakpointConfig returns the recommended cache breakpoint configuration.
// Places breakpoints every 10 messages for conversations with 5+ messages.
func DefaultCacheBreakpointConfig() CacheBreakpointConfig {
	return CacheBreakpointConfig{
		Enabled:               true,
		StandardInterval:      10,
		MinMessagesForCaching: 5,
	}
}

// anthropicProvider implements direct HTTP calls to Anthropic API
// This replaces the langchaingo wrapper for better control and proper tool parsing
type anthropicProvider struct {
	httpClient            *http.Client
	config                *AIConfig
	tokenExtractor        *TokenUsageExtractor
	tokenLogger           *TokenUsageLogger
	metricsStore          MetricsStore
	cacheBreakpointConfig CacheBreakpointConfig
}

func newAnthropicProvider(config *AIConfig, metricsStore MetricsStore) (*anthropicProvider, error) {
	return &anthropicProvider{
		httpClient:            &http.Client{Timeout: 5 * time.Minute},
		config:                config,
		tokenExtractor:        NewTokenUsageExtractor("anthropic"),
		tokenLogger:           NewTokenUsageLogger(),
		metricsStore:          metricsStore,
		cacheBreakpointConfig: DefaultCacheBreakpointConfig(),
	}, nil
}

func (p *anthropicProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Build Anthropic API messages
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	var systemPrompt string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		apiMessages = append(apiMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// Anthropic requires max_tokens
	maxTokens := p.config.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	reqBody := map[string]interface{}{
		"model":      p.config.Model,
		"messages":   apiMessages,
		"max_tokens": maxTokens,
		"stream":     true,
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}
	if p.config.Temperature > 0 {
		reqBody["temperature"] = p.config.Temperature
	}

	// Create output channel
	outputChan := make(chan string, 100)

	go func() {
		defer close(outputChan)

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			outputChan <- fmt.Sprintf("ERROR: failed to marshal request: %v", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
		if err != nil {
			outputChan <- fmt.Sprintf("ERROR: failed to create request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", p.config.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			if err != context.Canceled {
				outputChan <- fmt.Sprintf("ERROR: %v", err)
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			outputChan <- fmt.Sprintf("ERROR: Anthropic API error %d: %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Type == "content_block_delta" && event.Delta.Text != "" {
				select {
				case <-ctx.Done():
					return
				case outputChan <- event.Delta.Text:
				}
			}
		}
	}()

	return outputChan, nil
}

// extractTokenUsageFromAnthropicResponse extracts token usage from Anthropic API response body
// Includes prompt caching statistics for cost analysis
func (p *anthropicProvider) extractTokenUsageFromAnthropicResponse(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int) *TokenUsage {
	if inputTokens == 0 && outputTokens == 0 && cacheCreationTokens == 0 && cacheReadTokens == 0 {
		return nil
	}

	// Total input tokens = uncached + cache creation + cache read
	totalInput := inputTokens + cacheCreationTokens + cacheReadTokens

	return &TokenUsage{
		PromptTokens:        totalInput,
		CompletionTokens:    outputTokens,
		TotalTokens:         totalInput + outputTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
		Provider:            "anthropic",
		Model:               p.config.Model,
		Timestamp:           time.Now(),
	}
}

// logCacheStatistics logs comprehensive cache statistics for observability.
// Calculates and logs cache hit rate percentage when cache reads occur.
func (p *anthropicProvider) logCacheStatistics(cacheCreationTokens, cacheReadTokens, inputTokens int) {
	// Only log if there's any cache activity
	if cacheCreationTokens == 0 && cacheReadTokens == 0 {
		return
	}

	// Calculate cache hit rate if there are cache reads
	if cacheReadTokens > 0 {
		// Hit rate = (cache_read_tokens / total_input_context) * 100
		totalInputContext := inputTokens + cacheReadTokens + cacheCreationTokens
		if totalInputContext > 0 {
			hitRate := float64(cacheReadTokens) / float64(totalInputContext) * 100
			// 90% savings on cached tokens (cache reads cost 10% of input)
			estimatedSavings := float64(cacheReadTokens) * 0.9
			fmt.Printf("[CACHE HIT] Anthropic prompt cache: %.1f%% hit rate, %d tokens read, ~%.0f tokens saved\n",
				hitRate, cacheReadTokens, estimatedSavings)
		}
	} else if cacheCreationTokens > 0 {
		// Cache miss - new content being cached
		fmt.Printf("[CACHE MISS] Anthropic prompt cache: %d tokens cached (subsequent requests will use cached tokens)\n",
			cacheCreationTokens)
	}
}

// SupportsTools returns true for Anthropic provider
// All modern Claude models support tool use - no need for hardcoded model checks
func (p *anthropicProvider) SupportsTools() bool {
	return true
}

// StreamChatWithTools implements tool calling for Anthropic using direct API calls
// This bypasses langchaingo's broken tool parsing and uses Anthropic's native content blocks
func (p *anthropicProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error) {
	// Make direct API call to Anthropic to properly handle tool use content blocks
	return p.callAnthropicDirectly(ctx, messages, tools)
}

var invalidToolIDChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// sanitizeToolID ensures the ID matches Anthropic's pattern: ^[a-zA-Z0-9_-]+$
// Anthropic is strict about tool_use_id format and rejects IDs with other characters
func sanitizeToolID(id string) string {
	// Replace invalid characters with underscores
	sanitized := invalidToolIDChars.ReplaceAllString(id, "_")
	// Ensure we have at least some ID
	if sanitized == "" {
		sanitized = "tool_0"
	}
	return sanitized
}

// callAnthropicDirectly makes a direct HTTP call to Anthropic's Messages API
// This is necessary because langchaingo doesn't properly parse Anthropic's tool_use content blocks
func (p *anthropicProvider) callAnthropicDirectly(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error) {
	// Build Anthropic API request
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	var systemPrompt string

	// Find the last user message index to determine which tool messages to keep
	// We only keep tool messages that come AFTER the last user message
	// This prevents cross-model tool message errors when falling back from gpt-oss to Anthropic
	lastUserMessageIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessageIndex = i
			break
		}
	}

	// Track which tool_use IDs are actually included in apiMessages
	// This ensures we only include tool_results that have corresponding tool_use blocks
	includedToolUseIDs := make(map[string]bool)

	for i, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}

		// IMPORTANT: Only keep tool messages that come AFTER the most recent user message
		// When falling back from gpt-oss to Anthropic:
		// - Skip tool messages from BEFORE the last user message (these are from gpt-oss)
		// - Keep tool messages AFTER the last user message (these are from Anthropic in this conversation)
		// This prevents "unexpected tool_use_id" errors while preserving Anthropic's own tool call history
		if msg.Role == "tool_call" {
			if i < lastUserMessageIndex {
				// Skip tool messages from before the last user message
				continue
			}
			// Format tool_call (assistant message with tool_use blocks) for Anthropic
			if msg.ToolCall != nil {
				// Build content array for assistant message with tool_use
				contentBlocks := []map[string]interface{}{}

				// Add text content if present
				if msg.Content != "" {
					contentBlocks = append(contentBlocks, map[string]interface{}{
						"type": "text",
						"text": strings.TrimRight(msg.Content, " \t\n\r"),
					})
				}

				// Add tool_use block
				toolUseID := sanitizeToolID(msg.ToolCall.ID)
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    toolUseID,
					"name":  msg.ToolCall.Name,
					"input": msg.ToolCall.Args,
				})

				apiMessages = append(apiMessages, map[string]interface{}{
					"role":    "assistant",
					"content": contentBlocks,
				})

				// Track this tool_use ID as included
				includedToolUseIDs[toolUseID] = true
				// fmt.Printf("[DEBUG] Tracked tool_use ID: %s (name=%s)\n", toolUseID, msg.ToolCall.Name)
			}
			continue
		}

		if msg.Role == "tool_result" {
			if i < lastUserMessageIndex {
				// Skip tool messages from before the last user message
				continue
			}

			// Only include tool_results that have a corresponding tool_use in apiMessages
			// This prevents "unexpected tool_use_id" errors from Anthropic
			if msg.ToolResult != nil {
				toolResultID := sanitizeToolID(msg.ToolResult.ID)
				if !includedToolUseIDs[toolResultID] {
					// Skip this tool_result because its tool_call was filtered out
					continue
				}
			}

			// Format tool_result (user message with tool result) for Anthropic
			if msg.ToolResult != nil {
				resultContent := msg.ToolResult.Output
				if msg.ToolResult.Error != "" {
					resultContent = map[string]interface{}{
						"error": msg.ToolResult.Error,
					}
				}

				// Convert result to string if needed
				var resultStr string
				switch v := resultContent.(type) {
				case string:
					resultStr = v
				default:
					resultBytes, _ := json.Marshal(v)
					resultStr = string(resultBytes)
				}

				apiMessages = append(apiMessages, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":       "tool_result",
							"tool_use_id": sanitizeToolID(msg.ToolResult.ID),
							"content":    resultStr,
						},
					},
				})
			}
			continue
		}

		// Regular messages (user/assistant)
		// Anthropic requires that assistant content cannot end with trailing whitespace
		content := strings.TrimRight(msg.Content, " \t\n\r")

		apiMessages = append(apiMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": content,
		})
	}

	// Convert tools to Anthropic format
	// BP1: Add cache_control to the LAST tool for prompt caching
	apiTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}
		apiTools = append(apiTools, map[string]interface{}{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		})
	}
	// Add cache_control to the last tool (BP1) if caching is enabled
	if p.cacheBreakpointConfig.Enabled && len(apiTools) > 0 {
		lastToolIdx := len(apiTools) - 1
		apiTools[lastToolIdx]["cache_control"] = map[string]interface{}{
			"type": "ephemeral",
			"ttl":  "1h", // 1-hour TTL for stable tool definitions
		}
	}

	// Anthropic requires max_tokens to be set
	maxTokens := p.config.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 4096 // Default
	}

	// BP3/BP4: Add cache_control to messages at fixed positions
	// This ensures stable cache prefixes that don't change with every request
	apiMessages = p.addMessageCacheBreakpoints(apiMessages)

	reqBody := map[string]interface{}{
		"model":      p.config.Model,
		"messages":   apiMessages,
		"max_tokens": maxTokens,
	}

	// BP2: Add cache_control to system message for prompt caching
	if systemPrompt != "" {
		if p.cacheBreakpointConfig.Enabled {
			// Use array format with cache_control for proper caching
			reqBody["system"] = []map[string]interface{}{
				{
					"type": "text",
					"text": systemPrompt,
					"cache_control": map[string]interface{}{
						"type": "ephemeral",
						"ttl":  "1h", // 1-hour TTL for stable system prompts
					},
				},
			}
		} else {
			reqBody["system"] = systemPrompt
		}
	}
	if len(apiTools) > 0 {
		reqBody["tools"] = apiTools
		reqBody["tool_choice"] = map[string]interface{}{
			"type": "auto",
		}
	}
	if p.config.Temperature > 0 {
		reqBody["temperature"] = p.config.Temperature
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Debug: Print the actual JSON being sent to Anthropic
	// fmt.Printf("[DEBUG Anthropic Request] JSON: %s\n", string(bodyBytes))

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	// Enable extended cache TTL beta feature (required for 1-hour cache TTL)
	// Also enables computer-use and web-search beta features for compatibility
	if p.cacheBreakpointConfig.Enabled {
		req.Header.Set("anthropic-beta", "extended-cache-ttl-2025-04-11")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Role       string `json:"role"`
		Content    []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text,omitempty"`
			ID    string                 `json:"id,omitempty"`
			Name  string                 `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens             int `json:"input_tokens"`
			OutputTokens            int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"` // Tokens written to cache
			CacheReadInputTokens    int `json:"cache_read_input_tokens"`     // Tokens read from cache
		} `json:"usage"`
	}

	// Read response body for token extraction
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text and tool calls from content blocks
	textChan := make(chan string, 1000)
	var toolCalls []ToolCall
	var textContent string

	for _, block := range anthropicResp.Content {
		switch block.Type {
		case "thinking":
			// Include thinking content as part of the text output so users can see the reasoning
			if block.Text != "" {
				textContent += "[Thinking] " + block.Text + "\n\n"
			}
		case "text":
			textContent += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}

	// Send text content to channel and close immediately (non-streaming for now)
	go func() {
		if textContent != "" {
			textChan <- textContent
		}
		close(textChan)
	}()

	// Extract token usage from response (including cache statistics)
	tokenUsage := p.extractTokenUsageFromAnthropicResponse(
		anthropicResp.Usage.InputTokens,
		anthropicResp.Usage.OutputTokens,
		anthropicResp.Usage.CacheCreationInputTokens,
		anthropicResp.Usage.CacheReadInputTokens,
	)
	if tokenUsage != nil {
		p.tokenLogger.LogUsage(tokenUsage)
	}

	// Log cache statistics for observability
	p.logCacheStatistics(
		anthropicResp.Usage.CacheCreationInputTokens,
		anthropicResp.Usage.CacheReadInputTokens,
		anthropicResp.Usage.InputTokens,
	)

	// Record metrics to metrics store
	if tokenUsage != nil {
		metric := &ProviderMetric{
			ID:               fmt.Sprintf("anthropic-%d", time.Now().UnixNano()),
			Provider:         "anthropic",
			Model:            p.config.Model,
			PromptTokens:     tokenUsage.PromptTokens,
			CompletionTokens: tokenUsage.CompletionTokens,
			TotalTokens:      tokenUsage.TotalTokens,
			Cost:             CalculateAnthropicCost(p.config.Model, tokenUsage.PromptTokens, tokenUsage.CompletionTokens),
			DurationMs:       0, // TODO: Track from method start time
			Success:          err == nil,
			ErrorMessage:     "",
			Timestamp:        time.Now(),
		}
		if err != nil {
			metric.ErrorMessage = err.Error()
		}
		if p.metricsStore != nil {
			if recordErr := p.metricsStore.RecordProviderMetric(metric); recordErr != nil {
				fmt.Printf("[Metrics] Failed to record Anthropic provider metric: %v\n", recordErr)
			}
		}
	}

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
		StopReason:  anthropicResp.StopReason,
		TokenUsage:  tokenUsage,
	}, nil
}

// addMessageCacheBreakpoints adds cache_control to messages at FIXED positions (BP3/BP4).
// This uses up to 2 of Anthropic's 4 breakpoint limit for conversation history caching.
//
// Strategy: Place breakpoints at fixed message indices (e.g., 10, 20, 30) rather than
// relative positions (like "2nd-to-last"). Fixed positions maintain stable cache prefixes
// that can be reused across requests, while relative positions would invalidate the cache
// every time a new message is added.
//
// TTL ordering (Anthropic requirement): Earlier messages must have longer TTL.
// - First breakpoint (later message, higher index) → 5m TTL
// - Second breakpoint (earlier message, lower index) → 1h TTL
func (p *anthropicProvider) addMessageCacheBreakpoints(messages []map[string]interface{}) []map[string]interface{} {
	if !p.cacheBreakpointConfig.Enabled {
		return messages
	}

	if len(messages) < p.cacheBreakpointConfig.MinMessagesForCaching {
		return messages
	}

	// Find breakpoint positions from highest to lowest (most recent first)
	// We only add up to 2 breakpoints (BP3 and BP4)
	standardInterval := p.cacheBreakpointConfig.StandardInterval
	maxBreakpoints := 2
	breakpointsAdded := 0

	for msgIdx := len(messages) - 1; msgIdx >= 0 && breakpointsAdded < maxBreakpoints; msgIdx-- {
		// Check if this message is at a breakpoint interval (1-indexed position)
		// Message at index 9 is position 10, index 19 is position 20, etc.
		messagePosition := msgIdx + 1
		if messagePosition%standardInterval != 0 {
			continue
		}

		// Found a breakpoint position - add cache control
		msg := messages[msgIdx]

		// Determine TTL based on order found
		// First breakpoint (later message) → shorter TTL (5m)
		// Second breakpoint (earlier message) → longer TTL (1h)
		ttl := "5m"
		if breakpointsAdded > 0 {
			ttl = "1h"
		}

		cacheControl := map[string]interface{}{
			"type": "ephemeral",
			"ttl":  ttl,
		}

		// Add cache_control to the message content
		// Handle different content formats
		if content, ok := msg["content"].(string); ok {
			// Simple string content - convert to array with cache_control
			messages[msgIdx]["content"] = []map[string]interface{}{
				{
					"type":          "text",
					"text":          content,
					"cache_control": cacheControl,
				},
			}
		} else if contentBlocks, ok := msg["content"].([]map[string]interface{}); ok && len(contentBlocks) > 0 {
			// Array of content blocks - add cache_control to the last block
			lastIdx := len(contentBlocks) - 1
			contentBlocks[lastIdx]["cache_control"] = cacheControl
			messages[msgIdx]["content"] = contentBlocks
		}

		breakpointsAdded++
		fmt.Printf("[CACHE] Added BP%d cache breakpoint at message %d (TTL: %s)\n",
			breakpointsAdded+2, messagePosition, ttl) // BP3 or BP4
	}

	return messages
}

// customProvider is a placeholder for custom HTTP endpoint providers
type customProvider struct {
	config *AIConfig
}

func newCustomProvider(config *AIConfig) (*customProvider, error) {
	if config.ProviderURL == "" {
		return nil, fmt.Errorf("PROVIDER_URL is required for custom provider")
	}

	return &customProvider{
		config: config,
	}, nil
}

func (p *customProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Create output channel
	outputChan := make(chan string, 100)

	// Start streaming in goroutine
	go func() {
		defer close(outputChan)

		// TODO: Implement custom HTTP endpoint streaming
		// For now, return a placeholder error
		select {
		case <-ctx.Done():
		case outputChan <- "ERROR: Custom provider not yet implemented":
		}
	}()

	return outputChan, nil
}

// SupportsTools returns false for custom provider (not implemented yet)
func (p *customProvider) SupportsTools() bool {
	return false
}

// StreamChatWithTools is not supported for custom provider
func (p *customProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error) {
	return nil, fmt.Errorf("tool calling not supported for custom provider")
}

// CodeResultSummarizer handles intelligent summarization of code search results
// to reduce token usage while preserving critical information for AI decision-making
type CodeResultSummarizer struct {
	maxTokens int
}

// ResultMetadata contains extracted metadata from a code search result
type ResultMetadata struct {
	FilePath       string
	LineNumber     int
	FileType       string  // "ui", "backend", "test", "config", "other"
	RelevanceScore float64 // 0.0 to 1.0
	MatchType      string  // "exact", "partial", "contextual"
	ContextHint    string  // Brief context about the match
}

// NewCodeResultSummarizer creates a new summarizer with token limit
func NewCodeResultSummarizer(maxTokens int) *CodeResultSummarizer {
	if maxTokens <= 0 {
		maxTokens = 2000 // Default max tokens for summary
	}
	return &CodeResultSummarizer{
		maxTokens: maxTokens,
	}
}

// DetectFileType categorizes a file based on its path and extension
func (s *CodeResultSummarizer) DetectFileType(filePath string) string {
	lowerPath := strings.ToLower(filePath)

	// Test files
	if strings.Contains(lowerPath, "_test.go") || strings.Contains(lowerPath, ".test.tsx") ||
		strings.Contains(lowerPath, ".test.ts") || strings.Contains(lowerPath, ".spec.ts") ||
		strings.Contains(lowerPath, ".spec.tsx") {
		return "test"
	}

	// UI files
	if strings.Contains(lowerPath, "/ui/") || strings.Contains(lowerPath, "\\ui\\") {
		if strings.HasSuffix(lowerPath, ".tsx") || strings.HasSuffix(lowerPath, ".jsx") ||
			strings.HasSuffix(lowerPath, ".css") || strings.HasSuffix(lowerPath, ".scss") {
			return "ui"
		}
	}

	// Backend files
	if strings.HasSuffix(lowerPath, ".go") {
		return "backend"
	}

	// Config files
	if strings.HasSuffix(lowerPath, ".yaml") || strings.HasSuffix(lowerPath, ".yml") ||
		strings.HasSuffix(lowerPath, ".json") || strings.HasSuffix(lowerPath, ".env") ||
		strings.HasSuffix(lowerPath, ".toml") {
		return "config"
	}

	return "other"
}

// CalculateRelevanceScore determines how relevant a result is based on match quality
func (s *CodeResultSummarizer) CalculateRelevanceScore(matchType string, hasContextHint bool) float64 {
	score := 0.0

	// Base score by match type
	switch matchType {
	case "exact":
		score = 0.8
	case "partial":
		score = 0.5
	case "contextual":
		score = 0.3
	default:
		score = 0.2
	}

	// Bonus for context hint
	if hasContextHint {
		score += 0.15
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ExtractMetadata extracts metadata from a code search result
func (s *CodeResultSummarizer) ExtractMetadata(result map[string]interface{}) *ResultMetadata {
	metadata := &ResultMetadata{
		MatchType: "contextual",
	}

	// Extract file path
	if filePath, ok := result["filePath"].(string); ok {
		metadata.FilePath = filePath
		metadata.FileType = s.DetectFileType(filePath)
	}

	// Extract line number
	if lineNum, ok := result["lineNumber"].(float64); ok {
		metadata.LineNumber = int(lineNum)
	} else if lineNum, ok := result["lineNumber"].(int); ok {
		metadata.LineNumber = lineNum
	}

	// Extract match type if available
	if matchType, ok := result["matchType"].(string); ok {
		metadata.MatchType = matchType
	}

	// Extract context hint if available
	if context, ok := result["context"].(string); ok {
		if len(context) > 100 {
			metadata.ContextHint = context[:100] + "..."
		} else {
			metadata.ContextHint = context
		}
	}

	// Calculate relevance score
	metadata.RelevanceScore = s.CalculateRelevanceScore(metadata.MatchType, metadata.ContextHint != "")

	return metadata
}

// SummarizeResults generates an intelligent summary of code search results
func (s *CodeResultSummarizer) SummarizeResults(results []interface{}) string {
	if len(results) == 0 {
		return "No results found."
	}

	// Extract metadata from all results
	metadataList := make([]*ResultMetadata, 0, len(results))
	for _, result := range results {
		if resultMap, ok := result.(map[string]interface{}); ok {
			metadata := s.ExtractMetadata(resultMap)
			if metadata.FilePath != "" {
				metadataList = append(metadataList, metadata)
			}
		}
	}

	if len(metadataList) == 0 {
		return "No valid results found."
	}

	// Sort by relevance score (highest first)
	sort.Slice(metadataList, func(i, j int) bool {
		return metadataList[i].RelevanceScore > metadataList[j].RelevanceScore
	})

	// Group by file type
	categories := make(map[string][]*ResultMetadata)
	for _, metadata := range metadataList {
		categories[metadata.FileType] = append(categories[metadata.FileType], metadata)
	}

	// Build summary
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d results in %d categories:\n\n", len(metadataList), len(categories)))

	// Define category order for consistent output
	categoryOrder := []string{"ui", "backend", "test", "config", "other"}

	// Output each category
	for _, category := range categoryOrder {
		if items, exists := categories[category]; exists && len(items) > 0 {
			// Format category name
			categoryName := strings.ToUpper(category[:1]) + category[1:]
			if category == "ui" {
				categoryName = "UI Components"
			} else if category == "backend" {
				categoryName = "Backend Services"
			} else if category == "test" {
				categoryName = "Tests"
			} else if category == "config" {
				categoryName = "Configuration"
			}

			summary.WriteString(fmt.Sprintf("%s (%d files):\n", categoryName, len(items)))

			// List files in this category
			for _, item := range items {
				scoreStr := fmt.Sprintf("%.2f", item.RelevanceScore)
				if item.ContextHint != "" {
					summary.WriteString(fmt.Sprintf("  • %s:%d - %s (score: %s)\n",
						item.FilePath, item.LineNumber, item.ContextHint, scoreStr))
				} else {
					summary.WriteString(fmt.Sprintf("  • %s:%d (score: %s)\n",
						item.FilePath, item.LineNumber, scoreStr))
				}
			}
			summary.WriteString("\n")
		}
	}

	// Add most relevant file recommendation
	if len(metadataList) > 0 {
		topResult := metadataList[0]
		summary.WriteString(fmt.Sprintf("Most relevant: %s:%d (score: %.2f)\n",
			topResult.FilePath, topResult.LineNumber, topResult.RelevanceScore))
	}

	return summary.String()
}
