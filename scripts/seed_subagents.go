package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Subagent represents a specialist agent
type Subagent struct {
	ID           string    `bson:"_id" json:"id"`
	Name         string    `bson:"name" json:"name"`
	Description  string    `bson:"description" json:"description"`
	SystemPrompt string    `bson:"systemPrompt" json:"systemPrompt"`
	Tools        []string  `bson:"tools,omitempty" json:"tools,omitempty"`
	Category     string    `bson:"category,omitempty" json:"category,omitempty"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env.hyper"); err != nil {
		log.Printf("Warning: Error loading .env.hyper file: %v", err)
	}

	// Get MongoDB connection string
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is required")
	}

	mongoDBName := os.Getenv("MONGODB_DATABASE")
	if mongoDBName == "" {
		mongoDBName = "coordinator_db_max"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Get collection
	collection := client.Database(mongoDBName).Collection("subagents")

	// Define predefined subagents (matching the list from tools.go)
	subagents := []Subagent{
		{
			ID:           "go-dev",
			Name:         "go-dev",
			Description:  "Go backend development specialist",
			SystemPrompt: "You are a Go backend development specialist. Focus on writing clean, efficient Go code following best practices.",
			Category:     "backend",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "go-mcp-dev",
			Name:         "go-mcp-dev",
			Description:  "Go MCP (Model Context Protocol) development specialist",
			SystemPrompt: "You are a Go MCP development specialist. Focus on implementing MCP tools and protocols in Go.",
			Category:     "backend",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "backend-services-specialist",
			Name:         "Backend Services Specialist",
			Description:  "Go 1.25 microservices expert specializing in REST APIs, business logic, and service architecture",
			SystemPrompt: "You are a Backend Services Specialist focusing on Go 1.25 microservices, REST APIs, and service architecture within the Hyperion AI Platform.",
			Category:     "backend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "event-systems-specialist",
			Name:         "Event Systems Specialist",
			Description:  "NATS JetStream and MCP protocol expert specializing in event-driven architecture",
			SystemPrompt: "You are an Event Systems Specialist focusing on NATS JetStream, MCP protocols, event-driven architecture, and service orchestration.",
			Category:     "backend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "data-platform-specialist",
			Name:         "Data Platform Specialist",
			Description:  "MongoDB and coordinator knowledge optimization expert",
			SystemPrompt: "You are a Data Platform Specialist focusing on MongoDB, data modeling, query performance, migration strategies, and vector operations.",
			Category:     "data",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "ui-dev",
			Name:         "ui-dev",
			Description:  "Frontend UI development specialist",
			SystemPrompt: "You are a frontend UI development specialist. Focus on React, TypeScript, and modern UI frameworks.",
			Category:     "frontend",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "ui-tester",
			Name:         "ui-tester",
			Description:  "UI testing specialist",
			SystemPrompt: "You are a UI testing specialist. Focus on testing design, layout, and stability issues in web projects.",
			Category:     "frontend",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "frontend-experience-specialist",
			Name:         "Frontend Experience Specialist",
			Description:  "React 18 + TypeScript expert specializing in atomic design systems and user experience",
			SystemPrompt: "You are a Frontend Experience Specialist focusing on React 18, TypeScript, atomic design systems, user experience, accessibility, and component architecture.",
			Category:     "frontend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "ai-integration-specialist",
			Name:         "AI Integration Specialist",
			Description:  "Claude/GPT API expert and AI3 framework specialist",
			SystemPrompt: "You are an AI Integration Specialist focusing on Claude/GPT APIs, AI3 framework, model coordination, intelligent task orchestration, and AI-driven user experiences.",
			Category:     "ai",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "realtime-systems-specialist",
			Name:         "Real-time Systems Specialist",
			Description:  "WebSocket and real-time protocol expert",
			SystemPrompt: "You are a Real-time Systems Specialist focusing on WebSockets, real-time protocols, streaming data delivery, live connections, and real-time synchronization.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "sre",
			Name:         "sre",
			Description:  "Site Reliability Engineering and deployment specialist",
			SystemPrompt: "You are an SRE specialist. Focus on deployment, monitoring, reliability, and operational excellence.",
			Category:     "infrastructure",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "k8s-deployment-expert",
			Name:         "k8s-deployment-expert",
			Description:  "Kubernetes deployment and orchestration expert",
			SystemPrompt: "You are a Kubernetes deployment expert. Focus on creating, modifying, troubleshooting, and optimizing Kubernetes deployments, including manifests, rollouts, scaling strategies, and resource management.",
			Category:     "infrastructure",
			Tools:        []string{"*"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "infrastructure-automation-specialist",
			Name:         "Infrastructure Automation Specialist",
			Description:  "Google Cloud Platform and Kubernetes expert specializing in GKE deployment automation",
			SystemPrompt: "You are an Infrastructure Automation Specialist focusing on Google Cloud Platform, Kubernetes, GKE deployment automation, GitHub Actions CI/CD, and infrastructure orchestration.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "security-auth-specialist",
			Name:         "Security & Auth Specialist",
			Description:  "Security architecture and JWT authentication expert",
			SystemPrompt: "You are a Security & Auth Specialist focusing on security architecture, JWT authentication, identity management, access control, security policies, and threat protection.",
			Category:     "security",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "observability-specialist",
			Name:         "Observability Specialist",
			Description:  "Monitoring and observability expert",
			SystemPrompt: "You are an Observability Specialist focusing on monitoring, observability, metrics collection, distributed tracing, performance analysis, and operational insights.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "e2e-testing-coordinator",
			Name:         "End-to-End Testing Coordinator",
			Description:  "Cross-squad testing orchestrator specializing in end-to-end automation",
			SystemPrompt: "You are an End-to-End Testing Coordinator focusing on cross-squad testing orchestration, end-to-end automation, user journey validation, integration testing, and quality assurance coordination.",
			Category:     "testing",
			Tools:        []string{"hyper", "playwright-mcp", "@modelcontextprotocol/server-fetch", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "mcp-server-kubernetes", "mcp-server-mongodb"},
			CreatedAt:    time.Now(),
		},
	}

	// Clear existing subagents
	_, err = collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to clear existing subagents: %v", err)
	}
	fmt.Println("✓ Cleared existing subagents")

	// Insert subagents
	for _, subagent := range subagents {
		_, err := collection.InsertOne(ctx, subagent)
		if err != nil {
			log.Printf("Warning: Failed to insert subagent %s: %v", subagent.Name, err)
			continue
		}
		fmt.Printf("✓ Inserted subagent: %s\n", subagent.Name)
	}

	fmt.Printf("\n✓ Successfully seeded %d subagents\n", len(subagents))
}
