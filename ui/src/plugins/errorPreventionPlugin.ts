/**
 * Error Prevention Mode Plugin
 *
 * Enables AI to validate code and fix errors automatically.
 * When enabled, the AI system prompt is enhanced to include error prevention guidance.
 */

import type { ChatPlugin, PluginExecutionContext } from '@/types/plugin';
import type { ChatMessage, ToolResult } from '@/services/chatService';

export class ErrorPreventionPlugin implements ChatPlugin {
  name = 'error-prevention';
  version = '1.0.0';
  description = 'Enables AI to validate code and fix errors automatically';
  priority = 80; // High priority - runs early in the pipeline

  private enabled = false;
  private sessionErrorCounts: Map<string, number> = new Map();
  private readonly ERROR_THRESHOLD = 3; // Track errors per session

  async onInit(): Promise<void> {
    console.log('[ErrorPreventionPlugin] Initialized');
    this.sessionErrorCounts.clear();
  }

  async onDestroy(): Promise<void> {
    console.log('[ErrorPreventionPlugin] Destroyed');
    this.sessionErrorCounts.clear();
  }

  /**
   * Enable/disable error prevention for a session
   */
  setEnabled(enabled: boolean): void {
    this.enabled = enabled;
    console.log(
      `[ErrorPreventionPlugin] ${enabled ? '🛡️  Enabled' : '⏸️  Disabled'}`
    );
  }

  /**
   * Check if error prevention is enabled
   */
  isEnabled(): boolean {
    return this.enabled;
  }

  /**
   * Process incoming messages to add error prevention context
   */
  async onMessageReceived(
    message: ChatMessage,
    _context?: PluginExecutionContext
  ): Promise<ChatMessage> {
    if (!this.enabled) {
      return message;
    }

    // Add error prevention metadata to assistant messages
    if (message.role === 'assistant') {
      return {
        ...message,
        metadata: {
          ...(message as any).metadata,
          errorPreventionEnabled: true,
          errorPreventionTimestamp: new Date().toISOString(),
        },
      };
    }

    return message;
  }

  /**
   * Monitor tool results for errors
   */
  async onToolResult(
    result: ToolResult,
    context?: PluginExecutionContext
  ): Promise<ToolResult> {
    if (!this.enabled) {
      return result;
    }

    const sessionId = context?.sessionId;
    if (!sessionId) {
      return result;
    }

    // Track errors per session
    if (result.error) {
      const errorCount = (this.sessionErrorCounts.get(sessionId) || 0) + 1;
      this.sessionErrorCounts.set(sessionId, errorCount);

      console.warn(
        `[ErrorPreventionPlugin] ⚠️  Error detected in ${result.tool}: ${result.error} (Session errors: ${errorCount})`
      );

      // Add error prevention metadata
      return {
        ...result,
        metadata: {
          ...(result as any).metadata,
          errorPreventionDetected: true,
          sessionErrorCount: errorCount,
          shouldRetry: errorCount < this.ERROR_THRESHOLD,
        },
      };
    }

    return result;
  }

  /**
   * Get system prompt enhancement for error prevention
   */
  getSystemPromptEnhancement(): string {
    if (!this.enabled) {
      return '';
    }

    return `
## Error Prevention Mode: ENABLED

You are operating in Error Prevention Mode. Your responsibilities:

1. **Code Validation**: Before executing any code or tool calls, validate:
   - Syntax correctness
   - Type safety
   - Resource availability
   - Permission requirements

2. **Error Detection**: Monitor all tool results for:
   - Compilation errors
   - Runtime errors
   - Type mismatches
   - Missing dependencies

3. **Automatic Recovery**: When errors occur:
   - Analyze the error message
   - Identify the root cause
   - Suggest and implement fixes
   - Verify the fix works

4. **Proactive Prevention**: 
   - Ask clarifying questions if requirements are ambiguous
   - Suggest best practices
   - Warn about potential issues
   - Provide defensive code patterns

5. **Error Reporting**: When you detect an error:
   - Explain what went wrong
   - Show the error message
   - Describe your fix
   - Verify the fix resolves the issue

Remember: Your goal is to deliver working, error-free solutions.
`;
  }

  /**
   * Get error statistics for a session
   */
  getSessionStats(sessionId: string): {
    errorCount: number;
    isAboveThreshold: boolean;
  } {
    const errorCount = this.sessionErrorCounts.get(sessionId) || 0;
    return {
      errorCount,
      isAboveThreshold: errorCount >= this.ERROR_THRESHOLD,
    };
  }

  /**
   * Reset error count for a session
   */
  resetSessionStats(sessionId: string): void {
    this.sessionErrorCounts.delete(sessionId);
    console.log(`[ErrorPreventionPlugin] Reset error stats for session: ${sessionId}`);
  }

  /**
   * Handle session deletion to clean up tracking
   */
  async onSessionDeleted(
    sessionId: string,
    _context?: PluginExecutionContext
  ): Promise<void> {
    this.sessionErrorCounts.delete(sessionId);
    console.log(`[ErrorPreventionPlugin] Cleaned up stats for deleted session: ${sessionId}`);
  }
}

// Export singleton instance
export const errorPreventionPlugin = new ErrorPreventionPlugin();
