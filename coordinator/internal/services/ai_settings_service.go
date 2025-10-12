package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hyperion-coordinator/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// AISettingsService manages system prompts and subagents with MongoDB storage
type AISettingsService struct {
	systemPromptsCollection *mongo.Collection
	subagentsCollection     *mongo.Collection
	logger                  *zap.Logger
}

// NewAISettingsService creates a new AI settings service instance
func NewAISettingsService(db *mongo.Database, logger *zap.Logger) (*AISettingsService, error) {
	service := &AISettingsService{
		systemPromptsCollection: db.Collection("system_prompts"),
		subagentsCollection:     db.Collection("subagents"),
		logger:                  logger,
	}

	// Create indexes
	ctx := context.Background()

	// Index on system_prompts: {userId, companyId} for user prompt queries
	_, err := service.systemPromptsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system_prompts user index: %w", err)
	}

	// Index on subagents: {userId, companyId} for user subagent queries
	_, err = service.subagentsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subagents user index: %w", err)
	}

	// Index on subagents: {companyId} for company-level isolation
	_, err = service.subagentsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "companyId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subagents company index: %w", err)
	}

	logger.Info("AI settings service initialized with MongoDB indexes")
	return service, nil
}

// GetSystemPrompt retrieves the system prompt for a user
func (s *AISettingsService) GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error) {
	var systemPrompt models.SystemPrompt
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	err := s.systemPromptsCollection.FindOne(ctx, filter).Decode(&systemPrompt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty string if no custom prompt exists (not an error)
			return "", nil
		}
		return "", fmt.Errorf("failed to retrieve system prompt: %w", err)
	}

	return systemPrompt.Prompt, nil
}

// UpdateSystemPrompt updates or creates the system prompt for a user
func (s *AISettingsService) UpdateSystemPrompt(ctx context.Context, userID, companyID, prompt string) error {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	update := bson.M{
		"$set": bson.M{
			"userId":    userID,
			"companyId": companyID,
			"prompt":    prompt,
			"updatedAt": time.Now().UTC(),
		},
	}

	opts := options.Update().SetUpsert(true)

	result, err := s.systemPromptsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update system prompt: %w", err)
	}

	if result.UpsertedCount > 0 {
		s.logger.Info("System prompt created",
			zap.String("userId", userID),
			zap.String("companyId", companyID))
	} else {
		s.logger.Info("System prompt updated",
			zap.String("userId", userID),
			zap.String("companyId", companyID))
	}

	return nil
}

// ListSubagents retrieves all subagents for a user within their company
func (s *AISettingsService) ListSubagents(ctx context.Context, userID, companyID string) ([]models.Subagent, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Latest first

	cursor, err := s.subagentsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer cursor.Close(ctx)

	var subagents []models.Subagent
	if err := cursor.All(ctx, &subagents); err != nil {
		return nil, fmt.Errorf("failed to decode subagents: %w", err)
	}

	return subagents, nil
}

// GetSubagent retrieves a specific subagent by ID
func (s *AISettingsService) GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error) {
	var subagent models.Subagent
	filter := bson.M{
		"_id":       id,
		"companyId": companyID, // Company-level isolation
	}

	err := s.subagentsCollection.FindOne(ctx, filter).Decode(&subagent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subagent not found or access denied")
		}
		return nil, fmt.Errorf("failed to retrieve subagent: %w", err)
	}

	return &subagent, nil
}

// CreateSubagent creates a new subagent for a user
func (s *AISettingsService) CreateSubagent(ctx context.Context, userID, companyID, name, description, systemPrompt string) (*models.Subagent, error) {
	now := time.Now().UTC()
	subagent := &models.Subagent{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		CompanyID:    companyID,
		Name:         name,
		Description:  description,
		SystemPrompt: systemPrompt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.subagentsCollection.InsertOne(ctx, subagent)
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent: %w", err)
	}

	s.logger.Info("Subagent created",
		zap.String("subagentId", subagent.ID.Hex()),
		zap.String("name", name),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	return subagent, nil
}

// UpdateSubagent updates an existing subagent
func (s *AISettingsService) UpdateSubagent(ctx context.Context, id primitive.ObjectID, userID, companyID, name, description, systemPrompt string) (*models.Subagent, error) {
	// Verify subagent exists and belongs to user
	existingSubagent, err := s.GetSubagent(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	if existingSubagent.UserID != userID {
		return nil, fmt.Errorf("unauthorized: subagent does not belong to user")
	}

	update := bson.M{
		"$set": bson.M{
			"name":         name,
			"description":  description,
			"systemPrompt": systemPrompt,
			"updatedAt":    time.Now().UTC(),
		},
	}

	result, err := s.subagentsCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update subagent: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("subagent not found")
	}

	s.logger.Info("Subagent updated",
		zap.String("subagentId", id.Hex()),
		zap.String("userId", userID))

	// Retrieve updated subagent
	return s.GetSubagent(ctx, id, companyID)
}

// DeleteSubagent deletes a subagent
func (s *AISettingsService) DeleteSubagent(ctx context.Context, id primitive.ObjectID, userID, companyID string) error {
	// Verify subagent belongs to user and company (authorization)
	subagent, err := s.GetSubagent(ctx, id, companyID)
	if err != nil {
		return err
	}

	if subagent.UserID != userID {
		return fmt.Errorf("unauthorized: subagent does not belong to user")
	}

	// Delete the subagent
	result, err := s.subagentsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete subagent: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("subagent not found")
	}

	s.logger.Info("Subagent deleted",
		zap.String("subagentId", id.Hex()),
		zap.String("userId", userID))

	return nil
}

// ListClaudeAgents reads and parses .claude/agents/*.md files
func (s *AISettingsService) ListClaudeAgents(ctx context.Context) ([]models.ClaudeAgent, error) {
	agents := []models.ClaudeAgent{}

	// Read .claude/agents directory
	pattern := ".claude/agents/*.md"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to read .claude/agents directory: %w", err)
	}

	// Handle missing directory gracefully
	if len(files) == 0 {
		s.logger.Debug("No Claude agent files found in .claude/agents/")
		return agents, nil
	}

	// Parse each markdown file
	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			s.logger.Warn("Failed to read agent file",
				zap.String("file", filePath),
				zap.Error(err))
			continue
		}

		agent, err := s.parseClaudeAgentMarkdown(string(content))
		if err != nil {
			s.logger.Warn("Failed to parse agent file",
				zap.String("file", filePath),
				zap.Error(err))
			continue
		}

		agents = append(agents, agent)
	}

	s.logger.Info("Listed Claude agents",
		zap.Int("count", len(agents)))

	return agents, nil
}

// parseClaudeAgentMarkdown parses markdown with YAML frontmatter
func (s *AISettingsService) parseClaudeAgentMarkdown(content string) (models.ClaudeAgent, error) {
	// Split on '---' delimiters to extract frontmatter
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return models.ClaudeAgent{}, fmt.Errorf("invalid markdown format: missing frontmatter delimiters")
	}

	// Parse YAML frontmatter (second part, index 1)
	frontmatter := strings.TrimSpace(parts[1])
	var metadata map[string]interface{}
	err := yaml.Unmarshal([]byte(frontmatter), &metadata)
	if err != nil {
		return models.ClaudeAgent{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Extract system prompt (content after second '---')
	systemPrompt := strings.TrimSpace(strings.Join(parts[2:], "---"))

	// Build ClaudeAgent struct
	agent := models.ClaudeAgent{
		SystemPrompt: systemPrompt,
	}

	// Extract metadata fields with type assertions
	if name, ok := metadata["name"].(string); ok {
		agent.Name = name
	}
	if description, ok := metadata["description"].(string); ok {
		agent.Description = description
	}
	if model, ok := metadata["model"].(string); ok {
		agent.Model = model
	}
	if color, ok := metadata["color"].(string); ok {
		agent.Color = color
	}

	return agent, nil
}

// ImportClaudeAgents imports Claude agents as subagents
func (s *AISettingsService) ImportClaudeAgents(ctx context.Context, userID, companyID string, agentNames []string) (int, []string, error) {
	imported := 0
	errors := []string{}

	// Get existing subagents to check for duplicates
	existingSubagents, err := s.ListSubagents(ctx, userID, companyID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to check existing subagents: %w", err)
	}

	// Create map of existing names for quick lookup
	existingNames := make(map[string]bool)
	for _, subagent := range existingSubagents {
		existingNames[subagent.Name] = true
	}

	// Import each agent
	for _, agentName := range agentNames {
		// Skip if already exists
		if existingNames[agentName] {
			errors = append(errors, fmt.Sprintf("Agent '%s' already exists, skipping", agentName))
			s.logger.Debug("Skipping duplicate agent", zap.String("name", agentName))
			continue
		}

		// Read agent file
		filePath := fmt.Sprintf(".claude/agents/%s.md", agentName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read agent file '%s': %v", agentName, err)
			errors = append(errors, errMsg)
			s.logger.Warn("Failed to read agent file",
				zap.String("name", agentName),
				zap.Error(err))
			continue
		}

		// Parse markdown
		agent, err := s.parseClaudeAgentMarkdown(string(content))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to parse agent file '%s': %v", agentName, err)
			errors = append(errors, errMsg)
			s.logger.Warn("Failed to parse agent file",
				zap.String("name", agentName),
				zap.Error(err))
			continue
		}

		// Create subagent using existing CreateSubagent method
		_, err = s.CreateSubagent(ctx, userID, companyID, agent.Name, agent.Description, agent.SystemPrompt)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create subagent '%s': %v", agentName, err)
			errors = append(errors, errMsg)
			s.logger.Error("Failed to create subagent",
				zap.String("name", agentName),
				zap.Error(err))
			continue
		}

		imported++
		s.logger.Info("Imported Claude agent as subagent",
			zap.String("name", agent.Name),
			zap.String("userId", userID))
	}

	return imported, errors, nil
}
