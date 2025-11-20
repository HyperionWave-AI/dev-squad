package aiservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

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
type ToolResponse struct {
	TextChannel <-chan string // Channel for streaming text tokens
	ToolCalls   []ToolCall    // Tool calls requested by the AI
}

// NewChatProvider creates a ChatProvider based on the configuration
func NewChatProvider(config *AIConfig) (ChatProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	switch config.Provider {
	case "openai":
		return newOpenAIProvider(config)
	case "anthropic":
		return newAnthropicProvider(config)
	case "custom":
		return newCustomProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// openAIProvider wraps langchaingo's OpenAI client
type openAIProvider struct {
	llm    *openai.LLM
	config *AIConfig
}

func newOpenAIProvider(config *AIConfig) (*openAIProvider, error) {
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
		llm:    llm,
		config: config,
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

// SupportsTools returns true - all OpenAI-compatible endpoints support tools by default
// If a model doesn't support tools, it will simply ignore them
func (p *openAIProvider) SupportsTools() bool {
	return true
}

// StreamChatWithTools implements tool calling for OpenAI using LangChain
func (p *openAIProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
	// DEBUG: Log all messages BEFORE sending to AI
	fmt.Printf("\n[DEBUG PRE-AI REQUEST] ========================================\n")
	fmt.Printf("[DEBUG PRE-AI REQUEST] Total messages: %d\n", len(messages))
	totalSize := 0
	for i, msg := range messages {
		msgSize := len(msg.Content)
		totalSize += msgSize

		// Preview (first 250 chars)
		preview := msg.Content
		if len(preview) > 250 {
			preview = preview[:250] + "..."
		}

		// Log based on message type
		if msg.Role == "tool_call" && msg.ToolCall != nil {
			argsJSON, _ := json.Marshal(msg.ToolCall.Args)
			fmt.Printf("[DEBUG PRE-AI MSG %d] Role: %s, ToolName: %s, ToolID: %s, Size: %d bytes\n",
				i, msg.Role, msg.ToolCall.Name, msg.ToolCall.ID, msgSize)
			fmt.Printf("  Args: %s\n", string(argsJSON))
			if msg.Content != "" {
				fmt.Printf("  Content: %s\n", preview)
			}
		} else if msg.Role == "tool_result" && msg.ToolResult != nil {
			fmt.Printf("[DEBUG PRE-AI MSG %d] Role: %s, ToolName: %s, ToolID: %s, Size: %d bytes\n",
				i, msg.Role, msg.ToolResult.Name, msg.ToolResult.ID, msgSize)
			if msg.ToolResult.Error != "" {
				fmt.Printf("  Error: %s\n", msg.ToolResult.Error)
			} else {
				outputPreview := fmt.Sprintf("%v", msg.ToolResult.Output)
				if len(outputPreview) > 250 {
					outputPreview = outputPreview[:250] + "..."
				}
				fmt.Printf("  Output: %s\n", outputPreview)
			}
		} else {
			fmt.Printf("[DEBUG PRE-AI MSG %d] Role: %s, Size: %d bytes\n", i, msg.Role, msgSize)
			fmt.Printf("  Content: %s\n", preview)
		}
	}
	fmt.Printf("[DEBUG PRE-AI REQUEST] Total content size: %d bytes\n", totalSize)
	fmt.Printf("[DEBUG PRE-AI REQUEST] ========================================\n\n")

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
	for i, msg := range messages {
		fmt.Printf("[DEBUG MESSAGE CONVERSION %d] Role: %s\n", i, msg.Role)

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
					fmt.Printf("  [DEBUG] Added TextPart: %d bytes\n", len(msg.Content))
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
				fmt.Printf("  [DEBUG] Added ToolCall part: ID=%s, Name=%s\n", msg.ToolCall.ID, msg.ToolCall.Name)

				msgContent := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: parts,
				}
				msgContents = append(msgContents, msgContent)
				fmt.Printf("  [DEBUG] Created MessageContent with %d parts\n", len(parts))
			} else {
				// Fallback if no ToolCall data
				msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeAI, msg.Content))
				fmt.Printf("  [DEBUG] Fallback: using TextParts\n")
			}

		case "tool_result":
			// Tool results should be sent as ToolCallResponse parts
			if msg.ToolResult != nil {
				// Format result content
				var resultContent string
				if msg.ToolResult.Error != "" {
					resultContent = msg.ToolResult.Error
					fmt.Printf("  [DEBUG] ToolResult error: %s\n", msg.ToolResult.Error)
				} else {
					switch v := msg.ToolResult.Output.(type) {
					case string:
						resultContent = v
					default:
						resultContent = mustMarshalJSON(v)
					}
					fmt.Printf("  [DEBUG] ToolResult output: %d bytes\n", len(resultContent))
				}

				msgContent := llms.MessageContent{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: msg.ToolResult.ID,
							Name:       msg.ToolResult.Name,
							Content:    resultContent,
						},
					},
				}
				msgContents = append(msgContents, msgContent)
				fmt.Printf("  [DEBUG] Created ToolCallResponse: ToolCallID=%s, Name=%s\n", msg.ToolResult.ID, msg.ToolResult.Name)
			} else {
				// Fallback if no ToolResult data
				msgContents = append(msgContents, llms.TextParts(llms.ChatMessageTypeHuman, msg.Content))
				fmt.Printf("  [DEBUG] Fallback: using TextParts\n")
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

	if result.err != nil && result.err != context.Canceled {
		return nil, fmt.Errorf("failed to generate content: %w", result.err)
	}

	// Extract tool calls from response - USE AI-GENERATED IDs
	if result.resp != nil && len(result.resp.Choices) > 0 {
		choice := result.resp.Choices[0]

		// DEBUG: Log complete response structure from LangChain
		fmt.Printf("\n[DEBUG LANGCHAIN RESPONSE] ========================================\n")
		fmt.Printf("[DEBUG LANGCHAIN] Total Choices: %d\n", len(result.resp.Choices))
		fmt.Printf("[DEBUG LANGCHAIN] Choice 0 - Content: '%s'\n", choice.Content)
		fmt.Printf("[DEBUG LANGCHAIN] Choice 0 - Content Length: %d bytes\n", len(choice.Content))
		fmt.Printf("[DEBUG LANGCHAIN] Choice 0 - ToolCalls Count: %d\n", len(choice.ToolCalls))

		// Log each tool call in the choice
		if len(choice.ToolCalls) > 0 {
			for i, tc := range choice.ToolCalls {
				fmt.Printf("[DEBUG LANGCHAIN ToolCall %d]\n", i)
				fmt.Printf("  ID: '%s'\n", tc.ID)
				fmt.Printf("  Type: '%s'\n", tc.Type)
				if tc.FunctionCall != nil {
					fmt.Printf("  FunctionCall.Name: '%s'\n", tc.FunctionCall.Name)
					fmt.Printf("  FunctionCall.Arguments: '%s'\n", tc.FunctionCall.Arguments)
					fmt.Printf("  FunctionCall.Arguments Length: %d bytes\n", len(tc.FunctionCall.Arguments))
				}
			}
		}

		// Check if Content field contains the tool call JSON (this is the bug!)
		if strings.Contains(choice.Content, `[{"id":"`) {
			fmt.Printf("[DEBUG LANGCHAIN] ⚠️  WARNING: Content field contains tool call JSON!\n")
			previewLen := 200
			if len(choice.Content) < previewLen {
				previewLen = len(choice.Content)
			}
			fmt.Printf("[DEBUG LANGCHAIN] Content preview: %s...\n", choice.Content[:previewLen])
		}

		fmt.Printf("[DEBUG LANGCHAIN RESPONSE] ========================================\n\n")

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
						fmt.Printf("[DEBUG OpenAI] Extracted tool call: %s (id=%s)\n", tc.FunctionCall.Name, tc.ID)
					}
				}
			}
		}
	}

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
	}, nil
}

// anthropicProvider wraps langchaingo's Anthropic client
type anthropicProvider struct {
	llm    *anthropic.LLM
	config *AIConfig
}

func newAnthropicProvider(config *AIConfig) (*anthropicProvider, error) {
	opts := []anthropic.Option{
		anthropic.WithModel(config.Model),
		anthropic.WithToken(config.APIKey),
	}

	llm, err := anthropic.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	return &anthropicProvider{
		llm:    llm,
		config: config,
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
					fmt.Printf("[DEBUG] Skipped tool_result ID: %s (no matching tool_use)\n", toolResultID)
					continue
				}
				// fmt.Printf("[DEBUG] Including tool_result ID: %s (has matching tool_use)\n", toolResultID)
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

	// Debug: Show summary of tool_use/tool_result filtering
	fmt.Printf("[DEBUG] Message filtering complete: %d total messages, %d included tool_use IDs\n",
		len(apiMessages), len(includedToolUseIDs))

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
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
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
			fmt.Printf("[DEBUG] Extracted tool call: %s (id=%s)\n", block.Name, block.ID)
		}
	}

	// Send text content to channel and close immediately (non-streaming for now)
	go func() {
		if textContent != "" {
			textChan <- textContent
		}
		close(textChan)
	}()

	fmt.Printf("[DEBUG Anthropic Direct] StopReason: %s, ToolCalls: %d, Text: %d chars\n",
		anthropicResp.StopReason, len(toolCalls), len(textContent))

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
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
