package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB"
	}

	database := os.Getenv("MONGODB_DATABASE")
	if database == "" {
		database = "hyper_hyperion_megha"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)

	// Delete all system prompts
	result, err := db.Collection("system_prompts").DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error deleting system prompts: %v\n", err)
	} else {
		fmt.Printf("✓ Deleted %d system prompts\n", result.DeletedCount)
	}

	// Verify deletion
	count, err := db.Collection("system_prompts").CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Error counting system prompts: %v\n", err)
	} else {
		fmt.Printf("✓ Remaining system prompts: %d\n", count)
	}

	fmt.Println("\n✓ Database will now use DefaultSystemPrompt from chat_websocket.go!")
}
