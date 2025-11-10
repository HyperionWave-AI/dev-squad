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

	// Register knowledge_workflow_guide prompt
	if err := h.registerKnowledgeWorkflowGuide(server); err != nil {
		return fmt.Errorf("failed to register knowledge_workflow_guide prompt: %w", err)
	}

	// Register knowledge_voting_workflow prompt
	if err := h.registerKnowledgeVotingWorkflow(server); err != nil {
		return fmt.Errorf("failed to register knowledge_voting_workflow prompt: %w", err)
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
const results = await mcp__hyper__knowledge_find({
  collectionName: "[recommended-collection]",
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
await mcp__hyper__knowledge_store({
  collectionName: "[choose-collection]",
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

// registerKnowledgeWorkflowGuide registers the knowledge_workflow_guide prompt
func (h *KnowledgePromptHandler) registerKnowledgeWorkflowGuide(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "knowledge_workflow_guide",
		Description: "Guide agents through the complete knowledge discovery and usage workflow - when to search, how to search, and how to vote on results.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "taskDescription",
				Description: "What the agent is about to implement or solve",
				Required:    true,
			},
			{
				Name:        "domain",
				Description: "Technical domain (e.g., 'authentication', 'UI components', 'database')",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		taskDescription := ""
		domain := ""

		if req.Params != nil && req.Params.Arguments != nil {
			taskDescription = req.Params.Arguments["taskDescription"]
			domain = req.Params.Arguments["domain"]
		}

		if taskDescription == "" {
			return nil, fmt.Errorf("taskDescription is a required argument")
		}

		promptText := h.buildKnowledgeWorkflowGuide(taskDescription, domain)

		return &mcp.GetPromptResult{
			Description: "Knowledge base workflow guidance",
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

// registerKnowledgeVotingWorkflow registers the knowledge_voting_workflow prompt
func (h *KnowledgePromptHandler) registerKnowledgeVotingWorkflow(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "knowledge_voting_workflow",
		Description: "Teach agents when and how to vote on knowledge articles to improve quality and search ranking.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "articleId",
				Description: "The article ID being considered for voting",
				Required:    true,
			},
			{
				Name:        "articleTitle",
				Description: "Title of the article (optional)",
				Required:    false,
			},
			{
				Name:        "wasHelpful",
				Description: "Whether the article helped solve the problem (true/false)",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		articleId := ""
		articleTitle := ""
		wasHelpful := ""

		if req.Params != nil && req.Params.Arguments != nil {
			articleId = req.Params.Arguments["articleId"]
			articleTitle = req.Params.Arguments["articleTitle"]
			wasHelpful = req.Params.Arguments["wasHelpful"]
		}

		if articleId == "" {
			return nil, fmt.Errorf("articleId is a required argument")
		}

		promptText := h.buildKnowledgeVotingGuide(articleId, articleTitle, wasHelpful)

		return &mcp.GetPromptResult{
			Description: "Knowledge voting decision guidance",
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

// buildKnowledgeWorkflowGuide builds the knowledge workflow guide prompt
func (h *KnowledgePromptHandler) buildKnowledgeWorkflowGuide(taskDescription, domain string) string {
	domainSection := ""
	if domain != "" {
		domainSection = fmt.Sprintf(`
## Your Technical Domain
**%s**

This helps narrow down which collections are most relevant to your task.
`, domain)
	}

	return fmt.Sprintf(`# Knowledge Base Discovery Workflow

## Your Task
%s
%s
## Mission
Use the knowledge base EFFECTIVELY to avoid reinventing the wheel. Follow this workflow EVERY TIME before implementing something new.

---

## STEP 1: DISCOVER Available Collections

**ALWAYS start here - don't assume you know what exists!**

` + "```typescript" + `
const collections = await knowledge_list_collections();
` + "```" + `

**Why this matters:**
- Collections are organized by domain and purpose
- New collections are added regularly
- Each collection has a specific focus and quality level

**What to look for:**
- Collection names matching your domain (%s)
- Collection descriptions mentioning your technology
- Category tags relevant to your task type

**Example output:**
` + "```json" + `
{
  "collections": [
    {
      "name": "technical-knowledge",
      "description": "Reusable patterns, solutions, and best practices",
      "category": "Technical",
      "entryCount": 342
    },
    {
      "name": "ui-component-patterns",
      "description": "React components, Radix UI patterns",
      "category": "UI",
      "entryCount": 156
    }
  ]
}
` + "```" + `

---

## STEP 2: PICK the Right Collection

**Match your task domain to collection purpose:**

| **Your Domain** | **Best Collections** |
|----------------|---------------------|
| Authentication, API security | technical-knowledge, code-patterns |
| React, UI components | ui-component-patterns, ui-test-strategies |
| Database queries, schemas | technical-knowledge, data-contracts |
| Architecture decisions | adr (Architecture Decision Records) |
| Deployment, infrastructure | technical-knowledge (with infra tags) |
| Cross-squad coordination | team-coordination, agent-coordination |
| Code quality issues | code-quality-violations |

**Pro Tips:**
- ✅ Start with 1-2 most specific collections
- ✅ More focused = better results
- ❌ Don't search every collection hoping for hits
- ❌ Generic collections when specific ones exist

---

## STEP 3: SEARCH with Specific Queries

**Query Formula:** [Technology] + [Component] + [Problem/Pattern] + [Context]

**✅ GOOD Queries (Specific, Contextual):**
- "Go JWT middleware HS256 validation error handling pattern"
- "React Query mutation optimistic update task board UI"
- "MongoDB aggregation pipeline duplicate detection performance"
- "Playwright visual regression baseline update strategy"

**❌ BAD Queries (Too Generic):**
- "authentication" (too broad - 1000+ results)
- "error" (meaningless - every article mentions errors)
- "React component" (no context - which component? what problem?)
- "how to implement feature" (unfocused)

**Search Call:**
` + "```typescript" + `
const results = await knowledge_find({
  collectionName: "technical-knowledge", // From Step 2
  query: "Go JWT middleware HS256 validation error handling pattern", // Specific!
  limit: 5, // Start small, can search again
  retrieveMode: "chunk" // Don't overwhelm with full articles
});
` + "```" + `

---

## STEP 4: REVIEW Results Thoroughly

**Don't just skim - actually read and evaluate:**

` + "```typescript" + `
for (const article of results.entries) {
  console.log("Title:", article.metadata.title);
  console.log("Score:", article.score); // Relevance: 0.0-1.0
  console.log("Content preview:", article.content.substring(0, 200));

  // Read the full content if score > 0.7
  if (article.score > 0.7) {
    const full = await knowledge_get_by_id({
      id: article.id,
      retrieveMode: "full"
    });

    // Study the implementation details
    // Check if code examples apply to your case
    // Note any gotchas mentioned
  }
}
` + "```" + `

**Evaluation Criteria:**
- **Relevance:** Does this solve MY specific problem?
- **Accuracy:** Is the information correct and up-to-date?
- **Completeness:** Are there code examples and gotchas?
- **Applicability:** Can I use this pattern directly or adapt it?

---

## STEP 5: VOTE on Usefulness (MANDATORY!)

**This is NOT optional - voting improves the system for everyone!**

After reading articles, you MUST vote:

` + "```typescript" + `
// Article was helpful
await knowledge_vote({
  articleId: article.id,
  vote: 1, // +1 = helpful
  reason: "Exact pattern I needed - JWT validation with proper error handling. Code example worked perfectly."
});

// Article was NOT helpful
await knowledge_vote({
  articleId: article.id,
  vote: -1, // -1 = not helpful
  reason: "Outdated approach - uses deprecated library. Modern approach is different."
});
` + "```" + `

**When to vote +1 (Helpful):**
- ✅ Article helped solve your problem
- ✅ Provided useful context or background
- ✅ Saved you time researching
- ✅ Accurate and complete information
- ✅ Good code examples that work

**When to vote -1 (Not Helpful):**
- ❌ Outdated or incorrect information
- ❌ Unhelpful or irrelevant to the query
- ❌ Missing critical details
- ❌ Caused confusion or wasted time
- ❌ Code examples don't work

**Why voting matters:**
- Your votes improve search ranking
- Highly-voted articles surface first
- Low-voted articles get updated or removed
- Community learns what's valuable

---

## STEP 6: APPLY Patterns to Your Code

**Now implement using what you learned:**

` + "```typescript" + `
// Before: Would have written from scratch
function validateToken(token: string) {
  // ... 50 lines of trial and error ...
}

// After: Applied pattern from knowledge base
function validateToken(token: string) {
  // Pattern from "Go JWT middleware HS256 validation"
  // Key gotcha: Must verify algorithm matches (from article)
  const decoded = jwt.verify(token, secret, {
    algorithms: ['HS256'] // Prevents algorithm confusion attack
  });

  // Extract user context (from article example)
  return {
    userId: decoded.sub,
    email: decoded.email
  };
}
` + "```" + `

**Document what you learned:**
- If you made improvements to the pattern → update the article (vote with improvement notes)
- If you solved something NOT in knowledge base → store it for future agents
- If you found gotchas not mentioned → add them via voting reason

---

## WORKFLOW CHECKLIST

Before you start coding, verify:

- [ ] ✅ Called knowledge_list_collections to see what exists
- [ ] ✅ Picked 1-2 most relevant collections by domain
- [ ] ✅ Crafted specific query using [Technology + Component + Problem]
- [ ] ✅ Searched with retrieveMode: "chunk" first
- [ ] ✅ Read top results (score > 0.7) in full
- [ ] ✅ Evaluated relevance, accuracy, completeness
- [ ] ✅ **VOTED on every article you read** (mandatory!)
- [ ] ✅ Applied patterns or noted why they don't apply
- [ ] ✅ Documented improvements or new learnings

---

## ANTI-PATTERNS (DON'T DO THIS!)

❌ **Skip Step 1:** "I know what collections exist" → You don't, they change weekly
❌ **Generic queries:** "authentication" → Too broad, 1000+ results
❌ **Skip voting:** "I'll vote later" → You won't, and system quality degrades
❌ **Only read titles:** Content has critical gotchas in the details
❌ **Copy-paste blindly:** Understand the pattern, adapt to your context
❌ **Don't store learnings:** Next agent will waste time re-discovering

---

## Example End-to-End Workflow

**Task:** Implement JWT authentication middleware for Go service

` + "```typescript" + `
// Step 1: Discover
const collections = await knowledge_list_collections();
// Found: technical-knowledge, code-patterns

// Step 2: Pick collection
// → technical-knowledge (auth is cross-service pattern)

// Step 3: Search
const results = await knowledge_find({
  collectionName: "technical-knowledge",
  query: "Go JWT middleware HS256 validation error handling pattern",
  limit: 5,
  retrieveMode: "chunk"
});

// Step 4: Review
const topArticle = results.entries[0]; // score: 0.89
const full = await knowledge_get_by_id({
  id: topArticle.id,
  retrieveMode: "full"
});

// Step 5: Vote
await knowledge_vote({
  articleId: topArticle.id,
  vote: 1,
  reason: "Perfect example with HS256 validation and context extraction. Gotcha about algorithm confusion was crucial!"
});

// Step 6: Apply
// Implemented middleware based on pattern
// Added edge case handling for expired tokens (not in article)

// Step 7: Document improvement
await knowledge_vote({
  articleId: topArticle.id,
  vote: 1,
  reason: "Used successfully. Suggestion: add example for handling token expiration gracefully with 401 response."
});
` + "```" + `

---

## Now Apply This Workflow to Your Task:

**Your task:** %s
**Your domain:** %s

**Next steps:**
1. Run knowledge_list_collections
2. Pick the best collection(s) for your domain
3. Craft a specific query using the formula
4. Search, read, vote, apply

**Remember:** Every article you vote on makes the system better for all agents!`,
		taskDescription, domainSection, domain, taskDescription, domain)
}

// buildKnowledgeVotingGuide builds the knowledge voting guide prompt
func (h *KnowledgePromptHandler) buildKnowledgeVotingGuide(articleId, articleTitle, wasHelpful string) string {
	articleSection := fmt.Sprintf("**Article ID:** %s", articleId)
	if articleTitle != "" {
		articleSection = fmt.Sprintf("%s\n**Article Title:** %s", articleSection, articleTitle)
	}

	helpfulnessHint := ""
	if wasHelpful != "" {
		if wasHelpful == "true" {
			helpfulnessHint = `
**Initial Assessment:** You indicated this article WAS helpful.

Now let's determine if it deserves a +1 vote and craft a good reason.`
		} else if wasHelpful == "false" {
			helpfulnessHint = `
**Initial Assessment:** You indicated this article was NOT helpful.

Now let's determine if it deserves a -1 vote and craft a constructive reason.`
		}
	}

	return fmt.Sprintf(`# Knowledge Voting Decision Guide

## Article Under Review
%s
%s

---

## Why Voting Matters

**Your votes directly impact:**
- 🎯 **Search ranking** - Highly-voted articles appear first
- 📈 **Content quality** - Low-voted articles get reviewed and improved
- 🤝 **Community learning** - Future agents benefit from your feedback
- 🔍 **Discovery** - Good content becomes easier to find

**This is your responsibility as an agent - vote on EVERY article you read!**

---

## DECISION TREE: Should I Vote +1 (Helpful)?

### ✅ Vote +1 (Helpful) When:

**Accuracy & Completeness:**
- [ ] Information is **correct and up-to-date**
- [ ] Code examples **actually work** (you tested them or they're clearly valid)
- [ ] Covers **all critical aspects** of the topic
- [ ] Includes **gotchas and edge cases** that save debugging time

**Usefulness:**
- [ ] **Directly solved your problem** or gave you the solution approach
- [ ] **Saved significant time** (avoided research, trial-and-error, or dead ends)
- [ ] **Provided valuable context** or background you didn't have
- [ ] **Well-structured** and easy to understand

**Applicability:**
- [ ] Pattern/approach **applies to your specific use case**
- [ ] Examples are **relevant to current tech stack**
- [ ] Follows **current project standards and conventions**

**Quality Indicators:**
- [ ] Has **working code snippets** with explanations
- [ ] Documents **real-world gotchas** (not obvious stuff)
- [ ] Includes **testing approach** or verification steps
- [ ] References **specific files, line numbers, or PRs**

**Example +1 Vote Reasons:**
- ✅ "Exact pattern I needed for JWT validation. Code example worked perfectly. Gotcha about algorithm confusion saved me hours of debugging."
- ✅ "Comprehensive guide to React Query optimistic updates. Applied to task board - all edge cases covered. Voting pattern was particularly useful."
- ✅ "MongoDB aggregation pipeline example was production-ready. Used it directly, handles all our duplicate detection cases efficiently."

---

## DECISION TREE: Should I Vote -1 (Not Helpful)?

### ❌ Vote -1 (Not Helpful) When:

**Accuracy Issues:**
- [ ] Contains **incorrect or outdated information**
- [ ] Code examples **don't work** or use **deprecated APIs**
- [ ] Recommends **anti-patterns** or **bad practices**
- [ ] **Conflicts with current project standards**

**Completeness Issues:**
- [ ] **Missing critical details** needed to implement
- [ ] No **code examples** or examples too generic
- [ ] Doesn't mention **important gotchas** you discovered
- [ ] **Incomplete implementation** (only shows happy path)

**Relevance Issues:**
- [ ] **Irrelevant to the query** that led you to it
- [ ] Solves a **different problem** than advertised
- [ ] Technology/framework **doesn't match current stack**
- [ ] Too generic to be actionable

**Clarity Issues:**
- [ ] **Confusing or poorly structured**
- [ ] **Caused more confusion** than it resolved
- [ ] **Contradictory information** within the article
- [ ] **Wasted significant time** trying to understand

**Example -1 Vote Reasons:**
- ❌ "Outdated approach - uses deprecated jwt-simple library. Current project uses jsonwebtoken with different API. Update needed."
- ❌ "Missing critical error handling for expired tokens. Code example fails in production. Needs edge case coverage."
- ❌ "Irrelevant to query 'React Query mutations'. Article is about REST API design, not React Query hooks."
- ❌ "Too generic - 'use authentication' without any code examples or specific implementation guidance."

---

## How to Write GOOD Vote Reasons

**Vote reasons should be:**

1. **Specific** - What exactly was helpful or unhelpful?
2. **Actionable** - How can the article be improved?
3. **Constructive** - Help future readers and authors
4. **Honest** - Accurate assessment, not just +1 everything

**Template for +1 Votes:**
` + "```" + `
"[What problem it solved] [What specifically helped] [What was particularly valuable]"

Example:
"Solved JWT validation bug. HS256 algorithm specification prevented token confusion attack. Gotcha about health endpoint exclusion was crucial."
` + "```" + `

**Template for -1 Votes:**
` + "```" + `
"[What was wrong/missing] [Why it didn't help] [Suggestion for improvement]"

Example:
"Code example uses deprecated library (jwt-simple). Current stack uses jsonwebtoken with different API. Suggest updating to current library with migration notes."
` + "```" + `

---

## Common Voting Mistakes

**❌ DON'T:**
- Vote +1 without reading the full article (skimming titles doesn't count)
- Vote -1 because the topic isn't relevant to YOU right now (it might help others)
- Leave vague reasons like "helpful" or "not useful" (be specific!)
- Vote on articles you didn't actually read or try to apply
- Skip voting because "someone else will do it" (they won't!)

**✅ DO:**
- Read the full content before voting
- Vote based on accuracy, completeness, and usefulness
- Provide specific, actionable feedback in your reason
- Vote on EVERY article you read (mandatory workflow)
- Update your vote if you later discover issues or benefits

---

## Voting Workflow

` + "```typescript" + `
// After reading and evaluating an article
await knowledge_vote({
  articleId: "%s", // The article you're evaluating
  vote: 1,  // +1 for helpful, -1 for not helpful
  reason: "[Your specific, actionable reason following the templates above]"
});
` + "```" + `

---

## Your Voting Decision for This Article

**Article:** %s

**Evaluation Checklist:**

- [ ] Did this article help solve a problem?
- [ ] Is the information accurate and current?
- [ ] Are code examples working and relevant?
- [ ] Does it cover important gotchas?
- [ ] Is it complete enough to implement from?
- [ ] Did it save you time or prevent errors?

**Based on your checklist:**
- If ≥4 checkmarks → Vote +1 (Helpful)
- If <2 checkmarks → Vote -1 (Not Helpful)
- If 2-3 checkmarks → Vote +1 if it helped, -1 if it wasted time

**Your vote reason should include:**
1. What you were trying to accomplish
2. How this article helped (or didn't help)
3. Specific examples or suggestions for improvement

---

## Example Decision Process

**Scenario:** Looking for JWT validation pattern

**Article Found:** "Go JWT Middleware with HS256 Validation"

**Evaluation:**
- ✅ Information accurate - uses current jsonwebtoken library
- ✅ Code example works - tested locally
- ✅ Covers gotcha about algorithm confusion attack
- ✅ Includes error handling for expired tokens
- ✅ Shows health endpoint exclusion pattern
- ✅ Saved 2+ hours of research and debugging

**Decision:** Vote +1 (Helpful)

**Reason:** "Implemented JWT middleware using this pattern. HS256 algorithm specification prevented token confusion attack. Health endpoint exclusion saved me from debugging startup failures. Code example was production-ready."

---

## Now Make Your Voting Decision

**Article:** %s

**Ask yourself:**
1. Did this help me accomplish my task?
2. Would I want a colleague to find this article?
3. Is it accurate enough to recommend?
4. Does it have actionable information?

**Vote based on honest assessment, not politeness - quality feedback helps everyone!**`,
		articleSection, helpfulnessHint, articleId, articleId, articleId)
}
