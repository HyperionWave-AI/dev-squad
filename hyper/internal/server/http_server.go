package server

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	mcptools "hyper/internal/ai-service/tools/mcp"
	"hyper/internal/api"
	"hyper/internal/handlers"
	"hyper/internal/middleware"
	"hyper/internal/services"
	"hyper/internal/mcp/embeddings"
	mcphandlers "hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/watcher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/term"
)

// findProcessByPort finds the PID of the process using a specific port
func findProcessByPort(port string) (int, error) {
	// Try lsof first (macOS/BSD)
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", port))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		pidStr := strings.TrimSpace(string(output))
		// lsof can return multiple PIDs, take the first one
		if lines := strings.Split(pidStr, "\n"); len(lines) > 0 {
			pid, parseErr := strconv.Atoi(lines[0])
			if parseErr == nil {
				return pid, nil
			}
		}
	}

	// Fallback to netstat (works on macOS and some Linux)
	cmd = exec.Command("sh", "-c", fmt.Sprintf("netstat -anv -p tcp | grep '.%s' | grep LISTEN | awk '{print $9}' | head -n1", port))
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		pidStr := strings.TrimSpace(string(output))
		if pidStr != "" {
			pid, parseErr := strconv.Atoi(pidStr)
			if parseErr == nil {
				return pid, nil
			}
		}
	}

	return 0, fmt.Errorf("no process found on port %s", port)
}

// isInteractiveTerminal checks if the program is running in an interactive terminal
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// promptKillProcess asks the user if they want to kill the process using the port
func promptKillProcess(port string, pid int, logger *zap.Logger) (bool, error) {
	// Only prompt if running in an interactive terminal
	if !isInteractiveTerminal() {
		logger.Info("Not prompting to kill process - not running in interactive terminal")
		return false, nil
	}

	fmt.Printf("\n⚠️  Port %s is already in use by process %d\n", port, pid)
	fmt.Print("Kill the process and retry? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// killProcess attempts to gracefully terminate a process by PID
func killProcess(pid int, logger *zap.Logger) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try SIGTERM first for graceful shutdown
	logger.Info("Sending SIGTERM to process", zap.Int("pid", pid))
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	// Wait a bit for graceful shutdown
	time.Sleep(2 * time.Second)

	return nil
}

// StartHTTPServer starts the HTTP server with REST API + UI static serving + MCP HTTP endpoint
func StartHTTPServer(
	ctx context.Context,
	port string,
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	fileWatcher *watcher.FileWatcher,
	mcpServer *mcp.Server,
	embeddedUI http.FileSystem,
	hasEmbeddedUI bool,
	logger *zap.Logger,
	mongoDatabase *mongo.Database,
) error {
	// Create REST API handler
	restHandler := api.NewRESTAPIHandler(
		taskStorage,
		knowledgeStorage,
		codeIndexStorage,
		qdrantClient,
		embeddingClient,
		fileWatcher,
		logger,
	)

	// Initialize chat service
	chatService, err := services.NewChatService(mongoDatabase, logger)
	if err != nil {
		logger.Error("Failed to initialize chat service", zap.Error(err))
		return err
	}

	// Initialize AI settings service
	aiSettingsService, err := services.NewAISettingsService(mongoDatabase, logger)
	if err != nil {
		logger.Error("Failed to initialize AI settings service", zap.Error(err))
		return err
	}

	// Initialize AI service for real-time chat streaming
	aiConfig, err := aiservice.LoadAIConfig(".env.hyper")
	if err != nil {
		logger.Error("Failed to load AI config from .env.hyper", zap.Error(err))
		return err
	}
	logger.Info("AI configuration loaded",
		zap.String("provider", aiConfig.Provider),
		zap.String("model", aiConfig.Model),
		zap.Float64("temperature", aiConfig.Temperature))

	aiChatService, err := aiservice.NewChatService(aiConfig)
	if err != nil {
		logger.Error("Failed to initialize AI chat service", zap.Error(err))
		return err
	}
	logger.Info("AI chat service initialized successfully",
		zap.String("provider", aiConfig.Provider))

	// Initialize tools storage for HTTP tool management and MCP discovery
	toolsStorage, err := storage.NewToolsStorage(mongoDatabase, qdrantClient)
	if err != nil {
		logger.Error("Failed to create tools storage", zap.Error(err))
		return err
	}
	logger.Info("Tools storage initialized for HTTP tool management")

	// Create tools discovery handler for MCP management tools
	// Pass mcpServer instance for direct tool execution (no HTTP bridge needed)
	toolsDiscoveryHandler := mcphandlers.NewToolsDiscoveryHandler(toolsStorage, mcpServer)
	logger.Info("Tools discovery handler created with direct MCP server access")

	// Register MCP tools with the chat service
	logger.Info("Starting MCP tools registration...")
	toolRegistry := aiChatService.GetToolRegistry()

	// Initialize subchat storage (needed for execute_subagent tool)
	subchatStorage := storage.NewSubchatStorage(mongoDatabase, logger)

	// Register coordinator tools (task management, knowledge base, MCP management)
	logger.Info("Registering coordinator tools (task management, knowledge base, MCP management)...")
	beforeCount := len(toolRegistry.List())
	if err := mcptools.RegisterCoordinatorTools(
		toolRegistry,
		taskStorage,
		knowledgeStorage,
		toolsDiscoveryHandler,
		subchatStorage,
		aiChatService,  // AI service for sub-agent streaming
		chatService,    // Chat service for message storage
		aiSettingsService, // AI settings service for subagent prompts
		logger,         // Logger for debugging
	); err != nil {
		logger.Error("Failed to register coordinator tools", zap.Error(err))
		return err
	}
	afterCount := len(toolRegistry.List())
	coordinatorToolsCount := afterCount - beforeCount
	logger.Info("Coordinator tools registered",
		zap.Int("count", coordinatorToolsCount),
		zap.Int("totalSoFar", afterCount))
	for _, toolName := range toolRegistry.List()[beforeCount:afterCount] {
		logger.Debug("Registered coordinator tool", zap.String("name", toolName))
	}

	// Register Qdrant tools (semantic search and storage)
	logger.Info("Registering Qdrant tools (semantic search and storage)...")
	beforeCount = len(toolRegistry.List())
	if err := mcptools.RegisterQdrantTools(toolRegistry, qdrantClient); err != nil {
		logger.Error("Failed to register Qdrant tools", zap.Error(err))
		return err
	}
	afterCount = len(toolRegistry.List())
	qdrantToolsCount := afterCount - beforeCount
	logger.Info("Qdrant tools registered",
		zap.Int("count", qdrantToolsCount),
		zap.Int("totalSoFar", afterCount))
	for _, toolName := range toolRegistry.List()[beforeCount:afterCount] {
		logger.Debug("Registered Qdrant tool", zap.String("name", toolName))
	}

	// REMOVED: Code index tools (code_index_search, code_index_scan, code_index_add_folder)
	// These tools are stubs that require MCP server dependencies (embedding client, Qdrant, file scanner)
	// which are not available in the Chat context. They always fail with errors, confusing the AI.
	// Code indexing functionality should be accessed via the direct MCP endpoint at /mcp instead.
	// logger.Info("Registering code index tools (code search and indexing)...")
	// beforeCount = len(toolRegistry.List())
	// if err := mcptools.RegisterCodeIndexTools(toolRegistry, codeIndexStorage); err != nil {
	// 	logger.Error("Failed to register code index tools", zap.Error(err))
	// 	return err
	// }
	// afterCount = len(toolRegistry.List())
	codeIndexToolsCount := 0 // Set to 0 since tools are no longer registered
	// logger.Info("Code index tools registered",
	// 	zap.Int("count", codeIndexToolsCount),
	// 	zap.Int("totalSoFar", afterCount))
	// for _, toolName := range toolRegistry.List()[beforeCount:afterCount] {
	// 	logger.Debug("Registered code index tool", zap.String("name", toolName))
	// }

	// Register filesystem tools (bash, file operations, patch application)
	logger.Info("Registering filesystem tools (bash, file operations, patch application)...")
	beforeCount = len(toolRegistry.List())
	if err := tools.RegisterFilesystemTools(toolRegistry); err != nil {
		logger.Error("Failed to register filesystem tools", zap.Error(err))
		return err
	}
	afterCount = len(toolRegistry.List())
	filesystemToolsCount := afterCount - beforeCount
	logger.Info("Filesystem tools registered",
		zap.Int("count", filesystemToolsCount),
		zap.Int("totalSoFar", afterCount))
	for _, toolName := range toolRegistry.List()[beforeCount:afterCount] {
		logger.Debug("Registered filesystem tool", zap.String("name", toolName))
	}

	// Log final summary with all tool names
	allTools := toolRegistry.List()
	logger.Info("Chat service ready with MCP tools",
		zap.Int("totalTools", len(allTools)),
		zap.Int("coordinatorTools", coordinatorToolsCount),
		zap.Int("qdrantTools", qdrantToolsCount),
		zap.Int("codeIndexTools", codeIndexToolsCount),
		zap.Int("filesystemTools", filesystemToolsCount))
	logger.Info("All registered tools", zap.Strings("availableTools", allTools))

	// Create chat handlers
	chatHandler := handlers.NewChatHandler(chatService, logger)
	chatWebSocketHandler := handlers.NewChatWebSocketHandler(chatService, aiChatService, aiSettingsService, logger)

	// Create AI settings handler
	aiSettingsHandler := handlers.NewAISettingsHandler(aiSettingsService, logger)

	// Create HTTP tools handler
	httpToolsHandler, err := handlers.NewHTTPToolsHandler(mongoDatabase, toolsStorage, logger)
	if err != nil {
		logger.Error("Failed to initialize HTTP tools handler", zap.Error(err))
		return err
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure CORS for frontend
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		"http://localhost:5173",  // Vite dev server
		"http://localhost:5177",  // Alt Vite port
		"http://localhost:5178",  // Alt Vite port
		"http://localhost:7777",  // Main dev UI port
		"http://localhost:7779",  // Dev UI port
		"http://localhost:7780",  // Dev UI port (auto-assigned)
		"http://localhost:9173",  // Custom UI port
		"http://localhost:3000",  // React dev server
		"http://localhost",       // Docker UI
		"http://hyperion-ui",     // Docker internal network
		"http://hyperion-ui:80",  // Docker internal network with port
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "X-Request-ID", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Register optional JWT authentication middleware
	// Disabled by default (injects dev mock values)
	// Enable with ENABLE_JWT=true environment variable
	r.Use(middleware.OptionalJWTMiddleware())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "hyperion-coordinator-unified",
			"version": "2.0.0",
		})
	})

	// Register REST API routes
	restHandler.RegisterRESTRoutes(r)

	// Register chat routes
	chatGroup := r.Group("/api/v1/chat")
	{
		// REST endpoints for session management
		chatHandler.RegisterChatRoutes(chatGroup)

		// WebSocket endpoint for real-time streaming
		chatGroup.GET("/stream", chatWebSocketHandler.HandleChatWebSocket)
	}

	logger.Info("Chat API routes registered",
		zap.String("restPath", "/api/v1/chat/sessions"),
		zap.String("websocketPath", "/api/v1/chat/stream"))

	// Register AI settings routes
	aiSettingsGroup := r.Group("/api/v1/ai")
	{
		aiSettingsHandler.RegisterAISettingsRoutes(aiSettingsGroup)
	}

	logger.Info("AI Settings API routes registered",
		zap.String("systemPromptPath", "/api/v1/ai/system-prompt"),
		zap.String("subagentsPath", "/api/v1/ai/subagents"))

	// Register knowledge routes
	knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeStorage, logger)
	knowledgeGroup := r.Group("/api/v1/knowledge")
	{
		knowledgeHandler.RegisterRoutes(knowledgeGroup)
	}

	logger.Info("Knowledge API routes registered",
		zap.String("popularCollectionsPath", "/api/v1/knowledge/popular-collections"))

	// Subchat storage already initialized earlier for execute_subagent tool
	// Use it to seed system subagents and create handlers

	// Automatically seed system subagents on startup (idempotent - safe to run every time)
	logger.Info("Ensuring system subagents are seeded...")
	if err := subchatStorage.EnsureSystemSubagents(); err != nil {
		logger.Error("Failed to ensure system subagents", zap.Error(err))
		// Don't fail startup - log warning and continue
		logger.Warn("System subagents may not be available - some features may not work correctly")
	}

	subchatHandler := handlers.NewSubchatHandler(subchatStorage, taskStorage, logger)
	subagentHandler := handlers.NewSubagentHandler(subchatStorage, logger)

	// Register subchat routes
	subchatGroup := r.Group("/api/v1/subchats")
	{
		subchatGroup.POST("", subchatHandler.CreateSubchat)
		subchatGroup.GET("/:id", subchatHandler.GetSubchat)
		subchatGroup.PUT("/:id/status", subchatHandler.UpdateSubchatStatus)
	}

	// Register chat-subchats routes
	chatsGroup := r.Group("/api/v1/chats")
	{
		chatsGroup.GET("/:chatId/subchats", subchatHandler.GetSubchatsByParent)
	}

	// Register subagent routes
	subagentGroup := r.Group("/api/v1/subagents")
	{
		subagentGroup.GET("", subagentHandler.ListSubagents)
		subagentGroup.GET("/:name", subagentHandler.GetSubagent)
	}

	logger.Info("Subchat and Subagent API routes registered",
		zap.String("subchatsPath", "/api/v1/subchats"),
		zap.String("chatSubchatsPath", "/api/v1/chats/:chatId/subchats"),
		zap.String("subagentsPath", "/api/v1/subagents"))

	// Register HTTP tools routes
	httpToolsGroup := r.Group("/api/v1/tools/http")
	{
		httpToolsHandler.RegisterHTTPToolsRoutes(httpToolsGroup)
	}

	logger.Info("HTTP Tools API routes registered",
		zap.String("createPath", "/api/v1/tools/http"),
		zap.String("listPath", "/api/v1/tools/http"),
		zap.String("deletePath", "/api/v1/tools/http/:id"))

	// Create MCP HTTP handler using official go-sdk StreamableHTTPHandler
	// This implements the full MCP Streamable HTTP transport specification
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			// Return the same server instance for all requests
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			// Stateless: false means the handler will validate Mcp-Session-Id headers
			// and maintain proper session state across requests
			Stateless: false,
			// JSONResponse: false means responses will use text/event-stream (SSE)
			// for streamable responses, as per MCP spec
			JSONResponse: false,
		},
	)

	// Mount MCP handler at /mcp endpoint
	// This handles both GET (session info) and POST (JSON-RPC requests)
	// The StreamableHTTPHandler implements http.Handler interface
	r.Any("/mcp", gin.WrapH(mcpHandler))

	logger.Info("MCP HTTP transport initialized",
		zap.String("endpoint", "/mcp"),
		zap.String("transport", "StreamableHTTP"),
		zap.String("protocol", "2024-11-05"))

	// Serve UI static files
	// Priority: embedded UI (single binary) > filesystem (development)
	if hasEmbeddedUI && embeddedUI != nil {
		// Production mode: serve embedded UI from binary
		logger.Info("Serving embedded UI from binary (production mode)")
		r.StaticFS("/ui", embeddedUI)

		// SPA routing for embedded UI
		r.NoRoute(func(c *gin.Context) {
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/ui/" {
				// Serve index.html for SPA routes
				indexFile, err := embeddedUI.Open("index.html")
				if err == nil {
					defer indexFile.Close()
					c.DataFromReader(http.StatusOK, -1, "text/html; charset=utf-8", indexFile, nil)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		})
	} else {
		// Development mode: proxy to Vite dev server for hot reload
		viteURL := "http://localhost:5173"
		logger.Info("Proxying UI to Vite dev server (development mode)", zap.String("viteURL", viteURL))

		// Proxy all /ui requests to Vite dev server
		r.Any("/ui/*proxyPath", func(c *gin.Context) {
			proxy := &httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = "localhost:5173"
					// Strip /ui prefix - Vite serves from root in dev mode
					req.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/ui")
					if req.URL.Path == "" {
						req.URL.Path = "/"
					}
					req.Host = "localhost:5173"
				},
			}
			proxy.ServeHTTP(c.Writer, c.Request)
		})

		// Proxy root /ui to Vite
		r.GET("/ui", func(c *gin.Context) {
			proxy := &httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = "localhost:5173"
					// Vite serves from root in dev mode
					req.URL.Path = "/"
					req.Host = "localhost:5173"
				},
			}
			proxy.ServeHTTP(c.Writer, c.Request)
		})

		// Fallback for other routes
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		})
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Retry logic for port-busy errors
	maxRetries := 3
	var startErr error
	serverStarted := make(chan error, 1)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Start server in goroutine
		go func() {
			logger.Info("HTTP server starting",
				zap.String("port", port),
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", maxRetries))

			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverStarted <- err
			} else {
				serverStarted <- nil
			}
		}()

		// Wait briefly to see if server starts successfully or fails immediately
		select {
		case startErr = <-serverStarted:
			if startErr == nil {
				// Server started successfully
				logger.Info("HTTP server listening",
					zap.String("port", port),
					zap.String("apiEndpoints", "/api/tasks, /api/agent-tasks, /api/code-index"),
					zap.String("uiEndpoint", "/ui"))
				break
			}

			// Check if error is "address already in use"
			if !strings.Contains(startErr.Error(), "bind: address already in use") {
				// Different error, don't retry
				logger.Error("HTTP server error (not port-busy)", zap.Error(startErr))
				return startErr
			}

			// Port is busy - try to find and kill the process
			logger.Warn("Port is already in use", zap.String("port", port), zap.Int("attempt", attempt))

			pid, findErr := findProcessByPort(port)
			if findErr != nil {
				errMsg := fmt.Sprintf("Port %s is busy but couldn't find the process: %v\nManually kill the process with:\n  lsof -ti tcp:%s | xargs kill", port, findErr, port)
				logger.Error(errMsg)
				return fmt.Errorf("%s", errMsg)
			}

			// Prompt user to kill the process
			shouldKill, promptErr := promptKillProcess(port, pid, logger)
			if promptErr != nil {
				return fmt.Errorf("port %s is busy (PID: %d) but failed to prompt user: %w", port, pid, promptErr)
			}

			if !shouldKill {
				errMsg := fmt.Sprintf("Port %s is busy (PID: %d). Please kill the process manually:\n  kill %d\n  or use:\n  lsof -ti tcp:%s | xargs kill", port, pid, pid, port)
				logger.Error(errMsg)
				return fmt.Errorf("%s", errMsg)
			}

			// Kill the process
			logger.Info("Attempting to kill process", zap.Int("pid", pid))
			if killErr := killProcess(pid, logger); killErr != nil {
				return fmt.Errorf("failed to kill process %d: %w", pid, killErr)
			}

			fmt.Printf("✓ Killed process %d, retrying... (attempt %d/%d)\n", pid, attempt, maxRetries)
			logger.Info("Process killed, retrying server start",
				zap.Int("pid", pid),
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", maxRetries))

			// Wait a bit before retry
			time.Sleep(1 * time.Second)

			// Create new server instance for retry
			srv = &http.Server{
				Addr:    ":" + port,
				Handler: r,
			}

		case <-time.After(500 * time.Millisecond):
			// Server didn't fail immediately, assume it started successfully
			logger.Info("HTTP server listening",
				zap.String("port", port),
				zap.String("apiEndpoints", "/api/v1/tasks, /api/v1/agent-tasks, /api/v1/code-index, /api/v1/knowledge"),
				zap.String("uiEndpoint", "/ui"))
			startErr = nil
			break
		}

		// If server started successfully, break out of retry loop
		if startErr == nil {
			break
		}
	}

	// Check if we exhausted retries
	if startErr != nil {
		return fmt.Errorf("failed to start HTTP server after %d attempts: %w", maxRetries, startErr)
	}

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()
	logger.Info("HTTP server shutting down...")

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("HTTP server stopped")
	return nil
}
