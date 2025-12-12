package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB"
	dbName := "hyper_hyperion_megha"

	fmt.Println("Connecting to MongoDB...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	// Delete folder metadata
	foldersColl := db.Collection("indexed_folders")
	result, err := foldersColl.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to delete folder metadata: %v", err)
	}

	fmt.Printf("✓ Deleted %d folder metadata records\n", result.DeletedCount)

	// Delete file metadata
	filesColl := db.Collection("indexed_files")
	fileResult, err := filesColl.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to delete file metadata: %v", err)
	}

	fmt.Printf("✓ Deleted %d file metadata records\n", fileResult.DeletedCount)
	fmt.Println("\n✓ MongoDB index metadata cleared successfully!")
	fmt.Println("Restart the server to re-index with correct paths.")
}
