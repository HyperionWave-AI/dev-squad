package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	// Import the storage package from mcp-server
	"hyperion-coordinator-mcp/storage"
)

// REST API Data Transfer Objects (DTOs)
// These define clean JSON contracts for the REST API

type TaskDTO struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Status    string `json:"status"`
	Notes     string `json:"notes,omitempty"`
}

type TodoItemDTO struct {
	ID                        string  `json:"id"`
	Description               string  `json:"description"`
	Status                    string  `json:"status"`
	CreatedAt                 string  `json:"createdAt"`
	CompletedAt               *string `json:"completedAt,omitempty"`
	Notes                     string  `json:"notes,omitempty"`
	FilePath                  string  `json:"filePath,omitempty"`
	FunctionName              string  `json:"functionName,omitempty"`
	ContextHint               string  `json:"contextHint,omitempty"`
	HumanPromptNotes          string  `json:"humanPromptNotes,omitempty"`
	HumanPromptNotesAddedAt   *string `json:"humanPromptNotesAddedAt,omitempty"`
	HumanPromptNotesUpdatedAt *string `json:"humanPromptNotesUpdatedAt,omitempty"`
}

type AgentTaskDTO struct {
	ID                        string        `json:"id"`
	HumanTaskID               string        `json:"humanTaskId"`
	AgentName                 string        `json:"agentName"`
	Role                      string        `json:"role"`
	Todos                     []TodoItemDTO `json:"todos"`
	CreatedAt                 string        `json:"createdAt"`
	UpdatedAt                 string        `json:"updatedAt"`
	Status                    string        `json:"status"`
	Notes                     string        `json:"notes,omitempty"`
	ContextSummary            string        `json:"contextSummary,omitempty"`
	FilesModified             []string      `json:"filesModified,omitempty"`
	QdrantCollections         []string      `json:"qdrantCollections,omitempty"`
	PriorWorkSummary          string        `json:"priorWorkSummary,omitempty"`
	HumanPromptNotes          string        `json:"humanPromptNotes,omitempty"`
	HumanPromptNotesAddedAt   *string       `json:"humanPromptNotesAddedAt,omitempty"`
	HumanPromptNotesUpdatedAt *string       `json:"humanPromptNotesUpdatedAt,omitempty"`
}

type CreateHumanTaskRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type CreateHumanTaskResponse struct {
	Task TaskDTO `json:"task"`
}

type ListHumanTasksResponse struct {
	Tasks []TaskDTO `json:"tasks"`
	Count int       `json:"count"`
}

type UpdateTaskStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Notes  string `json:"notes,omitempty"`
}

type UpdateTaskStatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type CreateAgentTaskRequest struct {
	HumanTaskID       string                     `json:"humanTaskId" binding:"required"`
	AgentName         string                     `json:"agentName" binding:"required"`
	Role              string                     `json:"role" binding:"required"`
	Todos             []storage.TodoItemInput    `json:"todos" binding:"required"`
	ContextSummary    string                     `json:"contextSummary,omitempty"`
	FilesModified     []string                   `json:"filesModified,omitempty"`
	QdrantCollections []string                   `json:"qdrantCollections,omitempty"`
	PriorWorkSummary  string                     `json:"priorWorkSummary,omitempty"`
}

type CreateAgentTaskResponse struct {
	Task AgentTaskDTO `json:"task"`
}

type ListAgentTasksQuery struct {
	HumanTaskID string
	AgentName   string
	Offset      int
	Limit       int
}

type ListAgentTasksResponse struct {
	Tasks      []AgentTaskDTO `json:"tasks"`
	Count      int            `json:"count"`
	TotalCount int            `json:"totalCount"`
	Offset     int            `json:"offset"`
	Limit      int            `json:"limit"`
}

type GetAgentTaskResponse struct {
	Task AgentTaskDTO `json:"task"`
}

type UpdateTodoStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Notes  string `json:"notes,omitempty"`
}

type UpdateTodoStatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// REST API Handler - wraps TaskStorage
type RESTAPIHandler struct {
	taskStorage storage.TaskStorage
}

// NewRESTAPIHandler creates a new REST API handler
func NewRESTAPIHandler(taskStorage storage.TaskStorage) *RESTAPIHandler {
	return &RESTAPIHandler{
		taskStorage: taskStorage,
	}
}

// Conversion functions: storage models → DTOs

func convertTaskToDTO(task *storage.HumanTask) TaskDTO {
	return TaskDTO{
		ID:        task.ID,
		Prompt:    task.Prompt,
		CreatedAt: task.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt: task.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		Status:    string(task.Status),
		Notes:     task.Notes,
	}
}

func convertTodoItemToDTO(todo *storage.TodoItem) TodoItemDTO {
	dto := TodoItemDTO{
		ID:           todo.ID,
		Description:  todo.Description,
		Status:       string(todo.Status),
		CreatedAt:    todo.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		Notes:        todo.Notes,
		FilePath:     todo.FilePath,
		FunctionName: todo.FunctionName,
		ContextHint:  todo.ContextHint,
		HumanPromptNotes: todo.HumanPromptNotes,
	}

	if todo.CompletedAt != nil {
		completedStr := todo.CompletedAt.Format("2006-01-02T15:04:05.000Z")
		dto.CompletedAt = &completedStr
	}

	if todo.HumanPromptNotesAddedAt != nil {
		addedStr := todo.HumanPromptNotesAddedAt.Format("2006-01-02T15:04:05.000Z")
		dto.HumanPromptNotesAddedAt = &addedStr
	}

	if todo.HumanPromptNotesUpdatedAt != nil {
		updatedStr := todo.HumanPromptNotesUpdatedAt.Format("2006-01-02T15:04:05.000Z")
		dto.HumanPromptNotesUpdatedAt = &updatedStr
	}

	return dto
}

func convertAgentTaskToDTO(task *storage.AgentTask) AgentTaskDTO {
	todos := make([]TodoItemDTO, len(task.Todos))
	for i, todo := range task.Todos {
		todos[i] = convertTodoItemToDTO(&todo)
	}

	dto := AgentTaskDTO{
		ID:                task.ID,
		HumanTaskID:       task.HumanTaskID,
		AgentName:         task.AgentName,
		Role:              task.Role,
		Todos:             todos,
		CreatedAt:         task.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:         task.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		Status:            string(task.Status),
		Notes:             task.Notes,
		ContextSummary:    task.ContextSummary,
		FilesModified:     task.FilesModified,
		QdrantCollections: task.QdrantCollections,
		PriorWorkSummary:  task.PriorWorkSummary,
		HumanPromptNotes:  task.HumanPromptNotes,
	}

	if task.HumanPromptNotesAddedAt != nil {
		addedStr := task.HumanPromptNotesAddedAt.Format("2006-01-02T15:04:05.000Z")
		dto.HumanPromptNotesAddedAt = &addedStr
	}

	if task.HumanPromptNotesUpdatedAt != nil {
		updatedStr := task.HumanPromptNotesUpdatedAt.Format("2006-01-02T15:04:05.000Z")
		dto.HumanPromptNotesUpdatedAt = &updatedStr
	}

	return dto
}

// REST API Handlers

// CreateHumanTask creates a new human task
// POST /api/tasks
func (h *RESTAPIHandler) CreateHumanTask(c *gin.Context) {
	var req CreateHumanTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	task, err := h.taskStorage.CreateHumanTask(req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, CreateHumanTaskResponse{
		Task: convertTaskToDTO(task),
	})
}

// ListHumanTasks returns all human tasks
// GET /api/tasks
func (h *RESTAPIHandler) ListHumanTasks(c *gin.Context) {
	tasks := h.taskStorage.ListAllHumanTasks()

	dtos := make([]TaskDTO, len(tasks))
	for i, task := range tasks {
		dtos[i] = convertTaskToDTO(task)
	}

	c.JSON(http.StatusOK, ListHumanTasksResponse{
		Tasks: dtos,
		Count: len(dtos),
	})
}

// GetHumanTask returns a single human task by ID
// GET /api/tasks/:id
func (h *RESTAPIHandler) GetHumanTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.taskStorage.GetHumanTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": convertTaskToDTO(task)})
}

// UpdateTaskStatus updates the status of a task (human or agent)
// PUT /api/tasks/:id/status
func (h *RESTAPIHandler) UpdateTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	var req UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err := h.taskStorage.UpdateTaskStatus(taskID, storage.TaskStatus(req.Status), req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, UpdateTaskStatusResponse{
		Success: true,
		Message: fmt.Sprintf("Task status updated to %s", req.Status),
	})
}

// CreateAgentTask creates a new agent task
// POST /api/agent-tasks
func (h *RESTAPIHandler) CreateAgentTask(c *gin.Context) {
	var req CreateAgentTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	task, err := h.taskStorage.CreateAgentTask(
		req.HumanTaskID,
		req.AgentName,
		req.Role,
		req.Todos,
		req.ContextSummary,
		req.FilesModified,
		req.QdrantCollections,
		req.PriorWorkSummary,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent task: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, CreateAgentTaskResponse{
		Task: convertAgentTaskToDTO(task),
	})
}

// ListAgentTasks returns agent tasks with optional filters
// GET /api/agent-tasks?humanTaskId=...&agentName=...&offset=0&limit=50
func (h *RESTAPIHandler) ListAgentTasks(c *gin.Context) {
	query := ListAgentTasksQuery{
		HumanTaskID: c.Query("humanTaskId"),
		AgentName:   c.Query("agentName"),
		Offset:      0,
		Limit:       50,
	}

	if offset := c.Query("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil && val >= 0 {
			query.Offset = val
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 {
			query.Limit = val
			if query.Limit > 50 {
				query.Limit = 50 // Enforce max
			}
		}
	}

	allTasks := h.taskStorage.ListAllAgentTasks()

	// Apply filters
	var filteredTasks []*storage.AgentTask
	for _, task := range allTasks {
		if query.HumanTaskID != "" && task.HumanTaskID != query.HumanTaskID {
			continue
		}
		if query.AgentName != "" && task.AgentName != query.AgentName {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	totalCount := len(filteredTasks)

	// Apply pagination
	endIndex := query.Offset + query.Limit
	if query.Offset > totalCount {
		query.Offset = totalCount
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	paginatedTasks := filteredTasks[query.Offset:endIndex]

	// Convert to DTOs
	dtos := make([]AgentTaskDTO, len(paginatedTasks))
	for i, task := range paginatedTasks {
		dtos[i] = convertAgentTaskToDTO(task)
	}

	c.JSON(http.StatusOK, ListAgentTasksResponse{
		Tasks:      dtos,
		Count:      len(dtos),
		TotalCount: totalCount,
		Offset:     query.Offset,
		Limit:      query.Limit,
	})
}

// GetAgentTask returns a single agent task by ID
// GET /api/agent-tasks/:id
func (h *RESTAPIHandler) GetAgentTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.taskStorage.GetAgentTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent task not found"})
		return
	}

	c.JSON(http.StatusOK, GetAgentTaskResponse{
		Task: convertAgentTaskToDTO(task),
	})
}

// UpdateTodoStatus updates the status of a TODO item
// PUT /api/agent-tasks/:agentTaskId/todos/:todoId/status
func (h *RESTAPIHandler) UpdateTodoStatus(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	todoID := c.Param("todoId")

	var req UpdateTodoStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err := h.taskStorage.UpdateTodoStatus(agentTaskID, todoID, storage.TodoStatus(req.Status), req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update TODO status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, UpdateTodoStatusResponse{
		Success: true,
		Message: fmt.Sprintf("TODO status updated to %s", req.Status),
	})
}

// RegisterRESTRoutes registers all REST API routes
func (h *RESTAPIHandler) RegisterRESTRoutes(r *gin.Engine) {
	// Human Tasks
	tasks := r.Group("/api/tasks")
	{
		tasks.GET("", h.ListHumanTasks)           // List all human tasks
		tasks.POST("", h.CreateHumanTask)         // Create human task
		tasks.GET("/:id", h.GetHumanTask)         // Get single human task
		tasks.PUT("/:id/status", h.UpdateTaskStatus) // Update task status
	}

	// Agent Tasks
	agentTasks := r.Group("/api/agent-tasks")
	{
		agentTasks.GET("", h.ListAgentTasks)                                  // List agent tasks with filters
		agentTasks.POST("", h.CreateAgentTask)                                 // Create agent task
		agentTasks.GET("/:id", h.GetAgentTask)                                 // Get single agent task
		agentTasks.PUT("/:agentTaskId/todos/:todoId/status", h.UpdateTodoStatus) // Update TODO status
	}
}
