# Plugin System Documentation

## Overview

The chat system now features a **production-ready plugin architecture** that replaces hardcoded toggles with a flexible, extensible plugin registry system. This enables:

- ✅ **No more hardcoded toggles** - Modes are now plugins
- ✅ **Plugin registry** - Centralized plugin management
- ✅ **No duplicate code** - Plugins follow a single pattern
- ✅ **Full configuration** - Plugins are configurable and composable
- ✅ **Priority-based execution** - Plugins run in order
- ✅ **Lifecycle management** - Init/destroy hooks for setup/cleanup

## Architecture

### Plugin Interface

All plugins implement the `ChatPlugin` interface:

```typescript
interface ChatPlugin {
  // Metadata
  name: string;
  version: string;
  description?: string;
  priority: number; // 0-100, higher = runs first

  // Lifecycle hooks
  onInit?(): Promise<void>;
  onDestroy?(): Promise<void>;

  // Message processing hooks
  onMessageReceived?(message: ChatMessage): Promise<ChatMessage>;
  onMessageSent?(content: string): Promise<string>;

  // Tool execution hooks
  onToolCall?(toolCall: ToolCall): Promise<ToolCall>;
  onToolResult?(result: ToolResult): Promise<ToolResult>;

  // Session lifecycle hooks
  onSessionCreated?(sessionId: string): Promise<void>;
  onSessionDeleted?(sessionId: string): Promise<void>;
  onSessionChanged?(sessionId: string): Promise<void>;

  // Error handling hooks
  onError?(error: Error): Promise<void>;

  // Streaming hooks
  onStreamStart?(sessionId: string): Promise<void>;
  onStreamEnd?(sessionId: string): Promise<void>;
  onStreamChunk?(sessionId: string, chunk: string): Promise<string>;
}
```

### Plugin Registry

The `PluginRegistry` manages all plugins:

```typescript
interface IPluginRegistry {
  register(plugin: ChatPlugin): void;
  unregister(name: string): void;
  getPlugin(name: string): ChatPlugin | undefined;
  getPlugins(hook: keyof ChatPlugin): ChatPlugin[];
  executeHook<T>(hook: keyof ChatPlugin, data: T): Promise<T>;
  getAllPlugins(): ChatPlugin[];
  isPluginEnabled(name: string): boolean;
  setPluginEnabled(name: string, enabled: boolean): void;
}
```

## Built-in Plugins

### 1. Error Prevention Plugin

**Location:** `./ui/src/plugins/errorPreventionPlugin.ts`

**Purpose:** Enables AI to validate code and fix errors automatically.

**Features:**
- Tracks errors per session
- Monitors tool results for failures
- Provides system prompt enhancement
- Manages error statistics

**Usage:**
```typescript
import { errorPreventionPlugin } from '@/plugins/errorPreventionPlugin';

// Enable for a session
errorPreventionPlugin.setEnabled(true);

// Get error stats
const stats = errorPreventionPlugin.getSessionStats(sessionId);
console.log(`Errors: ${stats.errorCount}, Above threshold: ${stats.isAboveThreshold}`);

// Reset stats
errorPreventionPlugin.resetSessionStats(sessionId);
```

**System Prompt Enhancement:**
When enabled, adds guidance for:
- Code validation before execution
- Error detection and monitoring
- Automatic recovery strategies
- Proactive prevention measures
- Error reporting and verification

### 2. Complexity Analysis Plugin

**Location:** `./ui/src/plugins/complexityAnalysisPlugin.ts`

**Purpose:** Enables AI to analyze task complexity and suggest splitting large tasks.

**Features:**
- Analyzes message complexity
- Monitors tool calls for complexity indicators
- Tracks complexity metrics per session
- Provides system prompt enhancement

**Usage:**
```typescript
import { complexityAnalysisPlugin } from '@/plugins/complexityAnalysisPlugin';

// Enable for a session
complexityAnalysisPlugin.setEnabled(true);

// Get complexity metrics
const metrics = complexityAnalysisPlugin.getSessionMetrics(sessionId);
console.log(`Complexity: ${metrics.estimatedComplexity}, Time: ${metrics.estimatedTimeMinutes}m`);

// Set custom metrics
complexityAnalysisPlugin.setSessionMetrics(sessionId, {
  estimatedComplexity: 'high',
  estimatedTimeMinutes: 45,
  suggestedSplits: ['Phase 1: Setup', 'Phase 2: Implementation'],
  riskFactors: ['Breaking changes', 'Testing required'],
});

// Reset metrics
complexityAnalysisPlugin.resetSessionMetrics(sessionId);
```

**System Prompt Enhancement:**
When enabled, adds guidance for:
- Task assessment and complexity scoring
- Task splitting recommendations
- Risk analysis
- Estimated time and expertise requirements

## Using the Plugin System

### In React Components

Use the `usePluginRegistry` hook:

```typescript
import { usePluginRegistry } from '@/hooks/usePluginRegistry';

export function MyComponent() {
  const {
    registry,
    toggleErrorPrevention,
    toggleComplexityAnalysis,
    isErrorPreventionEnabled,
    isComplexityAnalysisEnabled,
  } = usePluginRegistry();

  // Toggle error prevention
  const handleToggleErrorPrevention = () => {
    toggleErrorPrevention(!isErrorPreventionEnabled());
  };

  // Toggle complexity analysis
  const handleToggleComplexityAnalysis = () => {
    toggleComplexityAnalysis(!isComplexityAnalysisEnabled());
  };

  return (
    <div>
      <button onClick={handleToggleErrorPrevention}>
        Error Prevention: {isErrorPreventionEnabled() ? 'ON' : 'OFF'}
      </button>
      <button onClick={handleToggleComplexityAnalysis}>
        Complexity Analysis: {isComplexityAnalysisEnabled() ? 'ON' : 'OFF'}
      </button>
    </div>
  );
}
```

### Direct Registry Access

```typescript
import { getPluginRegistry } from '@/services/pluginRegistry';

const registry = getPluginRegistry();

// Register a plugin
registry.register(myPlugin);

// Execute a hook
const result = await registry.executeHook('onMessageReceived', message);

// Get plugin stats
const stats = registry.getStats();
console.log(`Total plugins: ${stats.totalPlugins}, Enabled: ${stats.enabledPlugins}`);
```

## Creating Custom Plugins

### Example: Analytics Plugin

```typescript
import type { ChatPlugin, PluginExecutionContext } from '@/types/plugin';
import type { ToolCall, ToolResult } from '@/services/chatService';

export class AnalyticsPlugin implements ChatPlugin {
  name = 'analytics';
  version = '1.0.0';
  description = 'Tracks analytics for chat interactions';
  priority = 10; // Low priority - runs last

  private events: any[] = [];

  async onInit(): Promise<void> {
    console.log('[AnalyticsPlugin] Initialized');
    this.events = [];
  }

  async onToolCall(
    toolCall: ToolCall,
    context?: PluginExecutionContext
  ): Promise<ToolCall> {
    this.events.push({
      type: 'tool_call',
      tool: toolCall.tool,
      timestamp: new Date(),
      sessionId: context?.sessionId,
    });
    return toolCall;
  }

  async onToolResult(
    result: ToolResult,
    context?: PluginExecutionContext
  ): Promise<ToolResult> {
    this.events.push({
      type: 'tool_result',
      tool: result.tool,
      hasError: !!result.error,
      duration: result.durationMs,
      timestamp: new Date(),
      sessionId: context?.sessionId,
    });
    return result;
  }

  getEvents() {
    return this.events;
  }

  clearEvents() {
    this.events = [];
  }
}

// Register the plugin
import { getPluginRegistry } from '@/services/pluginRegistry';
const registry = getPluginRegistry();
registry.register(new AnalyticsPlugin());
```

## Plugin Execution Flow

### Hook Execution Pipeline

When a hook is executed, plugins run in priority order:

```
1. Registry.executeHook('onToolCall', toolCall)
   ↓
2. Sort plugins by priority (highest first)
   ↓
3. For each enabled plugin:
   a. Call plugin.onToolCall(toolCall)
   b. Pass result to next plugin
   c. Handle errors (log and continue)
   ↓
4. Return final result
```

### Error Handling

- Plugin errors are caught and logged
- Errors don't break the pipeline
- Other plugins continue executing
- `PluginError` class provides detailed error info

```typescript
try {
  const result = await registry.executeHook('onToolCall', toolCall);
} catch (error) {
  if (error instanceof PluginError) {
    console.error(`Plugin "${error.pluginName}" failed on hook "${error.hook}"`);
    console.error(`Original error: ${error.originalError?.message}`);
  }
}
```

## Migration from Hardcoded Toggles

### Before (Hardcoded)

```typescript
// State management
const [errorPreventionMode, setErrorPreventionMode] = useState(false);
const [complexityAnalysisMode, setComplexityAnalysisMode] = useState(false);

// Toggle handlers
const toggleErrorPrevention = async () => {
  const newMode = !errorPreventionMode;
  const result = await updateErrorPreventionMode(activeSessionId, newMode);
  setErrorPreventionMode(result.errorPreventionMode);
};

// UI rendering
<button onClick={toggleErrorPrevention}>
  {errorPreventionMode ? 'ON' : 'OFF'}
</button>
```

### After (Plugin-based)

```typescript
// Hook usage
const {
  toggleErrorPrevention,
  isErrorPreventionEnabled,
} = usePluginRegistry();

// Toggle handler
const handleToggle = () => {
  toggleErrorPrevention(!isErrorPreventionEnabled());
};

// UI rendering
<button onClick={handleToggle}>
  {isErrorPreventionEnabled() ? 'ON' : 'OFF'}
</button>
```

## Best Practices

### 1. Plugin Priority

- **80-100:** Core plugins (error prevention, complexity analysis)
- **50-79:** Processing plugins (caching, filtering)
- **20-49:** Enhancement plugins (analytics, logging)
- **0-19:** Utility plugins (cleanup, monitoring)

### 2. Error Handling

Always wrap plugin operations in try-catch:

```typescript
async onToolCall(toolCall: ToolCall): Promise<ToolCall> {
  try {
    // Your logic here
    return toolCall;
  } catch (error) {
    console.error(`[${this.name}] Error:`, error);
    // Return original data to prevent breaking the pipeline
    return toolCall;
  }
}
```

### 3. Resource Cleanup

Always implement `onDestroy` for cleanup:

```typescript
async onDestroy(): Promise<void> {
  // Clean up resources
  this.cache.clear();
  this.listeners.forEach(listener => listener.unsubscribe());
  console.log(`[${this.name}] Destroyed`);
}
```

### 4. Session Isolation

Use session IDs to track per-session state:

```typescript
private sessionData: Map<string, any> = new Map();

async onSessionCreated(sessionId: string): Promise<void> {
  this.sessionData.set(sessionId, { /* initial state */ });
}

async onSessionDeleted(sessionId: string): Promise<void> {
  this.sessionData.delete(sessionId);
}
```

## Testing Plugins

### Unit Test Example

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { ErrorPreventionPlugin } from '@/plugins/errorPreventionPlugin';

describe('ErrorPreventionPlugin', () => {
  let plugin: ErrorPreventionPlugin;

  beforeEach(() => {
    plugin = new ErrorPreventionPlugin();
  });

  it('should track errors per session', async () => {
    plugin.setEnabled(true);
    const sessionId = 'test-session';

    const result = {
      id: 'tool-1',
      tool: 'bash',
      result: null,
      error: 'Command failed',
      durationMs: 100,
    };

    await plugin.onToolResult(result, { sessionId });

    const stats = plugin.getSessionStats(sessionId);
    expect(stats.errorCount).toBe(1);
  });

  it('should provide system prompt enhancement', () => {
    plugin.setEnabled(true);
    const enhancement = plugin.getSystemPromptEnhancement();
    expect(enhancement).toContain('Error Prevention Mode');
  });
});
```

## Performance Considerations

- **Plugin Priority:** Higher priority plugins run first, so place expensive operations at lower priority
- **Hook Execution:** Hooks execute sequentially, so keep individual plugin operations fast
- **Memory:** Use session-based cleanup to prevent memory leaks
- **Error Handling:** Errors are caught per-plugin, so one failure doesn't affect others

## Future Enhancements

Potential Phase 2-4 improvements:

1. **Plugin Marketplace:** Discover and install community plugins
2. **Plugin Versioning:** Manage plugin versions and dependencies
3. **Plugin Configuration UI:** Configure plugins through UI
4. **Plugin Testing Framework:** Built-in testing utilities
5. **Plugin Middleware:** Intercept and transform hook data
6. **Plugin Persistence:** Save plugin state to database
7. **Plugin Metrics:** Track plugin performance and usage

## Troubleshooting

### Plugin not executing

Check if plugin is enabled:
```typescript
const registry = getPluginRegistry();
console.log(registry.isPluginEnabled('my-plugin'));
registry.setPluginEnabled('my-plugin', true);
```

### Plugin errors not showing

Enable debug logging:
```typescript
const registry = getPluginRegistry();
const stats = registry.getStats();
console.log('Plugin stats:', stats);
```

### Memory leaks

Ensure `onDestroy` is called:
```typescript
useEffect(() => {
  return () => {
    registry.destroyAll();
  };
}, []);
```

## Summary

The plugin system provides:

✅ **Extensibility** - Add new modes without modifying core code
✅ **Maintainability** - Each plugin is self-contained
✅ **Testability** - Plugins can be tested independently
✅ **Performance** - Priority-based execution and error isolation
✅ **Production-Ready** - Lifecycle management, error handling, cleanup

This architecture transforms the chat system from a monolithic application into a flexible, plugin-based platform.
