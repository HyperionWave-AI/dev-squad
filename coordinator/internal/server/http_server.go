package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	aiservice "hyperion-coordinator/ai-service"
	"hyperion-coordinator/ai-service/tools"
	mcptools "hyperion-coordinator/ai-service/tools/mcp"
	"hyperion-coordinator/internal/api"
	"hyperion-coordinator/internal/handlers"
	"hyperion-coordinator/internal/middleware"
	"hyperion-coordinator/internal/services"
	"hyperion-coordinator-mcp/embeddings"
	"hyperion-coordinator-mcp/storage"
	"hyperion-coordinator-mcp/watcher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

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
	// Empty string tells LoadAIConfig to read from environment variables
	// The dev-hot.sh script exports all AI config vars via: set -a; source .env.hyper; set +a
	aiConfig, err := aiservice.LoadAIConfig("")
	if err != nil {
		logger.Error("Failed to load AI config from environment variables", zap.Error(err))
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

	// Register MCP tools with the chat service
	toolRegistry := aiChatService.GetToolRegistry()

	// Register coordinator tools (task management, knowledge base)
	if err := mcptools.RegisterCoordinatorTools(toolRegistry, taskStorage, knowledgeStorage); err != nil {
		logger.Error("Failed to register coordinator tools", zap.Error(err))
		return err
	}
	logger.Info("Coordinator tools registered (16 tools)")

	// Register Qdrant tools (semantic search and storage)
	if err := mcptools.RegisterQdrantTools(toolRegistry, qdrantClient); err != nil {
		logger.Error("Failed to register Qdrant tools", zap.Error(err))
		return err
	}
	logger.Info("Qdrant tools registered (2 tools)")

	// Register code index tools (code search and indexing)
	if err := mcptools.RegisterCodeIndexTools(toolRegistry, codeIndexStorage); err != nil {
		logger.Error("Failed to register code index tools", zap.Error(err))
		return err
	}
	logger.Info("Code index tools registered (5 tools)")

	// Register filesystem tools (bash, file operations, patch application)
	if err := tools.RegisterFilesystemTools(toolRegistry); err != nil {
		logger.Error("Failed to register filesystem tools", zap.Error(err))
		return err
	}
	logger.Info("Filesystem tools registered (5 tools)")

	logger.Info("Chat service ready with MCP tools",
		zap.Int("totalTools", len(toolRegistry.List())),
		zap.Strings("availableTools", toolRegistry.List()))

	// Create chat handlers
	chatHandler := handlers.NewChatHandler(chatService, logger)
	chatWebSocketHandler := handlers.NewChatWebSocketHandler(chatService, aiChatService, aiSettingsService, logger)

	// Create AI settings handler
	aiSettingsHandler := handlers.NewAISettingsHandler(aiSettingsService, logger)

	// Initialize tools storage for HTTP tool management
	toolsStorage, err := storage.NewToolsStorage(mongoDatabase, qdrantClient)
	if err != nil {
		logger.Error("Failed to create tools storage", zap.Error(err))
		return err
	}
	logger.Info("Tools storage initialized for HTTP tool management")

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
	// Check for HOT_RELOAD environment variable to force proxy mode for UI hot reloading
	if os.Getenv("HOT_RELOAD") == "true" {
		hasEmbeddedUI = false
		logger.Info("Hot reload mode enabled - proxying UI to Vite dev server",
			zap.String("hotReload", "true"),
			zap.String("viteURL", "http://localhost:5173"))
	}

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
					req.URL.Path = c.Request.URL.Path[3:] // Remove /ui prefix
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

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server listening",
			zap.String("port", port),
			zap.String("apiEndpoints", "/api/tasks, /api/agent-tasks, /api/code-index"),
			zap.String("uiEndpoint", "/ui"))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

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
