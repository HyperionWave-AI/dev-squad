package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"hyperion-coordinator/internal/server"
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
	// Parse command-line flags
	mode := flag.String("mode", "both", "Server mode: http, mcp, or both")
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Unified Hyperion Coordinator",
		zap.String("mode", *mode))

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

	// Initialize Qdrant client
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://qdrant:6333"
	}
	qdrantClient := storage.NewQdrantClient(qdrantURL)
	logger.Info("Qdrant client initialized", zap.String("url", qdrantURL))

	// Initialize storage layers
	taskStorage, err := storage.NewMongoTaskStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize task storage", zap.Error(err))
	}
	logger.Info("Task storage initialized with MongoDB")

	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize knowledge storage", zap.Error(err))
	}
	logger.Info("Knowledge storage initialized with MongoDB + Qdrant")

	// Initialize code indexing components
	codeIndexStorage, err := storage.NewCodeIndexStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize code index storage", zap.Error(err))
	}
	logger.Info("Code index storage initialized")

	// Ensure Qdrant code index collection exists
	if err := qdrantClient.EnsureCodeIndexCollection(); err != nil {
		logger.Fatal("Failed to ensure code index collection in Qdrant", zap.Error(err))
	}

	// Initialize embedding client based on EMBEDDING environment variable
	var embeddingClient embeddings.EmbeddingClient
	embeddingMode := os.Getenv("EMBEDDING")
	if embeddingMode == "" {
		embeddingMode = "local" // Default to local TEI
	}

	switch embeddingMode {
	case "local":
		// Use local TEI service (Hugging Face Text Embeddings Inference)
		teiURL := os.Getenv("TEI_URL")
		if teiURL == "" {
			teiURL = "http://embedding-service:8080" // Default TEI URL
		}
		embeddingClient = embeddings.NewTEIClient(teiURL)
		logger.Info("Using local TEI embedding service",
			zap.String("url", teiURL),
			zap.String("model", "nomic-ai/nomic-embed-text-v1.5"),
			zap.Int("dimensions", 768))

	case "openai":
		// Use OpenAI embeddings
		openAIKey := os.Getenv("OPENAI_API_KEY")
		if openAIKey == "" {
			logger.Fatal("OPENAI_API_KEY is required when EMBEDDING=openai")
		}
		embeddingClient = embeddings.NewOpenAIClient(openAIKey)
		logger.Info("Using OpenAI embedding service",
			zap.String("model", "text-embedding-3-small"),
			zap.Int("dimensions", 1536))

	default:
		logger.Fatal("Invalid EMBEDDING mode. Use 'local' or 'openai'",
			zap.String("mode", embeddingMode))
	}

	// Initialize path mapper for Docker volume mapping
	pathMappingsEnv := os.Getenv("CODE_INDEX_PATH_MAPPINGS")
	pathMapper := watcher.NewPathMapper(pathMappingsEnv, logger)

	// Initialize file watcher
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
	if err != nil {
		logger.Fatal("Failed to create file watcher", zap.Error(err))
	}

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
	}

	// Auto-register folders from CODE_INDEX_FOLDERS environment variable
	codeIndexFolders := os.Getenv("CODE_INDEX_FOLDERS")
	if codeIndexFolders != "" {
		folderPaths := strings.Split(codeIndexFolders, ",")
		for _, folderPath := range folderPaths {
			folderPath = strings.TrimSpace(folderPath)
			if folderPath == "" {
				continue
			}

			existingFolder, err := codeIndexStorage.GetFolderByPath(folderPath)
			if err != nil || existingFolder != nil {
				continue
			}

			newFolder, err := codeIndexStorage.AddFolder(folderPath, "Auto-registered")
			if err != nil {
				logger.Warn("Failed to auto-register folder", zap.String("path", folderPath), zap.Error(err))
				continue
			}

			if err := fileWatcher.AddFolder(newFolder); err != nil {
				logger.Warn("Failed to add auto-registered folder to file watcher", zap.Error(err))
			}

			// Trigger initial scan if AUTO_SCAN is enabled
			if os.Getenv("CODE_INDEX_AUTO_SCAN") != "false" {
				go func(folder *storage.IndexedFolder) {
					if err := fileWatcher.ScanFolder(folder); err != nil {
						logger.Warn("Failed to scan auto-registered folder", zap.Error(err))
					}
				}(newFolder)
			}
		}
	}

	// Start file watcher
	if err := fileWatcher.Start(); err != nil {
		logger.Warn("Failed to start file watcher", zap.Error(err))
	} else {
		logger.Info("File watcher started successfully")
	}

	// Create MCP server instance (used by both HTTP and stdio modes)
	mcpServer := createMCPServer(taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mongoClient, logger)

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Start servers based on mode
	switch *mode {
	case "http":
		// HTTP mode only - REST API + UI serving
		httpPort := os.Getenv("HTTP_PORT")
		if httpPort == "" {
			httpPort = "7095"
		}
		logger.Info("Starting in HTTP-only mode", zap.String("port", httpPort))

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.StartHTTPServer(ctx, httpPort, taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mcpServer, logger); err != nil {
				logger.Fatal("HTTP server error", zap.Error(err))
			}
		}()

	case "mcp":
		// MCP mode only - stdio protocol
		logger.Info("Starting in MCP-only mode (stdio)")

		transport := &mcp.StdioTransport{}
		if err := mcpServer.Run(ctx, transport); err != nil {
			logger.Fatal("MCP server error", zap.Error(err))
		}

	case "both":
		// Both HTTP and MCP
		httpPort := os.Getenv("HTTP_PORT")
		if httpPort == "" {
			httpPort = "7095"
		}

		logger.Info("Starting in dual mode", zap.String("httpPort", httpPort))

		// Start HTTP server in goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.StartHTTPServer(ctx, httpPort, taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mcpServer, logger); err != nil {
				logger.Error("HTTP server error", zap.Error(err))
			}
		}()

		// Start MCP server in goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			transport := &mcp.StdioTransport{}
			if err := mcpServer.Run(ctx, transport); err != nil {
				logger.Error("MCP server error", zap.Error(err))
			}
		}()

	default:
		logger.Fatal("Invalid mode. Use: http, mcp, or both", zap.String("mode", *mode))
	}

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("Shutdown signal received, stopping servers...")

	// Wait for all servers to stop
	wg.Wait()
	logger.Info("Server shutdown complete")
}

// createMCPServer creates and configures the MCP server with all handlers
func createMCPServer(
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	fileWatcher *watcher.FileWatcher,
	mongoClient *mongo.Client,
	logger *zap.Logger,
) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "hyperion-coordinator-unified",
		Version: "2.0.0",
	}

	opts := &mcp.ServerOptions{
		HasResources: true,
		HasTools:     true,
		HasPrompts:   true,
	}

	server := mcp.NewServer(impl, opts)

	// Initialize and register all handlers
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

	// Register all handlers (panic on error)
	must := func(err error) {
		if err != nil {
			logger.Fatal("Failed to register handlers", zap.Error(err))
		}
	}

	must(resourceHandler.RegisterResourceHandlers(server))
	must(docResourceHandler.RegisterDocResources(server))
	must(workflowResourceHandler.RegisterWorkflowResources(server))
	must(knowledgeResourceHandler.RegisterKnowledgeResources(server))
	must(metricsResourceHandler.RegisterMetricsResources(server))
	must(toolHandler.RegisterToolHandlers(server))
	must(qdrantToolHandler.RegisterQdrantTools(server))
	must(codeToolsHandler.RegisterCodeIndexTools(server))
	must(planningPromptHandler.RegisterPlanningPrompts(server))
	must(knowledgePromptHandler.RegisterKnowledgePrompts(server))
	must(coordinationPromptHandler.RegisterCoordinationPrompts(server))
	must(documentationPromptHandler.RegisterDocumentationPrompts(server))

	logger.Info("MCP server configured with all handlers")

	return server
}
