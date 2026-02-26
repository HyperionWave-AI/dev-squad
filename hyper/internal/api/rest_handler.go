package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/scanner"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/watcher"
	"hyper/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Agent Communication DTOs
type AgentCommunicationRequest struct {
	AgentType         string                 `json:"agentType"`
	CommunicationType string                 `json:"communicationType"`
	Message           string                 `json:"message,omitempty"`
	TaskID            string                 `json:"taskId,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
}

type AgentCommunicationResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	AgentType string                 `json:"agentType"`
	Timestamp string                 `json:"timestamp"`
}

// Valid agent types
var validAgentTypes = map[string]bool{
	"ui-dev":       true,
	"go-dev":       true,
	"sre":          true,
	"coordinator":  true,
	"data-analyst": true,
	"qa":           true,
}

// Valid communication types
var validCommunicationTypes = map[string]bool{
	"execute":        true,
	"status":         true,
	"direct_message": true,
}

// Maximum message length
const maxMessageLength = 10000

// isFileWatcherDisabledByEnv returns true when watcher runtime is intentionally disabled.
func isFileWatcherDisabledByEnv() bool {
	raw := strings.TrimSpace(os.Getenv("ENABLE_FILE_WATCHER"))
	raw = strings.Trim(raw, "\"'")
	return strings.EqualFold(raw, "false")
}

// REST API Data Transfer Objects (DTOs)
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
	HumanTaskID       string                  `json:"humanTaskId" binding:"required"`
	AgentName         string                  `json:"agentName" binding:"required"`
	Role              string                  `json:"role" binding:"required"`
	Todos             []storage.TodoItemInput `json:"todos" binding:"required"`
	ContextSummary    string                  `json:"contextSummary,omitempty"`
	FilesModified     []string                `json:"filesModified,omitempty"`
	QdrantCollections []string                `json:"qdrantCollections,omitempty"`
	PriorWorkSummary  string                  `json:"priorWorkSummary,omitempty"`
}

type CreateAgentTaskResponse struct {
	Task AgentTaskDTO `json:"task"`
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

// Knowledge DTOs
type KnowledgeCollectionDTO struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Count       int      `json:"count"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type PopularCollectionDTO struct {
	Collection string `json:"collection"`
	Count      int    `json:"count"`
}

type KnowledgeEntryDTO struct {
	ID         string                 `json:"id"`
	Collection string                 `json:"collection"`
	Text       string                 `json:"text"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"createdAt"`
	Score      *float64               `json:"score,omitempty"`
}

type ListCollectionsResponse struct {
	Collections []KnowledgeCollectionDTO `json:"collections"`
}

type PopularCollectionsResponse struct {
	Collections []PopularCollectionDTO `json:"collections"`
}

type BrowseKnowledgeResponse struct {
	Entries []KnowledgeEntryDTO `json:"entries"`
	Count   int                 `json:"count"`
	Limit   int                 `json:"limit"`
}

type QueryKnowledgeRequest struct {
	Collection string `json:"collection" binding:"required"`
	Query      string `json:"query" binding:"required"`
	Limit      int    `json:"limit"`
}

type QueryKnowledgeResponse struct {
	Entries []KnowledgeEntryDTO `json:"entries"`
}

type RebuildCollectionCountsResponse struct {
	Success            bool                     `json:"success"`
	CollectionsUpdated int                      `json:"collectionsUpdated"`
	TotalEntries       int                      `json:"totalEntries"`
	Details            []map[string]interface{} `json:"details"`
	Errors             []string                 `json:"errors,omitempty"`
}

// Code Index DTOs
type AddFolderRequest struct {
	FolderPath      string   `json:"folderPath" binding:"required"`
	Description     string   `json:"description,omitempty"`
	IncludePatterns []string `json:"includePatterns,omitempty"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
	ChunkSize       string   `json:"chunkSize,omitempty"` // T-shirt size: s|m|l|xl
}

type FileDetailsDTO struct {
	ID           string `json:"id"`
	FolderPath   string `json:"folderPath"`
	RelativePath string `json:"relativePath"`
	Language     string `json:"language"`
	Size         int64  `json:"size"`
	LineCount    int    `json:"lineCount"`
	ChunkCount   int    `json:"chunkCount"`
	IndexedAt    string `json:"indexedAt"`
}

type FileChunkDetailsDTO struct {
	ChunkNum  int    `json:"chunkNum"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Content   string `json:"content"`
	ChunkType string `json:"chunkType"` // "ast" or "line-based"
	NodeType  string `json:"nodeType,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type ListFilesResponse struct {
	Files []FileDetailsDTO `json:"files"`
	Count int              `json:"count"`
}

type GetFileResponse struct {
	File FileDetailsDTO `json:"file"`
}

type GetFileChunksResponse struct {
	Chunks []FileChunkDetailsDTO `json:"chunks"`
	Count  int                   `json:"count"`
}

type AddFolderResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Folder  *storage.IndexedFolder `json:"folder"`
}

type RemoveFolderResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	FilesRemoved int    `json:"filesRemoved,omitempty"`
}

type ScanResponse struct {
	Success      bool `json:"success"`
	FilesIndexed int  `json:"filesIndexed"`
	FilesUpdated int  `json:"filesUpdated"`
	FilesSkipped int  `json:"filesSkipped"`
	TotalFiles   int  `json:"totalFiles"`
}

type SearchRequest struct {
	Query      string   `json:"query" binding:"required"`
	FileTypes  []string `json:"fileTypes,omitempty"`
	MinScore   float32  `json:"minScore,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	FolderPath string   `json:"folderPath,omitempty"`
	Retrieve   string   `json:"retrieve,omitempty"` // "chunk" or "full"
}

type SearchResultDTO struct {
	FileID            string  `json:"fileId"`
	FilePath          string  `json:"filePath"`
	RelativePath      string  `json:"relativePath"`
	Language          string  `json:"language"`
	ChunkNum          int     `json:"chunkNum,omitempty"`
	StartLine         int     `json:"startLine,omitempty"`
	EndLine           int     `json:"endLine,omitempty"`
	Content           string  `json:"content"`
	Score             float32 `json:"score"`
	FolderID          string  `json:"folderId"`
	FolderPath        string  `json:"folderPath"`
	FullFileRetrieved bool    `json:"fullFileRetrieved"`
	ChunkSize         string  `json:"chunkSize,omitempty"` // T-shirt size: s, m, l, xl
	// AST metadata
	ChunkType string `json:"chunkType,omitempty"`
	NodeType  string `json:"nodeType,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
}

type SearchResponse struct {
	Success      bool              `json:"success"`
	Query        string            `json:"query"`
	RetrieveMode string            `json:"retrieveMode"`
	Results      []SearchResultDTO `json:"results"`
	Count        int               `json:"count"`
}

type FolderDTO struct {
	ConfigId   string `json:"configId"`
	FolderPath string `json:"folderPath"`
	FileCount  int    `json:"fileCount"`
	Enabled    bool   `json:"enabled"`
}

type IndexStatusResponse struct {
	TotalFolders  int         `json:"totalFolders"`
	TotalFiles    int         `json:"totalFiles"`
	TotalSize     int64       `json:"totalSize"`
	LastScanTime  time.Time   `json:"lastScanTime,omitempty"`
	WatcherStatus string      `json:"watcherStatus"` // "running" or "stopped"
	Folders       []FolderDTO `json:"folders"`
}

type ClearAllIndexResponse struct {
	Success                  bool     `json:"success"`
	Message                  string   `json:"message"`
	FoldersRemoved           int      `json:"foldersRemoved"`
	FilesRemoved             int      `json:"filesRemoved"`
	ChunksRemoved            int      `json:"chunksRemoved"`
	QdrantCollectionsRemoved int      `json:"qdrantCollectionsRemoved"`
	Errors                   []string `json:"errors,omitempty"`
}

// RESTAPIHandler wraps TaskStorage for HTTP REST API
type RESTAPIHandler struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
	codeIndexStorage *storage.CodeIndexStorage
	qdrantClient     *storage.QdrantClient
	embeddingClient  embeddings.EmbeddingClient
	fileScanner      *scanner.FileScanner
	fileWatcher      *watcher.FileWatcher
	logger           *zap.Logger
}

// NewRESTAPIHandler creates a new REST API handler
func NewRESTAPIHandler(
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	fileWatcher *watcher.FileWatcher,
	logger *zap.Logger,
) *RESTAPIHandler {
	return &RESTAPIHandler{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
		codeIndexStorage: codeIndexStorage,
		qdrantClient:     qdrantClient,
		embeddingClient:  embeddingClient,
		fileScanner:      scanner.NewFileScanner(),
		fileWatcher:      fileWatcher,
		logger:           logger,
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
		ID:               todo.ID,
		Description:      todo.Description,
		Status:           string(todo.Status),
		CreatedAt:        todo.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		Notes:            todo.Notes,
		FilePath:         todo.FilePath,
		FunctionName:     todo.FunctionName,
		ContextHint:      todo.ContextHint,
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

// REST API Handlers - Direct TaskStorage access (NO MCP proxying)

// CreateHumanTask creates a new human task
// POST /api/v1/tasks
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

// ListHumanTasks returns all human tasks with optional status filtering
// GET /api/v1/tasks?status=...
func (h *RESTAPIHandler) ListHumanTasks(c *gin.Context) {
	statusFilter := c.Query("status")

	// Build MongoDB filter
	filter := make(map[string]interface{})
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	// Use storage method with MongoDB-level filtering (sorting by createdAt descending is built-in)
	tasks, err := h.taskStorage.ListHumanTasks(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks: " + err.Error()})
		return
	}

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
// GET /api/v1/tasks/:id
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
// PUT /api/v1/tasks/:id/status
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
// POST /api/v1/agent-tasks
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

// ListAgentTasks returns agent tasks with optional filters and pagination
// GET /api/v1/agent-tasks?humanTaskId=...&agentName=...&status=...&offset=0&limit=50
func (h *RESTAPIHandler) ListAgentTasks(c *gin.Context) {
	humanTaskID := c.Query("humanTaskId")
	agentName := c.Query("agentName")
	statusFilter := c.Query("status")
	offset := 0
	limit := 50

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 50 {
				limit = 50 // Enforce max
			}
		}
	}

	// Build MongoDB filter from query parameters
	filter := make(map[string]interface{})
	if humanTaskID != "" {
		filter["humanTaskId"] = humanTaskID
	}
	if agentName != "" {
		filter["agentName"] = agentName
	}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	// Use storage method with MongoDB-level filtering, pagination, and sorting (descending by createdAt)
	paginatedTasks, totalCount, err := h.taskStorage.ListAgentTasks(filter, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agent tasks: " + err.Error()})
		return
	}

	// Convert to DTOs
	dtos := make([]AgentTaskDTO, len(paginatedTasks))
	for i, task := range paginatedTasks {
		dtos[i] = convertAgentTaskToDTO(task)
	}

	c.JSON(http.StatusOK, ListAgentTasksResponse{
		Tasks:      dtos,
		Count:      len(dtos),
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	})
}

// GetAgentTask returns a single agent task by ID
// GET /api/v1/agent-tasks/:id
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
// PUT /api/v1/agent-tasks/:agentTaskId/todos/:todoId/status
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

// Knowledge Handlers

// ListCollections returns all knowledge collections with metadata
// GET /api/v1/knowledge/collections
func (h *RESTAPIHandler) ListCollections(c *gin.Context) {
	collections, err := h.knowledgeStorage.GetCollectionStatsWithMetadata()
	if err != nil {
		h.logger.Error("Failed to get collection stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve collections: " + err.Error()})
		return
	}

	// Convert to DTOs
	dtos := make([]KnowledgeCollectionDTO, len(collections))
	for i, col := range collections {
		dtos[i] = KnowledgeCollectionDTO{
			Name:        col.Name,
			Category:    col.Category,
			Count:       col.Count,
			Description: col.Description,
			Tags:        col.Tags,
		}
	}

	c.JSON(http.StatusOK, ListCollectionsResponse{
		Collections: dtos,
	})
}

// RebuildCollectionCounts recalculates entry counts for all collections
// POST /api/v1/knowledge/collections/rebuild-counts
func (h *RESTAPIHandler) RebuildCollectionCounts(c *gin.Context) {
	h.logger.Info("Rebuilding collection counts")

	// Call the storage method to rebuild counts
	stats, err := h.knowledgeStorage.(*storage.MongoKnowledgeStorage).RebuildCollectionCounts()
	if err != nil {
		h.logger.Error("Failed to rebuild collection counts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rebuild collection counts: " + err.Error()})
		return
	}

	// Extract stats
	collectionsUpdated := stats["collectionsUpdated"].(int)
	totalEntries := stats["totalEntries"].(int)
	details := stats["details"].([]map[string]interface{})
	errors := []string{}
	if errList, ok := stats["errors"].([]string); ok && len(errList) > 0 {
		errors = errList
	}

	h.logger.Info("Rebuild collection counts completed",
		zap.Int("collectionsUpdated", collectionsUpdated),
		zap.Int("totalEntries", totalEntries))

	c.JSON(http.StatusOK, RebuildCollectionCountsResponse{
		Success:            true,
		CollectionsUpdated: collectionsUpdated,
		TotalEntries:       totalEntries,
		Details:            details,
		Errors:             errors,
	})
}

// GetPopularCollections returns popular collections in frontend-compatible format
// GET /api/v1/knowledge/popular-collections?limit=20
func (h *RESTAPIHandler) GetPopularCollections(c *gin.Context) {
	// Parse limit parameter (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100
			}
		}
	}

	// Get popular collections from storage
	collections, err := h.knowledgeStorage.GetPopularCollections(limit)
	if err != nil {
		h.logger.Error("Failed to get popular collections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve popular collections"})
		return
	}

	// Convert to frontend-compatible DTOs (collection field instead of name)
	dtos := make([]PopularCollectionDTO, len(collections))
	for i, col := range collections {
		dtos[i] = PopularCollectionDTO{
			Collection: col.Collection,
			Count:      col.Count,
		}
	}

	c.JSON(http.StatusOK, PopularCollectionsResponse{
		Collections: dtos,
	})
}

// BrowseKnowledge retrieves knowledge entries without search (browse mode)
// GET /api/v1/knowledge/browse?collection=...&limit=10
func (h *RESTAPIHandler) BrowseKnowledge(c *gin.Context) {
	collection := c.Query("collection")
	limit := 10 // Default

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	// If no collection specified, browse across all major collections
	var collectionsToQuery []string
	if collection == "" || collection == "All Collections" {
		collectionsToQuery = []string{
			"technical-knowledge",
			"adr",
			"data-contracts",
			"team-coordination",
			"workflow-context",
		}
	} else {
		collectionsToQuery = []string{collection}
	}

	var allEntries []KnowledgeEntryDTO

	// List entries from each collection
	for _, col := range collectionsToQuery {
		entries, err := h.knowledgeStorage.ListKnowledge(col, limit)
		if err != nil {
			h.logger.Warn("Failed to list knowledge from collection",
				zap.String("collection", col),
				zap.Error(err))
			continue
		}

		// Convert to DTOs
		for _, entry := range entries {
			dto := KnowledgeEntryDTO{
				ID:         entry.ID,
				Collection: entry.Collection,
				Text:       entry.Text,
				Metadata:   entry.Metadata,
				CreatedAt:  entry.CreatedAt.Format(time.RFC3339),
			}
			allEntries = append(allEntries, dto)
		}
	}

	// Limit total results if browsing multiple collections
	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	h.logger.Info("Browse knowledge completed",
		zap.String("collection", collection),
		zap.Int("limit", limit),
		zap.Int("results", len(allEntries)))

	c.JSON(http.StatusOK, BrowseKnowledgeResponse{
		Entries: allEntries,
		Count:   len(allEntries),
		Limit:   limit,
	})
}

// QueryKnowledge searches the knowledge base with semantic search
// POST /api/v1/knowledge/query
func (h *RESTAPIHandler) QueryKnowledge(c *gin.Context) {
	var req QueryKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	// Query knowledge storage (no taskId filtering in this handler)
	results, err := h.knowledgeStorage.Query(req.Collection, req.Query, limit, nil)
	if err != nil {
		h.logger.Error("Failed to query knowledge",
			zap.String("collection", req.Collection),
			zap.String("query", req.Query),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query knowledge base"})
		return
	}

	// Transform QueryResult to DTOs
	entries := make([]KnowledgeEntryDTO, 0, len(results))
	for _, result := range results {
		score := float64(result.Score)
		dto := KnowledgeEntryDTO{
			ID:         result.Entry.ID,
			Collection: req.Collection,
			Text:       result.Entry.Text,
			Metadata:   result.Entry.Metadata,
			CreatedAt:  result.Entry.CreatedAt.Format(time.RFC3339),
			Score:      &score,
		}
		entries = append(entries, dto)
	}

	h.logger.Info("Query knowledge completed",
		zap.String("collection", req.Collection),
		zap.String("query", req.Query),
		zap.Int("limit", limit),
		zap.Int("results", len(entries)))

	c.JSON(http.StatusOK, QueryKnowledgeResponse{
		Entries: entries,
	})
}

// Code Index Handlers

// AddFolder adds a folder to the code index
// POST /api/v1/code-index/add-folder
func (h *RESTAPIHandler) AddFolder(c *gin.Context) {
	var req AddFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path: " + err.Error()})
		return
	}

	// Validate and set default chunk size
	chunkSize := req.ChunkSize
	if chunkSize == "" {
		chunkSize = "m" // Default to medium
	} else {
		// Validate chunk size is one of: s, m, l, xl
		if chunkSize != "s" && chunkSize != "m" && chunkSize != "l" && chunkSize != "xl" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chunk size. Must be one of: s, m, l, xl"})
			return
		}
	}

	// Check if folder already exists
	existing, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing folder: " + err.Error()})
		return
	}
	if existing != nil {
		message := "Folder already indexed."
		if isFileWatcherDisabledByEnv() {
			message = "Folder already indexed. File watcher is disabled by server configuration."
		} else if h.fileWatcher != nil && h.fileWatcher.IsFolderWatched(existing.Path) {
			message = "Folder already indexed. File watcher is monitoring changes."
		}

		c.JSON(http.StatusOK, AddFolderResponse{
			Success: true,
			Message: message,
			Folder:  existing,
		})
		return
	}

	// Add folder to storage with configuration
	folder, err := h.codeIndexStorage.AddFolderWithConfig(absPath, req.Description, req.IncludePatterns, req.ExcludePatterns, chunkSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add folder: " + err.Error()})
		return
	}

	// Add folder to file watcher only if watcher runtime is active.
	// If watcher runtime is stopped, keep folder indexed but mark watcher inactive for clarity in UI.
	watcherEnabled := false
	if h.fileWatcher != nil && h.fileWatcher.IsRunning() {
		if err := h.fileWatcher.AddFolder(folder); err != nil {
			h.logger.Warn("Failed to add folder to file watcher", zap.Error(err))
		} else {
			watcherEnabled = true
			h.logger.Info("Added folder to file watcher", zap.String("path", absPath))
		}
	}

	if !watcherEnabled {
		if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "inactive", ""); err != nil {
			h.logger.Warn("Failed to mark folder watcher as inactive",
				zap.String("folderID", folder.ID),
				zap.Error(err))
		}
	}

	h.logger.Info("Added folder to code index",
		zap.String("folderID", folder.ID),
		zap.String("path", absPath),
		zap.String("chunkSize", chunkSize),
		zap.Strings("includePatterns", req.IncludePatterns),
		zap.Strings("excludePatterns", req.ExcludePatterns))

	message := "Folder added successfully. Use /api/code-index/scan to index existing files."
	if watcherEnabled {
		message = "Folder added successfully. File watcher is now monitoring changes. Use /api/code-index/scan to index existing files."
	} else if isFileWatcherDisabledByEnv() {
		message = "Folder added successfully. File watcher is disabled by server configuration. Use /api/code-index/scan to index existing files."
	} else if h.fileWatcher != nil {
		message = "Folder added successfully. File watcher is currently stopped. Enable watcher to monitor changes. Use /api/code-index/scan to index existing files."
	}

	c.JSON(http.StatusCreated, AddFolderResponse{
		Success: true,
		Message: message,
		Folder:  folder,
	})
}

// RemoveFolder removes a folder from the code index
// DELETE /api/v1/code-index/remove-folder/:configId
func (h *RESTAPIHandler) RemoveFolder(c *gin.Context) {
	configID := c.Param("configId")

	// Get folder
	folder, err := h.codeIndexStorage.GetFolder(configID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found: " + configID})
		return
	}

	// Get all files to delete their vectors
	files, err := h.codeIndexStorage.ListFiles(folder.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files: " + err.Error()})
		return
	}

	// Get path mapping before deletions
	mapping, _ := h.codeIndexStorage.GetPathMapping(folder.Path)

	// Delete vectors from Qdrant - lookup collection from path mapping
	if len(files) > 0 && mapping != nil {
		err = h.qdrantClient.DeleteCodeIndexByFilter(mapping.QdrantCollection, map[string]interface{}{
			"must": []map[string]interface{}{
				{"key": "folderId", "match": map[string]interface{}{"value": folder.ID}},
			},
		})
		if err != nil {
			h.logger.Warn("Failed to delete vectors from Qdrant", zap.Error(err))
		}
	}

	// Delete the Qdrant collection
	if mapping != nil {
		if err := h.qdrantClient.DeleteCollection(mapping.QdrantCollection); err != nil {
			h.logger.Warn("Failed to delete Qdrant collection",
				zap.String("collection", mapping.QdrantCollection),
				zap.Error(err))
		} else {
			h.logger.Info("Deleted Qdrant collection",
				zap.String("collection", mapping.QdrantCollection))
		}
	}

	// Delete path mapping from MongoDB
	if err := h.codeIndexStorage.RemovePathMapping(folder.Path); err != nil {
		h.logger.Warn("Failed to remove path mapping",
			zap.String("path", folder.Path),
			zap.Error(err))
	} else {
		h.logger.Info("Removed path mapping",
			zap.String("path", folder.Path))
	}

	// Remove folder from file watcher
	if h.fileWatcher != nil {
		if err := h.fileWatcher.RemoveFolder(folder.Path); err != nil {
			h.logger.Warn("Failed to remove folder from file watcher", zap.Error(err))
		} else {
			h.logger.Info("Removed folder from file watcher", zap.String("path", folder.Path))
		}
	}

	// Remove folder from MongoDB (cascades to files and chunks)
	if err := h.codeIndexStorage.RemoveFolder(folder.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove folder: " + err.Error()})
		return
	}

	h.logger.Info("Removed folder from code index",
		zap.String("folderID", folder.ID),
		zap.String("path", folder.Path),
		zap.Int("filesRemoved", len(files)))

	c.JSON(http.StatusOK, RemoveFolderResponse{
		Success:      true,
		Message:      "Folder removed successfully",
		FilesRemoved: len(files),
	})
}

// ScanFolder triggers a scan of a folder
// POST /api/v1/code-index/scan
func (h *RESTAPIHandler) ScanFolder(c *gin.Context) {
	var req AddFolderRequest // Reuse same structure (only folderPath needed)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path: " + err.Error()})
		return
	}

	// Get folder
	folder, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found. Use /api/code-index/add-folder first: " + absPath})
		return
	}

	// Preserve watcher preference across scans.
	// A manual scan should not force a previously inactive watcher back to active.
	restoreStatus := "active"
	if folder.Status == "inactive" {
		restoreStatus = "inactive"
	}

	// Update folder status to scanning
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "scanning", ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder status: " + err.Error()})
		return
	}

	// Scan directory for files
	scannedFiles, err := h.fileScanner.ScanDirectory(absPath)
	if err != nil {
		h.codeIndexStorage.UpdateFolderStatus(folder.ID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan directory: " + err.Error()})
		return
	}

	filesIndexed := 0
	filesUpdated := 0
	filesSkipped := 0

	// Process each file
	for _, scannedFile := range scannedFiles {
		scannedFile.FolderID = folder.ID

		// Check if file already exists
		existingFile, _ := h.codeIndexStorage.GetFileByPath(scannedFile.Path)

		if existingFile != nil {
			// Check if file has changed
			if existingFile.SHA256 == scannedFile.SHA256 {
				filesSkipped++
				continue
			}
			filesUpdated++
			scannedFile.ID = existingFile.ID
		} else {
			filesIndexed++
			scannedFile.ID = uuid.New().String()
		}

		// Create chunks
		chunks, err := h.fileScanner.CreateFileChunks(scannedFile.ID, scannedFile.Path)
		if err != nil {
			h.logger.Warn("Failed to create chunks", zap.String("file", scannedFile.Path), zap.Error(err))
			continue
		}

		// Generate embeddings for chunks
		var qdrantPoints []storage.CodeIndexPoint
		for _, chunk := range chunks {
			// Generate embedding
			embedding, err := h.embeddingClient.CreateEmbedding(chunk.Content)
			if err != nil {
				h.logger.Warn("Failed to create embedding",
					zap.String("file", scannedFile.Path),
					zap.Int("chunk", chunk.ChunkNum),
					zap.Error(err))
				continue
			}

			// Create Qdrant point with deterministic UUID (not concatenated string)
			// Generate a deterministic UUID by hashing fileID + chunkNum
			pointID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s_chunk_%d", scannedFile.ID, chunk.ChunkNum))).String()
			chunk.VectorID = pointID

			point := storage.CodeIndexPoint{
				ID:     pointID,
				Vector: embedding,
				Payload: map[string]interface{}{
					"fileId":       scannedFile.ID,
					"folderId":     folder.ID,
					"folderPath":   folder.Path,
					"filePath":     scannedFile.Path,
					"relativePath": scannedFile.RelativePath,
					"language":     scannedFile.Language,
					"chunkNum":     chunk.ChunkNum,
					"startLine":    chunk.StartLine,
					"endLine":      chunk.EndLine,
					"content":      chunk.Content,
				},
			}
			qdrantPoints = append(qdrantPoints, point)

			// Save chunk to MongoDB
			if err := h.codeIndexStorage.UpsertChunk(chunk); err != nil {
				h.logger.Warn("Failed to save chunk", zap.Error(err))
			}
		}

		// Upload vectors to Qdrant - lookup collection from path mapping
		if len(qdrantPoints) > 0 {
			mapping, _ := h.codeIndexStorage.GetPathMapping(folder.Path)
			collectionName := storage.CodeIndexCollection // fallback to default
			if mapping != nil {
				collectionName = mapping.QdrantCollection
			}
			if err := h.qdrantClient.UpsertCodeIndexPoints(collectionName, qdrantPoints); err != nil {
				h.logger.Warn("Failed to upsert vectors", zap.String("file", scannedFile.Path), zap.Error(err))
			}
		}

		// Save file metadata to MongoDB
		if err := h.codeIndexStorage.UpsertFile(scannedFile); err != nil {
			h.logger.Warn("Failed to save file", zap.Error(err))
		}
	}

	// Update folder status and scan time
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, restoreStatus, ""); err != nil {
		h.logger.Warn("Failed to update folder status", zap.Error(err))
	}

	if err := h.codeIndexStorage.UpdateFolderScanTime(folder.ID, len(scannedFiles)); err != nil {
		h.logger.Warn("Failed to update scan time", zap.Error(err))
	}

	h.logger.Info("Completed folder scan",
		zap.String("folderID", folder.ID),
		zap.Int("filesIndexed", filesIndexed),
		zap.Int("filesUpdated", filesUpdated),
		zap.Int("filesSkipped", filesSkipped))

	c.JSON(http.StatusOK, ScanResponse{
		Success:      true,
		FilesIndexed: filesIndexed,
		FilesUpdated: filesUpdated,
		FilesSkipped: filesSkipped,
		TotalFiles:   len(scannedFiles),
	})
}

// fileExtensionToLanguage maps file extensions to language names
// This matches the language metadata stored during indexing
func fileExtensionToLanguage(extension string) string {
	extensionMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".py":    "python",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".rs":    "rust",
		".swift": "swift",
		".kt":    "kotlin",
		".m":     "objective-c",
		".scala": "scala",
		".r":     "r",
		".sql":   "sql",
		".sh":    "shell",
		".bash":  "shell",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".less":  "less",
		".vue":   "vue",
		".md":    "markdown",
	}

	// Normalize extension to lowercase with leading dot
	normalizedExt := strings.ToLower(extension)
	if !strings.HasPrefix(normalizedExt, ".") {
		normalizedExt = "." + normalizedExt
	}

	if lang, ok := extensionMap[normalizedExt]; ok {
		return lang
	}
	return ""
}

// buildFileTypeFilter creates a Qdrant filter for file types
// Returns nil if no file types specified
func buildFileTypeFilter(fileTypes []string) map[string]interface{} {
	if len(fileTypes) == 0 {
		return nil
	}

	// Convert file extensions to language names
	var languages []string
	for _, ext := range fileTypes {
		if lang := fileExtensionToLanguage(ext); lang != "" {
			languages = append(languages, lang)
		}
	}

	if len(languages) == 0 {
		return nil
	}

	// Build Qdrant filter with "should" clause for OR logic
	// Match any of the selected languages
	shouldClauses := make([]map[string]interface{}, 0, len(languages))
	for _, lang := range languages {
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"key": "language",
			"match": map[string]interface{}{
				"value": lang,
			},
		})
	}

	return map[string]interface{}{
		"should": shouldClauses,
	}
}

// SearchCode searches the code index
// POST /api/v1/code-index/search
func (h *RESTAPIHandler) SearchCode(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	retrieveModeParam := req.Retrieve
	if retrieveModeParam == "" {
		retrieveModeParam = "chunk"
	}
	if !utils.IsValidRetrieveMode(retrieveModeParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "retrieve must be one of: 'chunk', 'chunk-s', 'chunk-m', 'chunk-l', 'chunk-xl', or 'full'"})
		return
	}

	// Parse retrieve mode to get chunk size parameters
	retrieveType, chunkLines, tshirtSize := utils.ParseRetrieveMode(retrieveModeParam)

	// Limit full file retrieval to top 1 result to conserve bandwidth
	if retrieveType == "full" {
		limit = 1
	}

	// Generate embedding for query
	queryEmbedding, err := h.embeddingClient.CreateEmbedding(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create query embedding: " + err.Error()})
		return
	}

	// Determine which collection to search - if folderPath provided, use its mapping
	collectionName := storage.CodeIndexCollection // fallback to default
	if req.FolderPath != "" {
		mapping, _ := h.codeIndexStorage.GetPathMapping(req.FolderPath)
		if mapping != nil {
			collectionName = mapping.QdrantCollection
		}
	}

	// Build file type filter if fileTypes parameter is provided
	filter := buildFileTypeFilter(req.FileTypes)

	// Search in Qdrant with optional filter
	var searchResp *storage.CodeIndexSearchResponse
	if filter != nil {
		searchResp, err = h.qdrantClient.SearchCodeIndexWithFilter(collectionName, queryEmbedding, limit, filter)
	} else {
		searchResp, err = h.qdrantClient.SearchCodeIndex(collectionName, queryEmbedding, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search: " + err.Error()})
		return
	}

	// Build results
	var results []SearchResultDTO
	for _, hit := range searchResp.Result {
		result := SearchResultDTO{
			Score: hit.Score,
		}

		if fileID, ok := hit.Payload["fileId"].(string); ok {
			result.FileID = fileID
		}
		if folderID, ok := hit.Payload["folderId"].(string); ok {
			result.FolderID = folderID
		}
		if folderPath, ok := hit.Payload["folderPath"].(string); ok {
			result.FolderPath = folderPath
		}
		if filePath, ok := hit.Payload["filePath"].(string); ok {
			result.FilePath = filePath
		}

		// FILTER: Skip archived/deprecated files and tmp directories
		if strings.Contains(result.FilePath, "/.archived/") || strings.Contains(result.FilePath, "/.archive/") || strings.Contains(result.FilePath, "/tmp/") {
			continue
		}

		// FILTER: Skip files not matching folderPath filter
		if req.FolderPath != "" && !strings.HasPrefix(result.FilePath, req.FolderPath) {
			continue
		}

		if relativePath, ok := hit.Payload["relativePath"].(string); ok {
			result.RelativePath = relativePath
		}
		if language, ok := hit.Payload["language"].(string); ok {
			result.Language = language
		}
		if chunkNum, ok := hit.Payload["chunkNum"].(float64); ok {
			result.ChunkNum = int(chunkNum)
		}
		if startLine, ok := hit.Payload["startLine"].(float64); ok {
			result.StartLine = int(startLine)
		}
		if endLine, ok := hit.Payload["endLine"].(float64); ok {
			result.EndLine = int(endLine)
		}
		// Extract AST metadata
		if chunkType, ok := hit.Payload["chunkType"].(string); ok {
			result.ChunkType = chunkType
		}
		if nodeType, ok := hit.Payload["nodeType"].(string); ok {
			result.NodeType = nodeType
		}
		if nodeName, ok := hit.Payload["nodeName"].(string); ok {
			result.NodeName = nodeName
		}

		// Handle content based on retrieve mode
		if retrieveType == "chunk" {
			// Sized chunk retrieval: fetch N lines around the match
			if result.FileID != "" && chunkLines > 0 {
				// Calculate target line range centered around the match
				matchMidpoint := (result.StartLine + result.EndLine) / 2
				targetStart := matchMidpoint - (chunkLines / 2)
				targetEnd := targetStart + chunkLines - 1
				if targetStart < 1 {
					targetStart = 1
					targetEnd = targetStart + chunkLines - 1
				}

				// Fetch overlapping chunks from storage
				overlappingChunks, err := h.codeIndexStorage.GetChunksByFileIDAndLineRange(result.FileID, targetStart, targetEnd)
				if err != nil {
					h.logger.Warn("Failed to fetch sized chunk",
						zap.String("fileID", result.FileID),
						zap.Int("targetStart", targetStart),
						zap.Int("targetEnd", targetEnd),
						zap.Error(err))
					// Fallback to Qdrant chunk content
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				} else if len(overlappingChunks) > 0 {
					// Concatenate overlapping chunks and extract exact line range
					var fullContent strings.Builder
					minLine := overlappingChunks[0].StartLine
					for _, chunk := range overlappingChunks {
						fullContent.WriteString(chunk.Content)
					}

					// Extract exactly chunkLines from the concatenated content
					allLines := strings.Split(fullContent.String(), "\n")
					startIdx := targetStart - minLine
					endIdx := targetEnd - minLine + 1
					if startIdx < 0 {
						startIdx = 0
					}
					if endIdx > len(allLines) {
						endIdx = len(allLines)
					}

					extractedLines := allLines[startIdx:endIdx]
					result.Content = strings.Join(extractedLines, "\n")
					result.StartLine = targetStart
					result.EndLine = targetStart + len(extractedLines) - 1
					result.ChunkSize = tshirtSize
				} else {
					// Fallback to Qdrant chunk if no chunks found
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				}
			} else {
				// No sizing requested or invalid chunkLines, use Qdrant chunk
				if content, ok := hit.Payload["content"].(string); ok {
					result.Content = content
				}
			}
		} else if retrieveType == "full" {
			// Fetch entire file content from MongoDB
			if result.FileID != "" {
				allChunks, err := h.codeIndexStorage.GetChunksByFileID(result.FileID)
				if err != nil {
					h.logger.Warn("Failed to fetch full file content",
						zap.String("fileID", result.FileID),
						zap.Error(err))
					// Fallback to chunk content
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				} else {
					// Concatenate all chunks to build full file content
					var fullContent strings.Builder
					for _, chunk := range allChunks {
						fullContent.WriteString(chunk.Content)
					}
					result.Content = fullContent.String()
					result.FullFileRetrieved = true
				}
			}
		}

		results = append(results, result)
	}

	// Filter results by minimum score if specified
	if req.MinScore > 0 {
		filteredResults := make([]SearchResultDTO, 0, len(results))
		for _, result := range results {
			if result.Score >= req.MinScore {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}

	h.logger.Info("Code search completed",
		zap.String("query", req.Query),
		zap.String("retrieveMode", retrieveModeParam),
		zap.Int("results", len(results)))

	c.JSON(http.StatusOK, SearchResponse{
		Success:      true,
		Query:        req.Query,
		RetrieveMode: retrieveModeParam,
		Results:      results,
		Count:        len(results),
	})
}

// GetIndexStatus gets the current index status
// GET /api/v1/code-index/status
func (h *RESTAPIHandler) GetIndexStatus(c *gin.Context) {
	// Get index status
	status, err := h.codeIndexStorage.GetIndexStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get index status: " + err.Error()})
		return
	}

	// Get folder details
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list folders: " + err.Error()})
		return
	}

	// Calculate total size from all files
	totalSize := int64(0)
	for _, folder := range folders {
		files, _ := h.codeIndexStorage.ListFiles(folder.ID)
		for _, file := range files {
			totalSize += file.Size
		}
	}

	// Determine watcher status from actual watcher runtime and watched folder count.
	watcherStatus := "stopped"
	if h.fileWatcher != nil && h.fileWatcher.IsRunning() && h.fileWatcher.WatchedFolderCount() > 0 {
		watcherStatus = "running"
	}

	// Transform folders to UI format
	uiFolders := make([]FolderDTO, 0, len(folders))
	for _, folder := range folders {
		enabled := folder.Status == "active"
		if h.fileWatcher != nil {
			enabled = h.fileWatcher.IsFolderWatched(folder.Path)
		}

		uiFolders = append(uiFolders, FolderDTO{
			ConfigId:   folder.ID,
			FolderPath: folder.Path,
			FileCount:  folder.FileCount,
			Enabled:    enabled,
		})
	}

	c.JSON(http.StatusOK, IndexStatusResponse{
		TotalFolders:  status.TotalFolders,
		TotalFiles:    status.TotalFiles,
		TotalSize:     totalSize,
		LastScanTime:  status.LastScanTime,
		WatcherStatus: watcherStatus,
		Folders:       uiFolders,
	})
}

// EnableWatcher enables the file watcher for all indexed folders
// POST /api/v1/code-index/enable-watcher
func (h *RESTAPIHandler) EnableWatcher(c *gin.Context) {
	if h.fileWatcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "File watcher is not available",
		})
		return
	}

	if isFileWatcherDisabledByEnv() {
		c.JSON(http.StatusConflict, gin.H{
			"error": "File watcher is disabled by server configuration (ENABLE_FILE_WATCHER=false)",
		})
		return
	}

	// Ensure watcher runtime (event loop + workers) is active.
	if err := h.fileWatcher.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start file watcher runtime: " + err.Error(),
		})
		return
	}

	// Get all folders
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list folders: " + err.Error(),
		})
		return
	}

	// Start watching all folders
	addedCount := 0
	failedCount := 0
	for _, folder := range folders {
		if err := h.fileWatcher.AddFolder(folder); err != nil {
			h.logger.Warn("Failed to add folder to watcher",
				zap.String("path", folder.Path),
				zap.Error(err))
			failedCount++
			if statusErr := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "inactive", ""); statusErr != nil {
				h.logger.Warn("Failed to mark folder inactive after watcher add failure",
					zap.String("folderID", folder.ID),
					zap.Error(statusErr))
			}
		} else {
			addedCount++
			if statusErr := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "active", ""); statusErr != nil {
				h.logger.Warn("Failed to mark folder active after watcher enable",
					zap.String("folderID", folder.ID),
					zap.Error(statusErr))
			}
		}
	}

	h.logger.Info("Enabled file watcher", zap.Int("foldersAdded", addedCount))

	message := fmt.Sprintf("File watcher enabled for %d folders", addedCount)
	if failedCount > 0 {
		message = fmt.Sprintf("%s (%d failed)", message, failedCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        message,
		"foldersWatched": addedCount,
		"failed":         failedCount,
	})
}

// DisableWatcher disables the file watcher
// POST /api/v1/code-index/disable-watcher
func (h *RESTAPIHandler) DisableWatcher(c *gin.Context) {
	if h.fileWatcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "File watcher is not available",
		})
		return
	}

	// Get all folders
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list folders: " + err.Error(),
		})
		return
	}

	// Stop watching all folders
	removedCount := 0
	for _, folder := range folders {
		if err := h.fileWatcher.RemoveFolder(folder.Path); err != nil {
			h.logger.Warn("Failed to remove folder from watcher",
				zap.String("path", folder.Path),
				zap.Error(err))
		} else {
			removedCount++
		}

		if statusErr := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "inactive", ""); statusErr != nil {
			h.logger.Warn("Failed to mark folder inactive after watcher disable",
				zap.String("folderID", folder.ID),
				zap.Error(statusErr))
		}
	}

	h.logger.Info("Disabled file watcher", zap.Int("foldersRemoved", removedCount))

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        fmt.Sprintf("File watcher disabled, stopped watching %d folders", removedCount),
		"foldersStopped": removedCount,
	})
}

// ReindexAll reindexes all files in all indexed folders
// POST /api/v1/code-index/reindex-all
func (h *RESTAPIHandler) ReindexAll(c *gin.Context) {
	// Get all folders
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list folders: " + err.Error(),
		})
		return
	}

	if len(folders) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"message":           "No folders to reindex",
			"foldersReindexed":  0,
			"totalFilesIndexed": 0,
		})
		return
	}

	// Reindex each folder
	totalFilesIndexed := 0
	totalFilesUpdated := 0
	totalFilesSkipped := 0
	foldersReindexed := 0

	for _, folder := range folders {
		h.logger.Info("Reindexing folder", zap.String("path", folder.Path))

		// Preserve watcher preference across reindex operations.
		restoreStatus := "active"
		if folder.Status == "inactive" {
			restoreStatus = "inactive"
		}

		// Get collection name (with fallback to default)
		mapping, _ := h.codeIndexStorage.GetPathMapping(folder.Path)
		collectionName := storage.CodeIndexCollection // fallback to default if mapping not found
		if mapping != nil {
			collectionName = mapping.QdrantCollection
		}

		// Update folder status to scanning
		if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "scanning", ""); err != nil {
			h.logger.Error("Failed to update folder status",
				zap.String("path", folder.Path),
				zap.Error(err))
			continue
		}

		// Scan directory for files
		scannedFiles, err := h.fileScanner.ScanDirectory(folder.Path)
		if err != nil {
			h.codeIndexStorage.UpdateFolderStatus(folder.ID, "error", err.Error())
			h.logger.Error("Failed to scan directory",
				zap.String("path", folder.Path),
				zap.Error(err))
			continue
		}

		filesIndexed := 0
		filesUpdated := 0
		filesSkipped := 0

		// Process each file
		for _, scannedFile := range scannedFiles {
			scannedFile.FolderID = folder.ID

			// Check if file already exists
			existingFile, _ := h.codeIndexStorage.GetFileByPath(scannedFile.Path)

			if existingFile != nil {
				// Check if file has changed
				if existingFile.SHA256 == scannedFile.SHA256 {
					filesSkipped++
					continue
				}
				filesUpdated++
				scannedFile.ID = existingFile.ID
			} else {
				filesIndexed++
				scannedFile.ID = uuid.New().String()
			}

			// Create chunks
			chunks, err := h.fileScanner.CreateFileChunks(scannedFile.ID, scannedFile.Path)
			if err != nil {
				h.logger.Warn("Failed to create chunks", zap.String("file", scannedFile.Path), zap.Error(err))
				continue
			}

			// Generate embeddings for chunks
			var qdrantPoints []storage.CodeIndexPoint
			for _, chunk := range chunks {
				// Generate embedding
				embedding, err := h.embeddingClient.CreateEmbedding(chunk.Content)
				if err != nil {
					h.logger.Warn("Failed to create embedding",
						zap.String("file", scannedFile.Path),
						zap.Int("chunk", chunk.ChunkNum),
						zap.Error(err))
					continue
				}

				// Create Qdrant point with deterministic UUID
				pointID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s_chunk_%d", scannedFile.ID, chunk.ChunkNum))).String()
				chunk.VectorID = pointID

				point := storage.CodeIndexPoint{
					ID:     pointID,
					Vector: embedding,
					Payload: map[string]interface{}{
						"fileId":       scannedFile.ID,
						"folderId":     folder.ID,
						"folderPath":   folder.Path,
						"filePath":     scannedFile.Path,
						"relativePath": scannedFile.RelativePath,
						"language":     scannedFile.Language,
						"chunkNum":     chunk.ChunkNum,
						"startLine":    chunk.StartLine,
						"endLine":      chunk.EndLine,
						"content":      chunk.Content,
					},
				}
				qdrantPoints = append(qdrantPoints, point)

				// Save chunk to MongoDB
				if err := h.codeIndexStorage.UpsertChunk(chunk); err != nil {
					h.logger.Warn("Failed to save chunk", zap.Error(err))
				}
			}

			// Upload vectors to Qdrant
			if len(qdrantPoints) > 0 {
				if err := h.qdrantClient.UpsertCodeIndexPoints(collectionName, qdrantPoints); err != nil {
					h.logger.Warn("Failed to upsert vectors", zap.String("file", scannedFile.Path), zap.Error(err))
				}
			}

			// Save file metadata to MongoDB
			if err := h.codeIndexStorage.UpsertFile(scannedFile); err != nil {
				h.logger.Warn("Failed to save file", zap.Error(err))
			}
		}

		// Update folder status and scan time
		if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, restoreStatus, ""); err != nil {
			h.logger.Warn("Failed to update folder status", zap.Error(err))
		}

		if err := h.codeIndexStorage.UpdateFolderScanTime(folder.ID, len(scannedFiles)); err != nil {
			h.logger.Warn("Failed to update scan time", zap.Error(err))
		}

		totalFilesIndexed += filesIndexed
		totalFilesUpdated += filesUpdated
		totalFilesSkipped += filesSkipped
		foldersReindexed++

		h.logger.Info("Folder reindexed",
			zap.String("path", folder.Path),
			zap.Int("filesIndexed", filesIndexed),
			zap.Int("filesUpdated", filesUpdated),
			zap.Int("filesSkipped", filesSkipped))
	}

	h.logger.Info("Reindex all completed",
		zap.Int("foldersReindexed", foldersReindexed),
		zap.Int("totalFilesIndexed", totalFilesIndexed))

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           fmt.Sprintf("Reindexed %d folders", foldersReindexed),
		"foldersReindexed":  foldersReindexed,
		"totalFilesIndexed": totalFilesIndexed,
		"totalFilesUpdated": totalFilesUpdated,
		"totalFilesSkipped": totalFilesSkipped,
	})
}

// HandleListFiles lists all files in a folder with metadata
// GET /api/v1/code-index/files/:folderId
func (h *RESTAPIHandler) HandleListFiles(c *gin.Context) {
	folderID := c.Param("folderId")

	// Get files from storage
	files, err := h.codeIndexStorage.ListFiles(folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files: " + err.Error()})
		return
	}

	// Convert to DTOs
	dtos := make([]FileDetailsDTO, len(files))
	for i, file := range files {
		dtos[i] = FileDetailsDTO{
			ID:           file.ID,
			FolderPath:   file.FolderID,
			RelativePath: file.RelativePath,
			Language:     file.Language,
			Size:         file.Size,
			LineCount:    file.LineCount,
			ChunkCount:   file.ChunkCount,
			IndexedAt:    file.IndexedAt.Format(time.RFC3339),
		}
	}

	h.logger.Info("Listed files",
		zap.String("folderID", folderID),
		zap.Int("count", len(dtos)))

	c.JSON(http.StatusOK, ListFilesResponse{
		Files: dtos,
		Count: len(dtos),
	})
}

// HandleGetFile returns details for a single file
// GET /api/v1/code-index/file/:fileId
func (h *RESTAPIHandler) HandleGetFile(c *gin.Context) {
	fileID := c.Param("fileId")

	// Get file from storage
	file, err := h.codeIndexStorage.GetFile(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found: " + fileID})
		return
	}

	// Convert to DTO
	dto := FileDetailsDTO{
		ID:           file.ID,
		FolderPath:   file.Path,
		RelativePath: file.RelativePath,
		Language:     file.Language,
		Size:         file.Size,
		LineCount:    file.LineCount,
		ChunkCount:   file.ChunkCount,
		IndexedAt:    file.IndexedAt.Format(time.RFC3339),
	}

	h.logger.Info("Retrieved file details",
		zap.String("fileID", fileID),
		zap.String("path", file.RelativePath))

	c.JSON(http.StatusOK, GetFileResponse{
		File: dto,
	})
}

// HandleGetFileChunks returns chunks for a file with AST metadata
// GET /api/v1/code-index/file/:fileId/chunks
func (h *RESTAPIHandler) HandleGetFileChunks(c *gin.Context) {
	fileID := c.Param("fileId")

	// Get chunks from storage
	chunks, err := h.codeIndexStorage.GetChunksByFileID(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chunks: " + err.Error()})
		return
	}

	// Convert to DTOs (include content field for UI display)
	dtos := make([]FileChunkDetailsDTO, len(chunks))
	for i, chunk := range chunks {
		dtos[i] = FileChunkDetailsDTO{
			ChunkNum:  chunk.ChunkNum,
			StartLine: chunk.StartLine,
			EndLine:   chunk.EndLine,
			Content:   chunk.Content,
			ChunkType: chunk.ChunkType,
			NodeType:  chunk.NodeType,
			NodeName:  chunk.NodeName,
			Signature: chunk.Signature,
		}
	}

	h.logger.Info("Retrieved file chunks",
		zap.String("fileID", fileID),
		zap.Int("count", len(dtos)))

	c.JSON(http.StatusOK, GetFileChunksResponse{
		Chunks: dtos,
		Count:  len(dtos),
	})
}

// HandleRemoveFolderByID removes a folder from the code index by folder ID
// DELETE /api/v1/code-index/folders/:folderId
func (h *RESTAPIHandler) HandleRemoveFolderByID(c *gin.Context) {
	folderID := c.Param("folderId")

	// Get folder
	folder, err := h.codeIndexStorage.GetFolder(folderID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found: " + folderID})
		return
	}

	// Get all files to delete their vectors
	files, err := h.codeIndexStorage.ListFiles(folder.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files: " + err.Error()})
		return
	}

	// Get path mapping before deletions
	mapping, _ := h.codeIndexStorage.GetPathMapping(folder.Path)

	// Delete vectors from Qdrant - lookup collection from path mapping
	if len(files) > 0 && mapping != nil {
		err = h.qdrantClient.DeleteCodeIndexByFilter(mapping.QdrantCollection, map[string]interface{}{
			"must": []map[string]interface{}{
				{"key": "folderId", "match": map[string]interface{}{"value": folder.ID}},
			},
		})
		if err != nil {
			h.logger.Warn("Failed to delete vectors from Qdrant", zap.Error(err))
		}
	}

	// Delete the Qdrant collection
	if mapping != nil {
		if err := h.qdrantClient.DeleteCollection(mapping.QdrantCollection); err != nil {
			h.logger.Warn("Failed to delete Qdrant collection",
				zap.String("collection", mapping.QdrantCollection),
				zap.Error(err))
		} else {
			h.logger.Info("Deleted Qdrant collection",
				zap.String("collection", mapping.QdrantCollection))
		}
	}

	// Delete path mapping from MongoDB
	if err := h.codeIndexStorage.RemovePathMapping(folder.Path); err != nil {
		h.logger.Warn("Failed to remove path mapping",
			zap.String("path", folder.Path),
			zap.Error(err))
	} else {
		h.logger.Info("Removed path mapping",
			zap.String("path", folder.Path))
	}

	// Remove folder from file watcher
	if h.fileWatcher != nil {
		if err := h.fileWatcher.RemoveFolder(folder.Path); err != nil {
			h.logger.Warn("Failed to remove folder from file watcher", zap.Error(err))
		} else {
			h.logger.Info("Removed folder from file watcher", zap.String("path", folder.Path))
		}
	}

	// Remove folder from MongoDB (cascades to files and chunks)
	if err := h.codeIndexStorage.RemoveFolder(folder.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove folder: " + err.Error()})
		return
	}

	h.logger.Info("Removed folder from code index",
		zap.String("folderID", folder.ID),
		zap.String("path", folder.Path),
		zap.Int("filesRemoved", len(files)))

	c.JSON(http.StatusOK, RemoveFolderResponse{
		Success:      true,
		Message:      "Folder removed successfully",
		FilesRemoved: len(files),
	})
}

// ToggleWatcherRequest is the request body for toggling watcher
type ToggleWatcherRequest struct {
	Enabled bool `json:"enabled"`
}

// HandleToggleWatcher enables or disables the file watcher for a specific folder
// PATCH /api/v1/code-index/folders/:folderId/watcher
func (h *RESTAPIHandler) HandleToggleWatcher(c *gin.Context) {
	folderID := c.Param("folderId")

	var req ToggleWatcherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get folder
	folder, err := h.codeIndexStorage.GetFolder(folderID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found: " + folderID})
		return
	}

	if h.fileWatcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "File watcher is not available",
		})
		return
	}

	// Add or remove from file watcher
	if req.Enabled {
		if isFileWatcherDisabledByEnv() {
			c.JSON(http.StatusConflict, gin.H{"error": "File watcher is disabled by server configuration (ENABLE_FILE_WATCHER=false)"})
			return
		}

		if err := h.fileWatcher.Start(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start watcher runtime: " + err.Error()})
			return
		}

		if err := h.fileWatcher.AddFolder(folder); err != nil {
			h.logger.Error("Failed to add folder to watcher",
				zap.String("path", folder.Path),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable watcher: " + err.Error()})
			return
		}

		if err := h.codeIndexStorage.UpdateFolderStatus(folderID, "active", ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder status: " + err.Error()})
			return
		}

		h.logger.Info("Enabled watcher for folder", zap.String("path", folder.Path))
	} else {
		if err := h.fileWatcher.RemoveFolder(folder.Path); err != nil {
			h.logger.Warn("Failed to remove folder from watcher",
				zap.String("path", folder.Path),
				zap.Error(err))
		}

		if err := h.codeIndexStorage.UpdateFolderStatus(folderID, "inactive", ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder status: " + err.Error()})
			return
		}

		h.logger.Info("Disabled watcher for folder", zap.String("path", folder.Path))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"enabled": req.Enabled,
		"message": fmt.Sprintf("Watcher %s for folder", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]),
	})
}

// ClearAllIndexData removes ALL indexed data but preserves folder configurations
// DELETE /api/v1/code-index/clear-all
func (h *RESTAPIHandler) ClearAllIndexData(c *gin.Context) {
	h.logger.Info("Clearing all code index data (preserving folder configs)")

	var errors []string
	filesRemoved := 0
	chunksRemoved := 0
	qdrantCollectionsRemoved := 0

	// 1. Get all folders to stop watchers
	folders, err := h.codeIndexStorage.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list folders: " + err.Error()})
		return
	}

	// 2. Stop file watcher for all folders (will be restarted on reindex)
	if h.fileWatcher != nil {
		for _, folder := range folders {
			if err := h.fileWatcher.RemoveFolder(folder.Path); err != nil {
				h.logger.Warn("Failed to remove folder from watcher", zap.String("path", folder.Path), zap.Error(err))
			}
		}
	}

	// 3. Get all path mappings (for Qdrant collections)
	mappings, err := h.codeIndexStorage.ListAllPathMappings()
	if err != nil {
		errors = append(errors, "Failed to list path mappings: "+err.Error())
	} else {
		// 4. Delete all Qdrant collections
		for _, mapping := range mappings {
			if err := h.qdrantClient.DeleteCollection(mapping.QdrantCollection); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to delete Qdrant collection %s: %v", mapping.QdrantCollection, err))
			} else {
				qdrantCollectionsRemoved++
			}
		}
	}

	// 5. Count before delete
	filesRemoved, _ = h.codeIndexStorage.CountAllFiles()
	chunksRemoved, _ = h.codeIndexStorage.CountAllChunks()

	// 6. Clear MongoDB collections (preserves folder configs)
	if err := h.codeIndexStorage.ClearAllIndexData(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear index data: " + err.Error()})
		return
	}

	h.logger.Info("Cleared all code index data",
		zap.Int("foldersPreserved", len(folders)),
		zap.Int("filesRemoved", filesRemoved),
		zap.Int("chunksRemoved", chunksRemoved),
		zap.Int("qdrantCollectionsRemoved", qdrantCollectionsRemoved))

	c.JSON(http.StatusOK, ClearAllIndexResponse{
		Success:                  true,
		Message:                  fmt.Sprintf("All indexed data cleared. %d folder configuration(s) preserved for reindexing.", len(folders)),
		FoldersRemoved:           0, // Folders are preserved
		FilesRemoved:             filesRemoved,
		ChunksRemoved:            chunksRemoved,
		QdrantCollectionsRemoved: qdrantCollectionsRemoved,
		Errors:                   errors,
	})
}

// GetFileWithContentResponse includes full file content
type GetFileWithContentResponse struct {
	FileID     string `json:"fileId"`
	Path       string `json:"path"`
	Language   string `json:"language"`
	Content    string `json:"content"`
	LineCount  int    `json:"lineCount"`
	Size       int64  `json:"size"`
	ChunkCount int    `json:"chunkCount"`
	IndexedAt  string `json:"indexedAt"`
}

// HandleGetFileWithContent returns a file with its full content assembled from chunks
// GET /api/v1/code-index/files/:fileId
func (h *RESTAPIHandler) HandleGetFileWithContent(c *gin.Context) {
	fileID := c.Param("fileId")

	// Get file metadata
	file, err := h.codeIndexStorage.GetFile(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found: " + fileID})
		return
	}

	// Get all chunks and assemble content
	chunks, err := h.codeIndexStorage.GetChunksByFileID(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file chunks: " + err.Error()})
		return
	}

	// Concatenate all chunks to build full file content
	var fullContent strings.Builder
	for _, chunk := range chunks {
		fullContent.WriteString(chunk.Content)
	}

	h.logger.Info("Retrieved file with content",
		zap.String("fileID", fileID),
		zap.String("path", file.Path),
		zap.Int("chunks", len(chunks)))

	c.JSON(http.StatusOK, GetFileWithContentResponse{
		FileID:     file.ID,
		Path:       file.Path,
		Language:   file.Language,
		Content:    fullContent.String(),
		LineCount:  file.LineCount,
		Size:       file.Size,
		ChunkCount: len(chunks),
		IndexedAt:  file.IndexedAt.Format(time.RFC3339),
	})
}

// Agent Communication Handlers

// validateAgentType validates that the agent type is one of the allowed types
func (h *RESTAPIHandler) validateAgentType(agentType string) error {
	if !validAgentTypes[agentType] {
		return fmt.Errorf("invalid agent type: %s", agentType)
	}
	return nil
}

// validateCommunicationType validates that the communication type is valid
func (h *RESTAPIHandler) validateCommunicationType(commType string) error {
	if !validCommunicationTypes[commType] {
		return fmt.Errorf("invalid communication type: %s", commType)
	}
	return nil
}

// validateAndLogAgentRequest validates the request method and content type
func (h *RESTAPIHandler) validateAndLogAgentRequest(c *gin.Context) bool {
	// Check HTTP method
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, AgentCommunicationResponse{
			Success:   false,
			Message:   "Method not allowed",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return false
	}

	// Check content type (allow empty for some cases)
	contentType := c.GetHeader("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		c.JSON(http.StatusUnsupportedMediaType, AgentCommunicationResponse{
			Success:   false,
			Message:   "Content-Type must be application/json",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return false
	}

	return true
}

// validateExecuteRequest validates an execute type request
func (h *RESTAPIHandler) validateExecuteRequest(req AgentCommunicationRequest) error {
	// Check message length
	if len(req.Message) > maxMessageLength {
		return fmt.Errorf("message length exceeds maximum of %d characters", maxMessageLength)
	}

	// Execute requires either a message or parameters with command
	if req.Message == "" && req.Parameters == nil {
		return fmt.Errorf("execute communication requires either a message or parameters with command")
	}

	// If parameters provided, validate command
	if req.Parameters != nil {
		command, ok := req.Parameters["command"]
		if ok {
			cmdStr, isString := command.(string)
			if !isString || cmdStr == "" {
				return fmt.Errorf("command must be a non-empty string")
			}
		}
	}

	return nil
}

// validateDirectMessageRequest validates a direct_message type request
func (h *RESTAPIHandler) validateDirectMessageRequest(req AgentCommunicationRequest) error {
	// Check message length
	if len(req.Message) > maxMessageLength {
		return fmt.Errorf("message length exceeds maximum of %d characters", maxMessageLength)
	}

	// Direct message requires a message (either in Message field or in Parameters)
	message := req.Message
	if message == "" && req.Parameters != nil {
		if msg, ok := req.Parameters["message"].(string); ok {
			message = msg
		}
	}

	if message == "" {
		return fmt.Errorf("direct_message communication requires message content")
	}

	return nil
}

// AgentCommunicate handles agent-to-agent communication
// POST /api/v1/agents/communicate
func (h *RESTAPIHandler) AgentCommunicate(c *gin.Context) {
	// Validate request basics
	if !h.validateAndLogAgentRequest(c) {
		return
	}

	var req AgentCommunicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   "Invalid request body: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate agent type
	if err := h.validateAgentType(req.AgentType); err != nil {
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   err.Error(),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate communication type
	if err := h.validateCommunicationType(req.CommunicationType); err != nil {
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   err.Error(),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Handle based on communication type
	switch req.CommunicationType {
	case "status":
		h.handleStatusCommunication(c, req)
	case "execute":
		h.handleExecuteCommunication(c, req)
	case "direct_message":
		h.handleDirectMessageCommunication(c, req)
	default:
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   "Unsupported communication type",
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
}

// handleStatusCommunication handles status type communications
func (h *RESTAPIHandler) handleStatusCommunication(c *gin.Context, req AgentCommunicationRequest) {
	// If task ID provided, verify access to task
	if req.TaskID != "" {
		task, err := h.taskStorage.GetAgentTask(req.TaskID)
		if err != nil {
			c.JSON(http.StatusForbidden, AgentCommunicationResponse{
				Success:   false,
				Message:   "task not found or access denied",
				AgentType: req.AgentType,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, AgentCommunicationResponse{
			Success:   true,
			Message:   fmt.Sprintf("Status for task %s: %s", task.ID, task.Status),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]interface{}{
				"taskId": task.ID,
				"status": string(task.Status),
			},
		})
		return
	}

	// General status response
	c.JSON(http.StatusOK, AgentCommunicationResponse{
		Success:   true,
		Message:   fmt.Sprintf("Agent %s is active", req.AgentType),
		AgentType: req.AgentType,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleExecuteCommunication handles execute type communications
func (h *RESTAPIHandler) handleExecuteCommunication(c *gin.Context, req AgentCommunicationRequest) {
	// Validate execute request
	if err := h.validateExecuteRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   err.Error(),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Get command from parameters
	var command string
	if req.Parameters != nil {
		if cmd, ok := req.Parameters["command"].(string); ok {
			command = cmd
		}
	}

	// Handle supported commands
	switch command {
	case "create_agent_task":
		h.handleCreateAgentTaskCommand(c, req)
	case "list_subagents":
		c.JSON(http.StatusOK, AgentCommunicationResponse{
			Success:   true,
			Message:   "Subagents listed successfully",
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]interface{}{
				"subagents": []string{"ui-dev", "go-dev", "sre", "coordinator", "data-analyst", "qa"},
			},
		})
	case "query_knowledge":
		c.JSON(http.StatusOK, AgentCommunicationResponse{
			Success:   true,
			Message:   "Knowledge query executed",
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
			Data:      map[string]interface{}{"results": []interface{}{}},
		})
	default:
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   fmt.Sprintf("Unsupported command: %s", command),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
}

// handleCreateAgentTaskCommand handles the create_agent_task command
func (h *RESTAPIHandler) handleCreateAgentTaskCommand(c *gin.Context, req AgentCommunicationRequest) {
	humanTaskID, _ := req.Parameters["humanTaskId"].(string)
	role, _ := req.Parameters["role"].(string)

	// Get todos from parameters
	var todos []storage.TodoItemInput
	if todosRaw, ok := req.Parameters["todos"]; ok {
		if todosList, isList := todosRaw.([]storage.TodoItemInput); isList {
			todos = todosList
		}
	}

	task, err := h.taskStorage.CreateAgentTask(
		humanTaskID,
		req.AgentType,
		role,
		todos,
		"",  // contextSummary
		nil, // filesModified
		nil, // qdrantCollections
		"",  // priorWorkSummary
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AgentCommunicationResponse{
			Success:   false,
			Message:   "Failed to create agent task: " + err.Error(),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusCreated, AgentCommunicationResponse{
		Success:   true,
		Message:   fmt.Sprintf("Agent task %s created successfully", task.ID),
		AgentType: req.AgentType,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"taskId": task.ID,
		},
	})
}

// handleDirectMessageCommunication handles direct_message type communications
func (h *RESTAPIHandler) handleDirectMessageCommunication(c *gin.Context, req AgentCommunicationRequest) {
	// Validate direct message request
	if err := h.validateDirectMessageRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, AgentCommunicationResponse{
			Success:   false,
			Message:   err.Error(),
			AgentType: req.AgentType,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Log the message
	h.logger.Info("Direct message received",
		zap.String("agentType", req.AgentType),
		zap.String("message", req.Message))

	c.JSON(http.StatusOK, AgentCommunicationResponse{
		Success:   true,
		Message:   "Message received successfully",
		AgentType: req.AgentType,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// RegisterRESTRoutes registers all REST API routes under /api/v1
func (h *RESTAPIHandler) RegisterRESTRoutes(r *gin.Engine) {
	// Human Tasks
	tasks := r.Group("/api/v1/tasks")
	{
		tasks.GET("", h.ListHumanTasks)
		tasks.POST("", h.CreateHumanTask)
		tasks.GET("/:id", h.GetHumanTask)
		tasks.PUT("/:id/status", h.UpdateTaskStatus)
	}

	// Agent Tasks
	agentTasks := r.Group("/api/v1/agent-tasks")
	{
		agentTasks.GET("", h.ListAgentTasks)
		agentTasks.POST("", h.CreateAgentTask)
		agentTasks.GET("/:id", h.GetAgentTask)
		agentTasks.PUT("/:agentTaskId/todos/:todoId/status", h.UpdateTodoStatus)
	}

	// Knowledge routes are registered separately in http_server.go
	// to avoid duplication - see http_server.go line 344

	// Code Index
	codeIndex := r.Group("/api/v1/code-index")
	{
		codeIndex.POST("/add-folder", h.AddFolder)
		codeIndex.DELETE("/remove-folder/:configId", h.RemoveFolder)
		codeIndex.DELETE("/folders/:folderId", h.HandleRemoveFolderByID)
		codeIndex.PATCH("/folders/:folderId/watcher", h.HandleToggleWatcher)
		codeIndex.POST("/scan", h.ScanFolder)
		codeIndex.POST("/search", h.SearchCode)
		codeIndex.GET("/status", h.GetIndexStatus)
		codeIndex.POST("/enable-watcher", h.EnableWatcher)
		codeIndex.POST("/disable-watcher", h.DisableWatcher)
		codeIndex.POST("/reindex-all", h.ReindexAll)
		codeIndex.GET("/folders/:folderId/files", h.HandleListFiles)
		codeIndex.GET("/file/:fileId/content", h.HandleGetFileWithContent)
		codeIndex.GET("/file/:fileId", h.HandleGetFile)
		codeIndex.GET("/file/:fileId/chunks", h.HandleGetFileChunks)
		codeIndex.DELETE("/clear-all", h.ClearAllIndexData)
	}
}
