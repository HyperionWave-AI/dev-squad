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
	uri := "mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB"
	dbName := "hyper_hyperion_megha"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	fmt.Println("\n=== system_prompt_versions collection ===")
	cursor, err := db.Collection("system_prompt_versions").Find(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		count++
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nDocument %d:\n", count)
		fmt.Printf("  userId: %v\n", result["userId"])
		fmt.Printf("  companyId: %v\n", result["companyId"])
		fmt.Printf("  isActive: %v\n", result["isActive"])
		if prompt, ok := result["prompt"].(string); ok {
			fmt.Printf("  Prompt length: %d chars\n", len(prompt))
			if len(prompt) > 100 {
				fmt.Printf("  First 100 chars: %s...\n", prompt[:100])
			} else {
				fmt.Printf("  Prompt: %s\n", prompt)
			}
		}
	}

	if count == 0 {
		fmt.Println("  No documents found")
	} else {
		fmt.Printf("\nTotal: %d documents\n", count)
	}
}
