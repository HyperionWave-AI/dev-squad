package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/scanner"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/watcher"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
	HumanTaskID       string                    `json:"humanTaskId" binding:"required"`
	AgentName         string                    `json:"agentName" binding:"required"`
	Role              string                    `json:"role" binding:"required"`
	Todos             []storage.TodoItemInput   `json:"todos" binding:"required"`
	ContextSummary    string                    `json:"contextSummary,omitempty"`
	FilesModified     []string                  `json:"filesModified,omitempty"`
	QdrantCollections []string                  `json:"qdrantCollections,omitempty"`
	PriorWorkSummary  string                    `json:"priorWorkSummary,omitempty"`
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
	Name     string `json:"name"`
	Category string `json:"category"`
	Count    int    `json:"count"`
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

type BrowseKnowledgeResponse struct {
	Entries []KnowledgeEntryDTO `json:"entries"`
	Count   int                 `json:"count"`
	Limit   int                 `json:"limit"`
}

// Code Index DTOs
type AddFolderRequest struct {
	FolderPath  string `json:"folderPath" binding:"required"`
	Description string `json:"description,omitempty"`
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
	WatcherStatus string      `json:"watcherStatus"` // "running" or "stopped"
	Folders       []FolderDTO `json:"folders"`
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
	humanTaskID := c.Query("humanTaskId")
	agentName := c.Query("agentName")
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

	allTasks := h.taskStorage.ListAllAgentTasks()

	// Apply filters
	var filteredTasks []*storage.AgentTask
	for _, task := range allTasks {
		if humanTaskID != "" && task.HumanTaskID != humanTaskID {
			continue
		}
		if agentName != "" && task.AgentName != agentName {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	totalCount := len(filteredTasks)

	// Apply pagination
	endIndex := offset + limit
	if offset > totalCount {
		offset = totalCount
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	paginatedTasks := filteredTasks[offset:endIndex]

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

// Knowledge Handlers

// ListCollections returns all knowledge collections with metadata
// GET /api/knowledge/collections
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
			Name:     col.Name,
			Category: col.Category,
			Count:    col.Count,
		}
	}

	c.JSON(http.StatusOK, ListCollectionsResponse{
		Collections: dtos,
	})
}

// BrowseKnowledge retrieves knowledge entries without search (browse mode)
// GET /api/knowledge/browse?collection=...&limit=10
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
	collectionsToQuery := []string{}
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

// Code Index Handlers

// AddFolder adds a folder to the code index
// POST /api/code-index/add-folder
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

	// Check if folder already exists
	existing, err := h.codeIndexStorage.GetFolderByPath(absPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing folder: " + err.Error()})
		return
	}
	if existing != nil {
		c.JSON(http.StatusOK, AddFolderResponse{
			Success: true,
			Message: "Folder already indexed. File watcher is monitoring changes.",
			Folder:  existing,
		})
		return
	}

	// Add folder to storage
	folder, err := h.codeIndexStorage.AddFolder(absPath, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add folder: " + err.Error()})
		return
	}

	// Add folder to file watcher
	if h.fileWatcher != nil {
		if err := h.fileWatcher.AddFolder(folder); err != nil {
			h.logger.Warn("Failed to add folder to file watcher", zap.Error(err))
		} else {
			h.logger.Info("Added folder to file watcher", zap.String("path", absPath))
		}
	}

	h.logger.Info("Added folder to code index",
		zap.String("folderID", folder.ID),
		zap.String("path", absPath))

	c.JSON(http.StatusCreated, AddFolderResponse{
		Success: true,
		Message: "Folder added successfully. File watcher is now monitoring changes. Use /api/code-index/scan to index existing files.",
		Folder:  folder,
	})
}

// RemoveFolder removes a folder from the code index
// DELETE /api/code-index/remove-folder/:configId
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

	// Delete vectors from Qdrant
	if len(files) > 0 {
		err = h.qdrantClient.DeleteCodeIndexByFilter(map[string]interface{}{
			"must": []map[string]interface{}{
				{"key": "folderId", "match": map[string]interface{}{"value": folder.ID}},
			},
		})
		if err != nil {
			h.logger.Warn("Failed to delete vectors from Qdrant", zap.Error(err))
		}
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
// POST /api/code-index/scan
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

		// Upload vectors to Qdrant
		if len(qdrantPoints) > 0 {
			if err := h.qdrantClient.UpsertCodeIndexPoints(qdrantPoints); err != nil {
				h.logger.Warn("Failed to upsert vectors", zap.String("file", scannedFile.Path), zap.Error(err))
			}
		}

		// Save file metadata to MongoDB
		if err := h.codeIndexStorage.UpsertFile(scannedFile); err != nil {
			h.logger.Warn("Failed to save file", zap.Error(err))
		}
	}

	// Update folder status and scan time
	if err := h.codeIndexStorage.UpdateFolderStatus(folder.ID, "active", ""); err != nil {
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

// SearchCode searches the code index
// POST /api/code-index/search
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

	retrieveMode := req.Retrieve
	if retrieveMode == "" {
		retrieveMode = "chunk"
	}
	if retrieveMode != "chunk" && retrieveMode != "full" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "retrieve must be 'chunk' or 'full'"})
		return
	}

	// Generate embedding for query
	queryEmbedding, err := h.embeddingClient.CreateEmbedding(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create query embedding: " + err.Error()})
		return
	}

	// Search in Qdrant
	searchResp, err := h.qdrantClient.SearchCodeIndex(queryEmbedding, limit)
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

		// Handle content based on retrieve mode
		if retrieveMode == "chunk" {
			// Default: return just the matching chunk content from Qdrant
			if content, ok := hit.Payload["content"].(string); ok {
				result.Content = content
			}
		} else if retrieveMode == "full" {
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

	h.logger.Info("Code search completed",
		zap.String("query", req.Query),
		zap.String("retrieveMode", retrieveMode),
		zap.Int("results", len(results)))

	c.JSON(http.StatusOK, SearchResponse{
		Success:      true,
		Query:        req.Query,
		RetrieveMode: retrieveMode,
		Results:      results,
		Count:        len(results),
	})
}

// GetIndexStatus gets the current index status
// GET /api/code-index/status
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

	// Determine watcher status (running if file watcher exists and has active folders)
	watcherStatus := "stopped"
	if h.fileWatcher != nil && status.ActiveFolders > 0 {
		watcherStatus = "running"
	}

	// Transform folders to UI format
	uiFolders := make([]FolderDTO, 0, len(folders))
	for _, folder := range folders {
		uiFolders = append(uiFolders, FolderDTO{
			ConfigId:   folder.ID,
			FolderPath: folder.Path,
			FileCount:  folder.FileCount,
			Enabled:    folder.Status == "active",
		})
	}

	c.JSON(http.StatusOK, IndexStatusResponse{
		TotalFolders:  status.TotalFolders,
		TotalFiles:    status.TotalFiles,
		TotalSize:     totalSize,
		WatcherStatus: watcherStatus,
		Folders:       uiFolders,
	})
}

// RegisterRESTRoutes registers all REST API routes
func (h *RESTAPIHandler) RegisterRESTRoutes(r *gin.Engine) {
	// Human Tasks
	tasks := r.Group("/api/tasks")
	{
		tasks.GET("", h.ListHumanTasks)
		tasks.POST("", h.CreateHumanTask)
		tasks.GET("/:id", h.GetHumanTask)
		tasks.PUT("/:id/status", h.UpdateTaskStatus)
	}

	// Agent Tasks
	agentTasks := r.Group("/api/agent-tasks")
	{
		agentTasks.GET("", h.ListAgentTasks)
		agentTasks.POST("", h.CreateAgentTask)
		agentTasks.GET("/:id", h.GetAgentTask)
		agentTasks.PUT("/:agentTaskId/todos/:todoId/status", h.UpdateTodoStatus)
	}

	// Knowledge
	knowledge := r.Group("/api/knowledge")
	{
		knowledge.GET("/collections", h.ListCollections)
		knowledge.GET("/browse", h.BrowseKnowledge)
	}

	// Code Index
	codeIndex := r.Group("/api/code-index")
	{
		codeIndex.POST("/add-folder", h.AddFolder)
		codeIndex.DELETE("/remove-folder/:configId", h.RemoveFolder)
		codeIndex.POST("/scan", h.ScanFolder)
		codeIndex.POST("/search", h.SearchCode)
		codeIndex.GET("/status", h.GetIndexStatus)
	}
}
