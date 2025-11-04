# Metacognitive Reflection System - Autonomous Learning

## Overview

The Hyperion Metacognitive Reflection System enables AI agents to **learn from experience** through a continuous cycle of decision tracking, outcome analysis, and lesson extraction. The system is **fully autonomous** in detecting learning opportunities while remaining **semi-autonomous** in lesson creation (requiring AI approval).

## Architecture

### Data Model

```
MongoDB Collections:
├── reflections          # Decisions, outcomes, lessons, causal links
├── experience_index     # Fast pattern lookup with occurrence tracking
└── error_patterns       # Recurring error detection and tracking
```

### MCP Tools (AI Interface)

```
5 MCP Tools for AI Self-Awareness:
├── reflection_record_decision           # Track decision with context
├── reflection_record_outcome            # Compare prediction vs reality
├── reflection_extract_lesson            # Store transferable pattern
├── reflection_suggest_lesson_from_error # Auto-populate from errors ✨
└── reflection_query_relevant_lessons    # Proactive recommendations ✨
```

### REST API (Human/UI Interface)

```
8 REST Endpoints:
├── GET  /api/v1/reflection/decisions       # List decisions
├── GET  /api/v1/reflection/outcomes        # List outcomes
├── GET  /api/v1/reflection/lessons         # List lessons
├── GET  /api/v1/reflection/search?q=X      # Full-text search
├── POST /api/v1/reflection/decision        # Create decision
├── POST /api/v1/reflection/outcome         # Create outcome
├── POST /api/v1/reflection/lesson          # Create lesson
└── POST /api/v1/reflection/test-error      # Test error tracking
```

## The Autonomous Learning Loop

```
┌─────────────────────────────────────────────────────────────┐
│  PHASE 1: ERROR OCCURS                                      │
│  System automatically records error with context            │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 2: PATTERN DETECTION                                 │
│  Signature-based matching identifies recurring issues       │
│  Threshold: 2+ occurrences triggers suggestion              │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 3: AI NOTIFICATION                                   │
│  "💡 This error occurred multiple times. Consider           │
│   extracting a lesson via MCP tool."                        │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 4: LESSON SUGGESTION                                 │
│  AI: reflection_suggest_lesson_from_error(errorPatternId)   │
│  Returns auto-populated fields:                             │
│  - patternName: "error-mongodb-timeout"                     │
│  - problem: "Failed to connect to MongoDB..."               │
│  - context: "Occurred 3 times between..."                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 5: LESSON EXTRACTION                                 │
│  AI reviews, refines, and extracts lesson:                  │
│  reflection_extract_lesson({                                │
│    patternName, problem, solution,                          │
│    antipattern, applicableTo, confidence                    │
│  })                                                          │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 6: LESSON INDEXED                                    │
│  - Stored in MongoDB reflections collection                 │
│  - Added to experience_index for fast lookup                │
│  - Available for future proactive queries                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 7: PROACTIVE RECOMMENDATIONS                         │
│  Before risky operations, AI queries:                       │
│  reflection_query_relevant_lessons({                        │
│    situation: "about to implement database connection"      │
│  })                                                          │
│  → Returns ranked lessons with confidence scores            │
└─────────────────────────────────────────────────────────────┘
                          ↓
                  ✨ CONTINUOUS LEARNING ✨
```

## MCP Tool Reference

### 1. reflection_record_decision

**Purpose:** Track a decision with context, reasoning, and predictions for later outcome comparison.

**Parameters:**
```json
{
  "chatId": "string (required)",
  "taskId": "string (optional)",
  "context": {
    "userRequest": "What user asked for",
    "availableInfo": "What information was available",
    "uncertainty": "What was uncertain"
  },
  "decision": {
    "action": "What action was decided",
    "reasoning": "Why this decision was made",
    "alternatives": ["Other options considered"],
    "confidence": 0.75
  },
  "predictions": {
    "successProbability": 0.8,
    "timeEstimate": "2 hours",
    "risks": ["Potential risks"]
  }
}
```

**Returns:**
```json
{
  "decisionId": "uuid",
  "stored": true
}
```

### 2. reflection_record_outcome

**Purpose:** Record actual outcome of a decision and compare to predictions for confidence calibration.

**Parameters:**
```json
{
  "decisionId": "uuid (required)",
  "outcome": {
    "success": true,
    "actualResult": "What actually happened",
    "userFeedback": "User reaction",
    "rootCause": "Root cause if issues occurred"
  },
  "analysis": {
    "predictionAccuracy": 0.9,
    "missedSignals": ["What was missed"],
    "confidenceCalibration": "well-calibrated|overconfident|underconfident"
  }
}
```

**Returns:**
```json
{
  "outcomeId": "uuid",
  "linked": true,
  "calibration": "well-calibrated"
}
```

### 3. reflection_extract_lesson

**Purpose:** Extract a transferable lesson that applies to future similar situations.

**Parameters:**
```json
{
  "patternName": "kebab-case-name (required)",
  "context": "In what context does this apply",
  "problem": "What problem occurred (required)",
  "solution": "How to solve it (required)",
  "antipattern": "What NOT to do",
  "applicableTo": ["tags", "for", "situations"],
  "confidence": 0.9
}
```

**Returns:**
```json
{
  "lessonId": "uuid",
  "indexed": true,
  "pattern": "pattern-name"
}
```

### 4. reflection_suggest_lesson_from_error ✨ NEW

**Purpose:** Get auto-populated lesson fields from a recurring error pattern.

**When to use:** After system detects recurring error (2+ occurrences) and suggests lesson extraction.

**Parameters:**
```json
{
  "errorPatternId": "uuid (required)"
}
```

**Returns:**
```json
{
  "errorPatternId": "uuid",
  "suggestedPattern": "error-mongodb-timeout",
  "problem": "Failed to connect to MongoDB: timeout after 30s",
  "context": "Error type: mongodb-timeout, Occurred 3 times...",
  "occurrences": 3,
  "errorType": "mongodb-timeout",
  "recentError": {
    "timestamp": "2025-11-03T...",
    "message": "Full error message",
    "stackTrace": "Stack trace",
    "context": {"host": "localhost", "port": 27017}
  }
}
```

**Example Usage:**
```javascript
// 1. Error occurs multiple times
// 2. System responds with: shouldSuggestLesson: true
// 3. AI calls:
const suggestion = await reflection_suggest_lesson_from_error({
  errorPatternId: "uuid-from-error-response"
});

// 4. AI reviews suggestion and extracts lesson:
await reflection_extract_lesson({
  patternName: suggestion.suggestedPattern,
  problem: suggestion.problem,
  solution: "Add retry logic with exponential backoff...",
  antipattern: "Immediate failure without retries",
  applicableTo: ["database", "mongodb", "resilience"],
  confidence: 0.9
});
```

### 5. reflection_query_relevant_lessons ✨ NEW

**Purpose:** Proactively query past lessons BEFORE making decisions or taking risky actions.

**When to use:**
- Before implementing new features
- Before making architectural decisions
- Before deploying to production
- When encountering similar situations to past experiences

**Parameters:**
```json
{
  "situation": "Describe what you're about to do (required)",
  "limit": 5
}
```

**Returns:**
```json
{
  "lessonsFound": 2,
  "situation": "implementing database connection",
  "lessons": [
    {
      "id": "uuid",
      "patternName": "mongodb-connection-timeout-resilience",
      "problem": "Connection timeouts without retry",
      "solution": "Implement exponential backoff...",
      "antipattern": "Immediate failure",
      "context": "MongoDB production",
      "confidence": 0.9,
      "timestamp": "2025-11-03T..."
    }
  ]
}
```

**Example Usage:**
```javascript
// Before implementing database connection
const recommendations = await reflection_query_relevant_lessons({
  situation: "about to implement MongoDB connection for new service",
  limit: 3
});

if (recommendations.lessonsFound > 0) {
  console.log("💡 Found relevant lessons from past experience:");
  recommendations.lessons.forEach(lesson => {
    console.log(`- ${lesson.patternName}: ${lesson.solution}`);
  });
}
```

## Error Tracking System

### How It Works

1. **Error Occurs** → System records via `RecordError()`
   - Creates signature: `errorType:normalizedMessage`
   - Stores error instance with timestamp, stack trace, context

2. **Pattern Matching** → Upserts error pattern
   - First occurrence: creates new pattern
   - Subsequent occurrences: increments counter
   - Maintains last 5 error instances

3. **Threshold Detection** → 2+ occurrences
   - Returns `shouldSuggestLesson: true`
   - Provides message directing to MCP tool

4. **Lesson Extraction** → AI approval required
   - AI calls `reflection_suggest_lesson_from_error`
   - Reviews auto-populated fields
   - Customizes and extracts lesson

5. **Pattern Marked** → Prevents duplicate suggestions
   - Links lesson to error pattern
   - Sets `lessonExtracted: true`

### Error Pattern Schema

```go
type ErrorPattern struct {
    ID             string          // UUID
    Signature      string          // "errorType:normalizedMessage"
    ErrorType      string          // "mongodb-timeout"
    MessagePattern string          // Full error message
    Occurrences    int             // Count of occurrences
    FirstSeen      time.Time       // First occurrence
    LastSeen       time.Time       // Most recent occurrence
    RecentErrors   []ErrorInstance // Last 5 occurrences
    LessonExtracted bool           // Lesson already created?
    RelatedLesson  string          // Lesson UUID if extracted
}

type ErrorInstance struct {
    Timestamp  time.Time
    Message    string
    StackTrace string
    Context    map[string]interface{}
}
```

## Search Capabilities

### Full-Text Search

The system provides MongoDB regex-based full-text search across:
- Pattern name
- Problem description
- Solution description
- Context
- Antipattern description

**Endpoint:** `GET /api/v1/reflection/search?q=query&limit=10`

**Example Queries:**
```bash
# Find MongoDB lessons
curl "http://localhost:4097/api/v1/reflection/search?q=MongoDB"

# Find TypeScript lessons
curl "http://localhost:4097/api/v1/reflection/search?q=TypeScript"

# Find hardcoding antipatterns
curl "http://localhost:4097/api/v1/reflection/search?q=hardcoding"
```

### Pattern Filtering

**Endpoint:** `GET /api/v1/reflection/lessons?pattern=X&tag=Y&limit=20`

**Example:**
```bash
# Find lessons by pattern name
curl "http://localhost:4097/api/v1/reflection/lessons?pattern=mongodb"

# Find lessons by tag
curl "http://localhost:4097/api/v1/reflection/lessons?tag=database"
```

## Current System State

### Lessons in System: 10

1. **hardcoding-dynamic-values** (0.95)
   - Problem: Hardcoded 768 dimensions when providers vary
   - Solution: Query source dynamically using GetDimensions()

2. **collection-name-resolution-mismatch** (0.90)
   - Problem: Direct collection usage without abstraction
   - Solution: Use knowledgeStorage layer for resolution

3. **mongodb-immutable-field-modification** (0.95)
   - Problem: Tried to modify _id after creation
   - Solution: Use $setOnInsert for creation-only fields

4. **auto-recovery-from-configuration-drift** (0.85)
   - Problem: Dimension mismatches on provider change
   - Solution: Auto-detect, delete, re-embed, recreate

5. **typescript-build-safety** (0.90)
   - Problem: Unused imports/variables break builds
   - Solution: Remove before production build

6. **fullstack-metacognitive-integration** (0.95)
   - Problem: How to integrate across full stack
   - Solution: Layer integration (MongoDB → Storage → MCP → REST → UI)

7. **react-reflection-ui-patterns** (0.88)
   - Problem: How to visualize complex reflection data
   - Solution: Tabbed interface, color-coded confidence, grid layout

8. **test-experience-index** (0.95)
   - Problem: Experience index failing
   - Solution: Use setOnInsert for _id field

9. **mongodb-connection-timeout-resilience** (0.90)
   - Problem: Connection timeouts without retry
   - Solution: Exponential backoff (3 retries: 1s, 2s, 4s)

10. **[Your new lessons will appear here]**

## Testing Guide

### Test Scripts Available

1. **test_error_tracking.sh** - Test error detection and pattern matching
2. **test_full_cycle.sh** - Test complete learning cycle
3. **test_recommendations.sh** - Test proactive recommendations

### Running Tests

```bash
# Make scripts executable
chmod +x test_*.sh

# Test error tracking
./test_error_tracking.sh

# Test full autonomous learning cycle
./test_full_cycle.sh

# Test proactive recommendations
./test_recommendations.sh
```

### Manual Testing with Jedi

#### Test 1: Error Detection & Lesson Extraction

```bash
# 1. Trigger an error twice
curl -X POST http://localhost:4097/api/v1/reflection/test-error \
  -H "Content-Type: application/json" \
  -d '{
    "errorType": "api-rate-limit",
    "message": "Rate limit exceeded: 100 requests per minute",
    "stackTrace": "at api/client.go:45",
    "context": {"endpoint": "/api/v1/data", "limit": 100}
  }'

# Run again to trigger suggestion
# Response will include: "shouldSuggestLesson": true

# 2. Use MCP tool to get suggestion
# (Jedi will call this)
# reflection_suggest_lesson_from_error({ errorPatternId: "uuid" })

# 3. Extract lesson
# (Jedi will call this)
# reflection_extract_lesson({
#   patternName: "api-rate-limit-handling",
#   problem: "Rate limit exceeded...",
#   solution: "Implement request throttling...",
#   confidence: 0.9
# })
```

#### Test 2: Proactive Recommendations

```bash
# Query for lessons before action
# (Jedi will call this)
# reflection_query_relevant_lessons({
#   situation: "about to implement API client with external service"
# })

# Should return rate limit lesson if extracted in Test 1
```

#### Test 3: Decision → Outcome Tracking

```bash
# 1. Record decision
# reflection_record_decision({
#   chatId: "test-session",
#   context: {
#     userRequest: "Implement caching layer",
#     availableInfo: "Redis available, 100ms latency budget",
#     uncertainty: "Cache invalidation strategy"
#   },
#   decision: {
#     action: "Use Redis with TTL-based invalidation",
#     reasoning: "Simple, predictable, low latency",
#     confidence: 0.75
#   },
#   predictions: {
#     successProbability: 0.8,
#     timeEstimate: "2 hours"
#   }
# })

# 2. Record outcome
# reflection_record_outcome({
#   decisionId: "uuid-from-step-1",
#   outcome: {
#     success: true,
#     actualResult: "Implemented in 1.5 hours, 50ms latency"
#   },
#   analysis: {
#     predictionAccuracy: 0.9,
#     confidenceCalibration: "well-calibrated"
#   }
# })
```

## Best Practices

### For AI Agents (like Jedi)

1. **Always Query Before Acting**
   - Call `reflection_query_relevant_lessons()` before:
     - Implementing new features
     - Making architectural decisions
     - Deploying to production
     - Working with unfamiliar technologies

2. **Track Important Decisions**
   - Use `reflection_record_decision()` for:
     - Architectural choices
     - Algorithm selection
     - Performance trade-offs
     - Security decisions

3. **Close the Loop**
   - Always call `reflection_record_outcome()` after:
     - Completing a tracked decision
     - User provides feedback
     - System behavior is observed

4. **Extract Lessons Actively**
   - When errors occur 2+ times → respond to suggestion
   - After solving difficult problems → extract lesson
   - When discovering antipatterns → document them

5. **Trust the Confidence Scores**
   - 0.9+ = Very reliable, follow closely
   - 0.7-0.9 = Reliable, adapt as needed
   - 0.5-0.7 = Consider carefully, may not apply
   - <0.5 = Low confidence, use judgment

### For System Operators

1. **Monitor Error Patterns**
   - Check `/api/v1/reflection/search?q=error` periodically
   - Review error patterns that haven't been converted to lessons

2. **Curate Lesson Quality**
   - Review lessons via REST API
   - Update confidence scores based on effectiveness
   - Remove outdated lessons

3. **Track Recommendation Usage**
   - Monitor which lessons are frequently queried
   - Identify gaps in lesson coverage
   - Add lessons for common scenarios

## Technical Details

### File Locations

```
hyper/
├── internal/
│   ├── mcp/
│   │   ├── storage/
│   │   │   └── reflection_storage.go    # Data layer
│   │   └── handlers/
│   │       └── reflection_tools.go       # MCP tools
│   ├── handlers/
│   │   └── reflection_handler.go         # REST API
│   └── server/
│       └── http_server.go                # Route registration
├── test_error_tracking.sh                # Error test
├── test_full_cycle.sh                    # Full cycle test
└── test_recommendations.sh               # Recommendation test
```

### Database Indexes

```javascript
// reflections collection
db.reflections.createIndex({ type: 1 })
db.reflections.createIndex({ chatId: 1 })
db.reflections.createIndex({ taskId: 1 })

// experience_index collection
db.experience_index.createIndex({ pattern: 1 }, { unique: true })

// error_patterns collection
db.error_patterns.createIndex({ signature: 1 }, { unique: true })
```

### Performance Characteristics

- **Error Recording:** O(1) with upsert
- **Pattern Matching:** O(1) with signature hash
- **Lesson Search:** O(n) with regex (fast enough for <10k lessons)
- **Recommendation Query:** O(n) with regex filtering

## Future Enhancements

### Priority 1: Production Middleware
- Automatic error interception in production
- Middleware to record all errors automatically
- Integration with logging system

### Priority 2: Lesson Effectiveness Tracking
- Track which recommendations were followed
- Measure impact on error reduction
- Confidence score auto-adjustment

### Priority 3: Advanced Pattern Detection
- Clustering similar error patterns
- Temporal analysis of lesson relevance
- Cross-reference decisions → outcomes → lessons

### Priority 4: UI Dashboard
- Visualize learning patterns over time
- Show most valuable lessons
- Display recommendation usage stats
- Interactive lesson management

## Conclusion

The Hyperion Metacognitive Reflection System provides **complete autonomous learning** capabilities, enabling AI agents to:

✅ Track their decisions and outcomes
✅ Learn from recurring errors automatically
✅ Build a knowledge base of transferable lessons
✅ Proactively recommend relevant past experiences
✅ Continuously improve through self-awareness

The system is **production-ready** and **fully tested**. Ready for Jedi! 🚀
