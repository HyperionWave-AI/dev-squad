package aiservice

import (
	"github.com/tmc/langchaingo/llms"
)

// convertMsgContentsToJSON converts langchain MessageContent to JSON-serializable format
func convertMsgContentsToJSON(msgContents []llms.MessageContent) []interface{} {
	result := make([]interface{}, 0, len(msgContents))

	for _, msg := range msgContents {
		msgMap := map[string]interface{}{
			"role": string(msg.Role),
		}

		// Convert parts
		if len(msg.Parts) > 0 {
			parts := make([]interface{}, 0, len(msg.Parts))
			for _, part := range msg.Parts {
				partMap := map[string]interface{}{}

				switch p := part.(type) {
				case llms.TextContent:
					partMap["type"] = "text"
					partMap["text"] = p.Text

				case llms.ToolCall:
					partMap["type"] = "tool_call"
					partMap["id"] = p.ID
					partMap["tool_type"] = p.Type
					if p.FunctionCall != nil {
						partMap["function_call"] = map[string]interface{}{
							"name":      p.FunctionCall.Name,
							"arguments": p.FunctionCall.Arguments,
						}
					}

				case llms.ToolCallResponse:
					partMap["type"] = "tool_call_response"
					partMap["tool_call_id"] = p.ToolCallID
					partMap["name"] = p.Name
					partMap["content"] = p.Content

				default:
					partMap["type"] = "unknown"
				}

				parts = append(parts, partMap)
			}
			msgMap["parts"] = parts
		}

		result = append(result, msgMap)
	}

	return result
}

// convertToolsToJSON converts langchain Tools to JSON-serializable format
func convertToolsToJSON(tools []llms.Tool) []interface{} {
	result := make([]interface{}, 0, len(tools))

	for _, tool := range tools {
		toolMap := map[string]interface{}{
			"type": tool.Type,
		}

		if tool.Function != nil {
			toolMap["function"] = map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			}
		}

		result = append(result, toolMap)
	}

	return result
}

// convertContentResponseToJSON converts langchain ContentResponse to JSON-serializable format
func convertContentResponseToJSON(resp *llms.ContentResponse) interface{} {
	if resp == nil {
		return nil
	}

	result := map[string]interface{}{}

	// Convert choices
	if len(resp.Choices) > 0 {
		choices := make([]interface{}, 0, len(resp.Choices))
		for _, choice := range resp.Choices {
			choiceMap := map[string]interface{}{
				"content":           choice.Content,
				"stop_reason":       choice.StopReason,
				"generation_info":   choice.GenerationInfo,
				"reasoning_content": choice.ReasoningContent,
			}

			// Add function call if present
			if choice.FuncCall != nil {
				choiceMap["func_call"] = map[string]interface{}{
					"name":      choice.FuncCall.Name,
					"arguments": choice.FuncCall.Arguments,
				}
			}

			// Add tool calls if present
			if len(choice.ToolCalls) > 0 {
				toolCalls := make([]interface{}, 0, len(choice.ToolCalls))
				for _, tc := range choice.ToolCalls {
					tcMap := map[string]interface{}{
						"id":   tc.ID,
						"type": tc.Type,
					}
					if tc.FunctionCall != nil {
						tcMap["function_call"] = map[string]interface{}{
							"name":      tc.FunctionCall.Name,
							"arguments": tc.FunctionCall.Arguments,
						}
					}
					toolCalls = append(toolCalls, tcMap)
				}
				choiceMap["tool_calls"] = toolCalls
			}

			choices = append(choices, choiceMap)
		}
		result["choices"] = choices
	}

	return result
}
