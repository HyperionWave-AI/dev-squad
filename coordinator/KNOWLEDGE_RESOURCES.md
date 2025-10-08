# Knowledge Resources - MCP Coordinator

## Overview

The MCP Coordinator now provides **Phase 2 Knowledge Resources** to help agents efficiently discover and navigate Qdrant knowledge collections without making excessive queries.

## Resources

### 1. `hyperion://knowledge/collections`

**Purpose**: Complete directory of all Qdrant knowledge collections with metadata

**Use Cases**:
- Discover available knowledge collections
- Learn collection purposes and categories
- Get example query patterns
- Find which collections have actual data

**Response Structure**:
```json
{
  "collections": [
    {
      "name": "technical-knowledge",
      "category": "Tech",
      "purpose": "General technical patterns and solutions",
      "exampleQuery": "qdrant-find: 'JWT middleware implementation Go'",
      "useCases": [
        "Store reusable code patterns",
        "Document architectural decisions",
        "Share bug fixes and workarounds"
      ],
      "hasData": true
    }
  ],
  "totalDefined": 14,
  "totalWithData": 3,
  "lastUpdated": "2025-10-04T..."
}
```

**Collection Categories**:
- **Task**: Task-specific and coordination collections
  - `task:hyperion://task/human/{taskId}` - Task-specific knowledge
  - `team-coordination` - Cross-squad communication
  - `agent-coordination` - Agent workflow coordination

- **Tech**: Technical knowledge and patterns
  - `technical-knowledge` - General patterns
  - `code-patterns` - Specific implementations
  - `adr` - Architecture Decision Records
  - `data-contracts` - API schemas
  - `technical-debt-registry` - Debt tracking

- **UI**: Frontend patterns
  - `ui-component-patterns` - React components
  - `ui-test-strategies` - Playwright tests
  - `ui-accessibility-standards` - WCAG compliance
  - `ui-visual-regression-baseline` - Visual baselines

- **Ops**: Operations and quality
  - `mcp-operations` - MCP troubleshooting
  - `code-quality-violations` - Quality tracking

### 2. `hyperion://knowledge/recent-learnings`

**Purpose**: Recently stored knowledge entries from the last 24 hours

**Use Cases**:
- Discover what other agents have learned recently
- Stay updated on cross-squad coordination
- Find fresh solutions to similar problems
- Track knowledge creation activity

**Response Structure**:
```json
{
  "timeRange": {
    "start": "2025-10-03T12:00:00Z",
    "end": "2025-10-04T12:00:00Z"
  },
  "totalEntries": 5,
  "collectionsWithActivity": 3,
  "byCollection": {
    "technical-knowledge": [
      {
        "id": "entry-123",
        "collection": "technical-knowledge",
        "topic": "JWT Auth Pattern",
        "text": "JWT authentication pattern implementation...",
        "metadata": {
          "title": "JWT Auth Pattern",
          "agentName": "backend-dev",
          "taskId": "task-456"
        },
        "createdAt": "2025-10-04T11:00:00Z",
        "agentName": "backend-dev"
      }
    ],
    "team-coordination": [...]
  },
  "allEntries": [...]
}
```

## Agent Workflow Integration

### Before Starting Work

**Step 1: Check Collection Directory**
```typescript
// Read available collections
const collectionsResource = await mcp.readResource({
  uri: "hyperion://knowledge/collections"
});

// Parse response to find relevant collection
const collections = JSON.parse(collectionsResource.contents[0].text);
const relevantCollection = collections.collections.find(
  c => c.category === "Tech" && c.name.includes("pattern")
);
```

**Step 2: Check Recent Learnings**
```typescript
// See what's been learned recently
const recentResource = await mcp.readResource({
  uri: "hyperion://knowledge/recent-learnings"
});

const recent = JSON.parse(recentResource.contents[0].text);

// Check if someone already solved similar problem
const similarSolution = recent.allEntries.find(
  entry => entry.topic.includes("JWT") && entry.collection === "technical-knowledge"
);

if (similarSolution) {
  // Use existing solution instead of querying Qdrant
  console.log("Found recent solution:", similarSolution.text);
}
```

## Benefits

### 1. **Context Efficiency**
- No need to memorize collection names
- No need to guess which collection has what
- Reduces speculative Qdrant queries

### 2. **Knowledge Discovery**
- Find existing solutions before implementing
- Learn from other agents' recent work
- Discover patterns across squads

### 3. **Coordination**
- See what's happening across teams
- Track knowledge creation activity
- Avoid duplicate work

## Example: Complete Workflow

```typescript
// Agent starting a new task: "Implement JWT authentication"

// 1. Check if someone already solved this recently
const recent = await readResource("hyperion://knowledge/recent-learnings");
const recentData = JSON.parse(recent.contents[0].text);

const existingSolution = recentData.allEntries.find(
  e => e.topic.toLowerCase().includes("jwt")
);

if (existingSolution) {
  // Found it! Use existing solution
  console.log("Recent solution found:", existingSolution.text);
} else {
  // 2. Need to search Qdrant - check which collection
  const collections = await readResource("hyperion://knowledge/collections");
  const collectionsData = JSON.parse(collections.contents[0].text);

  const techCollection = collectionsData.collections.find(
    c => c.name === "technical-knowledge"
  );

  // 3. Now do ONE targeted Qdrant query
  const results = await qdrant_find({
    collection_name: "technical-knowledge",
    query: techCollection.exampleQuery // Use suggested query pattern
  });
}
```

## Implementation Details

### Handler: `knowledge_resources.go`

**Location**: `/coordinator/mcp-server/handlers/knowledge_resources.go`

**Key Functions**:
- `RegisterKnowledgeResources(server *mcp.Server)` - Registers both resources
- `handleCollectionsResource()` - Returns collection directory
- `handleRecentLearningsResource()` - Returns recent entries (24h)

**Dependencies**:
- `storage.KnowledgeStorage` - MongoDB knowledge storage
- MCP SDK `mcp.Server` - Resource registration

### Tests: `knowledge_resources_test.go`

**Coverage**:
- ✅ Collections resource structure
- ✅ Recent learnings time filtering
- ✅ Collection categories completeness
- ✅ JSON response validation

**Run Tests**:
```bash
cd coordinator/mcp-server
go test -v handlers/knowledge_resources.go handlers/knowledge_resources_test.go
```

## Future Enhancements

### Planned Features
1. **Search within resources** - Filter collections by category
2. **Time range parameters** - Configurable recent learning window
3. **Agent filtering** - See learnings from specific agents
4. **Collection statistics** - Entry counts, activity metrics

### Integration Opportunities
- **Task context enrichment** - Auto-suggest relevant collections
- **Smart routing** - Direct agents to best collection
- **Duplicate detection** - Warn when similar knowledge exists

## Troubleshooting

### No Recent Learnings Returned
**Cause**: No knowledge stored in last 24 hours
**Solution**: Normal - agents haven't stored knowledge recently

### Collection Shows `hasData: false`
**Cause**: Collection defined but no entries stored
**Solution**: Normal - collection exists but unused

### Missing Collections
**Cause**: New collection not in hardcoded list
**Solution**: Add to `handleCollectionsResource()` collections array

## Version

**Phase**: 2 - Knowledge Resources
**Created**: 2025-10-04
**Handler Version**: 1.0.0
**MCP SDK**: github.com/modelcontextprotocol/go-sdk/mcp
