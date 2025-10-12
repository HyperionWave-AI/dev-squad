package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// KnowledgePromptHandler manages knowledge management prompts for Qdrant optimization
type KnowledgePromptHandler struct{}

// NewKnowledgePromptHandler creates a new knowledge prompt handler
func NewKnowledgePromptHandler() *KnowledgePromptHandler {
	return &KnowledgePromptHandler{}
}

// RegisterKnowledgePrompts registers all knowledge management prompts with the MCP server
func (h *KnowledgePromptHandler) RegisterKnowledgePrompts(server *mcp.Server) error {
	// Register recommend_qdrant_query prompt
	if err := h.registerRecommendQdrantQuery(server); err != nil {
		return fmt.Errorf("failed to register recommend_qdrant_query prompt: %w", err)
	}

	// Register suggest_knowledge_structure prompt
	if err := h.registerSuggestKnowledgeStructure(server); err != nil {
		return fmt.Errorf("failed to register suggest_knowledge_structure prompt: %w", err)
	}

	// Register guide_knowledge_storage prompt
	if err := h.registerGuideKnowledgeStorage(server); err != nil {
		return fmt.Errorf("failed to register guide_knowledge_storage prompt: %w", err)
	}

	return nil
}

// registerRecommendQdrantQuery registers the recommend_qdrant_query prompt
func (h *KnowledgePromptHandler) registerRecommendQdrantQuery(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "recommend_qdrant_query",
		Description: "Analyze what an agent needs to know and recommend the optimal Qdrant query strategy to find it efficiently.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "agentQuestion",
				Description: "What the agent wants to know or problem they need to solve",
				Required:    true,
			},
			{
				Name:        "taskContext",
				Description: "Current task context including squad, service, feature being worked on",
				Required:    true,
			},
			{
				Name:        "availableCollections",
				Description: "Comma-separated list of available Qdrant collections to search (optional)",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		agentQuestion := ""
		taskContext := ""
		availableCollectionsStr := ""

		if req.Params != nil && req.Params.Arguments != nil {
			agentQuestion = req.Params.Arguments["agentQuestion"]
			taskContext = req.Params.Arguments["taskContext"]
			availableCollectionsStr = req.Params.Arguments["availableCollections"]
		}

		if agentQuestion == "" || taskContext == "" {
			return nil, fmt.Errorf("agentQuestion and taskContext are required arguments")
		}

		// Parse comma-separated collections
		var availableCollections []string
		if availableCollectionsStr != "" {
			parts := strings.Split(availableCollectionsStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					availableCollections = append(availableCollections, trimmed)
				}
			}
		}

		promptText := h.buildQdrantQueryRecommendation(agentQuestion, taskContext, availableCollections)

		return &mcp.GetPromptResult{
			Description: "Qdrant query optimization recommendation",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: promptText,
					},
				},
			},
		}, nil
	}

	server.AddPrompt(prompt, handler)
	return nil
}

// registerSuggestKnowledgeStructure registers the suggest_knowledge_structure prompt
func (h *KnowledgePromptHandler) registerSuggestKnowledgeStructure(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "suggest_knowledge_structure",
		Description: "Help agents structure their learnings and solutions for optimal Qdrant storage and future reuse.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "rawLearning",
				Description: "What the agent learned, discovered, or implemented (unstructured)",
				Required:    true,
			},
			{
				Name:        "context",
				Description: "JSON task context including squad, service, files modified, and task type",
				Required:    true,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		rawLearning := ""
		contextStr := ""

		if req.Params != nil && req.Params.Arguments != nil {
			rawLearning = req.Params.Arguments["rawLearning"]
			contextStr = req.Params.Arguments["context"]
		}

		if rawLearning == "" || contextStr == "" {
			return nil, fmt.Errorf("rawLearning and context are required arguments")
		}

		// Parse context JSON
		var taskContext map[string]interface{}
		if err := json.Unmarshal([]byte(contextStr), &taskContext); err != nil {
			return nil, fmt.Errorf("invalid context JSON: %w", err)
		}

		promptText := h.buildKnowledgeStructurePrompt(rawLearning, taskContext)

		return &mcp.GetPromptResult{
			Description: "Knowledge structuring guidance",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: promptText,
					},
				},
			},
		}, nil
	}

	server.AddPrompt(prompt, handler)
	return nil
}

// buildQdrantQueryRecommendation builds the Qdrant query recommendation prompt
func (h *KnowledgePromptHandler) buildQdrantQueryRecommendation(agentQuestion, taskContext string, availableCollections []string) string {
	collectionsSection := ""
	if len(availableCollections) > 0 {
		collectionsSection = fmt.Sprintf(`
## Available Qdrant Collections
%s

Consider which collection(s) are most likely to contain relevant knowledge.
`, "- "+strings.Join(availableCollections, "\n- "))
	} else {
		collectionsSection = `
## Standard Qdrant Collections

**Task Collections:**
- task:hyperion://task/human/{taskId} - Task-specific knowledge
- team-coordination - Cross-squad coordination
- agent-coordination - Agent handoffs

**Technical Collections:**
- technical-knowledge - Reusable patterns, solutions
- code-patterns - Code examples and snippets
- adr - Architecture Decision Records
- data-contracts - API schemas and contracts
- technical-debt-registry - Known technical debt

**UI Collections:**
- ui-component-patterns - React components, Radix UI
- ui-test-strategies - Testing approaches
- ui-accessibility-standards - ARIA, WCAG patterns
- ui-visual-regression-baseline - Visual test baselines

**Operations:**
- mcp-operations - MCP server operations
- code-quality-violations - Code quality issues
`
	}

	return fmt.Sprintf(`# Qdrant Query Optimization

## Agent's Question
%s

## Current Task Context
%s
%s
## Your Mission
Recommend the OPTIMAL Qdrant query strategy to help this agent find what they need EFFICIENTLY.

## Analysis Framework

### Step 1: Understand the Need
Analyze the agent's question:
- **Type of knowledge needed:** Pattern/example, bug solution, architecture decision, API contract, etc.
- **Specificity level:** Very specific (exact error) vs general (design pattern)
- **Urgency:** Blocking issue vs exploratory research
- **Scope:** Single service, cross-service, platform-wide

### Step 2: Select Best Collection(s)
Based on the need, choose 1-2 collections maximum:

**Choose based on:**
- Bug/error → technical-knowledge, code-patterns (specific to domain)
- Pattern/approach → code-patterns, adr
- Cross-squad coordination → team-coordination, agent-coordination
- Task-specific context → task:hyperion://task/human/{taskId}
- UI/React patterns → ui-component-patterns, ui-test-strategies
- API contracts → data-contracts
- Performance → technical-knowledge (with performance tags)

**DON'T:**
- ❌ Search multiple collections for same thing
- ❌ Use generic collections when specific ones exist
- ❌ Query Qdrant if task context already has the answer

### Step 3: Craft Optimal Query String
Make queries SPECIFIC and FOCUSED:

**✅ GOOD Queries (Specific, Contextual):**
- "Go JWT middleware HS256 validation error handling pattern"
- "React Query mutation optimistic update task board UI"
- "MongoDB aggregation pipeline duplicate detection performance"
- "Playwright visual regression baseline update strategy"

**❌ BAD Queries (Too Generic):**
- "authentication" (too broad)
- "error" (too vague)
- "React component" (no context)
- "how to implement feature" (unfocused)

**Query Formula:**
[Technology] + [Specific Component] + [Specific Problem/Pattern] + [Context]

### Step 4: Set Expectations
Estimate what results will look like:
- Code examples with line numbers
- Architectural decision rationale
- Step-by-step implementation guide
- Gotchas and edge cases
- Performance considerations

### Step 5: Define Fallback Strategy
If no results found:
1. Try broader query in same collection
2. Try related collection
3. Check task context (might already have answer)
4. Document as NEW knowledge after solving

## Your Recommendation

### Primary Query Strategy

**Collection:** [collection-name]
**Reason:** [Why this collection is best for this need]

**Query String:**
` + "`" + `
[Optimized query following the formula above]
` + "`" + `

**Expected Results:**
- [What kind of knowledge should be returned]
- [Format: code example, documentation, decision record]
- [Confidence level: High/Medium/Low]

### Alternative Query (if primary fails)

**Collection:** [alternative-collection]
**Query String:**
` + "`" + `
[Slightly broader or different angle query]
` + "`" + `

### Fallback Plan

If both queries return no results:
1. [First fallback action]
2. [Second fallback action]
3. Remember to DOCUMENT solution in Qdrant after solving

### Context Check
⚠️ Before querying, verify task context doesn't already contain:
- [ ] The exact pattern/approach in contextHint
- [ ] File locations in filesModified
- [ ] Similar solutions in priorWorkSummary
- [ ] Relevant collections in qdrantCollections field

**If task context has it → DON'T QUERY, use what's provided!**

## Example Query

Based on your question, here's a ready-to-use query:

` + "```typescript" + `
const results = await mcp__qdrant__qdrant_find({
  collection_name: "[recommended-collection]",
  query: "[optimized-query-string]",
  limit: 3 // Start small, can query again if needed
});
` + "```" + `

Now, analyze the agent's question and provide your recommendation:`, agentQuestion, taskContext, collectionsSection)
}

// buildKnowledgeStructurePrompt builds the knowledge structuring prompt
func (h *KnowledgePromptHandler) buildKnowledgeStructurePrompt(rawLearning string, taskContext map[string]interface{}) string {
	// Extract context fields
	squad := getStringField(taskContext, "squad", "unknown")
	service := getStringField(taskContext, "service", "unknown")
	taskType := getStringField(taskContext, "taskType", "implementation")

	filesModified := []string{}
	if files, ok := taskContext["filesModified"].([]interface{}); ok {
		for _, f := range files {
			if fileStr, ok := f.(string); ok {
				filesModified = append(filesModified, fileStr)
			}
		}
	}

	filesSection := "No files specified"
	if len(filesModified) > 0 {
		filesSection = strings.Join(filesModified, ", ")
	}

	return fmt.Sprintf(`# Knowledge Structuring Guide

## Raw Learning (What You Discovered)
%s

## Task Context
- **Squad:** %s
- **Service:** %s
- **Task Type:** %s
- **Files Modified:** %s

## Your Mission
Transform this raw learning into STRUCTURED, SEARCHABLE knowledge that future agents can discover and reuse.

## Knowledge Structure Template

### 1. Title (Concise & Searchable)
Create a title that appears in semantic search:

**Format:** [Technology] [Component] [Action/Problem] [Key Detail]

**Examples:**
- ✅ "Go JWT Middleware HS256 Token Validation with Error Handling"
- ✅ "React Query Optimistic Update for Task Board Mutations"
- ✅ "MongoDB Aggregation Pipeline for Duplicate Task Detection"
- ❌ "Authentication" (too generic)
- ❌ "Fix bug" (no context)
- ❌ "Implementation notes" (not searchable)

**Your Title:**
[Write a specific, searchable title]

---

### 2. Summary (The "Why" - 2-3 sentences)
Explain the BUSINESS/TECHNICAL CONTEXT:
- What problem did this solve?
- Why was this approach chosen?
- What's the user/system impact?

**Example:**
"Implemented JWT validation middleware to secure all API endpoints. The HS256 algorithm was chosen for compatibility with existing frontend auth. This prevents unauthorized access while maintaining < 5ms validation latency."

**Your Summary:**
[2-3 sentences explaining context and impact]

---

### 3. Implementation (The "How" - Step-by-step)

Provide ACTIONABLE steps with code examples:

**Structure:**
1. **Setup/Prerequisites**
   - Dependencies needed
   - Configuration required
   - Environment setup

2. **Core Implementation**
   ` + "```[language]" + `
   [Key code snippet with comments]
   ` + "```" + `
   - Line-by-line explanation if complex
   - Function signatures
   - Integration points

3. **Testing Approach**
   ` + "```[language]" + `
   [Test example]
   ` + "```" + `
   - How to verify it works
   - Edge cases covered

4. **Integration Steps**
   - How this connects to other components
   - API contracts established
   - Event flows

**Your Implementation:**
[Provide step-by-step with code examples]

---

### 4. Gotchas (Edge Cases & Common Mistakes)

Document the NON-OBVIOUS pitfalls:

**Format:**
- ⚠️ **Gotcha:** [What can go wrong]
  - **Why:** [Root cause]
  - **Solution:** [How to avoid/fix]
  - **Detection:** [How to recognize this issue]

**Examples:**
- ⚠️ **Gotcha:** JWT validation fails with "signature invalid" even with correct secret
  - **Why:** Token uses different algorithm (RS256 vs HS256)
  - **Solution:** Always verify algorithm in token header matches validation config
  - **Detection:** Check token header: ` + "`jwt.decode(token, verify=False)`" + `

- ⚠️ **Gotcha:** Middleware runs on /health endpoint causing startup failures
  - **Why:** Health check doesn't have auth token
  - **Solution:** Exclude /health in middleware registration: ` + "`router.Use(authMiddleware).Except(\"/health\")`" + `
  - **Detection:** Service fails readiness probe

**Your Gotchas:**
[List 2-4 critical gotchas with solutions]

---

### 5. Metadata Tags (For Searchability)

Choose 5-8 tags that help semantic search:

**Tag Categories:**
- **Technology:** go, typescript, react, mongodb, kubernetes, etc.
- **Domain:** auth, api, database, frontend, infrastructure, etc.
- **Pattern:** middleware, hook, aggregation, deployment, etc.
- **Problem:** performance, security, bug-fix, optimization, etc.

**Example Tags:**
` + "`[\"go\", \"jwt\", \"middleware\", \"authentication\", \"hs256\", \"api-security\", \"error-handling\"]`" + `

**Your Tags:**
` + "`[\"tag1\", \"tag2\", \"tag3\", ...]`" + `

---

## Quality Checklist

Before storing in Qdrant, verify:

- [ ] **Title** is specific enough to appear in relevant searches
- [ ] **Summary** explains WHY (business/technical context), not just WHAT
- [ ] **Implementation** has working code examples with comments
- [ ] **Gotchas** document at least 2 non-obvious pitfalls with solutions
- [ ] **Tags** cover technology, domain, pattern, and problem categories
- [ ] **Searchability:** Would future agent find this with semantic search?
- [ ] **Completeness:** Can future agent implement without re-researching?
- [ ] **Code quality:** Examples follow project standards (DRY, SOLID, etc.)

---

## Qdrant Storage Format

Once structured, store using:

` + "```typescript" + `
await mcp__qdrant__qdrant_store({
  collection_name: "[choose-collection]",
  information: ` + "`" + `
## [Your Title]

### Summary
[Your 2-3 sentence summary]

### Implementation
[Your step-by-step with code]

### Gotchas
[Your gotchas list]

### Related
- Files: [filesModified]
- Squad: [squad]
- Service: [service]
  ` + "`" + `,
  metadata: {
    knowledgeType: \"[pattern|solution|gotcha|adr]\",
    domain: \"[squad]\",
    service: \"[service]\",
    title: \"[Your Title]\",
    tags: [\"tag1\", \"tag2\", ...],
    linkedTaskId: \"[taskId if applicable]\",
    createdAt: new Date().toISOString()
  }
});
` + "```" + `

---

## Collection Selection Guide

**Choose collection based on reusability:**

**technical-knowledge** ← Most reusable patterns (JWT, error handling, etc.)
- Patterns used across multiple services
- Architecture best practices
- Cross-cutting concerns

**code-patterns** ← Specific code examples and snippets
- Language-specific implementations
- Framework usage examples
- Algorithm implementations

**adr** ← Architecture Decision Records
- Why certain approaches were chosen
- Trade-offs considered
- Long-term architectural direction

**[domain]-patterns** (ui-component-patterns, etc.) ← Domain-specific
- Squad-specific patterns
- Component libraries
- Domain conventions

**task:hyperion://task/human/{taskId}** ← Task-specific only
- One-off solutions
- Task-specific context
- Handoff information

**Recommendation for your learning:** [collection-name]
**Reason:** [Why this collection is best]

---

Now, structure your raw learning into the format above:`, rawLearning, squad, service, taskType, filesSection)
}

// getStringField safely extracts a string field from map with fallback
func getStringField(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// registerGuideKnowledgeStorage registers the guide_knowledge_storage prompt
func (h *KnowledgePromptHandler) registerGuideKnowledgeStorage(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "guide_knowledge_storage",
		Description: "Provide guidelines on HOW to store knowledge effectively in the Hyperion Coordinator with focus on concise, AI-optimized content.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "knowledgeType",
				Description: "Type of knowledge being stored: pattern, solution, gotcha, adr (optional)",
				Required:    false,
			},
			{
				Name:        "domain",
				Description: "Domain/squad context: backend, frontend, infrastructure, etc. (optional)",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract optional arguments
		knowledgeType := ""
		domain := ""

		if req.Params != nil && req.Params.Arguments != nil {
			knowledgeType = req.Params.Arguments["knowledgeType"]
			domain = req.Params.Arguments["domain"]
		}

		promptText := h.buildKnowledgeStorageGuide(knowledgeType, domain)

		return &mcp.GetPromptResult{
			Description: "Knowledge storage best practices guide",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: promptText,
					},
				},
			},
		}, nil
	}

	server.AddPrompt(prompt, handler)
	return nil
}

// buildKnowledgeStorageGuide builds the knowledge storage guidelines prompt
func (h *KnowledgePromptHandler) buildKnowledgeStorageGuide(knowledgeType, domain string) string {
	contextSection := ""
	if knowledgeType != "" || domain != "" {
		contextSection = "\n## Your Context\n"
		if knowledgeType != "" {
			contextSection += fmt.Sprintf("- **Knowledge Type:** %s\n", knowledgeType)
		}
		if domain != "" {
			contextSection += fmt.Sprintf("- **Domain:** %s\n", domain)
		}
		contextSection += "\n"
	}

	return fmt.Sprintf(`# Knowledge Storage Guide for Hyperion Coordinator%s
## Your Mission

Store knowledge that future AI agents will discover and reuse. **Make every word count.** No fluff, only what matters.

## Core Principle: Headline First (CRITICAL)

**EVERY knowledge entry MUST start with a headline (max 100 words) that answers:**
- **WHAT** is this knowledge? (Be specific - technology, component, action)
- **WHY** does it matter? (Impact, value, when to use it)

**Example Headline (GOOD):**

"**JWT HS256 Token Validation Middleware for Go Gin** - Prevents unauthorized API access by validating Bearer tokens in Authorization headers. Uses jwt.Parse() with 5ms latency. Critical for all authenticated endpoints. Handles exp/iss/aud claims validation and stores user context for downstream handlers. Supports token refresh and revocation via Redis cache."

**Example Headline (BAD - too generic):**

"Authentication middleware for API endpoints"

**Why Headline Matters:**
- AI agents scan headlines first (semantic search preview)
- Headline determines if they read further
- Good headline = instant context, no need to read entire entry
- Bad headline = wasted query, re-searching with different terms

---

## Content Structure (Required Sections)

After the headline, structure your knowledge with these sections:

### 1. What (Core Information)

**Keep it concise and actionable:**
- Technical details that matter
- Code examples (working, tested, with comments)
- API signatures, function names, file locations
- Configuration requirements
- **NO filler words** like "This section explains..." or "As you can see..."

**Example:**
` + "```go" + `
// ValidateJWT extracts and validates JWT token from Authorization header
func ValidateJWT() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid_token"})
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
            // Validate algorithm
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %%v", t.Header["alg"])
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !claims.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid_token"})
            return
        }

        // Store user ID in context
        if userID, ok := claims["sub"].(string); ok {
            c.Set("userID", userID)
        }
        c.Next()
    }
}
` + "```" + `

**Register in router:**
` + "```go" + `
api := router.Group("/api/v1")
api.Use(ValidateJWT()) // All /api/v1/* routes require auth
api.GET("/tasks", taskHandler.List)
` + "```" + `

---

### 2. Why (Importance & Value)

**Explain the business/technical impact:**
- What problem does this solve?
- Why was this approach chosen?
- When should agents use this?
- What's the user/system benefit?

**Example:**

"JWT validation prevents unauthorized access to user data and operations. HS256 symmetric signing was chosen for:
1. **Performance:** 5ms validation vs 15ms for RS256 asymmetric
2. **Simplicity:** Single secret management vs public/private key pairs
3. **Compatibility:** Frontend already uses HS256 for signing
4. **Cost:** No HSM or key rotation infrastructure needed

Use this middleware for ALL authenticated endpoints except /health, /metrics, and public documentation."

---

### 3. How (Implementation Guide)

**Step-by-step implementation:**
- Prerequisites (dependencies, env vars, config)
- Integration steps (where to add, how to wire up)
- Testing approach (unit tests, integration tests)
- Verification steps (how to confirm it works)

**Example:**

**Prerequisites:**
` + "```bash" + `
go get github.com/golang-jwt/jwt/v5
export JWT_SECRET="your-256-bit-secret"
` + "```" + `

**Integration:**
1. Add middleware to router before protected routes
2. Extract userID from context in handlers: ` + "`c.GetString(\"userID\")`" + `
3. Use userID for database queries with user-level isolation

**Testing:**
` + "```go" + `
func TestValidateJWT_Valid(t *testing.T) {
    token := generateTestToken(t, "user123")
    req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
}

func TestValidateJWT_Invalid(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 401, w.Code)
    assert.Contains(t, w.Body.String(), "invalid_token")
}
` + "```" + `

---

### 4. Important (Critical Gotchas & Non-Obvious Insights)

**Document what's NOT obvious from code:**
- Edge cases that cause failures
- Performance pitfalls
- Security vulnerabilities
- Common mistakes (with fixes)
- Integration gotchas

**Format:**
⚠️ **Gotcha:** [What can go wrong]
   - **Why:** [Root cause]
   - **Solution:** [How to avoid/fix]
   - **Detection:** [How to recognize this issue]

**Example:**

⚠️ **Gotcha:** Middleware runs on /health endpoint, causing Kubernetes readiness probe failures
   - **Why:** Health checks don't send Authorization headers
   - **Solution:** Exclude health endpoints from auth middleware
     ` + "```go" + `
     public := router.Group(\"/\")
     public.GET(\"/health\", healthHandler)

     api := router.Group(\"/api/v1\")
     api.Use(ValidateJWT()) // Auth only on /api/v1/*
     ` + "```" + `
   - **Detection:** Service shows "not ready" in kubectl, logs show 401 on /health

⚠️ **Gotcha:** Token validation fails with "signature invalid" even with correct JWT_SECRET
   - **Why:** Token was signed with different algorithm (RS256 vs HS256)
   - **Solution:** Always check token header algorithm matches validation:
     ` + "```go" + `
     if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
         return nil, fmt.Errorf(\"unexpected algorithm\")
     }
     ` + "```" + `
   - **Detection:** Decode token without verification: ` + "`jwt.DecodeSegment(tokenParts[0])`" + ` → check ` + "`alg`" + ` field

⚠️ **Gotcha:** JWT_SECRET loaded from .env file has trailing newline, all validations fail
   - **Why:** ` + "`os.Getenv(\"JWT_SECRET\")`" + ` returns value with \\n if file has newline
   - **Solution:** Trim whitespace: ` + "`strings.TrimSpace(os.Getenv(\"JWT_SECRET\"))`" + `
   - **Detection:** Compare byte lengths: ` + "`len(os.Getenv(\"JWT_SECRET\"))`" + ` vs expected 32

---

## Metadata for Semantic Search (REQUIRED)

Choose 5-8 tags that help AI agents discover this knowledge:

**Tag Categories:**
- **Technology:** go, typescript, react, mongodb, kubernetes, python, rust
- **Domain:** auth, api, database, frontend, infrastructure, deployment
- **Pattern:** middleware, hook, aggregation, pipeline, state-machine
- **Problem:** performance, security, bug-fix, optimization, scalability

**Example Tags:**
` + "`[\"go\", \"jwt\", \"middleware\", \"authentication\", \"hs256\", \"api-security\", \"gin\", \"error-handling\"]`" + `

**Tag Selection Rules:**
- Include language/framework (1-2 tags)
- Include domain/purpose (1-2 tags)
- Include pattern type (1 tag)
- Include problem solved (1-2 tags)
- Be specific: "jwt" > "auth", "gin-middleware" > "middleware"

---

## Quality Checklist (Before Storing)

Before calling ` + "`mcp__hyper__coordinator_upsert_knowledge`" + `, verify:

- [ ] **Headline** is <100 words and answers WHAT + WHY
- [ ] **What section** has tested code examples with comments
- [ ] **Why section** explains business/technical value
- [ ] **How section** provides step-by-step implementation
- [ ] **Important section** documents ≥2 non-obvious gotchas with solutions
- [ ] **Zero fluff:** Every sentence adds value
- [ ] **Tags:** 5-8 specific, searchable tags
- [ ] **Searchability test:** Would you find this with semantic search?
- [ ] **Completeness test:** Can agent implement without re-researching?

---

## Storage Command Template

` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "[choose-appropriate-collection]",
  text: ` + "`" + `
## [Headline - What + Why in <100 words]

### What
[Core technical details with code examples]

### Why
[Business/technical importance and when to use]

### How
[Step-by-step implementation with prerequisites, integration, testing]

### Important
[Critical gotchas with Why/Solution/Detection format]
  ` + "`" + `,
  metadata: {
    knowledgeType: "[pattern|solution|gotcha|adr]",
    domain: "[backend|frontend|infrastructure]",
    title: "[Searchable title matching headline]",
    tags: ["tag1", "tag2", "tag3", "tag4", "tag5"],
    linkedTaskId: "[taskId if applicable]",
    createdAt: new Date().toISOString()
  }
})
` + "```" + `

---

## Collection Selection Guide

**Choose collection based on reusability and scope:**

### Technical Collections (Most Reusable)

**technical-knowledge** ← Use for cross-service patterns
- Authentication/authorization patterns
- Error handling strategies
- Performance optimization techniques
- Architecture patterns (CQRS, Event Sourcing, etc.)

**code-patterns** ← Use for language-specific implementations
- Go middleware examples
- React hooks patterns
- Database query optimizations
- API client implementations

**adr** ← Use for architecture decisions
- Why we chose Technology X over Y
- Trade-offs considered
- Long-term implications
- Migration paths

### Domain-Specific Collections

**ui-component-patterns** ← Frontend components
- React component examples
- Radix UI integration
- Accessibility patterns
- State management

**ui-test-strategies** ← Testing approaches
- Playwright patterns
- Visual regression
- E2E test organization

**data-contracts** ← API schemas
- Request/response formats
- GraphQL schemas
- Event payloads
- Database models

### Task-Specific Collections

**task:hyperion://task/human/{taskId}** ← One-off solutions
- Task completion summaries
- Agent handoff information
- Task-specific gotchas
- NOT for reusable patterns

---

## Anti-Patterns (NEVER Do This)

❌ **Generic Headlines**
` + "```" + `
"Authentication implementation" ← WHAT authentication? WHY important?
` + "```" + `

❌ **Filler Content**
` + "```" + `
"This section will explain how to implement JWT validation..."
"As you can see from the code above..."
"It's important to note that..."
` + "```" + `

❌ **Untested Code**
` + "```go" + `
// Code that looks right but hasn't been tested
func ValidateJWT() { /* ... */ } // Does this compile? Does it work?
` + "```" + `

❌ **Missing Gotchas**
` + "```" + `
### Important
Make sure to test thoroughly. ← NOT HELPFUL
` + "```" + `

❌ **Vague Tags**
` + "```json" + `
{"tags": ["backend", "code", "implementation"]} ← Too generic
` + "```" + `

---

## Examples by Knowledge Type

### Pattern Example (Reusable Implementation)

**Headline:** "Go Gin Router Group Pattern with Middleware Chaining - Organize API endpoints by domain with shared authentication, logging, and rate limiting. Reduces code duplication by 60%% vs per-route middleware. Use for multi-tenant APIs with role-based access control."

**Collection:** ` + "`technical-knowledge`" + `
**Tags:** ` + "`[\"go\", \"gin\", \"router-pattern\", \"middleware\", \"api-organization\", \"rbac\"]`" + `

---

### Solution Example (Bug Fix or Specific Problem)

**Headline:** "MongoDB Aggregation Pipeline Duplicate Detection Performance Fix - Solves 30s+ query timeouts when checking for duplicate tasks across 1M+ documents. Uses compound index on (userId, title, status) + $lookup optimization. Reduces query time to <100ms."

**Collection:** ` + "`technical-knowledge`" + `
**Tags:** ` + "`[\"mongodb\", \"aggregation\", \"performance\", \"duplicate-detection\", \"indexing\"]`" + `

---

### Gotcha Example (Critical Non-Obvious Issue)

**Headline:** "Docker Volume Bind Mount File Watcher Issue - Go file watchers (fsnotify) fail silently on Docker bind mounts due to inotify incompatibility. Hot reload breaks in development. Use polling fallback or named volumes instead."

**Collection:** ` + "`technical-knowledge`" + `
**Tags:** ` + "`[\"docker\", \"file-watcher\", \"gotcha\", \"development\", \"hot-reload\", \"fsnotify\"]`" + `

---

### ADR Example (Architecture Decision)

**Headline:** "ADR-012: Dual-MCP Architecture (MongoDB + Qdrant) - Separates task coordination (relational) from knowledge discovery (semantic). MongoDB for real-time task board, Qdrant for AI agent context retrieval. Enables 90%% knowledge reuse vs 40%% with MongoDB-only approach."

**Collection:** ` + "`adr`" + `
**Tags:** ` + "`[\"architecture\", \"mcp\", \"mongodb\", \"qdrant\", \"vector-db\", \"knowledge-management\"]`" + `

---

## Remember: Headline is KING

**Future AI agents will see:**
1. Your headline (first 100 words) ← **MAKE IT COUNT**
2. Semantic similarity score
3. Collection name
4. Tags

**If headline is good:** Agent reads full content, implements solution, marks task complete.

**If headline is bad:** Agent skips, searches again, wastes context window.

**Your goal:** Write headlines so good that agents KNOW they found the right knowledge in <10 seconds.

---

Now, structure your knowledge following this guide and store it using ` + "`mcp__hyper__coordinator_upsert_knowledge`" + `.`, contextSection)
}
