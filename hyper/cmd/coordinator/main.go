package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"hyper/embed"
	"hyper/internal/server"
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

// ensureCodeIndexCollectionWithDimensions ensures the code index collection exists with the correct dimensions
// If a dimension mismatch is detected, prompts the user to recreate the collection
func ensureCodeIndexCollectionWithDimensions(qdrantClient *storage.QdrantClient, expectedDimensions int, logger *zap.Logger) error {
	// Try to create the collection with dimension check
	err := qdrantClient.EnsureCodeIndexCollection(expectedDimensions)
	if err == nil {
		logger.Info("Code index collection ready",
			zap.String("collection", storage.CodeIndexCollection),
			zap.Int("dimensions", expectedDimensions))
		return nil
	}

	// Check if it's a dimension mismatch error
	var dimErr *storage.DimensionMismatchError
	if !errors.As(err, &dimErr) {
		// Not a dimension mismatch, return the error
		return err
	}

	// Dimension mismatch detected - prompt user or auto-recreate
	logger.Warn("Vector dimension mismatch detected",
		zap.String("collection", dimErr.Collection),
		zap.Int("expected", dimErr.ExpectedDim),
		zap.Int("got", expectedDimensions))

	// Check if auto-recreate is enabled via environment variable
	autoRecreate := os.Getenv("CODE_INDEX_AUTO_RECREATE")
	if autoRecreate == "true" {
		logger.Info("CODE_INDEX_AUTO_RECREATE=true, automatically recreating collection")
	} else {
		// Prompt user for confirmation
		fmt.Printf("\n")
		fmt.Printf("⚠️  Vector Dimension Mismatch Detected\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Collection:      %s\n", dimErr.Collection)
		fmt.Printf("Current dims:    %d (in Qdrant)\n", dimErr.ExpectedDim)
		fmt.Printf("Expected dims:   %d (from %s)\n", expectedDimensions, os.Getenv("OLLAMA_MODEL"))
		fmt.Printf("\n")
		fmt.Printf("This usually happens when you switch embedding models.\n")
		fmt.Printf("\n")
		fmt.Printf("⚠️  WARNING: Recreating will DELETE ALL indexed code!\n")
		fmt.Printf("You will need to re-scan your folders after recreation.\n")
		fmt.Printf("\n")
		fmt.Printf("Do you want to recreate the collection? (yes/no): ")

		// Read user input
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" && response != "y" {
			return fmt.Errorf("user declined to recreate collection - cannot proceed with dimension mismatch")
		}
	}

	// User agreed - recreate the collection
	logger.Info("Recreating code index collection", zap.Int("newDimensions", expectedDimensions))
	if err := qdrantClient.RecreateCodeIndexCollection(expectedDimensions); err != nil {
		return fmt.Errorf("failed to recreate collection: %w", err)
	}

	fmt.Printf("\n")
	fmt.Printf("✅ Collection recreated successfully with %d dimensions\n", expectedDimensions)
	fmt.Printf("🔄 You can now re-scan your code folders\n")
	fmt.Printf("\n")

	logger.Info("Code index collection recreated",
		zap.String("collection", storage.CodeIndexCollection),
		zap.Int("dimensions", expectedDimensions))

	return nil
}

func main() {
	// Parse command-line flags
	mode := flag.String("mode", "both", "Server mode: http, mcp, or both")
	configPath := flag.String("config", "", "Path to config file (default: .env.hyper in executable or current dir)")
	flag.Parse()

	// Load .env.hyper file if it exists (prefer over system env vars)
	// This allows native binary to have its own configuration without affecting system

	// If custom config path provided, use it exclusively
	if *configPath != "" {
		if err := godotenv.Overload(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed to load config from custom path: %s\n", *configPath)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Loaded configuration from custom path: %s\n", *configPath)
	} else {
		// Default behavior: try executable dir, then current dir
		executable, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(executable)
			envFile := filepath.Join(execDir, ".env.hyper")

			// Try to load .env.hyper from executable directory
			if err := godotenv.Overload(envFile); err == nil {
				fmt.Printf("✓ Loaded configuration from: %s\n", envFile)
			} else {
				fmt.Printf("Debug: Failed to load %s: %v\n", envFile, err)
				// Also try current working directory
				if err := godotenv.Overload(".env.hyper"); err == nil {
					fmt.Println("✓ Loaded configuration from: ./.env.hyper")
				} else {
					fmt.Printf("Debug: Failed to load ./.env.hyper: %v\n", err)
					// Debug: Show why loading failed
					fmt.Printf("Warning: No .env.hyper found (checked: %s and ./.env.hyper)\n", envFile)
				}
			}
		}
	}

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

	// Initialize Qdrant collection name from environment (must be done before creating qdrant client)
	storage.InitCodeIndexCollection()

	// Get Qdrant configuration
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://qdrant:6333"
	}

	// Get Qdrant knowledge collection name from environment
	qdrantKnowledgeCollection := os.Getenv("QDRANT_KNOWLEDGE_COLLECTION")
	if qdrantKnowledgeCollection == "" {
		qdrantKnowledgeCollection = "dev_squad_knowledge"
	}

	// Initialize embedding client based on EMBEDDING environment variable
	// IMPORTANT: This must be created BEFORE qdrantClient to ensure correct embeddings are used
	var embeddingClient embeddings.EmbeddingClient
	embeddingMode := os.Getenv("EMBEDDING")
	if embeddingMode == "" {
		embeddingMode = "ollama" // Default to Ollama (GPU-accelerated llama.cpp as a service)
	}

	logger.Info("Initializing embedding client", zap.String("mode", embeddingMode))

	switch embeddingMode {
	case "ollama":
		// Use Ollama (default - GPU-accelerated llama.cpp as a service)
		// Requires: brew install ollama && ollama pull nomic-embed-text
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
			logger.Fatal("Failed to initialize Ollama embedding client",
				zap.Error(err),
				zap.String("url", ollamaURL),
				zap.String("model", ollamaModel),
				zap.String("hint", "Install: brew install ollama && ollama pull <model> && brew services start ollama"))
		}
		logger.Info("Using Ollama embeddings (GPU-accelerated via llama.cpp)",
			zap.String("url", ollamaURL),
			zap.String("model", ollamaModel),
			zap.Int("dimensions", embeddingClient.GetDimensions()),
			zap.String("backend", "Metal/CUDA/Vulkan (auto-detected by Ollama)"))

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

		// Allow optional model override via VOYAGE_MODEL env var
		voyageModel := os.Getenv("VOYAGE_MODEL")
		if voyageModel != "" {
			embeddingClient = embeddings.NewVoyageClientWithModel(voyageKey, voyageModel)
			logger.Info("Using Voyage AI embedding service",
				zap.String("model", voyageModel),
				zap.Int("dimensions", embeddingClient.GetDimensions()))
		} else {
			embeddingClient = embeddings.NewVoyageClient(voyageKey)
			logger.Info("Using Voyage AI embedding service",
				zap.String("model", "voyage-3"),
				zap.Int("dimensions", 1024),
				zap.String("pricing", "$0.06/1M tokens"))
		}

	default:
		logger.Fatal("Invalid EMBEDDING mode. Use 'ollama' (default), 'llama', 'local', 'openai', or 'voyage'",
			zap.String("mode", embeddingMode))
	}

	// Now create Qdrant client with the correct embedding client
	qdrantClient := storage.NewQdrantClientWithEmbeddingClient(qdrantURL, qdrantKnowledgeCollection, embeddingClient)
	logger.Info("Qdrant client initialized with embedding client",
		zap.String("url", qdrantURL),
		zap.String("knowledgeCollection", qdrantKnowledgeCollection),
		zap.Int("vectorDimensions", embeddingClient.GetDimensions()))

	// Initialize storage layers (NOW that qdrantClient is created with correct embeddings)
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

	// Initialize tools storage for tools discovery (with correct embedding client!)
	toolsStorage, err := storage.NewToolsStorage(db, qdrantClient)
	if err != nil {
		logger.Fatal("Failed to initialize tools storage", zap.Error(err))
	}
	logger.Info("Tools storage initialized",
		zap.String("embeddingMode", embeddingMode),
		zap.Int("vectorDimensions", embeddingClient.GetDimensions()))

	logger.Info("Code index collection configured", zap.String("collection", storage.CodeIndexCollection))

	// Ensure Qdrant code index collection exists with correct dimensions
	expectedDimensions := embeddingClient.GetDimensions()
	if err := ensureCodeIndexCollectionWithDimensions(qdrantClient, expectedDimensions, logger); err != nil {
		logger.Fatal("Failed to ensure code index collection", zap.Error(err))
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
	mcpServer := createMCPServer(taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mongoClient, toolsStorage, logger)

	// Check for embedded UI (production single-binary mode)
	hasEmbedded := embed.HasUI()
	var embeddedFS http.FileSystem
	if hasEmbedded {
		var err error
		embeddedFS, err = embed.GetUIFileSystem()
		if err != nil {
			logger.Warn("Failed to load embedded UI, will use filesystem", zap.Error(err))
			hasEmbedded = false
		} else {
			logger.Info("Embedded UI detected (single-binary mode)")
		}
	} else {
		logger.Info("No embedded UI detected (development mode - will serve from filesystem)")
	}

	// Handle graceful shutdown (cross-platform signal handling)
	ctx, stop := setupSignalHandler()
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
			if err := server.StartHTTPServer(ctx, httpPort, taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mcpServer, embeddedFS, hasEmbedded, logger, db); err != nil {
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
			if err := server.StartHTTPServer(ctx, httpPort, taskStorage, knowledgeStorage, codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, mcpServer, embeddedFS, hasEmbedded, logger, db); err != nil {
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
	toolsStorage *storage.ToolsStorage,
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

	// Get the database from mongoClient
	mongoDB := mongoClient.Database(os.Getenv("MONGODB_DATABASE"))
	if mongoDB == nil {
		mongoDB = mongoClient.Database("coordinator_db1")
	}

	// Create tool metadata registry for automatic tool indexing
	toolMetadataRegistry := handlers.NewToolMetadataRegistry()

	// Initialize and register all handlers
	resourceHandler := handlers.NewResourceHandler(taskStorage, knowledgeStorage)
	docResourceHandler := handlers.NewDocResourceHandler()
	workflowResourceHandler := handlers.NewWorkflowResourceHandler(taskStorage)
	knowledgeResourceHandler := handlers.NewKnowledgeResourceHandler(knowledgeStorage)
	metricsResourceHandler := handlers.NewMetricsResourceHandler(taskStorage)
	toolHandler := handlers.NewToolHandler(taskStorage, knowledgeStorage, mongoDB)
	qdrantToolHandler := handlers.NewQdrantToolHandler(qdrantClient)
	codeToolsHandler := handlers.NewCodeToolsHandler(codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, logger)
	planningPromptHandler := handlers.NewPlanningPromptHandler()
	knowledgePromptHandler := handlers.NewKnowledgePromptHandler()
	coordinationPromptHandler := handlers.NewCoordinationPromptHandler()
	documentationPromptHandler := handlers.NewDocumentationPromptHandler()
	filesystemToolHandler := handlers.NewFilesystemToolHandler(logger)
	toolsDiscoveryHandler := handlers.NewToolsDiscoveryHandler(toolsStorage, server)

	// Set metadata registry on all tool handlers for automatic indexing
	toolHandler.SetMetadataRegistry(toolMetadataRegistry)
	qdrantToolHandler.SetMetadataRegistry(toolMetadataRegistry)
	filesystemToolHandler.SetMetadataRegistry(toolMetadataRegistry)
	codeToolsHandler.SetMetadataRegistry(toolMetadataRegistry)
	toolsDiscoveryHandler.SetMetadataRegistry(toolMetadataRegistry)

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
	must(filesystemToolHandler.RegisterFilesystemTools(server))
	must(toolsDiscoveryHandler.RegisterToolsDiscoveryTools(server))
	must(planningPromptHandler.RegisterPlanningPrompts(server))
	must(knowledgePromptHandler.RegisterKnowledgePrompts(server))
	must(coordinationPromptHandler.RegisterCoordinationPrompts(server))
	must(documentationPromptHandler.RegisterDocumentationPrompts(server))

	logger.Info("MCP server configured with all handlers")

	// Index all MCP tools for discovery via discover_tools (using automatic registry)
	logger.Info("Indexing MCP tools for semantic discovery...")
	if count, err := handlers.IndexRegisteredTools(toolMetadataRegistry, toolsStorage, logger); err != nil {
		logger.Warn("Failed to index MCP tools (tools may not be discoverable)", zap.Error(err))
	} else {
		logger.Info("MCP tools indexed successfully",
			zap.Int("count", count),
			zap.String("collection", "mcp-tools"))
	}

	return server
}
