# Knowledge Management Prompts - Phase 2

## Overview

Phase 2 knowledge management prompts help agents optimize their Qdrant queries and structure their learnings for maximum reusability.

## Prompts

### 1. `recommend_qdrant_query`

**Purpose:** Suggest optimal Qdrant query strategy for an agent's need

**Arguments:**
- `agentQuestion` (string, required) - What agent wants to know
- `taskContext` (string, required) - Current task context
- `availableCollections` (string, optional) - Comma-separated collections

**Output:** Prompt recommending:
- Best collection to query
- Optimized query string (specific, focused)
- Expected result type
- Fallback strategies if no results

**Example Usage:**
```typescript
const prompt = await mcp__hyper__prompts_get({
  name: "recommend_qdrant_query",
  arguments: {
    agentQuestion: "How do I implement JWT authentication middleware?",
    taskContext: "Working on go-mcp-dev service, building auth middleware",
    availableCollections: "technical-knowledge,code-patterns"
  }
});
```

**Prompt Output Includes:**
- **Analysis Framework:** Step-by-step query optimization
- **Collection Selection:** Based on need type (bug/pattern/architecture/etc.)
- **Query Formula:** [Technology] + [Component] + [Problem] + [Context]
- **Primary Query Strategy:** Best collection + optimized query
- **Alternative Query:** Fallback if primary fails
- **Context Check:** Verify task doesn't already have answer

---

### 2. `suggest_knowledge_structure`

**Purpose:** Help structure learnings before storing in Qdrant

**Arguments:**
- `rawLearning` (string, required) - What the agent learned
- `context` (string, required) - JSON task context (squad, service, files)

**Output:** Prompt structuring knowledge with:
- Title (concise, searchable)
- Summary (2-3 sentences, the "why")
- Implementation (code examples, step-by-step)
- Gotchas (edge cases, common mistakes)
- Metadata tags (for searchability)

**Example Usage:**
```typescript
const contextJSON = JSON.stringify({
  squad: "go-mcp-dev",
  service: "auth-service",
  taskType: "implementation",
  filesModified: ["middleware/auth.go", "middleware/auth_test.go"]
});

const prompt = await mcp__hyper__prompts_get({
  name: "suggest_knowledge_structure",
  arguments: {
    rawLearning: "Implemented JWT validation using HS256. Had to handle token expiration carefully.",
    context: contextJSON
  }
});
```

**Prompt Output Includes:**
- **Title Format:** [Technology] [Component] [Action/Problem] [Key Detail]
- **Summary Template:** Business/technical context, approach, impact
- **Implementation Structure:** Setup, Core code, Testing, Integration
- **Gotcha Format:** What/Why/Solution/Detection
- **Metadata Tags:** Technology, Domain, Pattern, Problem categories
- **Quality Checklist:** Searchability, completeness, code quality
- **Qdrant Storage Format:** Ready-to-use store command
- **Collection Selection:** Guidance on which collection to use

---

## Workflow Integration

### For Agents Needing Knowledge:

1. **Before querying Qdrant:**
   ```typescript
   // Get query recommendation
   const queryPrompt = await recommend_qdrant_query({
     agentQuestion: "What I need to know",
     taskContext: "Current task details",
     availableCollections: task.qdrantCollections?.join(',')
   });

   // Use recommended query
   const results = await qdrant_find({
     collection_name: "<from recommendation>",
     query: "<optimized query from recommendation>"
   });
   ```

2. **Benefit:** Faster, more accurate knowledge discovery with minimal queries

### For Agents Storing Knowledge:

1. **After completing work:**
   ```typescript
   // Get structure recommendation
   const structurePrompt = await suggest_knowledge_structure({
     rawLearning: "What I discovered/implemented",
     context: JSON.stringify({
       squad: task.squad,
       service: task.service,
       taskType: task.type,
       filesModified: task.filesModified
     })
   });

   // Follow the structured format to store
   await qdrant_store({
     collection_name: "<from recommendation>",
     information: "<structured following template>",
     metadata: { /* from recommendation */ }
   });
   ```

2. **Benefit:** Consistent, searchable knowledge that future agents can discover

---

## Query Optimization Patterns

### Good Query Examples (from prompt):
- ✅ "Go JWT middleware HS256 validation error handling pattern"
- ✅ "React Query mutation optimistic update task board UI"
- ✅ "MongoDB aggregation pipeline duplicate detection performance"

### Bad Query Examples (from prompt):
- ❌ "authentication" (too broad)
- ❌ "error" (too vague)
- ❌ "React component" (no context)

---

## Knowledge Structure Best Practices

### Title Examples (from prompt):
- ✅ "Go JWT Middleware HS256 Token Validation with Error Handling"
- ✅ "React Query Optimistic Update for Task Board Mutations"
- ✅ "MongoDB Aggregation Pipeline for Duplicate Task Detection"

### Gotcha Format (from prompt):
```
- ⚠️ **Gotcha:** [What can go wrong]
  - **Why:** [Root cause]
  - **Solution:** [How to avoid/fix]
  - **Detection:** [How to recognize this issue]
```

### Metadata Tags Categories:
- **Technology:** go, typescript, react, mongodb, kubernetes
- **Domain:** auth, api, database, frontend, infrastructure
- **Pattern:** middleware, hook, aggregation, deployment
- **Problem:** performance, security, bug-fix, optimization

---

## Collection Selection Guide (from prompts)

### By Reusability:

**technical-knowledge** - Most reusable patterns (JWT, error handling)
- Patterns used across multiple services
- Architecture best practices
- Cross-cutting concerns

**code-patterns** - Specific code examples
- Language-specific implementations
- Framework usage examples
- Algorithm implementations

**adr** - Architecture Decision Records
- Why certain approaches were chosen
- Trade-offs considered
- Long-term architectural direction

**[domain]-patterns** - Domain-specific
- Squad-specific patterns (ui-component-patterns, etc.)
- Component libraries
- Domain conventions

**task:hyperion://task/human/{taskId}** - Task-specific only
- One-off solutions
- Task-specific context
- Handoff information

---

## Implementation Details

**File:** `coordinator/mcp-server/handlers/knowledge_prompts.go`

**Registration:** Automatically registered in `main.go`:
```go
knowledgePromptHandler := handlers.NewKnowledgePromptHandler()
if err := knowledgePromptHandler.RegisterKnowledgePrompts(server); err != nil {
    logger.Fatal("Failed to register knowledge prompts", zap.Error(err))
}
```

**Test Coverage:** 100% coverage in `knowledge_prompts_test.go`
- Tests for both prompts with various inputs
- Error handling validation
- Prompt content verification
- Collection handling (with and without)

---

## Success Metrics

**For Query Optimization:**
- Reduce Qdrant queries from 3-5 to 1-2 per task
- Increase query success rate (finding relevant knowledge)
- Faster knowledge discovery (< 30 seconds vs minutes)

**For Knowledge Structure:**
- Improve knowledge reuse rate (same pattern used across tasks)
- Increase search discoverability (agents find what they need)
- Better knowledge quality (actionable, complete, documented)

---

## Next Steps (Phase 3+)

Potential future prompts:
- `analyze_context_sufficiency` - Check if task has enough context
- `suggest_handoff_notes` - Help agents document for next phase
- `recommend_test_strategy` - Testing approach for implementations
- `estimate_complexity` - Help size tasks accurately
