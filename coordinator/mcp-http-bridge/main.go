package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
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
		api.GET("/tools", bridge.handleListTools)
		api.POST("/tools/call", bridge.handleCallTool)
		api.GET("/resources", bridge.handleListResources)
		api.GET("/resources/read", bridge.handleReadResource)
	}

	// Knowledge API endpoints (proxied to MCP tools)
	knowledge := r.Group("/api/knowledge")
	{
		knowledge.GET("/collections", bridge.handleListCollections)
		knowledge.GET("/search", bridge.handleSearchKnowledge)
		knowledge.POST("", bridge.handleCreateKnowledge)
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
}