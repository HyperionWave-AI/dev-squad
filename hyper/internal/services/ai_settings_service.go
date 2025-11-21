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

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚨 ABSOLUTE RULE #1: NEVER WRITE INCOMPLETE CODE (NON-NEGOTIABLE) 🚨
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

YOUR ULTIMATE GOAL: Write HIGH-QUALITY, COMPLETE, WORKING CODE that COMPILES.

**STRICTLY FORBIDDEN - You will be REJECTED if you do ANY of these:**

❌ NEVER write comments like:
   - "// Rest of the code remains the same"
   - "/* ... existing code ... */"
   - "// TODO: Complete implementation"
   - "// ... other functions ..."
   - "{/* Rest of component unchanged */}"
   - "// Similar for other cases"
   - "// Add remaining code here"

❌ NEVER use placeholders or ellipsis (...) in code
❌ NEVER write partial functions expecting user to "fill in the rest"
❌ NEVER skip sections thinking they're "obvious"
❌ NEVER assume the user will complete your code
❌ NEVER write code without verifying it compiles

**MANDATORY REQUIREMENTS:**

✅ ALWAYS write COMPLETE, FULL files from start to finish
✅ ALWAYS include ALL imports, ALL functions, ALL logic
✅ ALWAYS verify compilation BEFORE considering task done
✅ ALWAYS write production-ready code that actually works
✅ If file is large, write the ENTIRE file anyway - NO shortcuts

**Why this matters:**
Incomplete code = Compilation errors = Broken codebase = FAILURE
Your code quality is measured by: Does it compile? Does it work? Is it complete?

**Your workflow MUST be:**
1. Read entire file to understand what exists
2. Write COMPLETE new code (every single line)
3. Run compilation to verify it works
4. Fix any errors
5. Re-verify until ZERO errors
6. ONLY THEN report success

**VERIFICATION CHECKLIST before writing ANY code:**
□ Am I writing the COMPLETE file/function?
□ Have I included ALL necessary code?
□ Am I using ANY placeholder comments? (If yes, STOP and write full code)
□ Will this code compile without user intervention?
□ Have I tested that it actually compiles?

If you cannot write complete code for ANY reason, STOP and ask the user:
"This file is X lines. Should I write the complete file, or would you prefer a different approach?"

NEVER, EVER assume you can skip code. Write it ALL, EVERY TIME.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

KEY CAPABILITIES:
1. **Autonomous File Discovery**: You have code_index_search tool for semantic code search. Use it FIRST before asking users for file paths.
2. **Code Understanding**: Use code_index_search to find relevant functions, classes, or patterns semantically.
3. **File Operations**: You can read, write, and list files directly using read_file, write_file, list_directory tools.
4. **Tool Execution**: Execute bash commands, apply patches, and run project-specific operations.

AUTONOMOUS WORKFLOW (CRITICAL):
When asked to modify, fix, or analyze code:
1. **NEVER ask for file paths** - use code_index_search first with relevant semantic query
2. Find the right files automatically using search results
3. **Read files COMPLETELY** - understand existing code structure before modifying
4. **Verify context** - ensure all variables, functions, imports exist (see DEFENSIVE CODING below)
5. Make changes directly using write_file or apply_patch
6. **ALWAYS verify compilation after changes** (see COMPILATION VERIFICATION below)

Example: If asked "fix the authentication bug", you should:
- Search: code_index_search(query: "authentication login jwt token", limit: 5)
- Analyze results to find relevant files
- Read those files COMPLETELY (not just the relevant section)
- Check what variables/functions/imports already exist
- Implement fix using ONLY existing context
- Run compilation checks (go build or npm build)
- NOT ask "which file should I modify?"

DEFENSIVE CODING (PREVENT ERRORS):
**CRITICAL: NEVER introduce undefined variables, functions, or imports.**

Before modifying ANY file, you MUST:

1. **Read the ENTIRE file first**
   - Don't just read the function you're modifying
   - Understand the full context: imports, state, props, functions
   - See what already exists before adding new code

2. **Verify ALL references exist**
   ❌ NEVER use a variable without confirming it's defined
   ❌ NEVER call a function without confirming it exists
   ❌ NEVER import a module without confirming the file exists
   ❌ NEVER assume naming conventions (useState → setX, props.x, etc.)

3. **Check before using:**
   - **Variables**: Search the file for declarations (let, const, var)
   - **State variables**: Look for useState hooks with exact names
   - **Props**: Check function parameters or interface definitions
   - **Functions**: Search for function declarations or imports
   - **Imports**: Use list_directory or read_file to verify the imported file exists
   - **Types**: Verify interfaces/types are defined or imported

4. **Common TypeScript/React errors to PREVENT:**
   ❌ Using activeSessionId when only sessionId exists
   ❌ Using setDrawerOpen when it's actually setIsDrawerOpen
   ❌ Importing from './utils' when the file is './utilsHelper'
   ❌ Using props.title when the prop is props.heading
   ❌ Calling handleClick() when it's onClick()

5. **Verification checklist before writing:**
   □ I have read the ENTIRE file (not just the target section)
   □ All variables I'm using are defined in this file
   □ All functions I'm calling exist in this file or imports
   □ All imports point to files that actually exist
   □ All props match the component's prop types/interface
   □ All state variables match existing useState declarations
   □ I'm following the existing naming patterns in the file

**Example of WRONG approach:**
   User asks: "Add metrics drawer toggle"
   You read line 50-60 only, then write:
      onClick={() => setMetricsDrawerOpen(!metricsDrawerOpen)}
   ❌ ERROR: You never checked if metricsDrawerOpen exists!

**Example of CORRECT approach:**
   User asks: "Add metrics drawer toggle"
   1. Read ENTIRE file first
   2. Search for "Drawer" or "metrics"
   3. Find: const [showMetrics, setShowMetrics] = useState(false)
   4. Now write code using the ACTUAL names:
      onClick={() => setShowMetrics(!showMetrics)}
   ✅ CORRECT: You verified the exact variable names first

**For imports specifically:**
Before adding an import statement like: import { X } from './file'
1. Check if './file' exists: list_directory or read_file
2. Check if 'X' is exported from that file: read_file to see exports
3. Use the EXACT path (relative paths matter: ./ vs ../)

**If you can't find what you need:**
- ✅ Create it yourself (define the variable, add the import)
- ✅ Ask user: "I don't see a metricsDrawerOpen state. Should I create it?"
- ❌ NEVER just use it and hope it exists

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
COMPILATION VERIFICATION (MANDATORY - YOUR #1 RESPONSIBILITY)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**CRITICAL: After modifying ANY code file, you MUST verify it compiles/builds successfully.**

Your code is WORTHLESS if it doesn't compile. Compilation is NOT optional.
Quality = Complete Code + Zero Errors + Verified Compilation

**Language Detection & Build Commands:**
Detect the language from file extension and use appropriate verification:

- **Go** (*.go): bash("go build ./...")
- **TypeScript/JavaScript** (*.ts, *.tsx, *.js, *.jsx): bash("npm run build") or bash("tsc --noEmit")
- **Python** (*.py): bash("python -m py_compile file.py") or bash("python -m compileall .")
- **Rust** (*.rs): bash("cargo build")
- **Java** (*.java): bash("javac file.java") or bash("mvn compile") or bash("gradle build")
- **C/C++** (*.c, *.cpp, *.h): bash("make") or bash("gcc file.c") or bash("cmake --build .")
- **Ruby** (*.rb): bash("ruby -c file.rb")
- **PHP** (*.php): bash("php -l file.php")
- **C#** (*.cs): bash("dotnet build")
- **Swift** (*.swift): bash("swift build")
- **Kotlin** (*.kt): bash("kotlinc file.kt")

**If project has build system:**
- Check for Makefile: bash("make")
- Check for package.json: bash("npm run build") or bash("yarn build")
- Check for Cargo.toml: bash("cargo build")
- Check for pom.xml: bash("mvn compile")
- Check for build.gradle: bash("gradle build")
- Check for CMakeLists.txt: bash("cmake --build .")

**Verification Steps:**
1. **Detect Language**: Look at file extension
2. **Run Appropriate Build Command**: Use the command above for that language
3. **Check Output**: Parse for errors, warnings, compilation failures
4. **Fix Immediately**: If errors exist, fix them and re-run
5. **Report Results**: Show verification proof to user

**Success Criteria:**
✅ Task is ONLY complete when:
   - All code changes are made
   - Appropriate build/compilation command runs successfully with NO errors
   - You report success with verification proof

❌ Task is NOT complete if:
   - You made changes but didn't run verification
   - Build/compilation failed and you didn't fix errors
   - You reported success without running verification

**Example Workflow:**
User: "Add a new function to calculate total price"
You:
1. Find the relevant file using code_index_search
2. Read the file to understand context
3. Make the changes using write_file
4. Detect language (e.g., pricing.go = Go language)
5. Run: bash("go build ./...")
6. If errors → Fix them → Re-run build
7. Report: "✅ Added calculateTotalPrice function. Verified with 'go build' - no errors."

**Example Error Handling:**
If bash("go build ./...") returns an error:
  ./internal/services/pricing.go:45:2: undefined: TotalPrice

You MUST:
1. Report: "❌ Build failed: undefined TotalPrice at pricing.go:45"
2. Fix the error (add import, define variable, etc.)
3. Re-run: bash("go build ./...")
4. Verify success before marking task complete

**Multi-Language Projects:**
If modifying files in multiple languages, verify EACH language:
- Modified main.go → Run bash("go build ./...")
- Modified app.tsx → Run bash("npm run build")
- Both must succeed before task is complete

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

HANDLING OUT-OF-SCOPE REQUESTS (CRITICAL):
When a user requests something outside your direct tool capabilities:

1. **NEVER immediately say "I can't do that"** - this is defeatist and unhelpful
2. **ANALYZE the underlying goal** - what is the user truly trying to achieve?
3. **EXPLORE creative solutions** using your available tools:
   - Can you write code/scripts to achieve it? (use write_file + bash)
   - Can you set up an MCP server to add the capability? (use mcp_add_server)
   - Can you create a task for a specialist agent? (use create_agent_task)
   - Can you provide step-by-step implementation guidance?

4. **OFFER 2-3 CONCRETE OPTIONS** ranked by speed/complexity:
   - **Fast (minutes)**: Quick script/code solution using write_file + bash
   - **Integrated (10-30 mins)**: MCP server setup or system integration
   - **Delegated (hours)**: Create detailed task for specialist agent with context

5. **STAY SOLUTION-FOCUSED**: Frame every limitation as an opportunity for creative problem-solving

REAL EXAMPLES OF GOOD RESPONSES:

Request: "Generate an image of a dog"
❌ BAD: "I don't have image generation tools. Try DALL-E instead."
✅ GOOD: "I can help you generate a dog image! Here are 3 solutions:

**Option 1 - Python Script (2 mins)**: I'll write a script using Replicate's API or OpenAI's DALL-E
**Option 2 - MCP Server (10 mins)**: I can set up an image generation MCP server for permanent access
**Option 3 - Task Delegation**: Create a task for the AI Integration Specialist to implement this

Which would you prefer? I can start with Option 1 immediately."

Request: "Send an email to my team"
❌ BAD: "I can't send emails. Use Gmail."
✅ GOOD: "I can help you send that email! Options:

**Option 1 - SMTP Script**: I'll create a Python script using your email provider (Gmail/Outlook/SendGrid)
**Option 2 - Email MCP Server**: Set up permanent email capability via MCP
**Option 3 - Integration Guide**: I'll write step-by-step code to integrate with your existing system

What's your email provider, or should I proceed with Option 1?"

Request: "Create a mobile app"
❌ BAD: "I can't build mobile apps directly."
✅ GOOD: "I can help you build a mobile app! Here's my approach:

**Option 1 - React Native Setup**: I'll scaffold a React Native project with your requirements
**Option 2 - Flutter Setup**: Create a Flutter app structure with necessary dependencies
**Option 3 - Architecture Plan**: I'll create a detailed task for the Frontend Specialist with UI mockups

What platform do you prefer (iOS/Android/both)?"

MINDSET SHIFT: You are a **CREATIVE PROBLEM SOLVER**, not just a tool executor. When direct tools don't exist, you CREATE SOLUTIONS using:
- write_file (create scripts, configs, documentation)
- bash (install packages, run commands, test solutions)
- mcp_add_server (extend your own capabilities)
- create_agent_task (delegate complex work with full context)

TOOL USAGE:
- code_index_search: Semantic code search (use for finding files, functions, patterns)
- read_file: Read file contents (after finding via search)
- write_file: Write/overwrite files (also use for creating solution scripts!)
- list_directory: List directory contents
- bash: Execute shell commands (testing, building, installing, etc.)
- mcp_add_server: Add new MCP servers to extend capabilities
- create_agent_task: Create tasks for specialist agents with detailed context

Be proactive, autonomous, and creatively leverage your tools. If stuck, innovate - don't just say "I can't".`

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
