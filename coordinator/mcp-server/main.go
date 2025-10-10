package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"hyperion-coordinator-mcp/embeddings"
	"hyperion-coordinator-mcp/handlers"
	"hyperion-coordinator-mcp/storage"
	"hyperion-coordinator-mcp/watcher"

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
		logger.Fatal("MONGODB_URI environment variable is required")
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "coordinator_db1"
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

	// Initialize Qdrant client first (needed for knowledge storage)
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://qdrant:6333"
	}
	qdrantClient := storage.NewQdrantClient(qdrantURL)
	logger.Info("Qdrant client initialized", zap.String("url", qdrantURL))

	// Initialize storage with MongoDB
	taskStorage, err := storage.NewMongoTaskStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize task storage", zap.Error(err))
	}
	logger.Info("Task storage initialized with MongoDB")

	// Initialize knowledge storage with MongoDB + Qdrant
	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize knowledge storage", zap.Error(err))
	}
	logger.Info("Knowledge storage initialized with MongoDB + Qdrant vector search")

	// Create MCP server with capabilities
	impl := &mcp.Implementation{
		Name:    "hyperion-coordinator-mcp",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasResources: true,
		HasTools:     true,
		HasPrompts:   true,
	}

	server := mcp.NewServer(impl, opts)

	// Initialize code indexing components
	logger.Info("Initializing code indexing components")

	// Initialize code index storage
	codeIndexStorage, err := storage.NewCodeIndexStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize code index storage", zap.Error(err))
	}
	logger.Info("Code index storage initialized with MongoDB")

	// Ensure Qdrant code index collection exists
	if err := qdrantClient.EnsureCodeIndexCollection(); err != nil {
		logger.Fatal("Failed to ensure code index collection in Qdrant", zap.Error(err))
	}
	logger.Info("Qdrant code index collection ensured")

	// Initialize OpenAI embedding client
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		logger.Warn("OPENAI_API_KEY not set - code indexing features will be limited")
	}
	embeddingClient := embeddings.NewOpenAIClient(openAIKey)
	logger.Info("OpenAI embedding client initialized")

	// Initialize file watcher
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, logger)
	if err != nil {
		logger.Fatal("Failed to create file watcher", zap.Error(err))
	}
	logger.Info("File watcher initialized")

	// Load existing folders into file watcher
	folders, err := codeIndexStorage.ListFolders()
	if err != nil {
		logger.Warn("Failed to load existing folders for file watcher", zap.Error(err))
	} else {
		for _, folder := range folders {
			if folder.Status == "active" {
				if err := fileWatcher.AddFolder(folder); err != nil {
					logger.Warn("Failed to add folder to file watcher",
						zap.String("path", folder.Path),
						zap.Error(err))
				}
			}
		}
		logger.Info("Loaded existing folders into file watcher", zap.Int("count", len(folders)))
	}

	// Start file watcher
	if err := fileWatcher.Start(); err != nil {
		logger.Warn("Failed to start file watcher", zap.Error(err))
	} else {
		logger.Info("File watcher started successfully")
	}

	// Initialize handlers
	resourceHandler := handlers.NewResourceHandler(taskStorage, knowledgeStorage)
	docResourceHandler := handlers.NewDocResourceHandler()
	workflowResourceHandler := handlers.NewWorkflowResourceHandler(taskStorage)
	knowledgeResourceHandler := handlers.NewKnowledgeResourceHandler(knowledgeStorage)
	metricsResourceHandler := handlers.NewMetricsResourceHandler(taskStorage)
	toolHandler := handlers.NewToolHandler(taskStorage, knowledgeStorage)
	qdrantToolHandler := handlers.NewQdrantToolHandler(qdrantClient)
	codeToolsHandler := handlers.NewCodeToolsHandler(codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, logger)
	planningPromptHandler := handlers.NewPlanningPromptHandler()
	knowledgePromptHandler := handlers.NewKnowledgePromptHandler()
	coordinationPromptHandler := handlers.NewCoordinationPromptHandler()
	documentationPromptHandler := handlers.NewDocumentationPromptHandler()
	healthCheckHandler := handlers.NewHealthCheckHandler(mongoClient, qdrantClient, logger)

	// Register resource handlers
	if err := resourceHandler.RegisterResourceHandlers(server); err != nil {
		logger.Fatal("Failed to register resource handlers", zap.Error(err))
	}

	// Register documentation resource handlers
	if err := docResourceHandler.RegisterDocResources(server); err != nil {
		logger.Fatal("Failed to register documentation resource handlers", zap.Error(err))
	}

	// Register workflow resource handlers
	if err := workflowResourceHandler.RegisterWorkflowResources(server); err != nil {
		logger.Fatal("Failed to register workflow resource handlers", zap.Error(err))
	}

	// Register knowledge resource handlers
	if err := knowledgeResourceHandler.RegisterKnowledgeResources(server); err != nil {
		logger.Fatal("Failed to register knowledge resource handlers", zap.Error(err))
	}

	// Register metrics resource handlers
	if err := metricsResourceHandler.RegisterMetricsResources(server); err != nil {
		logger.Fatal("Failed to register metrics resource handlers", zap.Error(err))
	}

	// Register tool handlers
	if err := toolHandler.RegisterToolHandlers(server); err != nil {
		logger.Fatal("Failed to register tool handlers", zap.Error(err))
	}

	// Register Qdrant tool handlers
	if err := qdrantToolHandler.RegisterQdrantTools(server); err != nil {
		logger.Fatal("Failed to register Qdrant tool handlers", zap.Error(err))
	}

	// Register code indexing tool handlers
	if err := codeToolsHandler.RegisterCodeIndexTools(server); err != nil {
		logger.Fatal("Failed to register code indexing tool handlers", zap.Error(err))
	}

	// Register planning prompts
	if err := planningPromptHandler.RegisterPlanningPrompts(server); err != nil {
		logger.Fatal("Failed to register planning prompts", zap.Error(err))
	}

	// Register knowledge management prompts
	if err := knowledgePromptHandler.RegisterKnowledgePrompts(server); err != nil {
		logger.Fatal("Failed to register knowledge prompts", zap.Error(err))
	}

	// Register coordination prompts
	if err := coordinationPromptHandler.RegisterCoordinationPrompts(server); err != nil {
		logger.Fatal("Failed to register coordination prompts", zap.Error(err))
	}

	// Register documentation prompts
	if err := documentationPromptHandler.RegisterDocumentationPrompts(server); err != nil {
		logger.Fatal("Failed to register documentation prompts", zap.Error(err))
	}

	logger.Info("All handlers registered successfully",
		zap.Int("tools", 24), // 17 coordinator + 2 qdrant + 5 code indexing
		zap.Int("resources", 12), // 2 task + 3 doc + 3 workflow + 2 knowledge + 2 metrics
		zap.Int("prompts", 7))    // 2 planning + 2 knowledge + 2 coordination + 1 documentation

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

		// Add comprehensive health check endpoint
		mux.Handle("/health", healthCheckHandler)

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
