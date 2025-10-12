package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code-indexing-mcp/embeddings"
	"code-indexing-mcp/handlers"
	"code-indexing-mcp/storage"
	"code-indexing-mcp/watcher"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables
	_ = godotenv.Load("../.env")

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Code Indexing MCP Server")

	// Get configuration from environment
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		logger.Fatal("MONGODB_URI environment variable is required")
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "code_index_db"
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	qdrantAPIKey := os.Getenv("QDRANT_API_KEY")

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		logger.Fatal("OPENAI_API_KEY environment variable is required")
	}

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

	// Verify MongoDB connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}
	logger.Info("Successfully connected to MongoDB", zap.String("database", mongoDatabase))

	// Get database
	db := mongoClient.Database(mongoDatabase)

	// Initialize storage
	mongoStorage, err := storage.NewMongoStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize MongoDB storage", zap.Error(err))
	}
	logger.Info("MongoDB storage initialized")

	// Initialize Qdrant client
	qdrantClient := storage.NewQdrantClient(qdrantURL, qdrantAPIKey)
	logger.Info("Qdrant client initialized", zap.String("url", qdrantURL))

	// Ensure Qdrant collection exists
	if err := qdrantClient.EnsureCollection(); err != nil {
		logger.Fatal("Failed to ensure Qdrant collection", zap.Error(err))
	}
	logger.Info("Qdrant collection ready", zap.String("collection", storage.CodeIndexCollection))

	// Initialize OpenAI client
	embeddingClient := embeddings.NewOpenAIClient(openaiAPIKey)
	logger.Info("OpenAI embedding client initialized")

	// Initialize file watcher
	fileWatcher, err := watcher.NewFileWatcher(mongoStorage, qdrantClient, embeddingClient, logger)
	if err != nil {
		logger.Fatal("Failed to create file watcher", zap.Error(err))
	}

	// Start file watcher
	if err := fileWatcher.Start(); err != nil {
		logger.Fatal("Failed to start file watcher", zap.Error(err))
	}
	logger.Info("File watcher started successfully")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Cleanup function
	cleanup := func() {
		logger.Info("Shutting down gracefully...")
		if err := fileWatcher.Stop(); err != nil {
			logger.Error("Error stopping file watcher", zap.Error(err))
		}
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			logger.Error("Error disconnecting from MongoDB", zap.Error(err))
		}
	}

	// Create MCP server
	impl := &mcp.Implementation{
		Name:    "code-indexing-mcp",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasTools: true,
	}

	server := mcp.NewServer(impl, opts)

	// Initialize and register tool handlers with file watcher
	toolHandler := handlers.NewToolHandler(mongoStorage, qdrantClient, embeddingClient, logger)
	toolHandler.SetFileWatcher(fileWatcher)
	if err := toolHandler.RegisterToolHandlers(server); err != nil {
		logger.Fatal("Failed to register tool handlers", zap.Error(err))
	}

	logger.Info("All handlers registered successfully", zap.Int("tools", 5))

	// Start MCP server with stdio transport in a goroutine
	logger.Info("Starting MCP server with stdio transport")

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	serverErr := make(chan error, 1)
	go func() {
		transport := &mcp.StdioTransport{}
		if err := server.Run(serverCtx, transport); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
		cleanup()
	case err := <-serverErr:
		logger.Error("Server error", zap.Error(err))
		cleanup()
	}

	logger.Info("Server shutdown complete")
}
