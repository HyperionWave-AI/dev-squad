package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceHandler manages MCP resource operations
type ResourceHandler struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
}

// NewResourceHandler creates a new resource handler
func NewResourceHandler(taskStorage storage.TaskStorage, knowledgeStorage storage.KnowledgeStorage) *ResourceHandler {
	return &ResourceHandler{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
	}
}

// RegisterResourceHandlers registers all resource handlers with the MCP server
func (h *ResourceHandler) RegisterResourceHandlers(server *mcp.Server) error {
	// Register all human tasks as resources
	humanTasks := h.taskStorage.ListAllHumanTasks()
	for _, task := range humanTasks {
		resource := &mcp.Resource{
			URI:         fmt.Sprintf("hyperion://task/human/%s", task.ID),
			Name:        fmt.Sprintf("Human Task: %s", truncateText(task.Prompt, 50)),
			Description: fmt.Sprintf("Human task created at %s with status: %s", task.CreatedAt.Format("2006-01-02 15:04:05"), task.Status),
			MIMEType:    "application/json",
		}

		server.AddResource(resource, h.createResourceHandler(task.ID, "human", ""))
	}

	// Register all agent tasks as resources
	agentTasks := h.taskStorage.ListAllAgentTasks()
	for _, task := range agentTasks {
		resource := &mcp.Resource{
			URI:         fmt.Sprintf("hyperion://task/agent/%s/%s", task.AgentName, task.ID),
			Name:        fmt.Sprintf("Agent Task: %s - %s", task.AgentName, task.Role),
			Description: fmt.Sprintf("Agent task for %s (parent: %s) with status: %s", task.AgentName, task.HumanTaskID, task.Status),
			MIMEType:    "application/json",
		}

		server.AddResource(resource, h.createResourceHandler(task.ID, "agent", task.AgentName))
	}

	return nil
}

// createResourceHandler creates a handler function for a specific task resource
func (h *ResourceHandler) createResourceHandler(taskID, taskType, agentName string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		var task interface{}
		var uri string
		var err error

		switch taskType {
		case "human":
			task, err = h.taskStorage.GetHumanTask(taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve human task: %w", err)
			}
			uri = fmt.Sprintf("hyperion://task/human/%s", taskID)

		case "agent":
			agentTask, err := h.taskStorage.GetAgentTask(taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve agent task: %w", err)
			}

			// Verify agent name matches
			if agentTask.AgentName != agentName {
				return nil, fmt.Errorf("agent name mismatch: expected %s but task belongs to %s", agentName, agentTask.AgentName)
			}

			task = agentTask
			uri = fmt.Sprintf("hyperion://task/agent/%s/%s", agentName, taskID)

		default:
			return nil, fmt.Errorf("unknown task type: %s", taskType)
		}

		jsonData, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal task data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

// truncateText truncates text to a maximum length
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}