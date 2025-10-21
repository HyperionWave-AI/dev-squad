package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// SubchatStatus represents the status of a subchat
type SubchatStatus string

const (
	SubchatStatusActive    SubchatStatus = "active"
	SubchatStatusCompleted SubchatStatus = "completed"
	SubchatStatusFailed    SubchatStatus = "failed"
)

// Subchat represents a parallel workflow session
type Subchat struct {
	ID             string        `bson:"_id" json:"id"`
	ParentChatID   string        `bson:"parentChatId" json:"parentChatId"`
	SessionID      *string       `bson:"sessionId,omitempty" json:"sessionId,omitempty"` // Chat session ID for message storage
	SubagentName   string        `bson:"subagentName" json:"subagentName"`
	AssignedTaskID *string       `bson:"assignedTaskId,omitempty" json:"assignedTaskId,omitempty"`
	AssignedTodoID *string       `bson:"assignedTodoId,omitempty" json:"assignedTodoId,omitempty"`
	Status         SubchatStatus `bson:"status" json:"status"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// Subagent represents an available specialist agent
type Subagent struct {
	ID           string   `bson:"_id" json:"id"`
	Name         string   `bson:"name" json:"name"`
	Description  string   `bson:"description" json:"description"`
	SystemPrompt string   `bson:"systemPrompt" json:"systemPrompt"`
	Tools        []string `bson:"tools,omitempty" json:"tools,omitempty"`
	Category     string   `bson:"category,omitempty" json:"category,omitempty"`
}

// SubchatStorage handles subchat persistence
type SubchatStorage struct {
	collection        *mongo.Collection
	subagentCollection *mongo.Collection
	logger            *zap.Logger
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// NewSubchatStorage creates a new subchat storage
func NewSubchatStorage(db *mongo.Database, logger *zap.Logger) *SubchatStorage {
	return &SubchatStorage{
		collection:        db.Collection("subchats"),
		subagentCollection: db.Collection("subagents"),
		logger:            logger,
	}
}

// CreateSubchat creates a new subchat
func (s *SubchatStorage) CreateSubchat(parentChatID, subagentName string, taskID, todoID *string) (*Subchat, error) {
	subchat := &Subchat{
		ID:             uuid.New().String(),
		ParentChatID:   parentChatID,
		SubagentName:   subagentName,
		AssignedTaskID: taskID,
		AssignedTodoID: todoID,
		Status:         SubchatStatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.collection.InsertOne(ctx, subchat)
	if err != nil {
		s.logger.Error("Failed to create subchat", zap.Error(err))
		return nil, fmt.Errorf("failed to create subchat: %w", err)
	}

	s.logger.Info("Created subchat",
		zap.String("subchatId", subchat.ID),
		zap.String("parentChatId", parentChatID),
		zap.String("subagentName", subagentName))

	return subchat, nil
}

// GetSubchat retrieves a subchat by ID
func (s *SubchatStorage) GetSubchat(id string) (*Subchat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subchat Subchat
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&subchat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subchat not found: %s", id)
		}
		s.logger.Error("Failed to get subchat", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get subchat: %w", err)
	}

	return &subchat, nil
}

// GetSubchatsByParent retrieves all subchats for a parent chat
func (s *SubchatStorage) GetSubchatsByParent(parentChatID string) ([]*Subchat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"parentChatId": parentChatID})
	if err != nil {
		s.logger.Error("Failed to query subchats", zap.String("parentChatId", parentChatID), zap.Error(err))
		return nil, fmt.Errorf("failed to query subchats: %w", err)
	}
	defer cursor.Close(ctx)

	var subchats []*Subchat
	if err := cursor.All(ctx, &subchats); err != nil {
		s.logger.Error("Failed to decode subchats", zap.Error(err))
		return nil, fmt.Errorf("failed to decode subchats: %w", err)
	}

	return subchats, nil
}

// UpdateSubchatStatus updates the status of a subchat
func (s *SubchatStorage) UpdateSubchatStatus(id string, status SubchatStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		s.logger.Error("Failed to update subchat status", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to update subchat status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subchat not found: %s", id)
	}

	s.logger.Info("Updated subchat status",
		zap.String("subchatId", id),
		zap.String("status", string(status)))

	return nil
}

// ListSubagents retrieves all available subagents
func (s *SubchatStorage) ListSubagents() ([]*Subagent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.subagentCollection.Find(ctx, bson.M{})
	if err != nil {
		s.logger.Error("Failed to query subagents", zap.Error(err))
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer cursor.Close(ctx)

	var subagents []*Subagent
	if err := cursor.All(ctx, &subagents); err != nil {
		s.logger.Error("Failed to decode subagents", zap.Error(err))
		return nil, fmt.Errorf("failed to decode subagents: %w", err)
	}

	return subagents, nil
}

// GetSubagent retrieves a subagent by name
func (s *SubchatStorage) GetSubagent(name string) (*Subagent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subagent Subagent
	err := s.subagentCollection.FindOne(ctx, bson.M{"name": name}).Decode(&subagent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subagent not found: %s", name)
		}
		s.logger.Error("Failed to get subagent", zap.String("name", name), zap.Error(err))
		return nil, fmt.Errorf("failed to get subagent: %w", err)
	}

	return &subagent, nil
}

// UpdateSubchatAgentTask links an agent task ID to a subchat
func (s *SubchatStorage) UpdateSubchatAgentTask(subchatID string, agentTaskID *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": subchatID},
		bson.M{
			"$set": bson.M{
				"assignedTaskId": agentTaskID,
				"updatedAt":      time.Now(),
			},
		},
	)
	if err != nil {
		s.logger.Error("Failed to update subchat agent task", zap.String("subchatId", subchatID), zap.Error(err))
		return fmt.Errorf("failed to update subchat agent task: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subchat not found: %s", subchatID)
	}

	s.logger.Info("Updated subchat with agent task",
		zap.String("subchatId", subchatID),
		zap.Stringp("agentTaskId", agentTaskID))

	return nil
}

// UpdateSubchatSessionID links a chat session ID to a subchat for message storage
func (s *SubchatStorage) UpdateSubchatSessionID(subchatID string, sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": subchatID},
		bson.M{
			"$set": bson.M{
				"sessionId": sessionID,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		s.logger.Error("Failed to update subchat session ID", zap.String("subchatId", subchatID), zap.Error(err))
		return fmt.Errorf("failed to update subchat session ID: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subchat not found: %s", subchatID)
	}

	s.logger.Info("Updated subchat with session ID",
		zap.String("subchatId", subchatID),
		zap.String("sessionId", sessionID))

	return nil
}

// EnsureSystemSubagents ensures all predefined system subagents exist in the database.
// This is called automatically on startup to bootstrap new databases.
// It's idempotent - safe to call multiple times.
func (s *SubchatStorage) EnsureSystemSubagents() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define all system subagents
	systemSubagents := []Subagent{
		{
			ID:           "go-dev",
			Name:         "go-dev",
			Description:  "Go backend development specialist",
			SystemPrompt: "You are a Go backend development specialist. Focus on writing clean, efficient Go code following best practices.",
			Category:     "backend",
			Tools:        []string{"*"},
		},
		{
			ID:           "go-mcp-dev",
			Name:         "go-mcp-dev",
			Description:  "Go MCP (Model Context Protocol) development specialist",
			SystemPrompt: "You are a Go MCP development specialist. Focus on implementing MCP tools and protocols in Go.",
			Category:     "backend",
			Tools:        []string{"*"},
		},
		{
			ID:           "backend-services-specialist",
			Name:         "Backend Services Specialist",
			Description:  "Go 1.25 microservices expert specializing in REST APIs, business logic, and service architecture",
			SystemPrompt: "You are a Backend Services Specialist focusing on Go 1.25 microservices, REST APIs, and service architecture within the Hyperion AI Platform.",
			Category:     "backend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
		},
		{
			ID:           "event-systems-specialist",
			Name:         "Event Systems Specialist",
			Description:  "NATS JetStream and MCP protocol expert specializing in event-driven architecture",
			SystemPrompt: "You are an Event Systems Specialist focusing on NATS JetStream, MCP protocols, event-driven architecture, and service orchestration.",
			Category:     "backend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
		},
		{
			ID:           "data-platform-specialist",
			Name:         "Data Platform Specialist",
			Description:  "MongoDB and coordinator knowledge optimization expert",
			SystemPrompt: "You are a Data Platform Specialist focusing on MongoDB, data modeling, query performance, migration strategies, and vector operations.",
			Category:     "data",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-fetch", "mcp-server-mongodb"},
		},
		{
			ID:           "ui-dev",
			Name:         "ui-dev",
			Description:  "Frontend UI development specialist",
			SystemPrompt: "You are a frontend UI development specialist. Focus on React, TypeScript, and modern UI frameworks.",
			Category:     "frontend",
			Tools:        []string{"*"},
		},
		{
			ID:           "ui-tester",
			Name:         "ui-tester",
			Description:  "UI testing specialist",
			SystemPrompt: "You are a UI testing specialist. Focus on testing design, layout, and stability issues in web projects.",
			Category:     "frontend",
			Tools:        []string{"*"},
		},
		{
			ID:           "frontend-experience-specialist",
			Name:         "Frontend Experience Specialist",
			Description:  "React 18 + TypeScript expert specializing in atomic design systems and user experience",
			SystemPrompt: "You are a Frontend Experience Specialist focusing on React 18, TypeScript, atomic design systems, user experience, accessibility, and component architecture.",
			Category:     "frontend",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "ai-integration-specialist",
			Name:         "AI Integration Specialist",
			Description:  "Claude/GPT API expert and AI3 framework specialist",
			SystemPrompt: "You are an AI Integration Specialist focusing on Claude/GPT APIs, AI3 framework, model coordination, intelligent task orchestration, and AI-driven user experiences.",
			Category:     "ai",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "realtime-systems-specialist",
			Name:         "Real-time Systems Specialist",
			Description:  "WebSocket and real-time protocol expert",
			SystemPrompt: "You are a Real-time Systems Specialist focusing on WebSockets, real-time protocols, streaming data delivery, live connections, and real-time synchronization.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "playwright-mcp", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "sre",
			Name:         "sre",
			Description:  "Site Reliability Engineering and deployment specialist",
			SystemPrompt: "You are an SRE specialist. Focus on deployment, monitoring, reliability, and operational excellence.",
			Category:     "infrastructure",
			Tools:        []string{"*"},
		},
		{
			ID:           "k8s-deployment-expert",
			Name:         "k8s-deployment-expert",
			Description:  "Kubernetes deployment and orchestration expert",
			SystemPrompt: "You are a Kubernetes deployment expert. Focus on creating, modifying, troubleshooting, and optimizing Kubernetes deployments, including manifests, rollouts, scaling strategies, and resource management.",
			Category:     "infrastructure",
			Tools:        []string{"*"},
		},
		{
			ID:           "infrastructure-automation-specialist",
			Name:         "Infrastructure Automation Specialist",
			Description:  "Google Cloud Platform and Kubernetes expert specializing in GKE deployment automation",
			SystemPrompt: "You are an Infrastructure Automation Specialist focusing on Google Cloud Platform, Kubernetes, GKE deployment automation, GitHub Actions CI/CD, and infrastructure orchestration.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "security-auth-specialist",
			Name:         "Security & Auth Specialist",
			Description:  "Security architecture and JWT authentication expert",
			SystemPrompt: "You are a Security & Auth Specialist focusing on security architecture, JWT authentication, identity management, access control, security policies, and threat protection.",
			Category:     "security",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "observability-specialist",
			Name:         "Observability Specialist",
			Description:  "Monitoring and observability expert",
			SystemPrompt: "You are an Observability Specialist focusing on monitoring, observability, metrics collection, distributed tracing, performance analysis, and operational insights.",
			Category:     "infrastructure",
			Tools:        []string{"hyper", "mcp-server-kubernetes", "@modelcontextprotocol/server-github", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-fetch"},
		},
		{
			ID:           "e2e-testing-coordinator",
			Name:         "End-to-End Testing Coordinator",
			Description:  "Cross-squad testing orchestrator specializing in end-to-end automation",
			SystemPrompt: "You are an End-to-End Testing Coordinator focusing on cross-squad testing orchestration, end-to-end automation, user journey validation, integration testing, and quality assurance coordination.",
			Category:     "testing",
			Tools:        []string{"hyper", "playwright-mcp", "@modelcontextprotocol/server-fetch", "@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-github", "mcp-server-kubernetes", "mcp-server-mongodb"},
		},
	}

	// Check if subagents already exist
	count, err := s.subagentCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		s.logger.Error("Failed to count subagents", zap.Error(err))
		return fmt.Errorf("failed to count subagents: %w", err)
	}

	// If we have the expected number of system subagents, skip seeding
	// Use len(systemSubagents) to dynamically match the actual array size
	expectedCount := int64(len(systemSubagents))
	if count >= expectedCount {
		s.logger.Info("System subagents already exist",
			zap.Int64("count", count),
			zap.Int64("expected", expectedCount))
		return nil
	}

	s.logger.Info("Seeding system subagents",
		zap.Int64("existingCount", count),
		zap.Int64("expectedCount", expectedCount))

	// Insert or update each subagent using upsert
	seededCount := 0
	for _, subagent := range systemSubagents {
		opts := options.Update().SetUpsert(true)
		_, err := s.subagentCollection.UpdateOne(
			ctx,
			bson.M{"_id": subagent.ID},
			bson.M{"$set": subagent},
			opts,
		)
		if err != nil {
			s.logger.Warn("Failed to upsert subagent",
				zap.String("name", subagent.Name),
				zap.Error(err))
			continue
		}
		seededCount++
	}

	s.logger.Info("System subagents seeding complete",
		zap.Int("seededCount", seededCount),
		zap.Int("totalSystemSubagents", len(systemSubagents)))

	return nil
}
