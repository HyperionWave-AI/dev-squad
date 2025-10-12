package main

import (
	"context"
	"flag"
	"os"
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

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

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

	logger.Info("Starting Hyper Code Indexer MCP Server")

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		logger.Fatal("MONGODB_URI required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer mongoClient.Disconnect(context.Background())

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "coordinator_db1"
	}
	db := mongoClient.Database(mongoDatabase)

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

	codeIndexStorage, err := storage.NewCodeIndexStorage(db)
	if err != nil {
		logger.Fatal("Failed to initialize code index storage", zap.Error(err))
	}

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		logger.Fatal("OPENAI_API_KEY required")
	}
	embeddingClient := embeddings.NewOpenAIClient(openAIKey)
	logger.Info("Using OpenAI embeddings")

	pathMappingsEnv := os.Getenv("CODE_INDEX_PATH_MAPPINGS")
	pathMapper := watcher.NewPathMapper(pathMappingsEnv, logger)
	fileWatcher, err := watcher.NewFileWatcher(codeIndexStorage, qdrantClient, embeddingClient, pathMapper, logger)
	if err != nil {
		logger.Fatal("Failed to create file watcher", zap.Error(err))
	}

	if err := fileWatcher.Start(); err != nil {
		logger.Warn("Failed to start file watcher", zap.Error(err))
	} else {
		logger.Info("File watcher started")
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "hyper-code-indexer", Version: "2.0.0"}, &mcp.ServerOptions{HasTools: true})
	codeToolsHandler := handlers.NewCodeToolsHandler(codeIndexStorage, qdrantClient, embeddingClient, fileWatcher, logger)
	if err := codeToolsHandler.RegisterCodeIndexTools(server); err != nil {
		logger.Fatal("Failed to register code index tools", zap.Error(err))
	}

	logger.Info("Starting stdio server")
	transport := &mcp.StdioTransport{}
	if err := server.Run(context.Background(), transport); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}
