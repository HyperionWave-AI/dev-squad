package server

import (
	"context"
	"net/http"
	"time"

	"hyperion-coordinator/internal/api"
	"hyperion-coordinator-mcp/embeddings"
	"hyperion-coordinator-mcp/storage"
	"hyperion-coordinator-mcp/watcher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	logger *zap.Logger,
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
	// Check if UI dist directory exists
	uiDistPath := "./ui/dist"
	logger.Info("Serving UI static files", zap.String("path", uiDistPath))

	// Serve static files from /ui prefix
	r.Static("/ui", uiDistPath)

	// SPA routing: serve index.html for all /ui/* routes not matched by static files
	r.NoRoute(func(c *gin.Context) {
		// Only handle routes starting with /ui/
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/ui/" {
			c.File(uiDistPath + "/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		}
	})

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
