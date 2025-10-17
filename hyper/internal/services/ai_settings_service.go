package services

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// defaultSystemPrompt is the default system prompt that ships with the application
// This matches the constant in handlers/chat_websocket.go
const defaultSystemPrompt = `You are an AI development assistant with access to powerful tools for code analysis, file operations, and task execution.

KEY CAPABILITIES:
1. **Autonomous File Discovery**: You have code_index_search tool for semantic code search. Use it FIRST before asking users for file paths.
2. **Code Understanding**: Use code_index_search to find relevant functions, classes, or patterns semantically.
3. **File Operations**: You can read, write, and list files directly using read_file, write_file, list_directory tools.
4. **Tool Execution**: Execute bash commands, apply patches, and run project-specific operations.

AUTONOMOUS WORKFLOW (CRITICAL):
When asked to modify, fix, or analyze code:
1. **NEVER ask for file paths** - use code_index_search first with relevant semantic query
2. Find the right files automatically using search results
3. Read files to understand context
4. Make changes directly using write_file or apply_patch
5. Verify changes if requested

Example: If asked "fix the authentication bug", you should:
- Search: code_index_search(query: "authentication login jwt token", limit: 5)
- Analyze results to find relevant files
- Read those files
- Implement fix
- NOT ask "which file should I modify?"

TOOL USAGE RULES (PREVENT INFINITE LOOPS):
1. **NEVER call the same tool with identical arguments twice in a row**
2. **If a tool returns a result, USE that result** - don't call it again expecting different output
3. **Track what you've already done** - if you listed a directory and didn't find what you need, try a different approach (search, bash find, etc.)
4. **If a tool fails or returns empty, try a DIFFERENT tool or DIFFERENT arguments** - repeating won't help
5. **Circuit breaker protection**: System will stop you after 3 identical tool calls - avoid this by being smart about tool usage

Examples of BAD patterns to AVOID:
❌ list_directory(./components) → list_directory(./components) → list_directory(./components)
❌ read_file(config.ts) fails → read_file(config.ts) → read_file(config.ts)
❌ bash("find . -name foo") returns nothing → bash("find . -name foo") → bash("find . -name foo")

Examples of GOOD patterns:
✅ list_directory(./components) → see files → read_file(specific file)
✅ read_file(config.ts) fails → try bash("ls -la config.ts") or code_index_search
✅ bash("find . -name foo") returns nothing → try different search: bash("find . -name '*foo*'") or code_index_search

TOOL USAGE:
- code_index_search: Semantic code search (use for finding files, functions, patterns)
- read_file: Read file contents (after finding via search)
- write_file: Write/overwrite files
- list_directory: List directory contents
- bash: Execute shell commands (testing, building, etc.)

Be proactive, autonomous, and leverage your tools efficiently. If stuck, change your approach - don't retry the same failing operation.`

// AISettingsService manages system prompts and subagents with MongoDB storage
type AISettingsService struct {
	systemPromptsCollection         *mongo.Collection
	systemPromptVersionsCollection  *mongo.Collection
	subagentsCollection             *mongo.Collection
	logger                          *zap.Logger
}

// NewAISettingsService creates a new AI settings service instance
func NewAISettingsService(db *mongo.Database, logger *zap.Logger) (*AISettingsService, error) {
	service := &AISettingsService{
		systemPromptsCollection:        db.Collection("system_prompts"),
		systemPromptVersionsCollection: db.Collection("system_prompt_versions"),
		subagentsCollection:            db.Collection("subagents"),
		logger:                         logger,
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

	// Index on system_prompt_versions: {userId, companyId} for user version queries
	_, err = service.systemPromptVersionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system_prompt_versions user index: %w", err)
	}

	// Index on system_prompt_versions: {userId, companyId, isActive} for active version lookup
	_, err = service.systemPromptVersionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "companyId", Value: 1},
			{Key: "isActive", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system_prompt_versions active index: %w", err)
	}

	// Index on system_prompt_versions: {isDefault} for default prompt lookup
	_, err = service.systemPromptVersionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "isDefault", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system_prompt_versions default index: %w", err)
	}

	logger.Info("AI settings service initialized with MongoDB indexes")
	return service, nil
}

// GetSystemPrompt retrieves the active system prompt for a user from version control
func (s *AISettingsService) GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error) {
	// Get the active version from version control system
	activeVersion, err := s.GetActiveSystemPromptVersion(ctx, userID, companyID)
	if err != nil {
		return "", err
	}

	// If no active version, return empty (user hasn't created any custom versions)
	if activeVersion == nil {
		return "", nil
	}

	return activeVersion.Prompt, nil
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

// ========================================
// System Prompt Version Control Methods
// ========================================

// ListSystemPromptVersions retrieves all system prompt versions for a user
// Automatically migrates legacy system prompts to version control on first access
func (s *AISettingsService) ListSystemPromptVersions(ctx context.Context, userID, companyID string) ([]models.SystemPromptVersion, error) {
	// Check if migration is needed
	err := s.migrateLegacySystemPrompt(ctx, userID, companyID)
	if err != nil {
		s.logger.Warn("Failed to migrate legacy system prompt",
			zap.String("userId", userID),
			zap.String("companyId", companyID),
			zap.Error(err))
		// Continue even if migration fails
	}

	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	opts := options.Find().SetSort(bson.D{{Key: "version", Value: -1}}) // Latest version first

	cursor, err := s.systemPromptVersionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query system prompt versions: %w", err)
	}
	defer cursor.Close(ctx)

	var versions []models.SystemPromptVersion
	if err := cursor.All(ctx, &versions); err != nil {
		return nil, fmt.Errorf("failed to decode system prompt versions: %w", err)
	}

	return versions, nil
}

// GetSystemPromptVersion retrieves a specific system prompt version by ID
func (s *AISettingsService) GetSystemPromptVersion(ctx context.Context, id primitive.ObjectID, companyID string) (*models.SystemPromptVersion, error) {
	var version models.SystemPromptVersion
	filter := bson.M{
		"_id":       id,
		"companyId": companyID,
	}

	err := s.systemPromptVersionsCollection.FindOne(ctx, filter).Decode(&version)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("system prompt version not found or access denied")
		}
		return nil, fmt.Errorf("failed to retrieve system prompt version: %w", err)
	}

	return &version, nil
}

// GetActiveSystemPromptVersion retrieves the active system prompt version for a user
func (s *AISettingsService) GetActiveSystemPromptVersion(ctx context.Context, userID, companyID string) (*models.SystemPromptVersion, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
		"isActive":  true,
	}

	var version models.SystemPromptVersion
	err := s.systemPromptVersionsCollection.FindOne(ctx, filter).Decode(&version)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No active version found
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve active system prompt version: %w", err)
	}

	return &version, nil
}

// CreateSystemPromptVersion creates a new system prompt version
func (s *AISettingsService) CreateSystemPromptVersion(ctx context.Context, userID, companyID, prompt, description string, activate bool) (*models.SystemPromptVersion, error) {
	// Get the next version number
	nextVersion, err := s.getNextVersionNumber(ctx, userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	now := time.Now().UTC()
	version := &models.SystemPromptVersion{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		CompanyID:   companyID,
		Version:     nextVersion,
		Prompt:      prompt,
		Description: description,
		IsActive:    activate,
		IsDefault:   false,
		CreatedAt:   now,
		CreatedBy:   userID,
	}

	// If activate is true, deactivate all other versions first
	if activate {
		err = s.deactivateAllVersions(ctx, userID, companyID)
		if err != nil {
			return nil, fmt.Errorf("failed to deactivate existing versions: %w", err)
		}
	}

	// Insert the new version
	_, err = s.systemPromptVersionsCollection.InsertOne(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt version: %w", err)
	}

	s.logger.Info("System prompt version created",
		zap.String("versionId", version.ID.Hex()),
		zap.Int("version", version.Version),
		zap.Bool("isActive", version.IsActive),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	return version, nil
}

// ActivateSystemPromptVersion sets a specific version as active
func (s *AISettingsService) ActivateSystemPromptVersion(ctx context.Context, id primitive.ObjectID, userID, companyID string) error {
	// Verify version exists and belongs to user
	version, err := s.GetSystemPromptVersion(ctx, id, companyID)
	if err != nil {
		return err
	}

	if version.UserID != userID {
		return fmt.Errorf("unauthorized: version does not belong to user")
	}

	if version.IsDefault {
		return fmt.Errorf("cannot activate the default system prompt version")
	}

	// Deactivate all other versions
	err = s.deactivateAllVersions(ctx, userID, companyID)
	if err != nil {
		return fmt.Errorf("failed to deactivate existing versions: %w", err)
	}

	// Activate this version
	update := bson.M{
		"$set": bson.M{
			"isActive": true,
		},
	}

	result, err := s.systemPromptVersionsCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to activate version: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("version not found")
	}

	s.logger.Info("System prompt version activated",
		zap.String("versionId", id.Hex()),
		zap.Int("version", version.Version),
		zap.String("userId", userID))

	return nil
}

// DeleteSystemPromptVersion deletes a system prompt version
func (s *AISettingsService) DeleteSystemPromptVersion(ctx context.Context, id primitive.ObjectID, userID, companyID string) error {
	// Verify version exists and belongs to user
	version, err := s.GetSystemPromptVersion(ctx, id, companyID)
	if err != nil {
		return err
	}

	if version.UserID != userID {
		return fmt.Errorf("unauthorized: version does not belong to user")
	}

	if version.IsDefault {
		return fmt.Errorf("cannot delete the default system prompt version")
	}

	if version.IsActive {
		return fmt.Errorf("cannot delete the active version - activate another version first")
	}

	// Delete the version
	result, err := s.systemPromptVersionsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("version not found")
	}

	s.logger.Info("System prompt version deleted",
		zap.String("versionId", id.Hex()),
		zap.Int("version", version.Version),
		zap.String("userId", userID))

	return nil
}

// GetDefaultSystemPrompt retrieves the default system prompt (read-only)
// Returns the hardcoded default prompt constant
func (s *AISettingsService) GetDefaultSystemPrompt(ctx context.Context) (string, error) {
	// Return the default prompt constant that ships with the application
	return defaultSystemPrompt, nil
}

// ========================================
// Helper Methods
// ========================================

// getNextVersionNumber gets the next version number for a user
func (s *AISettingsService) getNextVersionNumber(ctx context.Context, userID, companyID string) (int, error) {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})

	var latestVersion models.SystemPromptVersion
	err := s.systemPromptVersionsCollection.FindOne(ctx, filter, opts).Decode(&latestVersion)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No versions exist yet, start at 1
			return 1, nil
		}
		return 0, fmt.Errorf("failed to find latest version: %w", err)
	}

	return latestVersion.Version + 1, nil
}

// deactivateAllVersions deactivates all active versions for a user
func (s *AISettingsService) deactivateAllVersions(ctx context.Context, userID, companyID string) error {
	filter := bson.M{
		"userId":    userID,
		"companyId": companyID,
		"isActive":  true,
	}

	update := bson.M{
		"$set": bson.M{
			"isActive": false,
		},
	}

	_, err := s.systemPromptVersionsCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate versions: %w", err)
	}

	return nil
}

// migrateLegacySystemPrompt migrates a legacy system prompt from system_prompts collection
// to the new system_prompt_versions collection as version 1
func (s *AISettingsService) migrateLegacySystemPrompt(ctx context.Context, userID, companyID string) error {
	// Check if there are already versions for this user
	existingVersionsFilter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}
	count, err := s.systemPromptVersionsCollection.CountDocuments(ctx, existingVersionsFilter)
	if err != nil {
		return fmt.Errorf("failed to count existing versions: %w", err)
	}

	// If versions already exist, no migration needed
	if count > 0 {
		return nil
	}

	// Try to find a legacy system prompt - first try exact match
	var legacyPrompt models.SystemPrompt
	legacyFilter := bson.M{
		"userId":    userID,
		"companyId": companyID,
	}

	err = s.systemPromptsCollection.FindOne(ctx, legacyFilter).Decode(&legacyPrompt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No exact match, try to find ANY legacy prompt in the collection
			// This handles the case where old prompts were created with different user IDs
			err = s.systemPromptsCollection.FindOne(ctx, bson.M{}).Decode(&legacyPrompt)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					// No legacy prompt exists at all, nothing to migrate
					return nil
				}
				return fmt.Errorf("failed to find any legacy system prompt: %w", err)
			}

			s.logger.Info("Found legacy system prompt with different user/company IDs, migrating to current user",
				zap.String("oldUserId", legacyPrompt.UserID),
				zap.String("oldCompanyId", legacyPrompt.CompanyID),
				zap.String("newUserId", userID),
				zap.String("newCompanyId", companyID))
		} else {
			return fmt.Errorf("failed to find legacy system prompt: %w", err)
		}
	}

	// Create version 1 from the legacy prompt
	now := time.Now().UTC()
	version := &models.SystemPromptVersion{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		CompanyID:   companyID,
		Version:     1,
		Prompt:      legacyPrompt.Prompt,
		Description: "Migrated from legacy system_prompts collection",
		IsActive:    true, // Make it active by default
		IsDefault:   false,
		CreatedAt:   now,
		CreatedBy:   userID,
	}

	_, err = s.systemPromptVersionsCollection.InsertOne(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to insert migrated version: %w", err)
	}

	s.logger.Info("Successfully migrated legacy system prompt to version control",
		zap.String("userId", userID),
		zap.String("companyId", companyID),
		zap.String("versionId", version.ID.Hex()))

	return nil
}
