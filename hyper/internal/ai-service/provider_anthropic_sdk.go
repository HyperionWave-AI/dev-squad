package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicSDKProvider implements ChatProvider and ToolCapableProvider using the official Anthropic Go SDK.
type anthropicSDKProvider struct {
	client       anthropic.Client
	config       *AIConfig
	tokenLogger  *TokenUsageLogger
	metricsStore MetricsStore
}

func newAnthropicSDKProvider(config *AIConfig, metricsStore MetricsStore) (*anthropicSDKProvider, error) {
	opts := []anthropicoption.RequestOption{
		anthropicoption.WithAPIKey(config.APIKey),
	}
	if config.ProviderURL != "" {
		opts = append(opts, anthropicoption.WithBaseURL(strings.TrimSuffix(config.ProviderURL, "/")))
	}

	return &anthropicSDKProvider{
		client:       anthropic.NewClient(opts...),
		config:       config,
		tokenLogger:  NewTokenUsageLogger(),
		metricsStore: metricsStore,
	}, nil
}

func (p *anthropicSDKProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	startTime := time.Now()
	outputChan := make(chan string, 100)

	go func() {
		defer close(outputChan)

		resp, err := p.requestMessage(ctx, messages, nil)
		if err != nil {
			if err != context.Canceled {
				select {
				case <-ctx.Done():
				case outputChan <- fmt.Sprintf("ERROR: %v", err):
				}
			}
			return
		}

		textContent, _, err := p.extractTextAndToolCalls(resp)
		if err != nil {
			select {
			case <-ctx.Done():
			case outputChan <- fmt.Sprintf("ERROR: %v", err):
			}
			return
		}

		if textContent != "" {
			select {
			case <-ctx.Done():
			case outputChan <- textContent:
			}
		}

		tokenUsage := p.extractTokenUsage(resp)
		p.tokenLogger.LogUsage(tokenUsage)
		p.recordMetrics(tokenUsage, startTime)
	}()

	return outputChan, nil
}

func (p *anthropicSDKProvider) SupportsTools() bool {
	return true
}

func (p *anthropicSDKProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ToolResponse, error) {
	startTime := time.Now()

	resp, err := p.requestMessage(ctx, messages, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}

	textContent, toolCalls, err := p.extractTextAndToolCalls(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	textChan := make(chan string, 1)
	go func() {
		if textContent != "" {
			textChan <- textContent
		}
		close(textChan)
	}()

	tokenUsage := p.extractTokenUsage(resp)
	p.tokenLogger.LogUsage(tokenUsage)
	p.recordMetrics(tokenUsage, startTime)

	stopReason := string(resp.StopReason)
	if stopReason == "" {
		stopReason = "end_turn"
	}

	return &ToolResponse{
		TextChannel: textChan,
		ToolCalls:   toolCalls,
		StopReason:  stopReason,
		TokenUsage:  tokenUsage,
	}, nil
}

func (p *anthropicSDKProvider) requestMessage(ctx context.Context, messages []Message, tools []Tool) (*anthropic.Message, error) {
	params, err := p.buildMessageParams(messages, tools)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *anthropicSDKProvider) buildMessageParams(messages []Message, tools []Tool) (anthropic.MessageNewParams, error) {
	anthropicMessages, systemBlocks := p.convertMessagesToAnthropic(messages)

	maxTokens := p.config.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(p.config.Model),
		MaxTokens:   int64(maxTokens),
		Messages:    anthropicMessages,
		Temperature: anthropic.Float(p.config.Temperature),
	}

	if len(systemBlocks) > 0 {
		params.System = systemBlocks
	}

	if len(tools) > 0 {
		anthropicTools, err := p.convertToolsToAnthropic(tools)
		if err != nil {
			return anthropic.MessageNewParams{}, err
		}
		params.Tools = anthropicTools
		params.ToolChoice = anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{},
		}
	}

	return params, nil
}

func (p *anthropicSDKProvider) convertMessagesToAnthropic(messages []Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam) {
	converted := make([]anthropic.MessageParam, 0, len(messages))
	systemBlocks := make([]anthropic.TextBlockParam, 0, 2)

	for i := 0; i < len(messages); {
		msg := messages[i]

		switch msg.Role {
		case "system":
			if strings.TrimSpace(msg.Content) != "" {
				systemBlocks = append(systemBlocks, anthropic.TextBlockParam{Text: msg.Content})
			}
			i++

		case "user":
			converted = append(converted, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			i++

		case "assistant":
			if msg.Content != "" {
				converted = append(converted, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
			i++

		case "tool_call":
			blocks := make([]anthropic.ContentBlockParamUnion, 0, 4)
			for i < len(messages) && messages[i].Role == "tool_call" {
				tcMsg := messages[i]
				if tcMsg.ToolCall != nil {
					toolID := sanitizeToolID(tcMsg.ToolCall.ID)
					if toolID == "" {
						toolID = fmt.Sprintf("tool_%d", i)
					}
					blocks = append(blocks, anthropic.NewToolUseBlock(toolID, tcMsg.ToolCall.Args, tcMsg.ToolCall.Name))
				}
				i++
			}
			if len(blocks) > 0 {
				converted = append(converted, anthropic.NewAssistantMessage(blocks...))
			}

		case "tool_result":
			if msg.ToolResult != nil {
				toolID := sanitizeToolID(msg.ToolResult.ID)
				if toolID == "" {
					toolID = msg.ToolResult.ID
				}
				content, isError := formatAnthropicToolResult(msg.ToolResult)
				converted = append(converted, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(toolID, content, isError),
				))
			}
			i++

		default:
			i++
		}
	}

	return converted, systemBlocks
}

func (p *anthropicSDKProvider) convertToolsToAnthropic(tools []Tool) ([]anthropic.ToolUnionParam, error) {
	converted := make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		if tool.Function == nil || tool.Function.Name == "" {
			continue
		}

		schema, err := convertJSONSchemaToAnthropic(tool.Function.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to parse input schema for tool %q: %w", tool.Function.Name, err)
		}

		toolParam := anthropic.ToolUnionParamOfTool(schema, tool.Function.Name)
		if toolParam.OfTool != nil && tool.Function.Description != "" {
			toolParam.OfTool.Description = anthropic.String(tool.Function.Description)
		}

		converted = append(converted, toolParam)
	}

	return converted, nil
}

func convertJSONSchemaToAnthropic(schema map[string]interface{}) (anthropic.ToolInputSchemaParam, error) {
	if len(schema) == 0 {
		return anthropic.ToolInputSchemaParam{}, nil
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return anthropic.ToolInputSchemaParam{}, err
	}

	var anthropicSchema anthropic.ToolInputSchemaParam
	if err := json.Unmarshal(schemaBytes, &anthropicSchema); err != nil {
		return anthropic.ToolInputSchemaParam{}, err
	}

	return anthropicSchema, nil
}

func formatAnthropicToolResult(result *ToolResult) (content string, isError bool) {
	if result == nil {
		return "", false
	}

	if result.Error != "" {
		return result.Error, true
	}

	switch v := result.Output.(type) {
	case string:
		return v, false
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), false
		}
		return string(data), false
	}
}

func (p *anthropicSDKProvider) extractTextAndToolCalls(resp *anthropic.Message) (string, []ToolCall, error) {
	if resp == nil {
		return "", nil, fmt.Errorf("response is nil")
	}

	var textBuilder strings.Builder
	toolCalls := make([]ToolCall, 0, 4)

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			text := block.AsText().Text
			if text != "" {
				textBuilder.WriteString(text)
			}

		case "tool_use":
			toolUse := block.AsToolUse()
			args := map[string]interface{}{}
			if len(toolUse.Input) > 0 {
				if err := json.Unmarshal(toolUse.Input, &args); err != nil {
					args = map[string]interface{}{"_raw": string(toolUse.Input)}
				}
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:   toolUse.ID,
				Name: toolUse.Name,
				Args: args,
			})
		}
	}

	return textBuilder.String(), toolCalls, nil
}

func (p *anthropicSDKProvider) extractTokenUsage(resp *anthropic.Message) *TokenUsage {
	if resp == nil {
		return nil
	}

	inputTokens := int(resp.Usage.InputTokens)
	cacheCreationTokens := int(resp.Usage.CacheCreationInputTokens)
	cacheReadTokens := int(resp.Usage.CacheReadInputTokens)
	outputTokens := int(resp.Usage.OutputTokens)
	totalInput := inputTokens + cacheCreationTokens + cacheReadTokens

	if totalInput == 0 && outputTokens == 0 {
		return nil
	}

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

func (p *anthropicSDKProvider) recordMetrics(tokenUsage *TokenUsage, startTime time.Time) {
	if tokenUsage == nil || p.metricsStore == nil {
		return
	}

	metric := &ProviderMetric{
		ID:               fmt.Sprintf("anthropic-%d", time.Now().UnixNano()),
		Provider:         "anthropic",
		Model:            p.config.Model,
		PromptTokens:     tokenUsage.PromptTokens,
		CompletionTokens: tokenUsage.CompletionTokens,
		TotalTokens:      tokenUsage.TotalTokens,
		Cost:             CalculateAnthropicCost(p.config.Model, tokenUsage.PromptTokens, tokenUsage.CompletionTokens),
		DurationMs:       time.Since(startTime).Milliseconds(),
		Success:          true,
		Timestamp:        time.Now(),
	}

	if err := p.metricsStore.RecordProviderMetric(metric); err != nil {
		fmt.Printf("[Metrics] Failed to record Anthropic provider metric: %v\n", err)
	}
}
