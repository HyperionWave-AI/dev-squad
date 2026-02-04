package aiservice

// OpenAI provider implementation using the official OpenAI Go SDK
// https://pkg.go.dev/github.com/openai/openai-go

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// openAIProvider implements ChatProvider and ToolCapableProvider using the official OpenAI Go SDK
type openAIProvider struct {
	client       openai.Client
	config       *AIConfig
	tokenLogger  *TokenUsageLogger
	metricsStore MetricsStore
}

// newOpenAIProvider creates a new OpenAI provider with the official SDK
func newOpenAIProvider(config *AIConfig, metricsStore MetricsStore) (*openAIProvider, error) {
	// Build options
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	// Support custom base URL (Ollama, Azure OpenAI, etc.)
	if config.ProviderURL != "" {
		baseURL := strings.TrimSuffix(config.ProviderURL, "/")
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &openAIProvider{
		client:       client,
		config:       config,
		tokenLogger:  NewTokenUsageLogger(),
		metricsStore: metricsStore,
	}, nil
}

// StreamChat implements ChatProvider for simple text streaming
func (p *openAIProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Convert messages to OpenAI format
	openAIMessages := p.convertMessagesToOpenAI(messages)

	// Build params
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.Model),
		Messages: openAIMessages,
	}

	if p.config.Temperature > 0 {
		params.Temperature = openai.Float(p.config.Temperature)
	}
	if p.config.MaxOutputTokens > 0 {
		params.MaxTokens = openai.Int(int64(p.config.MaxOutputTokens))
	}

	// Create output channel
	outputChan := make(chan string, 100)

	// Start streaming in goroutine
	go func() {
		defer close(outputChan)

		stream := p.client.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					select {
					case <-ctx.Done():
						return
					case outputChan <- content:
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			if err != context.Canceled {
				select {
				case <-ctx.Done():
				case outputChan <- fmt.Sprintf("ERROR: %v", err):
				}
			}
		}
	}()

	return outputChan, nil
}

// SupportsTools returns true - OpenAI models support function calling
func (p *openAIProvider) SupportsTools() bool {
	return true
}

// StreamChatWithTools implements ToolCapableProvider for tool-enabled chat
func (p *openAIProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error) {
	startTime := time.Now()

	// Convert messages and tools to OpenAI format
	openAIMessages := p.convertMessagesToOpenAI(messages)
	openAITools := p.convertToolsToOpenAI(tools)

	// Build params - using non-streaming for tool calls to get complete response
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.Model),
		Messages: openAIMessages,
	}

	if len(openAITools) > 0 {
		params.Tools = openAITools
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	}
	if p.config.Temperature > 0 {
		params.Temperature = openai.Float(p.config.Temperature)
	}
	if p.config.MaxOutputTokens > 0 {
		params.MaxTokens = openai.Int(int64(p.config.MaxOutputTokens))
	}

	// LOG REQUEST
	if httpLogger != nil {
		apiMessages := p.buildOpenAIMessagesForLogging(messages)
		apiTools := p.buildOpenAIToolsForLogging(tools)
		httpLogger.LogOpenAIRequest(
			p.config.Provider,
			apiMessages,
			apiTools,
			map[string]interface{}{
				"temperature":     p.config.Temperature,
				"maxOutputTokens": p.config.MaxOutputTokens,
				"model":           p.config.Model,
			},
		)
	}

	// Make the API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	// LOG RESPONSE
	if httpLogger != nil {
		httpLogger.LogOpenAIResponse(p.config.Provider, p.completionToLoggable(completion), nil)
	}

	// Extract text content and tool calls
	textChan := make(chan string, 1000)
	var toolCalls []ToolCall
	var textContent string
	var stopReason string

	if len(completion.Choices) > 0 {
		choice := completion.Choices[0]
		textContent = choice.Message.Content
		stopReason = choice.FinishReason

		// Extract tool calls
		for _, tc := range choice.Message.ToolCalls {
			args := p.parseToolCallArgs(tc.Function.Arguments)
			toolCalls = append(toolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: args,
			})
		}
	}

	// Send text content to channel and close
	go func() {
		if textContent != "" {
			textChan <- textContent
		}
		close(textChan)
	}()

	// Extract token usage
	tokenUsage := p.extractTokenUsage(completion)
	p.tokenLogger.LogUsage(tokenUsage)

	// Record metrics
	durationMs := time.Since(startTime).Milliseconds()
	metric := &ProviderMetric{
		ID:               fmt.Sprintf("openai-%d", time.Now().UnixNano()),
		Provider:         "openai",
		Model:            p.config.Model,
		PromptTokens:     tokenUsage.PromptTokens,
		CompletionTokens: tokenUsage.CompletionTokens,
		TotalTokens:      tokenUsage.TotalTokens,
		Cost:             CalculateOpenAICost(p.config.Model, tokenUsage.PromptTokens, tokenUsage.CompletionTokens),
		DurationMs:       durationMs,
		Success:          true,
		Timestamp:        time.Now(),
	}
	if p.metricsStore != nil {
		if err := p.metricsStore.RecordProviderMetric(metric); err != nil {
			fmt.Printf("[Metrics] Failed to record OpenAI provider metric: %v\n", err)
		}
	}

	// Map OpenAI finish_reason to our stop_reason format
	stopReason = p.mapStopReason(stopReason)

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
		StopReason:  stopReason,
		TokenUsage:  tokenUsage,
	}, nil
}

// convertMessagesToOpenAI converts internal Message format to OpenAI SDK format
// CRITICAL: Groups consecutive tool_call messages into a single assistant message
// The OpenAI API requires all tool calls from one AI turn to be in ONE assistant message
func (p *openAIProvider) convertMessagesToOpenAI(messages []Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	// First pass: collect consecutive tool_calls into groups
	i := 0
	for i < len(messages) {
		msg := messages[i]

		switch msg.Role {
		case "system":
			result = append(result, openai.SystemMessage(msg.Content))
			i++

		case "user":
			result = append(result, openai.UserMessage(msg.Content))
			i++

		case "assistant":
			result = append(result, openai.AssistantMessage(msg.Content))
			i++

		case "tool_call":
			// Collect ALL consecutive tool_call messages into ONE assistant message
			// This is CRITICAL for OpenAI API compliance
			var toolCalls []openai.ChatCompletionMessageToolCallParam
			var content string

			for i < len(messages) && messages[i].Role == "tool_call" {
				tc := messages[i]
				if tc.ToolCall != nil {
					argsJSON, _ := json.Marshal(tc.ToolCall.Args)
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID: tc.ToolCall.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.ToolCall.Name,
							Arguments: string(argsJSON),
						},
					})
					// Use content from first tool_call message
					if content == "" && tc.Content != "" {
						content = tc.Content
					}
				}
				i++
			}

			if len(toolCalls) > 0 {
				assistantMsg := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCalls,
				}
				if content != "" {
					assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(content),
					}
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				})
			}

		case "tool_result":
			// Tool response message
			if msg.ToolResult != nil {
				var content string
				if msg.ToolResult.Error != "" {
					content = msg.ToolResult.Error
				} else {
					switch v := msg.ToolResult.Output.(type) {
					case string:
						content = v
					default:
						outputJSON, _ := json.Marshal(v)
						content = string(outputJSON)
					}
				}
				result = append(result, openai.ToolMessage(content, msg.ToolResult.ID))
			}
			i++

		default:
			// Skip unknown roles
			i++
		}
	}

	return result
}

// convertToolsToOpenAI converts internal Tool format to OpenAI SDK format
func (p *openAIProvider) convertToolsToOpenAI(tools []Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, 0, len(tools))

	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}

		// Convert parameters map to FunctionParameters
		params := shared.FunctionParameters{}
		if tool.Function.Parameters != nil {
			params = shared.FunctionParameters(tool.Function.Parameters)
		}

		result = append(result, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: openai.String(tool.Function.Description),
				Parameters:  params,
			},
		})
	}

	return result
}

// parseToolCallArgs parses the JSON arguments string from a tool call
func (p *openAIProvider) parseToolCallArgs(argsJSON string) map[string]interface{} {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		// Return empty map on parse error
		return make(map[string]interface{})
	}
	return args
}

// extractTokenUsage extracts token usage from the completion response
func (p *openAIProvider) extractTokenUsage(completion *openai.ChatCompletion) *TokenUsage {
	if completion == nil {
		return &TokenUsage{
			Provider:  "openai",
			Model:     p.config.Model,
			Timestamp: time.Now(),
		}
	}

	return &TokenUsage{
		PromptTokens:     int(completion.Usage.PromptTokens),
		CompletionTokens: int(completion.Usage.CompletionTokens),
		TotalTokens:      int(completion.Usage.TotalTokens),
		Provider:         "openai",
		Model:            p.config.Model,
		Timestamp:        time.Now(),
	}
}

// mapStopReason maps OpenAI finish reasons to our internal stop reason format
func (p *openAIProvider) mapStopReason(finishReason string) string {
	switch finishReason {
	case "tool_calls":
		return "tool_use"
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "content_filter"
	default:
		return finishReason
	}
}

// buildOpenAIMessagesForLogging converts messages to a format suitable for logging
// CRITICAL: Groups consecutive tool_call messages to match actual API request format
func (p *openAIProvider) buildOpenAIMessagesForLogging(messages []Message) []map[string]interface{} {
	apiMessages := make([]map[string]interface{}, 0, len(messages))

	i := 0
	for i < len(messages) {
		msg := messages[i]

		switch msg.Role {
		case "system", "user", "assistant":
			apiMessages = append(apiMessages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
			i++

		case "tool_call":
			// Collect ALL consecutive tool_call messages into ONE assistant message
			var toolCalls []map[string]interface{}
			var content string

			for i < len(messages) && messages[i].Role == "tool_call" {
				tc := messages[i]
				if tc.ToolCall != nil {
					argsJSON, _ := json.Marshal(tc.ToolCall.Args)
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   tc.ToolCall.ID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      tc.ToolCall.Name,
							"arguments": string(argsJSON),
						},
					})
					if content == "" && tc.Content != "" {
						content = tc.Content
					}
				}
				i++
			}

			if len(toolCalls) > 0 {
				toolCallMsg := map[string]interface{}{
					"role":       "assistant",
					"tool_calls": toolCalls,
				}
				if content != "" {
					toolCallMsg["content"] = content
				}
				apiMessages = append(apiMessages, toolCallMsg)
			}

		case "tool_result":
			if msg.ToolResult != nil {
				var content string
				if msg.ToolResult.Error != "" {
					content = msg.ToolResult.Error
				} else {
					switch v := msg.ToolResult.Output.(type) {
					case string:
						content = v
					default:
						outputJSON, _ := json.Marshal(v)
						content = string(outputJSON)
					}
				}
				apiMessages = append(apiMessages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": msg.ToolResult.ID,
					"content":      content,
				})
			}
			i++

		default:
			i++
		}
	}

	return apiMessages
}

// buildOpenAIToolsForLogging converts tools to a format suitable for logging
func (p *openAIProvider) buildOpenAIToolsForLogging(tools []Tool) []map[string]interface{} {
	apiTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}
		apiTools = append(apiTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		})
	}
	return apiTools
}

// completionToLoggable converts an OpenAI completion to a loggable format
func (p *openAIProvider) completionToLoggable(completion *openai.ChatCompletion) openAICompletionResponse {
	resp := openAICompletionResponse{
		ID:      completion.ID,
		Object:  string(completion.Object),
		Created: completion.Created,
		Model:   completion.Model,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}

	for _, choice := range completion.Choices {
		var toolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}

		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		resp.Choices = append(resp.Choices, struct {
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
		}{
			Index: int(choice.Index),
			Message: struct {
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
			}{
				Role:      string(choice.Message.Role),
				Content:   choice.Message.Content,
				ToolCalls: toolCalls,
			},
			FinishReason: choice.FinishReason,
		})
	}

	return resp
}
