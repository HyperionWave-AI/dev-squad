package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
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

// StreamChatWithTools implements tool calling for OpenAI using GenerateContent
func (p *openAIProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
	// Convert messages to LangChain MessageContent format
	msgContents := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		case "system":
			msgType = llms.ChatMessageTypeSystem
		default:
			msgType = llms.ChatMessageTypeHuman
		}
		msgContents = append(msgContents, llms.TextParts(msgType, msg.Content))
	}

	// Create text channel for streaming
	textChan := make(chan string, 1000) // Larger buffer to prevent blocking
	var toolCalls []ToolCall

	// Prepare streaming function (non-blocking) with tool call filtering
	streamFunc := func(ctx context.Context, chunk []byte) error {
		chunkStr := string(chunk)

		// Filter out tool call JSON arrays that match the pattern:
		// [{"id":"call_*","type":"function","function":{...}}]
		// These are metadata that should not appear in the message content
		if strings.HasPrefix(strings.TrimSpace(chunkStr), "[{\"id\":\"call_") {
			// This looks like a tool call JSON array - skip it
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case textChan <- chunkStr:
			return nil
		default:
			// Channel full, skip chunk (non-blocking)
			// This prevents GenerateContent from hanging
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

	// Call GenerateContent in goroutine to avoid blocking
	type generateResult struct {
		resp *llms.ContentResponse
		err  error
	}
	resultChan := make(chan generateResult, 1)

	go func() {
		resp, err := p.llm.GenerateContent(ctx, msgContents, opts...)
		resultChan <- generateResult{resp: resp, err: err}
		close(textChan) // Close after generation completes
	}()

	// Wait for generation to complete
	result := <-resultChan

	if result.err != nil && result.err != context.Canceled {
		return nil, fmt.Errorf("failed to generate content: %w", result.err)
	}

	// Extract tool calls from response
	if result.resp != nil && len(result.resp.Choices) > 0 {
		choice := result.resp.Choices[0]

		// Check for function call (FuncCall field)
		if choice.FuncCall != nil {
			var args map[string]interface{}
			if choice.FuncCall.Arguments != "" {
				if err := json.Unmarshal([]byte(choice.FuncCall.Arguments), &args); err == nil {
					toolCalls = append(toolCalls, ToolCall{
						ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
						Name: choice.FuncCall.Name,
						Args: args,
					})
				}
			}
		}

		// Also check for tool calls in content (newer format)
		if choice.Content != "" {
			// Try to parse tool calls from JSON in content
			var toolCallData struct {
				ToolCalls []struct {
					ID       string                 `json:"id"`
					Type     string                 `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			}
			if err := json.Unmarshal([]byte(choice.Content), &toolCallData); err == nil {
				for _, tc := range toolCallData.ToolCalls {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
						toolCalls = append(toolCalls, ToolCall{
							ID:   tc.ID,
							Name: tc.Function.Name,
							Args: args,
						})
					}
				}
			}
		}
	}

	response := &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
	}

	return response, nil
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

// SupportsTools returns true for Anthropic models that support tool use
func (p *anthropicProvider) SupportsTools() bool {
	// Claude 3 models (Sonnet, Opus, Haiku) support tool use
	model := strings.ToLower(p.config.Model)
	return strings.Contains(model, "claude-3") || strings.Contains(model, "claude-3.5")
}

// StreamChatWithTools implements tool calling for Anthropic
func (p *anthropicProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []llms.Tool) (*ToolResponse, error) {
	content := p.messagesToContent(messages)

	// Create text channel
	textChan := make(chan string, 100)

	// Start streaming in goroutine
	go func() {
		defer close(textChan)

		// Build call options with tools
		callOpts := []llms.CallOption{
			llms.WithTemperature(p.config.Temperature),
			llms.WithTools(tools),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case textChan <- string(chunk):
					return nil
				}
			}),
		}

		// Add max tokens if configured
		if p.config.MaxOutputTokens > 0 {
			callOpts = append(callOpts, llms.WithMaxTokens(p.config.MaxOutputTokens))
		}

		// Call with tools
		_, err := p.llm.Call(ctx, content, callOpts...)
		if err != nil && err != context.Canceled {
			select {
			case <-ctx.Done():
			case textChan <- fmt.Sprintf("ERROR: %v", err):
			}
		}
	}()

	// TODO: Extract tool calls from response
	// For now, return empty tool calls - will be implemented when we have access to response messages
	response := &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   []ToolCall{},
	}

	return response, nil
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
