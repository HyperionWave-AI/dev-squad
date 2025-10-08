# Hyperion Coordinator - Metrics Resources

## Overview

The Hyperion Coordinator MCP server provides two powerful metrics resources to track squad performance and validate the 15x efficiency improvement goal.

## Available Resources

### 1. Squad Velocity Metrics (`hyperion://metrics/squad-velocity`)

Tracks task completion rates by squad across multiple time windows.

**URI:** `hyperion://metrics/squad-velocity`

**Response Structure:**
```json
{
  "squads": [
    {
      "squadName": "backend-services",
      "todayCompleted": 5,
      "weekCompleted": 23,
      "allTimeCompleted": 156,
      "todayTaskCount": 8,
      "weekTaskCount": 35,
      "allTimeTaskCount": 200,
      "completionRate": 78.0,
      "averageTodoDuration": 45.3,
      "lastActivity": "2025-10-04T14:30:00Z"
    },
    {
      "squadName": "frontend-experience",
      "todayCompleted": 3,
      "weekCompleted": 18,
      "allTimeCompleted": 98,
      "todayTaskCount": 6,
      "weekTaskCount": 28,
      "allTimeTaskCount": 130,
      "completionRate": 75.4,
      "averageTodoDuration": 38.7,
      "lastActivity": "2025-10-04T15:15:00Z"
    }
  ],
  "timestamp": "2025-10-04T16:00:00Z",
  "windows": {
    "today": "2025-10-04T00:00:00Z",
    "week": "2025-09-27T00:00:00Z"
  }
}
```

**Metrics Explained:**

- **todayCompleted**: Tasks completed in the last 24 hours
- **weekCompleted**: Tasks completed in the last 7 days
- **allTimeCompleted**: Total completed tasks
- **completionRate**: Percentage of all-time tasks that are completed
- **averageTodoDuration**: Average time to complete a TODO item (in minutes)
- **lastActivity**: Timestamp of most recent task update

**Usage Example:**
```bash
# Access via MCP client
mcp read hyperion://metrics/squad-velocity

# Or use the coordinator tool
coordinator_read_resource({ uri: "hyperion://metrics/squad-velocity" })
```

---

### 2. Context Efficiency Metrics (`hyperion://metrics/context-efficiency`)

Analyzes agent context usage patterns and performance to validate the context-first architecture.

**URI:** `hyperion://metrics/context-efficiency`

**Response Structure:**
```json
{
  "metrics": {
    "overallStats": {
      "averageCompletionTime": 2.3,
      "tasksPerDay": 8.5,
      "tasksPerWeek": 41.2,
      "totalTasksCompleted": 254,
      "activeSquads": 7,
      "efficiencyScore": 87.5
    },
    "squadStats": [
      {
        "squadName": "backend-services",
        "averageCompletionTime": 1.8,
        "tasksCompleted": 156,
        "averageTodoCount": 4.2,
        "successRate": 78.0
      },
      {
        "squadName": "frontend-experience",
        "averageCompletionTime": 2.1,
        "tasksCompleted": 98,
        "averageTodoCount": 3.8,
        "successRate": 75.4
      }
    ],
    "trendData": {
      "dailyCompletions": [
        {
          "date": "2025-09-28",
          "completed": 12,
          "inProgress": 5,
          "blocked": 1
        },
        {
          "date": "2025-09-29",
          "completed": 15,
          "inProgress": 4,
          "blocked": 0
        }
      ],
      "weeklyCompletions": [
        {
          "weekStart": "2025-09-23",
          "completed": 38,
          "avgPerDay": 5.4
        },
        {
          "weekStart": "2025-09-30",
          "completed": 41,
          "avgPerDay": 5.9
        }
      ]
    },
    "taskComplexity": {
      "averageTodoCount": 4.1,
      "todoDistribution": {
        "1-3": 45,
        "4-5": 38,
        "6-7": 12,
        "8+": 5
      },
      "complexTasksPercent": 17.0,
      "simpleTasksPercent": 45.0
    }
  },
  "timestamp": "2025-10-04T16:00:00Z",
  "analysis": {
    "efficiency": "Good - squad performance is above target",
    "trend": "Improving trend (+9.3% week-over-week)"
  }
}
```

**Metrics Explained:**

#### Overall Stats
- **averageCompletionTime**: Average hours to complete a task
- **tasksPerDay**: Average tasks completed per day
- **tasksPerWeek**: Average tasks completed per week
- **efficiencyScore**: 0-100 score combining completion rate, speed, and throughput
  - 90-100: Excellent
  - 75-89: Good
  - 60-74: Fair
  - 40-59: Poor
  - 0-39: Critical

#### Squad Stats
- **averageCompletionTime**: Squad-specific completion time
- **tasksCompleted**: Total completed by squad
- **averageTodoCount**: Average TODOs per task
- **successRate**: Percentage of squad's tasks completed

#### Trend Data
- **dailyCompletions**: Last 7 days of completion data
- **weeklyCompletions**: Last 4 weeks with week-over-week comparison

#### Task Complexity
- **todoDistribution**: Breakdown by TODO count ranges
  - 1-3 TODOs: Simple tasks
  - 4-5 TODOs: Medium tasks
  - 6-7 TODOs: Complex tasks
  - 8+ TODOs: Very complex (consider splitting)
- **complexTasksPercent**: Tasks with >5 TODOs
- **simpleTasksPercent**: Tasks with ≤3 TODOs

**Usage Example:**
```bash
# Access via MCP client
mcp read hyperion://metrics/context-efficiency

# Or use the coordinator tool
coordinator_read_resource({ uri: "hyperion://metrics/context-efficiency" })
```

---

## Performance Targets

Based on the 15x efficiency improvement goal:

### Velocity Targets
- **Tasks per day**: ≥8 tasks/day (indicates parallel work)
- **Completion rate**: ≥80% (indicates good task definition)
- **Average TODO duration**: ≤60 minutes (indicates context efficiency)

### Efficiency Targets
- **Efficiency score**: ≥75 (Good or Excellent)
- **Average completion time**: ≤2 hours (context-first architecture goal)
- **Week-over-week trend**: Positive or stable

### Complexity Targets
- **Simple tasks**: ≥40% (well-scoped work)
- **Complex tasks**: ≤20% (avoid god tasks)
- **Average TODO count**: 3-5 (optimal task size)

---

## Implementation Details

### Efficiency Score Formula

The efficiency score (0-100) is calculated as:

```
efficiencyScore = (completionRate × 0.4) + (speedScore × 0.3) + (throughputScore × 0.3)

where:
- completionRate = (completed / total) × 100
- speedScore = max(0, 100 - ((avgTime - 2) / 22) × 100)
  - Target: <2 hours = 100%, 24 hours = 0%
- throughputScore = min(100, tasksPerDay × 10)
  - Target: 10 tasks/day = 100%
```

### Data Sources
- **TaskStorage**: MongoDB collections (human_tasks, agent_tasks)
- **Time Windows**:
  - Today: Last 24 hours (UTC midnight to now)
  - Week: Last 7 days
  - All-time: Complete history

### Performance Considerations
- Metrics computed on-demand (no caching)
- Queries filter by time windows for efficiency
- Full task history analyzed for all-time metrics
- No external API calls required

---

## Integration Examples

### Workflow Coordinator Usage

```typescript
// Before planning a sprint, check squad velocity
const velocity = await coordinator_read_resource({
  uri: "hyperion://metrics/squad-velocity"
});

// Assign work to squads based on completion rates and capacity
const squads = velocity.squads.sort((a, b) => a.todayTaskCount - b.todayTaskCount);
const leastBusySquad = squads[0].squadName;

// Create task for least busy squad
await coordinator_create_agent_task({
  humanTaskId: "...",
  agentName: leastBusySquad,
  role: "Implement feature X",
  todos: [...]
});
```

### Performance Monitoring

```typescript
// Check if 15x efficiency goal is being met
const efficiency = await coordinator_read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

const score = efficiency.metrics.overallStats.efficiencyScore;
if (score < 75) {
  console.warn(`Efficiency below target: ${score}/100`);
  console.log(`Analysis: ${efficiency.analysis.efficiency}`);

  // Investigate bottlenecks
  const slowestSquad = efficiency.metrics.squadStats
    .sort((a, b) => b.averageCompletionTime - a.averageCompletionTime)[0];

  console.log(`Slowest squad: ${slowestSquad.squadName} (${slowestSquad.averageCompletionTime}h avg)`);
}
```

### Trend Analysis

```typescript
// Monitor week-over-week improvement
const efficiency = await coordinator_read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

const trend = efficiency.analysis.trend;
console.log(`Performance trend: ${trend}`);

// Alert if declining
if (trend.includes("decline") || trend.includes("Declining")) {
  // Take corrective action
  console.warn("Performance declining - review context quality and task complexity");
}
```

---

## Testing

Comprehensive test suite included in `handlers/metrics_resources_test.go`:

```bash
# Run all metrics tests
go test ./handlers -v -run TestMetricsResourceHandler

# Test specific metrics
go test ./handlers -v -run TestMetricsResourceHandler_SquadVelocity
go test ./handlers -v -run TestMetricsResourceHandler_ContextEfficiency
go test ./handlers -v -run TestCalculateComplexityMetrics
go test ./handlers -v -run TestEfficiencyAnalysis
```

---

## Future Enhancements

Potential additions for future iterations:

1. **Historical Comparison**: Compare current metrics to previous periods
2. **Squad Benchmarking**: Compare squad performance against platform averages
3. **Predictive Analytics**: Forecast completion times based on task complexity
4. **Anomaly Detection**: Alert on unusual patterns (sudden drops, spikes)
5. **Custom Time Windows**: Allow clients to specify arbitrary date ranges
6. **Export Formats**: CSV/Excel export for external analysis
7. **Real-time Streaming**: WebSocket updates for live dashboard

---

## Conclusion

These metrics resources provide comprehensive visibility into squad performance and validate the context-first architecture's effectiveness. By tracking velocity, efficiency, and complexity, teams can:

- **Validate 15x improvement**: Monitor efficiency scores and throughput
- **Optimize task planning**: Use velocity data to balance squad workloads
- **Improve context quality**: Track completion times and identify bottlenecks
- **Maintain quality**: Ensure tasks stay well-scoped (complexity metrics)

**Target Achievement Indicators:**
- ✅ Efficiency score ≥75
- ✅ Average completion time ≤2 hours
- ✅ Tasks per day ≥8
- ✅ Completion rate ≥80%
- ✅ Positive week-over-week trends

Monitor these metrics regularly to ensure the parallel squad system delivers on its 15x efficiency promise.
