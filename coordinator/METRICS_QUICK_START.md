# Metrics Resources - Quick Start Guide

## 🚀 Accessing Metrics

### Via MCP Resources

```typescript
// 1. Get squad velocity
const velocity = await mcp.read_resource({
  uri: "hyperion://metrics/squad-velocity"
});

console.log(velocity);
// Output:
// {
//   "squads": [
//     {
//       "squadName": "backend-services",
//       "todayCompleted": 5,
//       "weekCompleted": 23,
//       "allTimeCompleted": 156,
//       "completionRate": 78.0,
//       "averageTodoDuration": 45.3,
//       "lastActivity": "2025-10-04T14:30:00Z"
//     }
//   ],
//   "timestamp": "2025-10-04T16:00:00Z"
// }

// 2. Get context efficiency
const efficiency = await mcp.read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

console.log(`Platform efficiency: ${efficiency.metrics.overallStats.efficiencyScore}/100`);
console.log(`Analysis: ${efficiency.analysis.efficiency}`);
// Output:
// Platform efficiency: 87.5/100
// Analysis: Good - squad performance is above target
```

## 📊 Common Use Cases

### 1. Load Balancing Across Squads

```typescript
// Get squad velocity to find least busy squad
const velocity = await mcp.read_resource({
  uri: "hyperion://metrics/squad-velocity"
});

// Sort by current workload (ascending)
const squads = velocity.squads.sort((a, b) =>
  a.todayTaskCount - b.todayTaskCount
);

const leastBusySquad = squads[0].squadName;
console.log(`Assigning task to: ${leastBusySquad}`);

// Create task for squad with lowest workload
await coordinator_create_agent_task({
  humanTaskId: "task-123",
  agentName: leastBusySquad,
  role: "Implement feature X",
  todos: [...]
});
```

### 2. Monitor Platform Health

```typescript
// Check if platform is meeting 15x efficiency goal
const efficiency = await mcp.read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

const score = efficiency.metrics.overallStats.efficiencyScore;
const threshold = 75; // Good performance threshold

if (score < threshold) {
  console.warn(`⚠️ Efficiency below target: ${score}/${threshold}`);

  // Investigate slowest squad
  const slowest = efficiency.metrics.squadStats
    .sort((a, b) => b.averageCompletionTime - a.averageCompletionTime)[0];

  console.log(`Bottleneck: ${slowest.squadName} (${slowest.averageCompletionTime}h avg)`);

  // Take action
  await sendAlert({
    type: "efficiency_warning",
    score: score,
    bottleneck: slowest.squadName
  });
} else {
  console.log(`✅ Platform performing well: ${score}/100`);
}
```

### 3. Track Week-over-Week Trends

```typescript
// Monitor performance trends
const efficiency = await mcp.read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

const weekly = efficiency.metrics.trendData.weeklyCompletions;
if (weekly.length >= 2) {
  const current = weekly[weekly.length - 1];
  const previous = weekly[weekly.length - 2];

  const change = ((current.avgPerDay - previous.avgPerDay) / previous.avgPerDay) * 100;

  console.log(`Week-over-week change: ${change.toFixed(1)}%`);
  console.log(`Current: ${current.avgPerDay.toFixed(1)} tasks/day`);
  console.log(`Previous: ${previous.avgPerDay.toFixed(1)} tasks/day`);

  if (change < -10) {
    console.warn("⚠️ Performance declining - review task quality and context");
  } else if (change > 10) {
    console.log("🎉 Performance improving - keep it up!");
  }
}
```

### 4. Ensure Tasks are Well-Scoped

```typescript
// Check task complexity distribution
const efficiency = await mcp.read_resource({
  uri: "hyperion://metrics/context-efficiency"
});

const complexity = efficiency.metrics.taskComplexity;

console.log(`Average TODOs per task: ${complexity.averageTodoCount.toFixed(1)}`);
console.log(`Complex tasks (>5 TODOs): ${complexity.complexTasksPercent.toFixed(1)}%`);
console.log(`Simple tasks (≤3 TODOs): ${complexity.simpleTasksPercent.toFixed(1)}%`);

// Alert if too many complex tasks (god tasks)
if (complexity.complexTasksPercent > 20) {
  console.warn("⚠️ Too many complex tasks - consider breaking down into smaller tasks");
  console.log("Distribution:", complexity.todoDistribution);
}

// Alert if average is too high
if (complexity.averageTodoCount > 5) {
  console.warn("⚠️ Tasks are too large on average - aim for 3-5 TODOs per task");
}
```

## 🎯 Performance Targets

Use these metrics to validate the 15x efficiency goal:

```typescript
async function validatePlatformEfficiency() {
  const velocity = await mcp.read_resource({
    uri: "hyperion://metrics/squad-velocity"
  });

  const efficiency = await mcp.read_resource({
    uri: "hyperion://metrics/context-efficiency"
  });

  const targets = {
    efficiencyScore: { target: 75, actual: efficiency.metrics.overallStats.efficiencyScore },
    avgCompletionTime: { target: 2, actual: efficiency.metrics.overallStats.averageCompletionTime },
    tasksPerDay: { target: 8, actual: efficiency.metrics.overallStats.tasksPerDay },
    completionRate: { target: 80, actual: 0 } // Calculate from velocity
  };

  // Calculate platform-wide completion rate
  let totalCompleted = 0;
  let totalTasks = 0;
  velocity.squads.forEach(squad => {
    totalCompleted += squad.allTimeCompleted;
    totalTasks += squad.allTimeTaskCount;
  });
  targets.completionRate.actual = (totalCompleted / totalTasks) * 100;

  // Check each target
  const results = {
    efficiencyScore: targets.efficiencyScore.actual >= targets.efficiencyScore.target,
    avgCompletionTime: targets.avgCompletionTime.actual <= targets.avgCompletionTime.target,
    tasksPerDay: targets.tasksPerDay.actual >= targets.tasksPerDay.target,
    completionRate: targets.completionRate.actual >= targets.completionRate.target
  };

  const allTargetsMet = Object.values(results).every(r => r);

  console.log("=== 15x Efficiency Validation ===");
  console.log(`✅ Efficiency Score: ${targets.efficiencyScore.actual}/100 (target: ≥${targets.efficiencyScore.target})`);
  console.log(`${results.avgCompletionTime ? '✅' : '❌'} Avg Completion: ${targets.avgCompletionTime.actual.toFixed(1)}h (target: ≤${targets.avgCompletionTime.target}h)`);
  console.log(`${results.tasksPerDay ? '✅' : '❌'} Tasks/Day: ${targets.tasksPerDay.actual.toFixed(1)} (target: ≥${targets.tasksPerDay.target})`);
  console.log(`${results.completionRate ? '✅' : '❌'} Completion Rate: ${targets.completionRate.actual.toFixed(1)}% (target: ≥${targets.completionRate.target}%)`);
  console.log(`\n${allTargetsMet ? '🎉 ALL TARGETS MET!' : '⚠️ Some targets not met - review and improve'}`);

  return allTargetsMet;
}

// Run validation
validatePlatformEfficiency();
```

## 📈 Dashboard Integration

```typescript
// Example: Real-time metrics dashboard
async function buildMetricsDashboard() {
  const [velocity, efficiency] = await Promise.all([
    mcp.read_resource({ uri: "hyperion://metrics/squad-velocity" }),
    mcp.read_resource({ uri: "hyperion://metrics/context-efficiency" })
  ]);

  return {
    summary: {
      platformScore: efficiency.metrics.overallStats.efficiencyScore,
      totalSquads: efficiency.metrics.overallStats.activeSquads,
      todayCompleted: efficiency.metrics.overallStats.tasksPerDay,
      weekCompleted: efficiency.metrics.overallStats.tasksPerWeek,
      trend: efficiency.analysis.trend
    },
    squads: velocity.squads.map(squad => ({
      name: squad.squadName,
      todayCompleted: squad.todayCompleted,
      todayTotal: squad.todayTaskCount,
      completionRate: squad.completionRate,
      avgDuration: squad.averageTodoDuration,
      status: squad.todayTaskCount > 10 ? 'busy' : 'available'
    })),
    trends: {
      daily: efficiency.metrics.trendData.dailyCompletions,
      weekly: efficiency.metrics.trendData.weeklyCompletions
    },
    complexity: efficiency.metrics.taskComplexity
  };
}

// Refresh dashboard every 5 minutes
setInterval(async () => {
  const dashboard = await buildMetricsDashboard();
  updateUI(dashboard);
}, 5 * 60 * 1000);
```

## 🔔 Alerting Examples

```typescript
// Set up automated alerts
async function setupMetricsAlerts() {
  const efficiency = await mcp.read_resource({
    uri: "hyperion://metrics/context-efficiency"
  });

  const alerts = [];

  // Alert 1: Efficiency below threshold
  if (efficiency.metrics.overallStats.efficiencyScore < 75) {
    alerts.push({
      severity: 'warning',
      type: 'efficiency_low',
      message: `Platform efficiency is ${efficiency.metrics.overallStats.efficiencyScore}/100`,
      recommendation: 'Review task context quality and squad workload distribution'
    });
  }

  // Alert 2: Declining trend
  if (efficiency.analysis.trend.includes('decline') || efficiency.analysis.trend.includes('Declining')) {
    alerts.push({
      severity: 'warning',
      type: 'trend_declining',
      message: efficiency.analysis.trend,
      recommendation: 'Investigate recent changes in task planning or execution'
    });
  }

  // Alert 3: High task complexity
  if (efficiency.metrics.taskComplexity.complexTasksPercent > 20) {
    alerts.push({
      severity: 'info',
      type: 'complexity_high',
      message: `${efficiency.metrics.taskComplexity.complexTasksPercent.toFixed(1)}% of tasks are complex (>5 TODOs)`,
      recommendation: 'Break down complex tasks into smaller, more manageable units'
    });
  }

  // Alert 4: Squad bottleneck
  const slowestSquad = efficiency.metrics.squadStats
    .sort((a, b) => b.averageCompletionTime - a.averageCompletionTime)[0];

  if (slowestSquad.averageCompletionTime > 4) {
    alerts.push({
      severity: 'warning',
      type: 'squad_bottleneck',
      message: `${slowestSquad.squadName} has high completion time (${slowestSquad.averageCompletionTime.toFixed(1)}h avg)`,
      recommendation: 'Review squad capacity, task complexity, or context quality'
    });
  }

  // Send alerts
  if (alerts.length > 0) {
    await sendAlerts(alerts);
  }

  return alerts;
}

// Run alerts check every hour
setInterval(setupMetricsAlerts, 60 * 60 * 1000);
```

## 📝 Summary

### Quick Access
- **Squad Velocity**: `hyperion://metrics/squad-velocity`
- **Context Efficiency**: `hyperion://metrics/context-efficiency`

### Key Metrics to Monitor
1. **Efficiency Score** (target: ≥75)
2. **Avg Completion Time** (target: ≤2h)
3. **Tasks Per Day** (target: ≥8)
4. **Completion Rate** (target: ≥80%)
5. **Complex Tasks %** (target: ≤20%)

### When to Use
- **Load Balancing**: Use velocity to distribute work evenly
- **Performance Monitoring**: Check efficiency score daily
- **Trend Analysis**: Review weekly trends for continuous improvement
- **Quality Assurance**: Monitor task complexity to prevent god tasks

**For complete documentation, see:** [METRICS_RESOURCES.md](./METRICS_RESOURCES.md)
