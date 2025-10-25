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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)
	coll := db.Collection("code_index_map")

	// Force new collection name
	newName := "code_index_full_reindex"
	
	result, err := coll.UpdateOne(
		ctx,
		bson.M{"path": "/Users/meghaneelamana/dev-squad"},
		bson.M{"$set": bson.M{
			"qdrantCollection": newName,
			"lastIndexed": time.Now().UnixMilli(),
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Fatalf("Failed to update: %v", err)
	}

	fmt.Printf("✓ Collection name updated to: %s\n", newName)
	fmt.Printf("  Matched: %d, Modified: %d, Upserted: %v\n", 
		result.MatchedCount, result.ModifiedCount, result.UpsertedID)
}
