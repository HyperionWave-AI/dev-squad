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

	// Delete from both collections (old and new versioned system)
	collections := []string{"system_prompts", "system_prompt_versions"}

	for _, collName := range collections {
		result, err := db.Collection(collName).DeleteMany(ctx, bson.M{})
		if err != nil {
			log.Printf("Error deleting from %s: %v\n", collName, err)
		} else {
			fmt.Printf("✓ Deleted %d documents from %s\n", result.DeletedCount, collName)
		}

		// Verify deletion
		count, err := db.Collection(collName).CountDocuments(ctx, bson.M{})
		if err != nil {
			log.Printf("Error counting in %s: %v\n", collName, err)
		} else {
			fmt.Printf("✓ Remaining in %s: %d\n", collName, count)
		}
	}

	fmt.Println("\n✓ Database will now use DefaultSystemPrompt from chat_websocket.go!")
}
