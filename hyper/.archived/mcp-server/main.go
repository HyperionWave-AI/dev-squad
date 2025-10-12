package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/watcher"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to config file (default: .env.hyper)")
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Hyperion MCP Server (unified)")

	// Load environment variables
	if *configPath != "" {
		// Load from custom config path
		if err := godotenv.Load(*configPath); err != nil {
			logger.Fatal("Failed to load config from custom path",
				zap.String("path", *configPath),
				zap.Error(err))
		}
		logger.Info("Loaded configuration from custom path", zap.String("path", *configPath))
	} else {
		// Load from default location
		if err := godotenv.Load(".env.hyper"); err != nil {
			logger.Warn("Could not load .env.hyper", zap.Error(err))
		}
	}

	// Get MongoDB configuration
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		logger.Fatal("MONGODB_URI environment variable is required")
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "coordinator_db1"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}

	db := mongoClient.Database(mongoDatabase)
	logger.Info("Connected to MongoDB", zap.String("database", mongoDatabase))

	// Initialize Qdrant
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://qdrant:6333"
	}

	knowledgeCollection := os.Getenv("QDRANT_KNOWLEDGE_COLLECTION")
	if knowledgeCollection == "" {
		knowledgeCollection = "dev_squad_knowledge"
	}

	qdrantClient := storage.NewQdrantClient(qdrantURL, knowledgeCollection)
	logger.Info("Qdrant client initialized", zap.String("url", qdrantURL))

	// Initialize storage layers
	taskStorage, err := storage.NewMongoTaskStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize task storage", zap.Error(err))
	}

	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize knowledge storage", zap.Error(err))
	}

	toolsStorage, err := storage.NewToolsStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize tools storage", zap.Error(err))
	}

	codeIndexStorage, err := storage.NewCodeIndexStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize code index storage", zap.Error(err))
	}

	// Initialize embedding client
	var embeddingClient embeddings.EmbeddingClient
	embeddingMode := os.Getenv("EMBEDDING")
	if embeddingMode == "" {
		embeddingMode = "ollama"
	}

	switch embeddingMode {
	case "ollama":
		ollamaURL := os.Getenv("OLLAMA_URL")
		if ollamaURL == "" {
			ollamaURL = "http://localhost:11434"
		}
		ollamaModel := os.Getenv("OLLAMA_MODEL")
		if ollamaModel == "" {
			ollamaModel = "nomic-embed-text"
		}
		embeddingClient, err = embeddings.NewOllamaClient(ollamaURL, ollamaModel)
		if err != nil {
			logger.Fatal("Failed to initialize Ollama client", zap.Error(err))
		}
		logger.Info("Using Ollama embeddings", zap.String("model", ollamaModel))
	case "openai":
		openAIKey := os.Getenv("OPENAI_API_KEY")
		if openAIKey == "" {
			logger.Fatal("OPENAI_API_KEY is required when EMBEDDING=openai")
		}
		embeddingClient = embeddings.NewOpenAIClient(openAIKey)
		logger.Info("Using OpenAI embeddings")
	default:
		logger.Fatal("Invalid EMBEDDING mode", zap.String("mode", embeddingMode))
	}

	// Initialize file watcher
	pathMappingsEnv := os.Getenv("CODE_INDEX_PATH_MAPPINGS")
	pathMapper := watcher.NewPathMapper(pathMappingsEnv, logger)
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
	if err != nil {
		logger.Fatal("Failed to create file watcher", zap.Error(err))
	}

	// Load existing folders
	folders, err := codeIndexStorage.ListFolders()
	if err == nil {
		for _, folder := range folders {
			if folder.Status == "active" {
				fileWatcher.AddFolder(folder)
			}
		}
	}

	// Auto-register folders
	codeIndexFolders := os.Getenv("CODE_INDEX_FOLDERS")
	if codeIndexFolders != "" {
		for _, folderPath := range strings.Split(codeIndexFolders, ",") {
			folderPath = strings.TrimSpace(folderPath)
			if folderPath == "" {
				continue
			}
			if existing, _ := codeIndexStorage.GetFolderByPath(folderPath); existing != nil {
				continue
			}
			if newFolder, err := codeIndexStorage.AddFolder(folderPath, "Auto-registered"); err == nil {
				fileWatcher.AddFolder(newFolder)
			}
		}
	}

	fileWatcher.Start()
	logger.Info("File watcher started")

	// Create MCP server
	impl := &mcp.Implementation{
		Name:    "hyper-mcp-server",
		Version: "2.0.0",
	}

	server := mcp.NewServer(impl, &mcp.ServerOptions{
		HasResources: true,
		HasTools:     true,
		HasPrompts:   true,
	})

	// Register all handlers
	handlers.NewResourceHandler(taskStorage, knowledgeStorage).RegisterResourceHandlers(server)
	handlers.NewDocResourceHandler().RegisterDocResources(server)
	handlers.NewWorkflowResourceHandler(taskStorage).RegisterWorkflowResources(server)
	handlers.NewKnowledgeResourceHandler(knowledgeStorage).RegisterKnowledgeResources(server)
	handlers.NewMetricsResourceHandler(taskStorage).RegisterMetricsResources(server)
	handlers.NewToolHandler(taskStorage, knowledgeStorage).RegisterToolHandlers(server)
	handlers.NewQdrantToolHandler(qdrantClient).RegisterQdrantTools(server)
	handlers.NewCodeToolsHandler(codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, logger).RegisterCodeIndexTools(server)
	handlers.NewFilesystemToolHandler(logger).RegisterFilesystemTools(server)
	handlers.NewToolsDiscoveryHandler(toolsStorage).RegisterToolsDiscoveryTools(server)
	handlers.NewPlanningPromptHandler().RegisterPlanningPrompts(server)
	handlers.NewKnowledgePromptHandler().RegisterKnowledgePrompts(server)
	handlers.NewCoordinationPromptHandler().RegisterCoordinationPrompts(server)
	handlers.NewDocumentationPromptHandler().RegisterDocumentationPrompts(server)

	logger.Info("All handlers registered successfully")

	// Start server
	transportMode := os.Getenv("TRANSPORT_MODE")
	if transportMode == "http" {
		mcpPort := os.Getenv("MCP_PORT")
		if mcpPort == "" {
			mcpPort = "7778"
		}
		logger.Info("Starting HTTP server", zap.String("port", mcpPort))

		handler := mcp.NewStreamableHTTPHandler(
			func(req *http.Request) *mcp.Server { return server },
			&mcp.StreamableHTTPOptions{Stateless: false, JSONResponse: true},
		)

		mux := http.NewServeMux()
		mux.Handle("/mcp", handler)
		mux.Handle("/health", handlers.NewHealthCheckHandler(mongoClient, qdrantClient, logger))

		http.ListenAndServe(fmt.Sprintf(":%s", mcpPort), mux)
	} else {
		logger.Info("Starting stdio server")
		transport := &mcp.StdioTransport{}
		server.Run(context.Background(), transport)
	}
}
