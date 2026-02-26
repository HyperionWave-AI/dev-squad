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
	"hyper/internal/ai-service/tools"
	hyperinit "hyper/internal/init"
	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/handlers"
	"hyper/internal/mcp/indexer"
	"hyper/internal/mcp/parser"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/summarizer"
	"hyper/internal/mcp/watcher"
	"hyper/internal/server"
	"hyper/internal/validation"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Build information set via ldflags at build time
var (
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func isFileWatcherDisabledByEnv() bool {
	raw := strings.TrimSpace(os.Getenv("ENABLE_FILE_WATCHER"))
	raw = strings.Trim(raw, "\"'")
	return strings.EqualFold(raw, "false")
}

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
		// Prompt user for confirmation - log the dimension mismatch for audit trail
		logger.Warn("User interaction required for dimension mismatch",
			zap.String("collection", dimErr.Collection),
			zap.Int("currentDimensions", dimErr.ExpectedDim),
			zap.Int("expectedDimensions", expectedDimensions),
			zap.String("embeddingModel", os.Getenv("OLLAMA_MODEL")))

		// Display user prompt
		logger.Info("⚠️  Vector Dimension Mismatch Detected")
		logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Info("Dimension details",
			zap.String("collection", dimErr.Collection),
			zap.Int("currentDims", dimErr.ExpectedDim),
			zap.Int("expectedDims", expectedDimensions),
			zap.String("model", os.Getenv("OLLAMA_MODEL")))
		logger.Info("This usually happens when you switch embedding models")
		logger.Warn("⚠️  WARNING: Recreating will DELETE ALL indexed code!")
		logger.Info("You will need to re-scan your folders after recreation")
		logger.Info("Do you want to recreate the collection? (yes/no): ")

		// Read user input
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		logger.Info("User decision received", zap.String("response", response))

		if response != "yes" && response != "y" {
			logger.Warn("User declined collection recreation",
				zap.String("collection", dimErr.Collection))
			return fmt.Errorf("user declined to recreate collection - cannot proceed with dimension mismatch")
		}
		logger.Info("User approved collection recreation",
			zap.String("collection", dimErr.Collection))
	}

	// User agreed - recreate the collection
	logger.Info("Recreating code index collection", zap.Int("newDimensions", expectedDimensions))
	if err := qdrantClient.RecreateCodeIndexCollection(expectedDimensions); err != nil {
		return fmt.Errorf("failed to recreate collection: %w", err)
	}

	logger.Info("✅ Collection recreated successfully",
		zap.Int("dimensions", expectedDimensions))
	logger.Info("🔄 You can now re-scan your code folders")

	logger.Info("Code index collection recreated",
		zap.String("collection", storage.CodeIndexCollection),
		zap.Int("dimensions", expectedDimensions))

	return nil
}

// initLogger creates a logger that outputs to both console and file
func initLogger() (*zap.Logger, error) {
	// Ensure logs directory exists
	logsDir := "./logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate log filename with timestamp
	logFilePath := filepath.Join(logsDir, fmt.Sprintf("coordinator-%s.log", time.Now().Format("2006-01-02")))

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Configure encoder for console (colorized, human-readable)
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure encoder for file (JSON, machine-readable)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create cores
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(fileEncoderConfig),
		zapcore.AddSync(logFile),
		zapcore.InfoLevel, // Log Info and above to file
	)

	// Combine cores
	core := zapcore.NewTee(consoleCore, fileCore)

	// Create logger with caller information
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

func main() {
	// Initialize logger early for startup messages
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize project root detection
	if err := tools.InitProjectRoot(); err != nil {
		logger.Fatal("Failed to detect project root", zap.Error(err))
	}
	logger.Info("Project root detected", zap.String("path", tools.GetProjectRoot()))

	// Initialize AST parsers for code indexing
	parser.InitializeParsers()
	logger.Info("AST parsers initialized for Go, JavaScript/TypeScript, and Python")

	// Check for init command before parsing flags
	if len(os.Args) > 1 && os.Args[1] == "init" {
		// Parse init-specific flags
		initFlags := flag.NewFlagSet("init", flag.ExitOnError)
		provider := initFlags.String("provider", "", "AI provider (openai, litellm, anthropic, voyage, ollama)")
		model := initFlags.String("model", "", "AI model name (e.g., gpt-4o-mini, claude-sonnet-4, voyage-3)")
		token := initFlags.String("token", "", "API token/key (required for cloud providers)")
		apiURL := initFlags.String("api-url", "", "Custom API URL (optional)")

		// Parse remaining args (skip "init")
		if err := initFlags.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		// Create provider config
		var config *hyperinit.ProviderConfig
		if *provider != "" {
			config = &hyperinit.ProviderConfig{
				Provider: *provider,
				Model:    *model,
				Token:    *token,
				APIURL:   *apiURL,
			}
		}

		if err := hyperinit.Init(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Parse command-line flags
	mode := flag.String("mode", "both", "Server mode: http, mcp, or both")
	configPath := flag.String("config", "", "Path to config file (default: .env.hyper in executable or current dir)")
	flag.Parse()

	// Load .env.hyper file if it exists (prefer over system env vars)
	// This allows native binary to have its own configuration without affecting system

	// If custom config path provided, use it exclusively
	if *configPath != "" {
		if err := godotenv.Overload(*configPath); err != nil {
			logger.Fatal("Failed to load config from custom path",
				zap.String("path", *configPath),
				zap.Error(err))
		}
		logger.Info("Configuration loaded from custom path", zap.String("path", *configPath))
	} else {
		// Default behavior: try .env.hyper.hot first (hot-reload dev), then .env.hyper (standard)
		// Priority: exec dir .env.hyper.hot → exec dir .env.hyper → current dir .env.hyper.hot → current dir .env.hyper
		executable, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(executable)
			hotEnvFile := filepath.Join(execDir, ".env.hyper.hot")
			envFile := filepath.Join(execDir, ".env.hyper")

			// Try to load .env.hyper.hot from executable directory first
			if err := godotenv.Overload(hotEnvFile); err == nil {
				logger.Info("Configuration loaded", zap.String("path", hotEnvFile))
			} else {
				logger.Debug("Failed to load config from executable directory",
					zap.String("path", hotEnvFile),
					zap.Error(err))

				// Fallback to .env.hyper from executable directory
				if err := godotenv.Overload(envFile); err == nil {
					logger.Info("Configuration loaded", zap.String("path", envFile))
				} else {
					logger.Debug("Failed to load config from executable directory",
						zap.String("path", envFile),
						zap.Error(err))

					// Also try current working directory - hot first, then standard
					if err := godotenv.Overload(".env.hyper.hot"); err == nil {
						logger.Info("Configuration loaded", zap.String("path", "./.env.hyper.hot"))
					} else {
						logger.Debug("Failed to load config from current directory",
							zap.String("path", "./.env.hyper.hot"),
							zap.Error(err))

						// Final fallback to .env.hyper in current directory
						if err := godotenv.Overload(".env.hyper"); err == nil {
							logger.Info("Configuration loaded", zap.String("path", "./.env.hyper"))
						} else {
							logger.Debug("Failed to load config from current directory",
								zap.String("path", "./.env.hyper"),
								zap.Error(err))
							logger.Warn("No .env.hyper or .env.hyper.hot found",
								zap.Strings("checkedPaths", []string{hotEnvFile, envFile, "./.env.hyper.hot", "./.env.hyper"}))
						}
					}
				}
			}
		}
	}

	logger.Info("Starting Unified Hyperion Coordinator",
		zap.String("mode", *mode),
		zap.String("buildTime", BuildTime),
		zap.String("gitCommit", GitCommit))

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

		logger.Info("Initialized Ollama embedding client",
			zap.String("url", ollamaURL),
			zap.String("model", ollamaModel))

	case "openai":
		// Use OpenAI API
		openaiKey := os.Getenv("OPENAI_API_KEY")
		if openaiKey == "" {
			logger.Fatal("OPENAI_API_KEY environment variable is required when EMBEDDING=openai")
		}

		embeddingClient = embeddings.NewOpenAIClient(openaiKey)
		logger.Info("Initialized OpenAI embedding client")

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
		logger.Fatal("Unknown embedding mode",
			zap.String("mode", embeddingMode),
			zap.String("hint", "Set EMBEDDING=ollama, EMBEDDING=openai, or EMBEDDING=voyage"))
	}

	// Append dimension suffix to knowledge collection name
	// This allows seamless switching between embedding providers without data loss
	// Each provider's vectors are stored in a separate collection (e.g., dev_squad_knowledge_1024, dev_squad_knowledge_768)
	dimensions := embeddingClient.GetDimensions()
	qdrantKnowledgeCollectionWithDim := fmt.Sprintf("%s_%d", qdrantKnowledgeCollection, dimensions)

	// Initialize Qdrant client with the same embedding client used for code indexing
	// This ensures knowledge_store and knowledge_find use the configured embedding provider (Voyage/Ollama/OpenAI)
	logger.Info("Connecting to Qdrant",
		zap.String("url", qdrantURL),
		zap.String("baseCollection", qdrantKnowledgeCollection),
		zap.String("activeCollection", qdrantKnowledgeCollectionWithDim),
		zap.String("embeddingProvider", embeddingMode),
		zap.Int("dimensions", dimensions))

	qdrantClient := storage.NewQdrantClientWithEmbeddingClient(qdrantURL, qdrantKnowledgeCollectionWithDim, embeddingClient)

	logger.Info("Successfully connected to Qdrant - using dimension-specific collection",
		zap.String("collection", qdrantKnowledgeCollectionWithDim),
		zap.String("embeddingProvider", embeddingMode),
		zap.Int("dimensions", dimensions))

	// Initialize code index storage
	codeIndexStorage, err := storage.NewCodeIndexStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize code index storage", zap.Error(err))
	}

	// Ensure code index collection exists with correct dimensions
	embeddingDimensions := embeddingClient.GetDimensions()
	logger.Info("Embedding dimensions", zap.Int("dimensions", embeddingDimensions))

	// Set code index collection name with user ID and dimension suffix
	// This allows:
	// 1. User isolation - each user gets their own code index (if CODE_INDEX_USER_ID is set)
	// 2. Seamless provider switching - different embedding dimensions use separate collections
	// Pattern: {base}_{userID}_{dimensions} or {base}_{dimensions} if no user ID
	storage.SetCodeIndexCollectionWithDimensions(embeddingDimensions)

	if storage.CodeIndexUserID != "" {
		logger.Info("Code index collection configured with user isolation",
			zap.String("baseCollection", storage.CodeIndexCollectionBase),
			zap.String("userId", storage.CodeIndexUserID),
			zap.String("activeCollection", storage.CodeIndexCollection),
			zap.Int("dimensions", embeddingDimensions))
	} else {
		logger.Info("Code index collection configured (shared mode)",
			zap.String("baseCollection", storage.CodeIndexCollectionBase),
			zap.String("activeCollection", storage.CodeIndexCollection),
			zap.Int("dimensions", embeddingDimensions),
			zap.String("note", "Set CODE_INDEX_USER_ID env var for user isolation"))
	}

	if err := ensureCodeIndexCollectionWithDimensions(qdrantClient, embeddingDimensions, logger); err != nil {
		logger.Fatal("Failed to ensure code index collection", zap.Error(err))
	}

	// Initialize path mapper for file watcher
	pathMappingsEnv := os.Getenv("PATH_MAPPINGS")
	pathMapper := watcher.NewPathMapper(pathMappingsEnv, logger)

	// Initialize file watcher
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
	if err != nil {
		logger.Fatal("Failed to initialize file watcher", zap.Error(err))
	}

	// Start file watcher worker loop unless explicitly disabled.
	// Even if no folders are watched yet, this keeps watcher enable/disable APIs consistent.
	if isFileWatcherDisabledByEnv() {
		logger.Info("File watcher startup disabled via ENABLE_FILE_WATCHER=false")
	} else {
		if err := fileWatcher.Start(); err != nil {
			logger.Warn("Failed to start file watcher at startup", zap.Error(err))
		}
	}

	// Auto-index project root at startup
	projectRoot := tools.GetProjectRoot()
	logger.Info("Auto-indexing project root", zap.String("path", projectRoot))

	// Check if project root already has a mapping
	existingMapping, err := codeIndexStorage.GetPathMapping(projectRoot)
	if err != nil {
		logger.Warn("Failed to check existing path mapping", zap.Error(err))
	}

	// Create auto-indexer instance
	autoIndexer := indexer.NewAutoIndexer(codeIndexStorage, qdrantClient, embeddingClient, logger)

	var collectionName string
	needsIndexing := false

	ensureProjectRootWatched := func() {
		if fileWatcher == nil || !fileWatcher.IsRunning() {
			return
		}

		folder, err := codeIndexStorage.GetFolderByPath(projectRoot)
		if err != nil {
			logger.Warn("Failed to fetch project root folder for watcher registration",
				zap.String("path", projectRoot),
				zap.Error(err))
			return
		}
		if folder == nil || folder.Status != "active" {
			return
		}

		if err := fileWatcher.AddFolder(folder); err != nil {
			logger.Warn("Failed to register project root with file watcher",
				zap.String("path", projectRoot),
				zap.Error(err))
		}
	}

	if existingMapping == nil {
		// No mapping exists - use the configured QDRANT_CODE_COLLECTION
		collectionName = storage.CodeIndexCollection

		// Save mapping to MongoDB
		err = codeIndexStorage.AddPathMapping(projectRoot, collectionName)
		if err != nil {
			logger.Warn("Failed to save path mapping",
				zap.String("path", projectRoot),
				zap.Error(err))
		}

		logger.Info("Created code index mapping for project root",
			zap.String("path", projectRoot),
			zap.String("collection", collectionName))
		needsIndexing = true
	} else {
		collectionName = existingMapping.QdrantCollection

		// Check if mapping uses the correct collection (from env var)
		if collectionName != storage.CodeIndexCollection {
			logger.Warn("Existing mapping uses wrong collection - updating",
				zap.String("path", projectRoot),
				zap.String("oldCollection", collectionName),
				zap.String("newCollection", storage.CodeIndexCollection))

			// Update mapping to use the configured collection
			collectionName = storage.CodeIndexCollection
			err = codeIndexStorage.AddPathMapping(projectRoot, collectionName)
			if err != nil {
				logger.Error("Failed to update path mapping",
					zap.String("path", projectRoot),
					zap.Error(err))
			} else {
				logger.Info("Updated mapping to correct collection",
					zap.String("path", projectRoot),
					zap.String("collection", collectionName))
				needsIndexing = true
			}
		} else {
			logger.Info("Project root mapping found",
				zap.String("path", projectRoot),
				zap.String("collection", collectionName))
		}

		// Check if collection is empty (mapping exists but no vectors)
		isEmpty, err := autoIndexer.CheckCollectionEmpty(context.Background(), collectionName)
		if err != nil {
			// If collection doesn't exist (404), re-create it and trigger indexing
			logger.Warn("Collection check failed - will recreate and index",
				zap.String("collection", collectionName),
				zap.Error(err))

			// Get folder to clear its file metadata
			folder, err := codeIndexStorage.GetFolderByPath(projectRoot)
			if err == nil && folder != nil {
				logger.Info("Clearing stale file metadata before re-indexing",
					zap.String("folderId", folder.ID))

				// Clear all file and chunk records to force full re-index
				if err := codeIndexStorage.RemoveFolder(folder.ID); err != nil {
					logger.Warn("Failed to clear file metadata", zap.Error(err))
				} else {
					logger.Info("Cleared stale file metadata successfully")

					// Recreate the folder entry
					if _, err := codeIndexStorage.AddFolder(projectRoot, "Auto-indexed project root"); err != nil {
						logger.Error("Failed to recreate folder metadata", zap.Error(err))
					}
				}
			}

			// Recreate the specific collection for this path (768 = nomic-embed-text dimensions)
			if err := qdrantClient.EnsureCollection(collectionName, embeddingDimensions); err != nil {
				logger.Error("Failed to recreate collection", zap.Error(err))
			} else {
				logger.Info("Recreated collection successfully",
					zap.String("collection", collectionName))
				needsIndexing = true
			}
		} else if isEmpty {
			logger.Warn("Collection exists but is empty - triggering indexing",
				zap.String("collection", collectionName))
			needsIndexing = true
		} else {
			// Check for FORCE_REINDEX env var
			forceReindex := os.Getenv("FORCE_REINDEX") == "true"
			if forceReindex {
				logger.Warn("FORCE_REINDEX=true - will re-index despite existing vectors",
					zap.String("collection", collectionName))
				needsIndexing = true
			} else {
				logger.Info("Collection already has vectors - skipping indexing",
					zap.String("collection", collectionName))
			}
		}
	}

	// Start indexing in background if needed
	if needsIndexing && collectionName != "" {
		logger.Info("Starting background file indexing...",
			zap.String("path", projectRoot),
			zap.String("collection", collectionName))

		// Index in background goroutine to not block server startup
		go func() {
			startTime := time.Now()
			result := autoIndexer.IndexProjectRoot(context.Background(), projectRoot, collectionName)

			if result.Error != nil {
				logger.Error("File indexing failed",
					zap.String("path", projectRoot),
					zap.String("collection", collectionName),
					zap.Error(result.Error))
			} else {
				duration := time.Since(startTime)
				logger.Info("File indexing complete",
					zap.String("path", projectRoot),
					zap.String("collection", collectionName),
					zap.Int("filesIndexed", result.FilesIndexed),
					zap.Int("filesUpdated", result.FilesUpdated),
					zap.Int("filesSkipped", result.FilesSkipped),
					zap.Int("totalFiles", result.TotalFiles),
					zap.Duration("duration", duration))

				// Verify collection has vectors
				pointCount, err := qdrantClient.GetCollectionPointCount(collectionName)
				if err != nil {
					logger.Error("Failed to verify collection point count", zap.Error(err))
				} else {
					logger.Info("Collection verification complete",
						zap.String("collection", collectionName),
						zap.Int("pointCount", pointCount))
				}

				// Ensure watcher registration for project root after metadata creation/reindex.
				ensureProjectRootWatched()
			}
		}()
	} else if !needsIndexing {
		// When indexing is skipped, still ensure watcher registration for existing active project root metadata.
		ensureProjectRootWatched()
		logger.Info("Skipping background indexing - collection already populated")
	} else {
		logger.Warn("Skipping background indexing - collection name is empty")
	}

	// Initialize knowledge storage (needed by task storage and coordinator tools)
	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(db, qdrantClient, logger)
	if err != nil {
		logger.Fatal("Failed to initialize knowledge storage", zap.Error(err))
	}

	// Initialize reflection storage (metacognitive layer) with Qdrant for semantic search
	reflectionStorage, err := storage.NewReflectionStorage(db, qdrantClient, logger)
	if err != nil {
		logger.Fatal("Failed to initialize reflection storage", zap.Error(err))
	}

	// Initialize AI summarizer for task summarization (optional - if fails, task storage will use fallback)
	var taskSummarizer storage.TaskSummarizer
	// Note: summarizer will be nil if initialization fails, which is acceptable (will use fallback)
	// The summarizer requires AI service configuration from .env.hyper
	// If not configured, tasks will still be created but summaries will be truncated text instead of AI-generated

	// Initialize task storage (needed by coordinator tools)
	taskStorage, err := storage.NewMongoTaskStorage(db, knowledgeStorage, taskSummarizer, logger)
	if err != nil {
		logger.Fatal("Failed to initialize task storage", zap.Error(err))
	}

	// Initialize tools storage for MCP hub tools (discover_tools, execute_tool, etc.)
	toolsStorage, err := storage.NewToolsStorage(db, qdrantClient, logger)
	if err != nil {
		logger.Fatal("Failed to initialize tools storage", zap.Error(err))
	}

	// Initialize code summarizer for search results
	var codeSummarizer summarizer.CodeSummarizer
	summarizerConfig := summarizer.LoadSummarizerConfig()
	if summarizerConfig.Enabled {
		llmSummarizer, err := summarizer.NewLLMSummarizer(summarizerConfig, logger)
		if err != nil {
			logger.Warn("Failed to initialize code summarizer, search results will not include summaries",
				zap.Error(err))
			// Continue without summarizer - graceful degradation
		} else {
			codeSummarizer = llmSummarizer
			logger.Info("Code summarizer initialized successfully",
				zap.String("model", summarizerConfig.Model),
				zap.Int("maxTokens", summarizerConfig.MaxTokens))
		}
	}

	// Initialize MCP handlers
	codeToolsHandler := handlers.NewCodeToolsHandler(codeIndexStorage, codeSummarizer, qdrantClient, embeddingClient, fileWatcher, logger)

	// Create MCP server with Implementation and ServerOptions
	impl := &mcp.Implementation{
		Name:    "hyper-coordinator",
		Version: "1.0.0",
	}
	mcpServer := mcp.NewServer(impl, &mcp.ServerOptions{})

	// Register code indexing tools
	if err := codeToolsHandler.RegisterCodeIndexTools(mcpServer); err != nil {
		logger.Fatal("Failed to register code indexing tools", zap.Error(err))
	}

	// Check if MCP hub tools should be registered (default: true, disable with MCP_HUB=false)
	mcpHubEnabled := os.Getenv("MCP_HUB") != "false"
	logger.Info("MCP hub tools registration",
		zap.Bool("enabled", mcpHubEnabled),
		zap.String("MCP_HUB", os.Getenv("MCP_HUB")))

	// Conditionally register MCP hub tools for external MCP client discovery
	if mcpHubEnabled {
		toolsDiscoveryHandler := handlers.NewToolsDiscoveryHandler(toolsStorage, mcpServer, logger)
		if err := toolsDiscoveryHandler.RegisterToolsDiscoveryTools(mcpServer); err != nil {
			logger.Fatal("Failed to register MCP hub tools", zap.Error(err))
		}
		logger.Info("MCP hub tools registered to MCP server",
			zap.Strings("tools", []string{
				"discover_tools",
				"get_tool_schema",
				"execute_tool",
				"mcp_add_server",
				"mcp_rediscover_server",
				"mcp_remove_server",
			}))
	} else {
		logger.Info("MCP hub tools NOT registered (MCP_HUB=false)")
	}

	// Register coordinator tools (task management, knowledge, subagents)
	logger.Info("Registering coordinator tools to MCP server...")
	toolHandler := handlers.NewToolHandler(taskStorage, knowledgeStorage, db)
	if err := toolHandler.RegisterToolHandlers(mcpServer); err != nil {
		logger.Fatal("Failed to register coordinator tools to MCP server", zap.Error(err))
	}
	logger.Info("Coordinator tools registered to MCP server", zap.Int("count", 19))

	// Register Qdrant tools (semantic search and storage)
	logger.Info("Registering Qdrant tools to MCP server...")
	qdrantHandler := handlers.NewQdrantToolHandler(qdrantClient)
	qdrantHandler.SetKnowledgeStorage(knowledgeStorage) // Enable MongoDB storage for knowledge_store
	if err := qdrantHandler.RegisterQdrantTools(mcpServer); err != nil {
		logger.Fatal("Failed to register Qdrant tools to MCP server", zap.Error(err))
	}
	logger.Info("Qdrant tools registered to MCP server", zap.Int("count", 2))

	// Register reflection tools (metacognitive self-awareness layer)
	logger.Info("Registering reflection tools to MCP server...")
	reflectionToolHandler := handlers.NewReflectionToolHandler(reflectionStorage)
	if err := reflectionToolHandler.RegisterReflectionTools(mcpServer); err != nil {
		logger.Fatal("Failed to register reflection tools to MCP server", zap.Error(err))
	}
	logger.Info("Reflection tools registered to MCP server", zap.Int("count", 7))

	// Initialize code validator for error prevention mode (per-session control via context)
	logger.Info("Initializing code validator for error prevention...")
	validator := validation.NewCodeValidator(logger, tools.GetProjectRoot())
	logger.Info("Code validator initialized (per-session control)")

	// Register filesystem tools (bash, file operations, patch application)
	logger.Info("Registering filesystem tools to MCP server...")
	filesystemHandler := handlers.NewFilesystemToolHandler(logger, validator)
	if err := filesystemHandler.RegisterFilesystemTools(mcpServer); err != nil {
		logger.Fatal("Failed to register filesystem tools to MCP server", zap.Error(err))
	}
	logger.Info("Filesystem tools registered to MCP server", zap.Int("count", 5))

	// Get UI filesystem
	embeddedUI, err := embed.GetUIFileSystem()
	if err != nil {
		logger.Warn("Failed to get embedded UI", zap.Error(err))
	}
	hasEmbeddedUI := embed.HasUI()

	// Start servers based on mode
	var wg sync.WaitGroup
	var serverErrors []error
	var mu sync.Mutex

	if *mode == "http" || *mode == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Read HTTP port from environment, default to 8080
			httpPort := os.Getenv("HTTP_PORT")
			if httpPort == "" {
				httpPort = "8080"
			}
			logger.Info("Starting HTTP server", zap.String("port", httpPort))
			ctx := context.Background()
			if err := server.StartHTTPServer(
				ctx,
				httpPort,
				taskStorage,
				knowledgeStorage,
				reflectionStorage,
				codeIndexStorage,
				qdrantClient,
				embeddingClient,
				codeSummarizer,
				fileWatcher,
				mcpServer,
				embeddedUI,
				hasEmbeddedUI,
				logger,
				db,
				toolsStorage,
			); err != nil && err != http.ErrServerClosed {
				mu.Lock()
				serverErrors = append(serverErrors, fmt.Errorf("HTTP server error: %w", err))
				mu.Unlock()
				logger.Error("HTTP server error", zap.Error(err))
			}
		}()
	}

	if *mode == "mcp" || *mode == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("Starting MCP server on stdio")
			ctx := context.Background()
			if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
				mu.Lock()
				serverErrors = append(serverErrors, fmt.Errorf("MCP server error: %w", err))
				mu.Unlock()
				logger.Error("MCP server error", zap.Error(err))
			}
		}()
	}

	// Wait for servers to complete
	wg.Wait()

	// Check for errors
	if len(serverErrors) > 0 {
		logger.Error("Server errors occurred")
		for _, err := range serverErrors {
			logger.Error("", zap.Error(err))
		}
		os.Exit(1)
	}
}
