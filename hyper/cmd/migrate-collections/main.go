package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// Load environment
	if err := godotenv.Load(".env.hyper"); err != nil {
		log.Printf("Warning: .env.hyper not found: %v", err)
	}

	// Get MongoDB URI
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is required")
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "max_hyper_coordinator_db_dev_squad"
	}

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Ping MongoDB
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	logger.Info("Successfully connected to MongoDB", zap.String("database", dbName))

	// Create MongoKnowledgeStorage
	db := client.Database(dbName)
	knowledgeStorage, err := storage.NewMongoKnowledgeStorage(
		db,
		nil, // qdrantClient - not needed for migration
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to create knowledge storage: %v", err)
	}

	// Run migration
	logger.Info("Starting collection migration...")
	logger.Info("This will:")
	logger.Info("  1. Create Collection objects from existing collection strings")
	logger.Info("  2. Update knowledge entries with collectionId references")
	logger.Info("  3. Preserve existing Qdrant collections (QdrantName = old collection string)")
	logger.Info("  4. Migrate metadata from collection_metadata")
	logger.Info("")

	stats, err := knowledgeStorage.MigrateToCollectionObjects()
	if err != nil {
		logger.Error("Migration failed", zap.Error(err), zap.Any("stats", stats))
		os.Exit(1)
	}

	logger.Info("✅ Migration completed successfully!")
	logger.Info("Migration statistics:")
	for key, value := range stats {
		logger.Info(fmt.Sprintf("  %s: %v", key, value))
	}
}
