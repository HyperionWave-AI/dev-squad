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

	fmt.Printf("Connecting to MongoDB...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	// Get recent agent tasks (last 10)
	agentTasksCollection := db.Collection("agent_tasks")
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(10)
	cursor, err := agentTasksCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Fatalf("Failed to query agent_tasks: %v", err)
	}
	defer cursor.Close(ctx)

	fmt.Println("\n=== RECENT AGENT TASKS ===")
	var tasks []bson.M
	if err = cursor.All(ctx, &tasks); err != nil {
		log.Fatalf("Failed to decode tasks: %v", err)
	}

	for i, task := range tasks {
		fmt.Printf("\n[%d] Task ID: %v\n", i+1, task["_id"])
		fmt.Printf("    Agent: %v\n", task["agentName"])
		fmt.Printf("    Role: %v\n", task["role"])
		fmt.Printf("    Status: %v\n", task["status"])
		fmt.Printf("    Created: %v\n", task["createdAt"])
		fmt.Printf("    Human Task ID: %v\n", task["humanTaskId"])
		if filesModified, ok := task["filesModified"].(bson.A); ok && len(filesModified) > 0 {
			fmt.Printf("    Files Modified: %v\n", filesModified)
		}
		if todos, ok := task["todos"].(bson.A); ok {
			fmt.Printf("    TODOs: %d items\n", len(todos))
		}
	}

	// Get recent human tasks
	humanTasksCollection := db.Collection("human_tasks")
	humanCursor, err := humanTasksCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Fatalf("Failed to query human_tasks: %v", err)
	}
	defer humanCursor.Close(ctx)

	fmt.Println("\n\n=== RECENT HUMAN TASKS ===")
	var humanTasks []bson.M
	if err = humanCursor.All(ctx, &humanTasks); err != nil {
		log.Fatalf("Failed to decode human tasks: %v", err)
	}

	for i, task := range humanTasks {
		fmt.Printf("\n[%d] Task ID: %v\n", i+1, task["_id"])
		fmt.Printf("    Prompt: %v\n", task["prompt"])
		fmt.Printf("    Status: %v\n", task["status"])
		fmt.Printf("    Created: %v\n", task["createdAt"])
	}
}
