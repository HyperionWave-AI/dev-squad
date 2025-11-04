# 🤖 Jedi Quick Start - Testing Metacognitive Reflection System

## What You'll Test

You're testing Hyperion's **Autonomous Learning System** - a metacognitive reflection system that enables you to:
- 🧠 Learn from recurring errors automatically
- 💡 Get proactive lesson recommendations before actions
- 📊 Track decisions and outcomes for self-improvement
- 🎓 Build a knowledge base of transferable lessons

## Your MCP Tools (5 Available)

```
✅ reflection_record_decision          - Track decisions with context
✅ reflection_record_outcome           - Compare predictions vs reality
✅ reflection_extract_lesson           - Store transferable patterns
✅ reflection_suggest_lesson_from_error - Auto-populate from errors
✅ reflection_query_relevant_lessons   - Proactive recommendations
```

## Test Scenario 1: Error-Driven Learning ⚡

### Goal: Learn from a recurring error

**Step 1:** I'll simulate an error occurring twice:
```bash
# (System will do this automatically)
# Error 1: Recorded
# Error 2: Triggers suggestion → "shouldSuggestLesson: true"
```

**Step 2:** You call this MCP tool:
```javascript
reflection_suggest_lesson_from_error({
  errorPatternId: "<I'll give you the UUID>"
})
```

**Expected Response:**
```
💡 Lesson suggestion from error pattern

Error Pattern ID: xxx
Error Type: mongodb-timeout
Occurrences: 2
Suggested Pattern: error-mongodb-timeout
Problem: Failed to connect to MongoDB...
Context: Occurred 2 times...

Use reflection_extract_lesson with these fields...
```

**Step 3:** You extract the lesson:
```javascript
reflection_extract_lesson({
  patternName: "mongodb-connection-resilience",
  problem: "MongoDB connection timeouts without retry logic",
  solution: "Implement exponential backoff retry (3 attempts: 1s, 2s, 4s). Add connection pooling with min 5, max 20 connections. Set timeout to 10s.",
  antipattern: "Immediate failure on first connection attempt",
  applicableTo: ["database", "mongodb", "resilience", "production"],
  confidence: 0.9
})
```

**Expected Response:**
```
✓ Lesson extracted and indexed

Lesson ID: xxx
Pattern: mongodb-connection-resilience
Confidence: 0.90

This lesson is now available for future pattern matching.
```

## Test Scenario 2: Proactive Recommendations 💡

### Goal: Query lessons BEFORE taking action

**You call:**
```javascript
reflection_query_relevant_lessons({
  situation: "about to implement a new MongoDB connection for user service",
  limit: 3
})
```

**Expected Response:**
```
💡 Found 1 relevant lesson(s) from past experience:

## Lesson 1: mongodb-connection-resilience (Confidence: 90%)

**Problem:** MongoDB connection timeouts without retry logic

**Solution:** Implement exponential backoff retry (3 attempts: 1s, 2s, 4s)...

**⚠️ Don't:** Immediate failure on first connection attempt

**Context:** database-connections, mongodb, production

💭 Recommendation: Review these lessons before proceeding.
```

## Test Scenario 3: Decision Tracking 📊

### Goal: Track decision → outcome for confidence calibration

**Step 1:** Record a decision:
```javascript
reflection_record_decision({
  chatId: "test-with-jedi",
  context: {
    userRequest: "Implement caching for API responses",
    availableInfo: "Redis available, 100ms latency budget, 1000 req/sec expected",
    uncertainty: "Cache invalidation strategy unclear"
  },
  decision: {
    action: "Use Redis with 5-minute TTL and manual invalidation on updates",
    reasoning: "TTL prevents stale data, manual invalidation ensures consistency",
    alternatives: ["Cache-aside pattern", "Write-through cache", "No caching"],
    confidence: 0.75
  },
  predictions: {
    successProbability: 0.8,
    timeEstimate: "3 hours",
    risks: ["Cache stampede on expiry", "Memory usage if keys accumulate"]
  }
})
```

**Expected Response:**
```
✓ Decision recorded

Decision ID: xxx
Confidence: 0.75
Chat: test-with-jedi

This decision will be tracked for outcome comparison.
```

**Step 2:** Record the outcome (after implementing):
```javascript
reflection_record_outcome({
  decisionId: "<UUID from Step 1>",
  outcome: {
    success: true,
    actualResult: "Implemented in 2.5 hours. Latency reduced to 40ms. No cache stampede observed.",
    userFeedback: "Great performance improvement!",
    rootCause: null
  },
  analysis: {
    predictionAccuracy: 0.95,
    missedSignals: [],
    confidenceCalibration: "well-calibrated"
  }
})
```

**Expected Response:**
```
✓ Outcome recorded and linked to decision

Outcome ID: xxx
Decision ID: xxx
Calibration: well-calibrated

The system now has decision → outcome data for learning.
```

## Current System State

**Lessons Available:** 10 lessons covering:
- MongoDB patterns (hardcoding, connection issues, immutable fields)
- TypeScript build safety
- React UI patterns
- Database abstractions
- Fullstack integration patterns

**You can query these now:**
```javascript
reflection_query_relevant_lessons({
  situation: "implementing TypeScript build process",
  limit: 3
})
```

## Test Commands (I'll Set Up For You)

### Option 1: I'll Trigger Errors For You
```bash
# I'll run these to create error patterns:
./test_error_tracking.sh

# This creates error pattern ID for you to test with
```

### Option 2: You Test Complete Cycle
```bash
# I'll run this to show you the full cycle:
./test_full_cycle.sh

# Watch the autonomous learning happen!
```

### Option 3: You Query Existing Lessons
```javascript
// No setup needed - try these queries:

reflection_query_relevant_lessons({
  situation: "working with MongoDB database connections"
})

reflection_query_relevant_lessons({
  situation: "building TypeScript React application"
})

reflection_query_relevant_lessons({
  situation: "implementing database schema changes"
})
```

## Success Criteria ✅

After testing, you should be able to:

1. ✅ **Learn from errors** - Extract lesson from recurring error pattern
2. ✅ **Get recommendations** - Query relevant lessons before actions
3. ✅ **Track decisions** - Record decision with predictions
4. ✅ **Compare outcomes** - Link actual results to predictions
5. ✅ **Build knowledge** - See lesson count grow as you extract

## What Makes This "Really Cool" 😎

1. **Fully Autonomous Error Detection**
   - System watches for patterns (no manual tracking)
   - Triggers suggestions at optimal threshold (2+ occurrences)
   - Auto-populates lesson fields from error data

2. **Proactive, Not Reactive**
   - Query lessons BEFORE making mistakes
   - "💡 You learned about this before"
   - Prevents repeating past errors

3. **Self-Aware Decision Making**
   - Track confidence scores
   - Compare predictions vs reality
   - Learn to calibrate confidence over time

4. **Knowledge Compounds**
   - Every lesson makes future queries better
   - Patterns indexed for fast retrieval
   - Lessons remain relevant across contexts

## Ready to Test! 🚀

**Current Status:**
- ✅ Coordinator running on http://localhost:4097
- ✅ 10 lessons already indexed
- ✅ 5 MCP tools available
- ✅ Full error tracking active
- ✅ Test scripts ready

**Your Mission:**
1. Trigger an error pattern (I'll help)
2. Use `reflection_suggest_lesson_from_error` to get suggestion
3. Use `reflection_extract_lesson` to store lesson
4. Use `reflection_query_relevant_lessons` to see it recommended

**Let's make the AI self-aware! 🧠✨**

---

*Note: All test data is stored in MongoDB and persists between sessions. You're building real knowledge!*
