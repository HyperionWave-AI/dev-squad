package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"hyperion-coordinator-mcp/embeddings"
	"hyperion-coordinator-mcp/handlers"
	"hyperion-coordinator-mcp/storage"
	"hyperion-coordinator-mcp/watcher"

	"github.com/joho/godotenv"
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

	// Load environment variables from .env.hyper if it exists in current working directory
	// This allows the service to work from any directory
	if err := godotenv.Load(".env.hyper"); err != nil {
		// Not finding .env.hyper is not an error - variables may be set externally
		logger.Info("No .env.hyper file found in current directory (environment variables may be set externally)")
	} else {
		// Get current working directory for logging
		cwd, _ := os.Getwd()
		logger.Info("Loaded environment from .env.hyper",
			zap.String("workingDirectory", cwd))
	}

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

	// Get knowledge collection name from environment
	knowledgeCollection := os.Getenv("QDRANT_KNOWLEDGE_COLLECTION")
	if knowledgeCollection == "" {
		knowledgeCollection = "dev_squad_knowledge"
	}

	qdrantClient := storage.NewQdrantClient(qdrantURL, knowledgeCollection)
	logger.Info("Qdrant client initialized",
		zap.String("url", qdrantURL),
		zap.String("knowledgeCollection", knowledgeCollection))

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

	// Initialize tools storage with MongoDB + Qdrant
	toolsStorage, err := storage.NewToolsStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize tools storage", zap.Error(err))
	}
	logger.Info("Tools storage initialized with MongoDB + Qdrant")

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

	case "voyage":
		// Use Voyage AI embeddings (Anthropic's recommended provider)
		voyageKey := os.Getenv("VOYAGE_API_KEY")
		if voyageKey == "" {
			logger.Fatal("VOYAGE_API_KEY is required when EMBEDDING=voyage")
		}
		embeddingClient = embeddings.NewVoyageClient(voyageKey)
		logger.Info("Using Voyage AI embedding service",
			zap.String("model", "voyage-3"),
			zap.Int("dimensions", 1024))

	case "ollama":
		// Use Ollama embeddings (local llama.cpp with GPU acceleration via REST API)
		ollamaURL := os.Getenv("OLLAMA_URL")
		if ollamaURL == "" {
			ollamaURL = "http://localhost:11434"
		}
		ollamaModel := os.Getenv("OLLAMA_MODEL")
		if ollamaModel == "" {
			ollamaModel = "nomic-embed-text"
		}
		var err error
		embeddingClient, err = embeddings.NewOllamaClient(ollamaURL, ollamaModel)
		if err != nil {
			logger.Fatal("Failed to initialize Ollama client", zap.Error(err))
		}
		logger.Info("Using Ollama embedding service",
			zap.String("url", ollamaURL),
			zap.String("model", ollamaModel),
			zap.Int("dimensions", embeddingClient.GetDimensions()))

	default:
		logger.Fatal("Invalid EMBEDDING mode. Use 'local', 'openai', 'voyage', or 'ollama'",
			zap.String("mode", embeddingMode))
	}

	// Initialize path mapper for Docker volume mapping
	pathMappingsEnv := os.Getenv("CODE_INDEX_PATH_MAPPINGS")
	pathMapper := watcher.NewPathMapper(pathMappingsEnv, logger)
	if pathMapper.HasMappings() {
		logger.Info("Path mapper configured",
			zap.Int("mappings", len(pathMapper.GetMappings())))
		for host, container := range pathMapper.GetMappings() {
			logger.Info("Path mapping",
				zap.String("host", host),
				zap.String("container", container))
		}
	} else {
		logger.Info("No path mappings configured - running on host")
	}

	// Initialize file watcher (always enabled)
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
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

	// Auto-register folders from CODE_INDEX_FOLDERS environment variable
	codeIndexFolders := os.Getenv("CODE_INDEX_FOLDERS")
	if codeIndexFolders != "" {
		folderPaths := strings.Split(codeIndexFolders, ",")
		logger.Info("Auto-registering folders from CODE_INDEX_FOLDERS", zap.Int("count", len(folderPaths)))

		for _, folderPath := range folderPaths {
			folderPath = strings.TrimSpace(folderPath)
			if folderPath == "" {
				continue
			}

			// Check if folder already exists
			existingFolder, err := codeIndexStorage.GetFolderByPath(folderPath)
			if err != nil {
				logger.Warn("Failed to check existing folder",
					zap.String("path", folderPath),
					zap.Error(err))
				continue
			}

			if existingFolder != nil {
				logger.Info("Folder already registered, skipping",
					zap.String("path", folderPath))
				continue
			}

			// Add new folder
			newFolder, err := codeIndexStorage.AddFolder(folderPath, "Auto-registered from CODE_INDEX_FOLDERS")
			if err != nil {
				logger.Warn("Failed to auto-register folder",
					zap.String("path", folderPath),
					zap.Error(err))
				continue
			}

			// Add to file watcher
			if err := fileWatcher.AddFolder(newFolder); err != nil {
				logger.Warn("Failed to add auto-registered folder to file watcher",
					zap.String("path", folderPath),
					zap.Error(err))
			}

			logger.Info("Auto-registered folder",
				zap.String("path", folderPath),
				zap.String("folderID", newFolder.ID))

			// Trigger initial scan if AUTO_SCAN is enabled
			if os.Getenv("CODE_INDEX_AUTO_SCAN") != "false" {
				go func(folder *storage.IndexedFolder) {
					logger.Info("Starting initial scan for auto-registered folder",
						zap.String("path", folder.Path),
						zap.String("folderID", folder.ID))

					if err := fileWatcher.ScanFolder(folder); err != nil {
						logger.Warn("Failed to scan auto-registered folder",
							zap.String("path", folder.Path),
							zap.Error(err))
					} else {
						logger.Info("Completed initial scan",
							zap.String("path", folder.Path),
							zap.String("folderID", folder.ID))
					}
				}(newFolder)
			}
		}
	}

	// Trigger initial scan for existing folders if AUTO_SCAN is enabled
	if os.Getenv("CODE_INDEX_AUTO_SCAN") != "false" {
		existingFolders, err := codeIndexStorage.ListFolders()
		if err == nil {
			for _, folder := range existingFolders {
				if folder.FileCount == 0 && folder.Status == "active" {
					go func(f *storage.IndexedFolder) {
						logger.Info("Starting initial scan for existing folder",
							zap.String("path", f.Path),
							zap.String("folderID", f.ID))

						if err := fileWatcher.ScanFolder(f); err != nil {
							logger.Warn("Failed to scan existing folder",
								zap.String("path", f.Path),
								zap.Error(err))
						} else {
							logger.Info("Completed initial scan",
								zap.String("path", f.Path),
								zap.String("folderID", f.ID))
						}
					}(folder)
				}
			}
		}
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
	filesystemToolsHandler := handlers.NewFilesystemToolHandler(logger)
	toolsDiscoveryHandler := handlers.NewToolsDiscoveryHandler(toolsStorage)
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

	// Register filesystem tool handlers
	if err := filesystemToolsHandler.RegisterFilesystemTools(server); err != nil {
		logger.Fatal("Failed to register filesystem tool handlers", zap.Error(err))
	}

	// Register tools discovery handlers
	if err := toolsDiscoveryHandler.RegisterToolsDiscoveryTools(server); err != nil {
		logger.Fatal("Failed to register tools discovery handlers", zap.Error(err))
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
		zap.Int("tools", 31), // 17 coordinator + 2 qdrant + 5 code indexing + 4 filesystem + 3 tools discovery
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

		// Add comprehensive health check endpoints
		mux.Handle("/health", healthCheckHandler)
		mux.HandleFunc("/health/qdrant", healthCheckHandler.ServeQdrantHealth)
		mux.HandleFunc("/health/ollama", healthCheckHandler.ServeOllamaHealth)

		httpServer := &http.Server{
			Addr:    fmt.Sprintf(":%s", mcpPort),
			Handler: mux,
		}

		logger.Info("HTTP server listening",
			zap.String("address", httpServer.Addr),
			zap.String("mcp_endpoint", "/mcp"),
			zap.String("health_endpoint", "/health"),
			zap.String("qdrant_health_endpoint", "/health/qdrant"),
			zap.String("ollama_health_endpoint", "/health/ollama"))

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
