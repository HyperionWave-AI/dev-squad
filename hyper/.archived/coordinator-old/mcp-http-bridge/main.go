package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// HTTPBridge provides HTTP access to the MCP server via stdio
type HTTPBridge struct {
	mcpServerPath  string
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	mu             sync.Mutex
	pendingReqs    map[interface{}]chan MCPResponse
	pendingReqsMu  sync.RWMutex
	responseReader *json.Decoder
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *MCPError              `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewHTTPBridge creates a new HTTP bridge to the MCP server
func NewHTTPBridge(mcpServerPath string) (*HTTPBridge, error) {
	bridge := &HTTPBridge{
		mcpServerPath: mcpServerPath,
		pendingReqs:   make(map[interface{}]chan MCPResponse),
	}

	if err := bridge.start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	return bridge, nil
}

// start launches the MCP server process
func (b *HTTPBridge) start() error {
	b.cmd = exec.Command(b.mcpServerPath)

	var err error
	b.stdin, err = b.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	b.stdout, err = b.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Also capture stderr for debugging
	b.cmd.Stderr = os.Stderr

	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	log.Printf("MCP server started with PID: %d", b.cmd.Process.Pid)

	// Initialize response reader
	b.responseReader = json.NewDecoder(b.stdout)

	// Start background response handler
	go b.handleResponses()

	// Initialize MCP connection
	if err := b.initialize(); err != nil {
		b.stop()
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	return nil
}

// initialize sends the initialize request to the MCP server
func (b *HTTPBridge) initialize() error {
	initRequest := MCPRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "hyperion-coordinator-http-bridge",
				"version": "1.0.0",
			},
		},
	}

	_, err := b.sendRequest(initRequest)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Send initialized notification
	initializedNotification := MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	notifJSON, _ := json.Marshal(initializedNotification)
	if _, err := b.stdin.Write(append(notifJSON, '\n')); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	log.Println("MCP connection initialized successfully")
	return nil
}

// handleResponses continuously reads responses from the MCP server and routes them to pending requests
func (b *HTTPBridge) handleResponses() {
	for {
		var resp MCPResponse
		if err := b.responseReader.Decode(&resp); err != nil {
			if err == io.EOF {
				log.Println("MCP server connection closed")
				return
			}
			log.Printf("Error reading response: %v", err)
			continue
		}

		log.Printf("Received MCP response for ID: %v", resp.ID)

		// Route response to the appropriate pending request
		b.pendingReqsMu.RLock()
		respChan, exists := b.pendingReqs[resp.ID]
		b.pendingReqsMu.RUnlock()

		if exists {
			// Send response to waiting goroutine
			select {
			case respChan <- resp:
				log.Printf("Response delivered for ID: %v", resp.ID)
			default:
				log.Printf("Warning: response channel full for ID: %v", resp.ID)
			}

			// Clean up pending request
			b.pendingReqsMu.Lock()
			delete(b.pendingReqs, resp.ID)
			close(respChan)
			b.pendingReqsMu.Unlock()
		} else {
			log.Printf("Warning: received response for unknown request ID: %v", resp.ID)
		}
	}
}

// sendRequest sends a request to the MCP server and returns the response
func (b *HTTPBridge) sendRequest(req MCPRequest) (map[string]interface{}, error) {
	// Create response channel for this request
	respChan := make(chan MCPResponse, 1)

	// Register pending request
	b.pendingReqsMu.Lock()
	b.pendingReqs[req.ID] = respChan
	b.pendingReqsMu.Unlock()

	// Ensure cleanup even if we timeout
	defer func() {
		b.pendingReqsMu.Lock()
		if _, exists := b.pendingReqs[req.ID]; exists {
			delete(b.pendingReqs, req.ID)
			close(respChan)
		}
		b.pendingReqsMu.Unlock()
	}()

	// Marshal request
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("Sending MCP request: %s", string(reqJSON))

	// Send request (locked to prevent interleaved writes)
	b.mu.Lock()
	if _, err := b.stdin.Write(append(reqJSON, '\n')); err != nil {
		b.mu.Unlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	b.mu.Unlock()

	// Use longer timeout for initialization, shorter for regular requests
	timeout := 10 * time.Second
	if req.Method == "initialize" {
		timeout = 30 * time.Second // MongoDB connection can take time
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		log.Printf("Received MCP response: %+v", resp.Result)
		return resp.Result, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
}

// stop terminates the MCP server process
func (b *HTTPBridge) stop() error {
	if b.cmd != nil && b.cmd.Process != nil {
		log.Printf("Stopping MCP server (PID: %d)", b.cmd.Process.Pid)
		if err := b.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill MCP server: %w", err)
		}
		b.cmd.Wait()
	}
	return nil
}

// Handler functions for HTTP endpoints

func (b *HTTPBridge) handleListTools(c *gin.Context) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/list",
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleCallTool(c *gin.Context) {
	var body struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      body.Name,
			"arguments": body.Arguments,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleListResources(c *gin.Context) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "resources/list",
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleReadResource(c *gin.Context) {
	uri := c.Query("uri")
	if uri == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uri parameter required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "resources/read",
		Params: map[string]interface{}{
			"uri": uri,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleListCollections(c *gin.Context) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "resources/read",
		Params: map[string]interface{}{
			"uri": "hyperion://knowledge/collections",
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleSearchKnowledge(c *gin.Context) {
	collectionName := c.Query("collectionName")
	query := c.Query("query")
	limit := c.DefaultQuery("limit", "10")

	if collectionName == "" || query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collectionName and query parameters required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_query_knowledge",
			"arguments": map[string]interface{}{
				"collection": collectionName,
				"query":      query,
				"limit":      limit,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleCreateKnowledge(c *gin.Context) {
	var body struct {
		CollectionName string                 `json:"collectionName" binding:"required"`
		Information    string                 `json:"information" binding:"required"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: collectionName and information required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_upsert_knowledge",
			"arguments": map[string]interface{}{
				"collection": body.CollectionName,
				"text":       body.Information,
				"metadata":   body.Metadata,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// REST API handlers for Human Tasks

func (b *HTTPBridge) handleListHumanTasks(c *gin.Context) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "coordinator_list_human_tasks",
			"arguments": map[string]interface{}{},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse result - check for tasks array or content[0].text
	if tasks, ok := result["tasks"].([]interface{}); ok {
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
		return
	}

	// Try parsing from content[0].text
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				// Extract JSON array from text (it may have a header like "✓ Retrieved X tasks\n\nTasks:\n[...]")
				// Find the start of the JSON array
				start := strings.Index(text, "[")
				if start >= 0 {
					jsonText := text[start:]
					var tasks []interface{}
					if err := json.Unmarshal([]byte(jsonText), &tasks); err == nil {
						c.JSON(http.StatusOK, gin.H{"tasks": tasks})
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleGetHumanTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "resources/read",
		Params: map[string]interface{}{
			"uri": fmt.Sprintf("hyperion://task/human/%s", taskID),
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		if fmt.Sprint(err) == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse taskData.contents[0].text as JSON
	if contents, ok := result["contents"].([]interface{}); ok && len(contents) > 0 {
		if contentObj, ok := contents[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				var task map[string]interface{}
				if err := json.Unmarshal([]byte(text), &task); err == nil {
					c.JSON(http.StatusOK, task)
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleCreateHumanTask(c *gin.Context) {
	var body struct {
		Prompt string `json:"prompt" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_create_human_task",
			"arguments": map[string]interface{}{
				"prompt": body.Prompt,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract taskId from result content using regex
	var taskID string
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				// Use regex to extract task ID: Task ID: <uuid>
				re := regexp.MustCompile(`Task ID:\s*([a-f0-9-]+)`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					taskID = matches[1]
				}
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"taskId":  taskID,
		"success": true,
	})
}

func (b *HTTPBridge) handleUpdateTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID required"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_update_task_status",
			"arguments": map[string]interface{}{
				"taskId": taskID,
				"status": body.Status,
				"notes":  body.Notes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// REST API handlers for Agent Tasks

func (b *HTTPBridge) handleListAgentTasks(c *gin.Context) {
	agentName := c.Query("agentName")
	humanTaskID := c.Query("humanTaskId")

	// Parse pagination parameters
	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 50 {
				limit = 50 // Enforce max limit
			}
		}
	}

	args := map[string]interface{}{
		"offset": float64(offset),
		"limit":  float64(limit),
	}

	if agentName != "" {
		args["agentName"] = agentName
	}

	if humanTaskID != "" {
		args["humanTaskId"] = humanTaskID
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "coordinator_list_agent_tasks",
			"arguments": args,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse result - check for tasks array or content[0].text
	if tasks, ok := result["tasks"].([]interface{}); ok {
		// Return with pagination metadata
		response := gin.H{"tasks": tasks}
		if count, ok := result["count"].(float64); ok {
			response["count"] = int(count)
		}
		if totalCount, ok := result["totalCount"].(float64); ok {
			response["totalCount"] = int(totalCount)
		}
		if resultOffset, ok := result["offset"].(float64); ok {
			response["offset"] = int(resultOffset)
		}
		if resultLimit, ok := result["limit"].(float64); ok {
			response["limit"] = int(resultLimit)
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Try parsing from content[0].text
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				// Extract JSON array from text (it may have a header)
				start := strings.Index(text, "[")
				if start >= 0 {
					jsonText := text[start:]
					var tasks []interface{}
					if err := json.Unmarshal([]byte(jsonText), &tasks); err == nil {
						c.JSON(http.StatusOK, gin.H{
							"tasks":      tasks,
							"count":      len(tasks),
							"totalCount": len(tasks),
							"offset":     offset,
							"limit":      limit,
						})
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleCreateAgentTask(c *gin.Context) {
	var body struct {
		HumanTaskID      string                   `json:"humanTaskId" binding:"required"`
		AgentName        string                   `json:"agentName" binding:"required"`
		Role             string                   `json:"role" binding:"required"`
		Todos            []interface{}            `json:"todos" binding:"required"`
		ContextSummary   string                   `json:"contextSummary"`
		FilesModified    []string                 `json:"filesModified"`
		QdrantCollection []string                 `json:"qdrantCollections"`
		PriorWorkSummary string                   `json:"priorWorkSummary"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "humanTaskId, agentName, role, and todos are required"})
		return
	}

	args := map[string]interface{}{
		"humanTaskId": body.HumanTaskID,
		"agentName":   body.AgentName,
		"role":        body.Role,
		"todos":       body.Todos,
	}

	if body.ContextSummary != "" {
		args["contextSummary"] = body.ContextSummary
	}
	if len(body.FilesModified) > 0 {
		args["filesModified"] = body.FilesModified
	}
	if len(body.QdrantCollection) > 0 {
		args["qdrantCollections"] = body.QdrantCollection
	}
	if body.PriorWorkSummary != "" {
		args["priorWorkSummary"] = body.PriorWorkSummary
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "coordinator_create_agent_task",
			"arguments": args,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract taskId from response
	var taskID string
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				re := regexp.MustCompile(`Task ID:\s*([a-f0-9-]+)`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					taskID = matches[1]
				}
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"taskId":  taskID,
		"success": true,
	})
}

func (b *HTTPBridge) handleUpdateTodoStatus(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	todoID := c.Param("todoId")

	if agentTaskID == "" || todoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId and todoId required"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_update_todo_status",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"todoId":      todoID,
				"status":      body.Status,
				"notes":       body.Notes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// REST API handlers for Prompt Notes (Task-level)

func (b *HTTPBridge) handleAddTaskPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	if agentTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId required"})
		return
	}

	var body struct {
		PromptNotes string `json:"promptNotes" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "promptNotes field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_add_task_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"promptNotes": body.PromptNotes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

func (b *HTTPBridge) handleUpdateTaskPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	if agentTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId required"})
		return
	}

	var body struct {
		PromptNotes string `json:"promptNotes" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "promptNotes field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_update_task_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"promptNotes": body.PromptNotes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

func (b *HTTPBridge) handleClearTaskPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	if agentTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_clear_task_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// REST API handlers for Prompt Notes (TODO-level)

func (b *HTTPBridge) handleAddTodoPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	todoID := c.Param("todoId")

	if agentTaskID == "" || todoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId and todoId required"})
		return
	}

	var body struct {
		PromptNotes string `json:"promptNotes" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "promptNotes field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_add_todo_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"todoId":      todoID,
				"promptNotes": body.PromptNotes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

func (b *HTTPBridge) handleUpdateTodoPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	todoID := c.Param("todoId")

	if agentTaskID == "" || todoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId and todoId required"})
		return
	}

	var body struct {
		PromptNotes string `json:"promptNotes" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "promptNotes field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_update_todo_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"todoId":      todoID,
				"promptNotes": body.PromptNotes,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

func (b *HTTPBridge) handleClearTodoPromptNotes(c *gin.Context) {
	agentTaskID := c.Param("agentTaskId")
	todoID := c.Param("todoId")

	if agentTaskID == "" || todoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agentTaskId and todoId required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_clear_todo_prompt_notes",
			"arguments": map[string]interface{}{
				"agentTaskId": agentTaskID,
				"todoId":      todoID,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// REST API handlers for Knowledge operations

func (b *HTTPBridge) handleQueryKnowledge(c *gin.Context) {
	var body struct {
		Collection string `json:"collection" binding:"required"`
		Query      string `json:"query" binding:"required"`
		Limit      int    `json:"limit"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection and query fields required"})
		return
	}

	if body.Limit == 0 {
		body.Limit = 10
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_query_knowledge",
			"arguments": map[string]interface{}{
				"collection": body.Collection,
				"query":      body.Query,
				"limit":      body.Limit,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse result.content[0].text as JSON array
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				var entries []interface{}
				if err := json.Unmarshal([]byte(text), &entries); err == nil {
					c.JSON(http.StatusOK, gin.H{"entries": entries})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleUpsertKnowledge(c *gin.Context) {
	var body struct {
		CollectionName string                 `json:"collectionName" binding:"required"`
		Information    string                 `json:"information" binding:"required"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collectionName and information fields required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_upsert_knowledge",
			"arguments": map[string]interface{}{
				"collection": body.CollectionName,
				"text":       body.Information,
				"metadata":   body.Metadata,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract ID from response
	var id string
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				// Look for "ID: <uuid>" pattern
				re := regexp.MustCompile(`ID:\s*([a-f0-9-]+)`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					id = matches[1]
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"success": true,
		"result":  result,
	})
}

func (b *HTTPBridge) handlePopularCollections(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "5")
	limit := 5
	if val, err := strconv.Atoi(limitStr); err == nil {
		limit = val
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "coordinator_get_popular_collections",
			"arguments": map[string]interface{}{
				"limit": limit,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse result.content[0].text as JSON array
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				var collections []interface{}
				if err := json.Unmarshal([]byte(text), &collections); err == nil {
					c.JSON(http.StatusOK, collections)
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// REST API handlers for Code Index operations

func (b *HTTPBridge) handleCodeIndexAddFolder(c *gin.Context) {
	var body struct {
		FolderPath string `json:"folderPath" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folderPath field required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "code_index_add_folder",
			"arguments": map[string]interface{}{
				"folderPath": body.FolderPath,
				"description": body.Description,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract config ID from response
	var configID string
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				re := regexp.MustCompile(`Config ID:\s*([a-f0-9-]+)`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					configID = matches[1]
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"configId": configID,
	})
}

func (b *HTTPBridge) handleCodeIndexRemoveFolder(c *gin.Context) {
	folderPath := c.Query("folderPath")
	if folderPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folderPath query parameter required"})
		return
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "code_index_remove_folder",
			"arguments": map[string]interface{}{
				"folderPath": folderPath,
			},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": result,
	})
}

func (b *HTTPBridge) handleCodeIndexScan(c *gin.Context) {
	folderPath := c.Query("folderPath")

	args := map[string]interface{}{}
	if folderPath != "" {
		args["folderPath"] = folderPath
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "code_index_scan",
			"arguments": args,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": result,
	})
}

func (b *HTTPBridge) handleCodeSearch(c *gin.Context) {
	var body struct {
		Query     string   `json:"query" binding:"required"`
		FolderPath string   `json:"folderPath"`
		Limit     int      `json:"limit"`
		Retrieve  string   `json:"retrieve"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query field required"})
		return
	}

	args := map[string]interface{}{
		"query": body.Query,
	}

	if body.FolderPath != "" {
		args["folderPath"] = body.FolderPath
	}
	if body.Limit > 0 {
		args["limit"] = body.Limit
	}
	if body.Retrieve != "" {
		args["retrieve"] = body.Retrieve
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "code_index_search",
			"arguments": args,
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse results from response text
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				// Try to parse as JSON array
				var results []interface{}
				if err := json.Unmarshal([]byte(text), &results); err == nil {
					c.JSON(http.StatusOK, gin.H{"results": results})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func (b *HTTPBridge) handleCodeIndexStatus(c *gin.Context) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.GetHeader("X-Request-ID"),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "code_index_status",
			"arguments": map[string]interface{}{},
		},
	}

	result, err := b.sendRequest(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse status from response text
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if contentObj, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentObj["text"].(string); ok {
				var status map[string]interface{}
				if err := json.Unmarshal([]byte(text), &status); err == nil {
					c.JSON(http.StatusOK, status)
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func main() {
	// Get MCP server path from environment or use default
	mcpServerPath := os.Getenv("MCP_SERVER_PATH")
	if mcpServerPath == "" {
		mcpServerPath = "../mcp-server/hyperion-coordinator-mcp"
	}

	// Verify MCP server exists
	if _, err := os.Stat(mcpServerPath); os.IsNotExist(err) {
		log.Fatalf("MCP server not found at: %s", mcpServerPath)
	}

	// Create HTTP bridge
	bridge, err := NewHTTPBridge(mcpServerPath)
	if err != nil {
		log.Fatalf("Failed to create HTTP bridge: %v", err)
	}
	defer bridge.stop()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure CORS for frontend
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",     // Vite dev server (host)
		"http://localhost:5177",     // Alt Vite port
		"http://localhost:5178",     // Alt Vite port
		"http://localhost:7777",     // Main dev UI port
		"http://localhost:7779",     // Dev UI port
		"http://localhost:7780",     // Dev UI port (auto-assigned)
		"http://localhost:9173",     // Custom UI port (via docker-compose.override.yml)
		"http://localhost:3000",     // React dev server
		"http://localhost",          // Docker UI (mapped to host)
		"http://hyperion-ui",        // Docker internal network
		"http://hyperion-ui:80",     // Docker internal network with port
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "X-Request-ID"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "hyperion-coordinator-http-bridge",
			"version": "1.0.0",
		})
	})

	// MCP API endpoints
	api := r.Group("/api/mcp")
	{
		// Root endpoint for connection validation (GET) and MCP protocol messages (POST)
		api.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"server":          "hyperion-coordinator-mcp",
				"version":         "1.0.0",
				"protocolVersion": "2024-11-05",
				"capabilities": gin.H{
					"tools":     gin.H{"listChanged": false},
					"resources": gin.H{"subscribe": false, "listChanged": false},
				},
			})
		})
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"server":          "hyperion-coordinator-mcp",
				"version":         "1.0.0",
				"protocolVersion": "2024-11-05",
				"capabilities": gin.H{
					"tools":     gin.H{"listChanged": false},
					"resources": gin.H{"subscribe": false, "listChanged": false},
				},
			})
		})
		// Handle MCP JSON-RPC requests at root (for Claude Code HTTP transport)
		api.POST("", func(c *gin.Context) {
			var jsonRPCReq struct {
				JSONRPC string                 `json:"jsonrpc"`
				ID      interface{}            `json:"id"`
				Method  string                 `json:"method"`
				Params  map[string]interface{} `json:"params,omitempty"`
			}

			if err := c.BindJSON(&jsonRPCReq); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON-RPC request"})
				return
			}

			// Convert to MCP request and forward
			mcpReq := MCPRequest{
				JSONRPC: jsonRPCReq.JSONRPC,
				ID:      jsonRPCReq.ID,
				Method:  jsonRPCReq.Method,
				Params:  jsonRPCReq.Params,
			}

			result, err := bridge.sendRequest(mcpReq)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"jsonrpc": "2.0",
					"id":      jsonRPCReq.ID,
					"error": gin.H{
						"code":    -32603,
						"message": err.Error(),
					},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      jsonRPCReq.ID,
				"result":  result,
			})
		})
		api.POST("/", func(c *gin.Context) {
			var jsonRPCReq struct {
				JSONRPC string                 `json:"jsonrpc"`
				ID      interface{}            `json:"id"`
				Method  string                 `json:"method"`
				Params  map[string]interface{} `json:"params,omitempty"`
			}

			if err := c.BindJSON(&jsonRPCReq); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON-RPC request"})
				return
			}

			// Convert to MCP request and forward
			mcpReq := MCPRequest{
				JSONRPC: jsonRPCReq.JSONRPC,
				ID:      jsonRPCReq.ID,
				Method:  jsonRPCReq.Method,
				Params:  jsonRPCReq.Params,
			}

			result, err := bridge.sendRequest(mcpReq)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"jsonrpc": "2.0",
					"id":      jsonRPCReq.ID,
					"error": gin.H{
						"code":    -32603,
						"message": err.Error(),
					},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      jsonRPCReq.ID,
				"result":  result,
			})
		})
		api.GET("/tools", bridge.handleListTools)
		api.POST("/tools/call", bridge.handleCallTool)
		api.GET("/resources", bridge.handleListResources)
		api.GET("/resources/read", bridge.handleReadResource)
	}

	// Knowledge API endpoints (proxied to MCP tools)
	knowledge := r.Group("/api/knowledge")
	{
		knowledge.GET("/collections", bridge.handleListCollections)
		knowledge.GET("/search", bridge.handleSearchKnowledge) // Deprecated - use POST /query
		knowledge.POST("/query", bridge.handleQueryKnowledge)
		knowledge.POST("", bridge.handleUpsertKnowledge)
		knowledge.GET("/popular-collections", bridge.handlePopularCollections)
	}

	// Human Tasks API endpoints
	tasks := r.Group("/api/tasks")
	{
		tasks.GET("", bridge.handleListHumanTasks)
		tasks.GET("/:id", bridge.handleGetHumanTask)
		tasks.POST("", bridge.handleCreateHumanTask)
		tasks.PUT("/:id/status", bridge.handleUpdateTaskStatus)
	}

	// Agent Tasks API endpoints
	agentTasks := r.Group("/api/agent-tasks")
	{
		agentTasks.GET("", bridge.handleListAgentTasks)
		agentTasks.POST("", bridge.handleCreateAgentTask)
		agentTasks.PUT("/:agentTaskId/todos/:todoId/status", bridge.handleUpdateTodoStatus)

		// Task-level prompt notes
		agentTasks.POST("/:agentTaskId/prompt-notes", bridge.handleAddTaskPromptNotes)
		agentTasks.PUT("/:agentTaskId/prompt-notes", bridge.handleUpdateTaskPromptNotes)
		agentTasks.DELETE("/:agentTaskId/prompt-notes", bridge.handleClearTaskPromptNotes)

		// TODO-level prompt notes
		agentTasks.POST("/:agentTaskId/todos/:todoId/prompt-notes", bridge.handleAddTodoPromptNotes)
		agentTasks.PUT("/:agentTaskId/todos/:todoId/prompt-notes", bridge.handleUpdateTodoPromptNotes)
		agentTasks.DELETE("/:agentTaskId/todos/:todoId/prompt-notes", bridge.handleClearTodoPromptNotes)
	}

	// Code Index API endpoints
	codeIndex := r.Group("/api/code-index")
	{
		codeIndex.POST("/add-folder", bridge.handleCodeIndexAddFolder)
		codeIndex.DELETE("/remove-folder", bridge.handleCodeIndexRemoveFolder)
		codeIndex.POST("/scan", bridge.handleCodeIndexScan)
		codeIndex.POST("/search", bridge.handleCodeSearch)
		codeIndex.GET("/status", bridge.handleCodeIndexStatus)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "7095"
	}

	log.Printf("HTTP bridge listening on port %s", port)
	log.Printf("MCP server path: %s", mcpServerPath)
	log.Printf("Frontend CORS enabled for: http://localhost:5173, http://localhost:3000")

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}// test comment
