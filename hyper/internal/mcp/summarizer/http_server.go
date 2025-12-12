package summarizer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ============================================================================
// HTTP Server Configuration
// ============================================================================

// HTTPServerConfig holds configuration for the HTTP server
type HTTPServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	EnableCORS      bool
	EnableMetrics   bool
}

// DefaultHTTPServerConfig returns default HTTP server configuration
func DefaultHTTPServerConfig() HTTPServerConfig {
	return HTTPServerConfig{
		Port:            8080,
		Host:            "0.0.0.0",
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableCORS:      true,
		EnableMetrics:   true,
	}
}

// ============================================================================
// HTTP Server
// ============================================================================

// HTTPServer wraps the Gin engine and provides summarizer API endpoints
type HTTPServer struct {
	engine      *gin.Engine
	server      *http.Server
	config      HTTPServerConfig
	handlers    *SummarizerHandlers
	logger      *zap.Logger
}

// NewHTTPServer creates a new HTTP server for the summarizer API
func NewHTTPServer(summarizer CodeSummarizer, config HTTPServerConfig, logger *zap.Logger) (*HTTPServer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if config.Port <= 0 || config.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", config.Port)
	}

	// Create handlers
	handlers := NewSummarizerHandlers(summarizer, logger)

	// Create Gin engine
	engine := gin.New()

	// Apply middleware
	engine.Use(handlers.ErrorHandlingMiddleware())
	engine.Use(handlers.RequestIDMiddleware())
	engine.Use(handlers.LoggingMiddleware())

	// Add CORS middleware if enabled
	if config.EnableCORS {
		engine.Use(corsMiddleware())
	}

	// Register routes
	registerRoutes(engine, handlers)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &HTTPServer{
		engine:   engine,
		server:   httpServer,
		config:   config,
		handlers: handlers,
		logger:   logger,
	}, nil
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.String("address", s.server.Addr),
		zap.Int("port", s.config.Port),
	)

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
		return err
	}

	s.logger.Info("HTTP server stopped successfully")
	return nil
}

// ============================================================================
// Route Registration
// ============================================================================

// registerRoutes registers all API routes
func registerRoutes(engine *gin.Engine, handlers *SummarizerHandlers) {
	// Health check endpoints
	engine.GET("/api/health", handlers.HandleHealth)
	engine.GET("/api/ready", handlers.HandleReadiness)
	engine.GET("/api/live", handlers.HandleLiveness)

	// Summarization endpoints
	engine.POST("/api/summarize", handlers.HandleSummarize)
	engine.POST("/api/summarize/batch", handlers.HandleBatchSummarize)

	// Metrics endpoint
	engine.GET("/api/metrics", handlers.HandleMetrics)

	// Root endpoint
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "code-summarizer",
			"version": "1.0.0",
			"endpoints": gin.H{
				"health":      "GET /api/health",
				"ready":       "GET /api/ready",
				"live":        "GET /api/live",
				"summarize":   "POST /api/summarize",
				"batch":       "POST /api/summarize/batch",
				"metrics":     "GET /api/metrics",
			},
		})
	})

	// 404 handler
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "Not found",
			Message:   fmt.Sprintf("Route %s %s not found", c.Request.Method, c.Request.URL.Path),
			Code:      "NOT_FOUND",
			Timestamp: time.Now(),
		})
	})
}

// ============================================================================
// Middleware
// ============================================================================

// corsMiddleware adds CORS headers to responses
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
