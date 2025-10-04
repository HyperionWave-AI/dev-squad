package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"hyperion-coordinator-mcp/handlers"
	"hyperion-coordinator-mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Hyperion Coordinator MCP Server")

	// Get MongoDB configuration from environment
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB"
		logger.Info("Using default MongoDB Atlas URI")
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "coordinator_db"
	}

	logger.Info("Connecting to MongoDB Atlas",
		zap.String("database", mongoDatabase))

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			logger.Error("Error disconnecting from MongoDB", zap.Error(err))
		}
	}()

	// Verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}

	logger.Info("Successfully connected to MongoDB Atlas")

	// Get database
	db := mongoClient.Database(mongoDatabase)

	// Initialize storage with MongoDB
	taskStorage, err := storage.NewMongoTaskStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize task storage", zap.Error(err))
	}
	logger.Info("Task storage initialized with MongoDB")

	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize knowledge storage", zap.Error(err))
	}
	logger.Info("Knowledge storage initialized with MongoDB")

	// Create MCP server with capabilities
	impl := &mcp.Implementation{
		Name:    "hyperion-coordinator-mcp",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasResources: true,
		HasTools:     true,
	}

	server := mcp.NewServer(impl, opts)

	// Initialize handlers
	resourceHandler := handlers.NewResourceHandler(taskStorage, knowledgeStorage)
	toolHandler := handlers.NewToolHandler(taskStorage, knowledgeStorage)

	// Register resource handlers
	if err := resourceHandler.RegisterResourceHandlers(server); err != nil {
		logger.Fatal("Failed to register resource handlers", zap.Error(err))
	}

	// Register tool handlers
	if err := toolHandler.RegisterToolHandlers(server); err != nil {
		logger.Fatal("Failed to register tool handlers", zap.Error(err))
	}

	logger.Info("All handlers registered successfully",
		zap.Int("tools", 9),
		zap.Int("resources", 2))

	// Get transport mode from environment (default: stdio)
	transportMode := os.Getenv("TRANSPORT_MODE")
	if transportMode == "" {
		transportMode = "stdio"
	}

	mcpPort := os.Getenv("MCP_PORT")
	if mcpPort == "" {
		mcpPort = "7778"
	}

	switch transportMode {
	case "http":
		// Start HTTP Streamable transport
		logger.Info("Starting MCP server with HTTP Streamable transport",
			zap.String("port", mcpPort),
			zap.String("endpoint", fmt.Sprintf("http://localhost:%s/mcp", mcpPort)))

		// Create HTTP handler with streamable transport
		handler := mcp.NewStreamableHTTPHandler(
			func(req *http.Request) *mcp.Server {
				return server
			},
			&mcp.StreamableHTTPOptions{
				Stateless:    false, // Support stateful sessions
				JSONResponse: true,  // Use application/json for responses
			},
		)

		// Setup HTTP server
		mux := http.NewServeMux()
		mux.Handle("/mcp", handler)

		// Add health check endpoint
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		httpServer := &http.Server{
			Addr:    fmt.Sprintf(":%s", mcpPort),
			Handler: mux,
		}

		logger.Info("HTTP server listening",
			zap.String("address", httpServer.Addr),
			zap.String("mcp_endpoint", "/mcp"),
			zap.String("health_endpoint", "/health"))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}

	default:
		// Start stdio transport (default for Claude Code)
		logger.Info("Starting MCP server with stdio transport")

		transport := &mcp.StdioTransport{}
		if err := server.Run(context.Background(), transport); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	}

	logger.Info("Server shutdown complete")
}