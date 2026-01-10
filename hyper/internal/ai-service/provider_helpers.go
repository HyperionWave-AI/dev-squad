package aiservice

// convertToolsToJSON converts native Tool type to JSON-serializable format
// Used for HTTP logging of tool definitions
func convertToolsToJSON(tools []Tool) []interface{} {
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

// convertMessagesToJSON converts internal Message type to JSON-serializable format
// Used for HTTP logging of conversation history
func convertMessagesToJSON(messages []Message) []interface{} {
	result := make([]interface{}, 0, len(messages))

	for _, msg := range messages {
		msgMap := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}

		if msg.ToolCall != nil {
			msgMap["toolCall"] = map[string]interface{}{
				"id":   msg.ToolCall.ID,
				"name": msg.ToolCall.Name,
				"args": msg.ToolCall.Args,
			}
		}

		if msg.ToolResult != nil {
			msgMap["toolResult"] = map[string]interface{}{
				"id":     msg.ToolResult.ID,
				"name":   msg.ToolResult.Name,
				"output": msg.ToolResult.Output,
				"error":  msg.ToolResult.Error,
			}
		}

		if msg.Provider != "" {
			msgMap["provider"] = msg.Provider
		}

		result = append(result, msgMap)
	}

	return result
}
