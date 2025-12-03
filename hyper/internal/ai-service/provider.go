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

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
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
	StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error)
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

// openAIProvider wraps langchaingo's OpenAI client
type openAIProvider struct {
	llm              *openai.LLM
	config           *AIConfig
	tokenExtractor   *TokenUsageExtractor
	tokenLogger      *TokenUsageLogger
	metricsStore     MetricsStore
}

func newOpenAIProvider(config *AIConfig, metricsStore MetricsStore) (*openAIProvider, error) {
	opts := []openai.Option{
		openai.WithModel(config.Model),
		openai.WithToken(config.APIKey),
	}

	// Add custom base URL if ProviderURL is set (for Ollama or other OpenAI-compatible endpoints)
	if config.ProviderURL != "" {
		opts = append(opts, openai.WithBaseURL(config.ProviderURL))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &openAIProvider{
		llm:            llm,
		config:         config,
		tokenExtractor: NewTokenUsageExtractor("openai"),
		tokenLogger:    NewTokenUsageLogger(),
		metricsStore:   metricsStore,
	}, nil
}

func (p *openAIProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Convert messages to langchaingo format
	content := p.messagesToContent(messages)

	// Create output channel
	outputChan := make(chan string, 100)

	// Start streaming in goroutine
	go func() {
		defer close(outputChan)

		// Build call options
		callOpts := []llms.CallOption{
			llms.WithTemperature(p.config.Temperature),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case outputChan <- string(chunk):
					return nil
				}
			}),
		}

		// Add max tokens if configured
		if p.config.MaxOutputTokens > 0 {
			callOpts = append(callOpts, llms.WithMaxTokens(p.config.MaxOutputTokens))
		}

		// Stream with callback
		_, err := p.llm.Call(ctx, content, callOpts...)

		if err != nil && err != context.Canceled {
			// Send error as last message (with error prefix so caller can detect)
			select {
			case <-ctx.Done():
			case outputChan <- fmt.Sprintf("ERROR: %v", err):
			}
		}
	}()

	return outputChan, nil
}

func (p *openAIProvider) messagesToContent(messages []Message) string {
	// Simple concatenation for now - langchaingo will handle formatting
	var content string
	for _, msg := range messages {
		content += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	return content
}

// extractTokenUsageFromOpenAI attempts to extract token usage from OpenAI response
// Note: LangChain's ContentResponse doesn't expose token usage directly,
// so we log a message indicating that token usage should be captured from HTTP headers
func (p *openAIProvider) extractTokenUsageFromOpenAI(resp *llms.ContentResponse) *TokenUsage {
	// TODO: Capture token usage from HTTP response headers
	// OpenAI includes usage in response headers:
	// - x-openai-input-tokens: prompt token count
	// - x-openai-output-tokens: completion token count
	// This requires intercepting the HTTP response before LangChain processes it
	
	// For now, return nil - token usage will be captured via HTTP logging
	fmt.Printf("[TOKEN USAGE] OpenAI token usage extraction via LangChain not yet implemented\n")
	return nil
}

// SupportsTools returns true - all OpenAI-compatible endpoints support tools by default
// If a model doesn't support tools, it will simply ignore them
func (p *openAIProvider) SupportsTools() bool {
	return true
}

// StreamChatWithTools implements tool calling for OpenAI using LangChain
func (p *openAIProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
	// Helper function for JSON marshaling
	mustMarshalJSON := func(v interface{}) string {
		if v == nil {
			return "{}"
		}
		bytes, err := json.Marshal(v)
		if err != nil {
			return "{}"
		}
		return string(bytes)
	}

	// Convert messages to LangChain MessageContent format
	msgContents := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))

		case "assistant":
			msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeAI, msg.Content))

		case "system":
			msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeSystem, msg.Content))

		case "tool_call":
			// Tool calls should be sent as assistant messages with ToolCall parts
			if msg.ToolCall != nil {
				parts := []llms.ContentPart{}

				// Add text content if present
				if msg.Content != "" {
					parts = append(parts, llms.TextPart(msg.Content))
				}

				// Add ToolCall part
				toolCallPart := llms.ToolCall{
					ID:   msg.ToolCall.ID,
					Type: "function",
					FunctionCall: &llms.FunctionCall{
						Name:      msg.ToolCall.Name,
						Arguments: mustMarshalJSON(msg.ToolCall.Args),
					},
				}
				parts = append(parts, toolCallPart)

				msgContent := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: parts,
				}
				msgContents = append(msgContents, msgContent)
			} else {
				// Fallback if no ToolCall data
				msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeAI, msg.Content))
			}

		case "tool_result":
			// Tool results should be sent as Tool messages with ToolCallResponse parts
			if msg.ToolResult != nil {
				// Format result content - pass through as-is without forcing JSON
				var resultContent string
				if msg.ToolResult.Error != "" {
					resultContent = msg.ToolResult.Error
				} else {
					switch v := msg.ToolResult.Output.(type) {
					case string:
						resultContent = v
					default:
						// Try JSON marshal, fallback to fmt.Sprintf
						if outputJSON, err := json.Marshal(v); err == nil {
							resultContent = string(outputJSON)
						} else {
							resultContent = fmt.Sprintf("%v", v)
						}
					}
				}

				msgContent := llms.MessageContent{
					Role: llms.ChatMessageTypeTool, // FIXED: Use Tool role, not Human
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: msg.ToolResult.ID,
							Name:       msg.ToolResult.Name,
							Content:    resultContent,
						},
					},
				}
				msgContents = append(msgContents, msgContent)
			} else {
				// Fallback if no ToolResult data
				msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))
			}

		default:
			msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))
		}
	}

	// Create text channel for streaming
	textChan := make(chan string, 1000)
	var toolCalls []ToolCall

	// Prepare streaming function (non-blocking)
	streamFunc := func(ctx context.Context, chunk []byte) error {
		chunkStr := string(chunk)

		// Strip tool call JSON if present (Groq/some providers append it to text)
		// Pattern: [{"id":"functions.X:Y" or [{"id":"call_X"
		if idx := strings.Index(chunkStr, `[{"id":"`); idx >= 0 {
			// Keep only the text before the JSON array
			chunkStr = chunkStr[:idx]
		}

		// Skip empty chunks
		if strings.TrimSpace(chunkStr) == "" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case textChan <- chunkStr:
			return nil
		default:
			return nil
		}
	}

	// Build options
	opts := []llms.CallOption{
		llms.WithTemperature(p.config.Temperature),
		llms.WithTools(tools),
		llms.WithStreamingFunc(streamFunc),
	}

	if p.config.MaxOutputTokens > 0 {
		opts = append(opts, llms.WithMaxTokens(p.config.MaxOutputTokens))
	}

	// LOG REQUEST: Serialize msgContents and tools before AI call
	httpLogger.LogLangChainRequest(
		p.config.Provider,
		convertMsgContentsToJSON(msgContents),
		convertToolsToJSON(tools),
		map[string]interface{}{
			"temperature":     p.config.Temperature,
			"maxOutputTokens": p.config.MaxOutputTokens,
			"model":           p.config.Model,
		},
	)

	// Call GenerateContent in goroutine
	type generateResult struct {
		resp *llms.ContentResponse
		err  error
	}
	resultChan := make(chan generateResult, 1)

	go func() {
		resp, err := p.llm.GenerateContent(ctx, msgContents, opts...)
		resultChan <- generateResult{resp: resp, err: err}
		close(textChan)
	}()

	// Wait for generation to complete
	result := <-resultChan

	// LOG RESPONSE: Log the response immediately after AI call
	httpLogger.LogLangChainResponse(
		p.config.Provider,
		convertContentResponseToJSON(result.resp),
		result.err,
	)

	if result.err != nil && result.err != context.Canceled {
		return nil, fmt.Errorf("failed to generate content: %w", result.err)
	}

	// Extract tool calls and stop reason from response - USE AI-GENERATED IDs
	var stopReason string
	if result.resp != nil && len(result.resp.Choices) > 0 {
		choice := result.resp.Choices[0]
		stopReason = choice.StopReason // Capture stop reason from LangChain

		// Check ToolCalls array first (preferred - has real IDs from AI)
		if len(choice.ToolCalls) > 0 {
			for _, tc := range choice.ToolCalls {
				if tc.FunctionCall != nil {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err == nil {
						toolCalls = append(toolCalls, ToolCall{
							ID:   tc.ID, // Use AI-generated ID from ToolCalls
							Name: tc.FunctionCall.Name,
							Args: args,
						})
					}
				}
			}
		}
	}

	// Extract token usage from response
	var tokenUsage *TokenUsage
	if result.resp != nil {
		tokenUsage = p.extractTokenUsageFromOpenAI(result.resp)
		if tokenUsage != nil {
			p.tokenLogger.LogUsage(tokenUsage)
		}
	}

	// Record metrics to metrics store
	if tokenUsage != nil {
		metric := &ProviderMetric{
			ID:               fmt.Sprintf("openai-%d", time.Now().UnixNano()),
			Provider:         "openai",
			Model:            p.config.Model,
			PromptTokens:     tokenUsage.PromptTokens,
			CompletionTokens: tokenUsage.CompletionTokens,
			TotalTokens:      tokenUsage.TotalTokens,
			Cost:             CalculateOpenAICost(p.config.Model, tokenUsage.PromptTokens, tokenUsage.CompletionTokens),
			DurationMs:       0, // TODO: Track from method start time
			Success:          result.err == nil,
			ErrorMessage:     "",
			Timestamp:        time.Now(),
		}
		if result.err != nil {
			metric.ErrorMessage = result.err.Error()
		}
		if p.metricsStore != nil {
			if err := p.metricsStore.RecordProviderMetric(metric); err != nil {
				fmt.Printf("[Metrics] Failed to record OpenAI provider metric: %v\n", err)
			}
		}
	}

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
		StopReason:  stopReason,
		TokenUsage:  tokenUsage,
	}, nil
}

// anthropicProvider wraps langchaingo's Anthropic client
type anthropicProvider struct {
	llm              *anthropic.LLM
	config           *AIConfig
	tokenExtractor   *TokenUsageExtractor
	tokenLogger      *TokenUsageLogger
	metricsStore     MetricsStore
}

func newAnthropicProvider(config *AIConfig, metricsStore MetricsStore) (*anthropicProvider, error) {
	opts := []anthropic.Option{
		anthropic.WithModel(config.Model),
		anthropic.WithToken(config.APIKey),
	}

	llm, err := anthropic.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	return &anthropicProvider{
		llm:            llm,
		config:         config,
		tokenExtractor: NewTokenUsageExtractor("anthropic"),
		tokenLogger:    NewTokenUsageLogger(),
		metricsStore:   metricsStore,
	}, nil
}

func (p *anthropicProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Convert messages to langchaingo format
	content := p.messagesToContent(messages)

	// Create output channel
	outputChan := make(chan string, 100)

	// Start streaming in goroutine
	go func() {
		defer close(outputChan)

		// Build call options
		callOpts := []llms.CallOption{
			llms.WithTemperature(p.config.Temperature),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case outputChan <- string(chunk):
					return nil
				}
			}),
		}

		// Add max tokens if configured
		if p.config.MaxOutputTokens > 0 {
			callOpts = append(callOpts, llms.WithMaxTokens(p.config.MaxOutputTokens))
		}

		// Stream with callback
		_, err := p.llm.Call(ctx, content, callOpts...)

		if err != nil && err != context.Canceled {
			// Send error as last message
			select {
			case <-ctx.Done():
			case outputChan <- fmt.Sprintf("ERROR: %v", err):
			}
		}
	}()

	return outputChan, nil
}

func (p *anthropicProvider) messagesToContent(messages []Message) string {
	// Simple concatenation for now
	var content string
	for _, msg := range messages {
		content += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	return content
}

// extractTokenUsageFromAnthropicResponse extracts token usage from Anthropic API response body
func (p *anthropicProvider) extractTokenUsageFromAnthropicResponse(inputTokens, outputTokens int) *TokenUsage {
	if inputTokens == 0 && outputTokens == 0 {
		return nil
	}
	
	return &TokenUsage{
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      inputTokens + outputTokens,
		Provider:         "anthropic",
		Model:            p.config.Model,
		Timestamp:        time.Now(),
	}
}

// SupportsTools returns true for Anthropic provider
// All modern Claude models support tool use - no need for hardcoded model checks
func (p *anthropicProvider) SupportsTools() bool {
	return true
}

// StreamChatWithTools implements tool calling for Anthropic using direct API calls
// This bypasses langchaingo's broken tool parsing and uses Anthropic's native content blocks
func (p *anthropicProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
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
func (p *anthropicProvider) callAnthropicDirectly(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
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

	// Anthropic requires max_tokens to be set
	maxTokens := p.config.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 4096 // Default
	}

	reqBody := map[string]interface{}{
		"model":      p.config.Model,
		"messages":   apiMessages,
		"max_tokens": maxTokens,
	}

	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
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
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
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

	// Extract token usage from response
	tokenUsage := p.extractTokenUsageFromAnthropicResponse(anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens)
	if tokenUsage != nil {
		p.tokenLogger.LogUsage(tokenUsage)
	}

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
func (p *customProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
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
