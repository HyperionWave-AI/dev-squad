package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Qdrant connection details
	qdrantURL := "2445de59-e696-4952-8934-a89c7c7cfec0.us-east4-0.gcp.cloud.qdrant.io:6334"
	apiKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3MiOiJyZWFkLXdyaXRlIn0.kPNB3b-cJMBLyL3vG_WqQJYH8VkLGzgGEpqvCJlDq8M"
	collectionName := "code_index_cc212e13"

	fmt.Printf("⚠️  WARNING: This will DELETE the code index collection: %s\n", collectionName)
	fmt.Println("This will force a re-index on next server startup with correct paths.")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to cancel, or wait 5 seconds to proceed...")
	time.Sleep(5 * time.Second)

	fmt.Println("\nConnecting to Qdrant...")

	// Connect to Qdrant
	conn, err := grpc.Dial(
		qdrantURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(&apiKeyAuth{apiKey: apiKey}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer conn.Close()

	client := qdrant.NewCollectionsClient(conn)
	ctx := context.Background()

	// Delete the collection
	fmt.Printf("Deleting collection: %s...\n", collectionName)
	_, err = client.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: collectionName,
	})
	if err != nil {
		log.Fatalf("Failed to delete collection: %v", err)
	}

	fmt.Println("✓ Collection deleted successfully!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("1. Restart the server (make run)")
	fmt.Println("2. Server will auto-index /Users/meghaneelamana/dev-squad with correct paths")
	fmt.Println("3. Wait for indexing to complete (check logs for 'Collection ready')")
}

// apiKeyAuth implements credentials.PerRPCCredentials for API key authentication
type apiKeyAuth struct {
	apiKey string
}

func (a *apiKeyAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"api-key": a.apiKey,
	}, nil
}

func (a *apiKeyAuth) RequireTransportSecurity() bool {
	return false
}
