package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WorkflowResourceHandler manages workflow visibility resources
type WorkflowResourceHandler struct {
	taskStorage storage.TaskStorage
}

// NewWorkflowResourceHandler creates a new workflow resource handler
func NewWorkflowResourceHandler(taskStorage storage.TaskStorage) *WorkflowResourceHandler {
	return &WorkflowResourceHandler{
		taskStorage: taskStorage,
	}
}

// AgentStatus represents the current status of an agent
type AgentStatus struct {
	AgentName      string                 `json:"agentName"`
	Status         string                 `json:"status"` // "working", "blocked", "idle"
	CurrentTask    *storage.AgentTask     `json:"currentTask,omitempty"`
	TaskCount      int                    `json:"taskCount"`
	CompletedCount int                    `json:"completedCount"`
	BlockedCount   int                    `json:"blockedCount"`
	LastUpdated    time.Time              `json:"lastUpdated"`
}

// TaskQueueItem represents a pending task in the queue
type TaskQueueItem struct {
	TaskID      string       `json:"taskId"`
	AgentName   string       `json:"agentName"`
	Role        string       `json:"role"`
	Priority    int          `json:"priority"`
	CreatedAt   time.Time    `json:"createdAt"`
	TodoCount   int          `json:"todoCount"`
	HumanTaskID string       `json:"humanTaskId"`
}

// TaskDependency represents a relationship between tasks
type TaskDependency struct {
	TaskID        string   `json:"taskId"`
	AgentName     string   `json:"agentName"`
	Role          string   `json:"role"`
	Status        string   `json:"status"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	BlockedBy     []string `json:"blockedBy,omitempty"`
	Blocks        []string `json:"blocks,omitempty"`
}

// RegisterWorkflowResources registers all workflow resources with the MCP server
func (h *WorkflowResourceHandler) RegisterWorkflowResources(server *mcp.Server) error {
	// Register active-agents resource
	activeAgentsResource := &mcp.Resource{
		URI:         "hyperion://workflow/active-agents",
		Name:        "Active Agents Status",
		Description: "Real-time status of all agents (working, blocked, idle)",
		MIMEType:    "application/json",
	}
	server.AddResource(activeAgentsResource, h.handleActiveAgents)

	// Register task-queue resource
	taskQueueResource := &mcp.Resource{
		URI:         "hyperion://workflow/task-queue",
		Name:        "Task Queue",
		Description: "Pending tasks ordered by priority and creation time",
		MIMEType:    "application/json",
	}
	server.AddResource(taskQueueResource, h.handleTaskQueue)

	// Register dependencies resource
	dependenciesResource := &mcp.Resource{
		URI:         "hyperion://workflow/dependencies",
		Name:        "Task Dependencies",
		Description: "Task dependency graph showing blocking relationships",
		MIMEType:    "application/json",
	}
	server.AddResource(dependenciesResource, h.handleDependencies)

	return nil
}

// handleActiveAgents computes and returns active agent status
func (h *WorkflowResourceHandler) handleActiveAgents(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	allAgentTasks := h.taskStorage.ListAllAgentTasks()

	// Group tasks by agent name
	agentTaskMap := make(map[string][]*storage.AgentTask)
	for _, task := range allAgentTasks {
		agentTaskMap[task.AgentName] = append(agentTaskMap[task.AgentName], task)
	}

	// Build agent status list
	agentStatuses := make([]AgentStatus, 0, len(agentTaskMap))
	for agentName, tasks := range agentTaskMap {
		status := AgentStatus{
			AgentName:      agentName,
			Status:         "idle",
			TaskCount:      len(tasks),
			CompletedCount: 0,
			BlockedCount:   0,
			LastUpdated:    time.Now().UTC(),
		}

		var mostRecentTask *storage.AgentTask
		mostRecentTime := time.Time{}

		for _, task := range tasks {
			// Count task statuses
			switch task.Status {
			case storage.TaskStatusCompleted:
				status.CompletedCount++
			case storage.TaskStatusBlocked:
				status.BlockedCount++
			}

			// Find most recent task
			if task.UpdatedAt.After(mostRecentTime) {
				mostRecentTime = task.UpdatedAt
				mostRecentTask = task
			}
		}

		// Determine agent status based on most recent task
		if mostRecentTask != nil {
			status.LastUpdated = mostRecentTask.UpdatedAt
			status.CurrentTask = mostRecentTask

			switch mostRecentTask.Status {
			case storage.TaskStatusInProgress:
				status.Status = "working"
			case storage.TaskStatusBlocked:
				status.Status = "blocked"
			case storage.TaskStatusPending:
				status.Status = "idle"
			case storage.TaskStatusCompleted:
				// Check if there are pending tasks
				hasPending := false
				for _, task := range tasks {
					if task.Status == storage.TaskStatusPending {
						hasPending = true
						break
					}
				}
				if hasPending {
					status.Status = "idle"
				} else {
					status.Status = "idle"
				}
			}
		}

		agentStatuses = append(agentStatuses, status)
	}

	// Sort by agent name for consistent output
	sort.Slice(agentStatuses, func(i, j int) bool {
		return agentStatuses[i].AgentName < agentStatuses[j].AgentName
	})

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"agents":     agentStatuses,
		"totalCount": len(agentStatuses),
		"timestamp":  time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent status: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://workflow/active-agents",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleTaskQueue returns pending tasks ordered by priority
func (h *WorkflowResourceHandler) handleTaskQueue(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	allAgentTasks := h.taskStorage.ListAllAgentTasks()

	// Filter pending tasks
	pendingTasks := make([]TaskQueueItem, 0)
	for _, task := range allAgentTasks {
		if task.Status == storage.TaskStatusPending {
			queueItem := TaskQueueItem{
				TaskID:      task.ID,
				AgentName:   task.AgentName,
				Role:        task.Role,
				Priority:    calculatePriority(task),
				CreatedAt:   task.CreatedAt,
				TodoCount:   len(task.Todos),
				HumanTaskID: task.HumanTaskID,
			}
			pendingTasks = append(pendingTasks, queueItem)
		}
	}

	// Sort by priority (descending) and creation time (ascending)
	sort.Slice(pendingTasks, func(i, j int) bool {
		if pendingTasks[i].Priority != pendingTasks[j].Priority {
			return pendingTasks[i].Priority > pendingTasks[j].Priority
		}
		return pendingTasks[i].CreatedAt.Before(pendingTasks[j].CreatedAt)
	})

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"queue":      pendingTasks,
		"totalCount": len(pendingTasks),
		"timestamp":  time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task queue: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://workflow/task-queue",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleDependencies analyzes and returns task dependency graph
func (h *WorkflowResourceHandler) handleDependencies(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	allAgentTasks := h.taskStorage.ListAllAgentTasks()

	// Build dependency graph
	dependencies := make([]TaskDependency, 0, len(allAgentTasks))
	taskMap := make(map[string]*storage.AgentTask)

	// First pass: build task map
	for _, task := range allAgentTasks {
		taskMap[task.ID] = task
	}

	// Second pass: analyze dependencies
	for _, task := range allAgentTasks {
		dep := TaskDependency{
			TaskID:    task.ID,
			AgentName: task.AgentName,
			Role:      task.Role,
			Status:    string(task.Status),
			DependsOn: []string{},
			BlockedBy: []string{},
			Blocks:    []string{},
		}

		// Analyze task notes and context for dependency keywords
		// Look for patterns like "depends on", "blocked by", "waiting for"
		if task.Status == storage.TaskStatusBlocked {
			// Extract blocking information from notes
			if task.Notes != "" {
				// Simple heuristic: if blocked, it might depend on other tasks
				// In a real implementation, this would parse the notes for task IDs
				dep.BlockedBy = extractTaskReferences(task.Notes, taskMap)
			}
		}

		// Check if other tasks depend on this one
		for _, otherTask := range allAgentTasks {
			if otherTask.ID != task.ID {
				if containsTaskReference(otherTask.Notes, task.ID) ||
					containsTaskReference(otherTask.PriorWorkSummary, task.ID) {
					dep.Blocks = append(dep.Blocks, otherTask.ID)
				}
			}
		}

		dependencies = append(dependencies, dep)
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"dependencies": dependencies,
		"totalCount":   len(dependencies),
		"timestamp":    time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dependencies: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://workflow/dependencies",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// calculatePriority determines task priority based on various factors
func calculatePriority(task *storage.AgentTask) int {
	priority := 0

	// Base priority on TODO count (more TODOs = higher priority)
	priority += len(task.Todos) * 10

	// Increase priority for tasks with context (ready to work on)
	if task.ContextSummary != "" {
		priority += 50
	}

	// Increase priority if files are specified (well-defined task)
	if len(task.FilesModified) > 0 {
		priority += 30
	}

	// Increase priority for tasks with prior work (continuation tasks)
	if task.PriorWorkSummary != "" {
		priority += 40
	}

	// Age factor: older tasks get higher priority
	daysSinceCreation := time.Since(task.CreatedAt).Hours() / 24
	priority += int(daysSinceCreation * 5)

	return priority
}

// extractTaskReferences extracts task IDs from text (simple heuristic)
func extractTaskReferences(text string, taskMap map[string]*storage.AgentTask) []string {
	references := []string{}
	// Simple implementation: look for UUID patterns in text
	// In production, this would use more sophisticated pattern matching
	for taskID := range taskMap {
		if containsTaskReference(text, taskID) {
			references = append(references, taskID)
		}
	}
	return references
}

// containsTaskReference checks if text contains a reference to a task ID
func containsTaskReference(text, taskID string) bool {
	// Simple substring check - in production, use regex or more sophisticated matching
	if len(taskID) == 0 || len(text) == 0 {
		return false
	}

	if stringContains(text, taskID) {
		return true
	}

	// Check for short UUID prefix (first 8 chars if available)
	if len(taskID) >= 8 {
		return stringContains(text, fmt.Sprintf("task %s", taskID[:8]))
	}

	return false
}

// stringContains is a simple case-insensitive contains check
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

// findSubstring performs simple substring search
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
