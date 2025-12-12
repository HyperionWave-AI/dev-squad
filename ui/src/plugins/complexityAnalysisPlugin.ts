/**
 * Complexity Analysis Mode Plugin
 *
 * Enables AI to analyze task complexity and suggest splitting large tasks.
 * When enabled, the AI system prompt is enhanced to include complexity analysis guidance.
 */

import type { ChatPlugin, PluginExecutionContext } from '@/types/plugin';
import type { ChatMessage, ToolCall } from '@/services/chatService';

export interface ComplexityMetrics {
  estimatedComplexity: 'low' | 'medium' | 'high' | 'extreme';
  estimatedTimeMinutes: number;
  suggestedSplits: string[];
  riskFactors: string[];
}

export class ComplexityAnalysisPlugin implements ChatPlugin {
  name = 'complexity-analysis';
  version = '1.0.0';
  description = 'Enables AI to analyze task complexity and suggest splitting large tasks';
  priority = 75; // High priority - runs early in the pipeline

  private enabled = false;
  private sessionMetrics: Map<string, ComplexityMetrics> = new Map();

  async onInit(): Promise<void> {
    console.log('[ComplexityAnalysisPlugin] Initialized');
    this.sessionMetrics.clear();
  }

  async onDestroy(): Promise<void> {
    console.log('[ComplexityAnalysisPlugin] Destroyed');
    this.sessionMetrics.clear();
  }

  /**
   * Enable/disable complexity analysis for a session
   */
  setEnabled(enabled: boolean): void {
    this.enabled = enabled;
    console.log(
      `[ComplexityAnalysisPlugin] ${enabled ? '📊 Enabled' : '⏸️  Disabled'}`
    );
  }

  /**
   * Check if complexity analysis is enabled
   */
  isEnabled(): boolean {
    return this.enabled;
  }

  /**
   * Process incoming messages to add complexity analysis context
   */
  async onMessageReceived(
    message: ChatMessage,
    _context?: PluginExecutionContext
  ): Promise<ChatMessage> {
    if (!this.enabled) {
      return message;
    }

    // Add complexity analysis metadata to user messages
    if (message.role === 'user') {
      const complexity = this.analyzeMessageComplexity(message.content);
      return {
        ...message,
        metadata: {
          ...(message as any).metadata,
          complexityAnalysisEnabled: true,
          estimatedComplexity: complexity.estimatedComplexity,
          estimatedTimeMinutes: complexity.estimatedTimeMinutes,
        },
      };
    }

    return message;
  }

  /**
   * Monitor tool calls for complexity indicators
   */
  async onToolCall(
    toolCall: ToolCall,
    context?: PluginExecutionContext
  ): Promise<ToolCall> {
    if (!this.enabled) {
      return toolCall;
    }

    const sessionId = context?.sessionId;
    if (!sessionId) {
      return toolCall;
    }

    // Track tool calls as complexity indicators
    const metrics = this.sessionMetrics.get(sessionId) || {
      estimatedComplexity: 'low',
      estimatedTimeMinutes: 5,
      suggestedSplits: [],
      riskFactors: [],
    };

    // Increment complexity based on tool type
    const toolComplexityMap: Record<string, number> = {
      'create_agent_task': 15,
      'execute_subagent': 20,
      'code_index_search': 5,
      'read_file': 2,
      'write_file': 3,
      'bash': 10,
      'apply_patch': 8,
    };

    const toolComplexity = toolComplexityMap[toolCall.tool] || 5;
    metrics.estimatedTimeMinutes += toolComplexity;

    // Update complexity level based on time estimate
    if (metrics.estimatedTimeMinutes > 60) {
      metrics.estimatedComplexity = 'extreme';
    } else if (metrics.estimatedTimeMinutes > 30) {
      metrics.estimatedComplexity = 'high';
    } else if (metrics.estimatedTimeMinutes > 15) {
      metrics.estimatedComplexity = 'medium';
    }

    this.sessionMetrics.set(sessionId, metrics);

    console.log(
      `[ComplexityAnalysisPlugin] 📊 Tool: ${toolCall.tool} (Estimated time: ${metrics.estimatedTimeMinutes}m, Complexity: ${metrics.estimatedComplexity})`
    );

    return toolCall;
  }

  /**
   * Get system prompt enhancement for complexity analysis
   */
  getSystemPromptEnhancement(): string {
    if (!this.enabled) {
      return '';
    }

    return `
## Complexity Analysis Mode: ENABLED

You are operating in Complexity Analysis Mode. Your responsibilities:

1. **Task Assessment**: For each task, analyze:
   - Number of files to modify
   - Scope of changes (localized vs. system-wide)
   - Dependencies and cross-system impacts
   - Estimated time to complete
   - Risk factors and potential issues

2. **Complexity Scoring**: Rate tasks as:
   - **LOW** (< 15 min): Simple, isolated changes
   - **MEDIUM** (15-30 min): Moderate scope, some dependencies
   - **HIGH** (30-60 min): Large scope, multiple systems affected
   - **EXTREME** (> 60 min): Major refactoring, high risk

3. **Task Splitting**: When complexity is HIGH or EXTREME:
   - Identify logical breakpoints
   - Suggest splitting into 2-3 smaller tasks
   - Explain dependencies between subtasks
   - Recommend execution order
   - Provide clear handoff points

4. **Risk Analysis**: Identify:
   - Breaking changes
   - Performance implications
   - Security concerns
   - Testing requirements
   - Rollback procedures

5. **Recommendations**: Provide:
   - Estimated time for each subtask
   - Required expertise
   - Testing strategy
   - Deployment considerations

Format your analysis clearly so users can make informed decisions about task splitting.
`;
  }

  /**
   * Get metrics for a session
   */
  getSessionMetrics(sessionId: string): ComplexityMetrics | undefined {
    return this.sessionMetrics.get(sessionId);
  }

  /**
   * Set metrics for a session
   */
  setSessionMetrics(sessionId: string, metrics: ComplexityMetrics): void {
    this.sessionMetrics.set(sessionId, metrics);
    console.log(
      `[ComplexityAnalysisPlugin] Updated metrics for session ${sessionId}:`,
      metrics
    );
  }

  /**
   * Reset metrics for a session
   */
  resetSessionMetrics(sessionId: string): void {
    this.sessionMetrics.delete(sessionId);
    console.log(`[ComplexityAnalysisPlugin] Reset metrics for session: ${sessionId}`);
  }

  /**
   * Handle session deletion to clean up tracking
   */
  async onSessionDeleted(
    sessionId: string,
    _context?: PluginExecutionContext
  ): Promise<void> {
    this.sessionMetrics.delete(sessionId);
    console.log(
      `[ComplexityAnalysisPlugin] Cleaned up metrics for deleted session: ${sessionId}`
    );
  }

  /**
   * Analyze message complexity based on content
   */
  private analyzeMessageComplexity(content: string): ComplexityMetrics {
    const metrics: ComplexityMetrics = {
      estimatedComplexity: 'low',
      estimatedTimeMinutes: 5,
      suggestedSplits: [],
      riskFactors: [],
    };

    // Analyze content for complexity indicators
    const contentLower = content.toLowerCase();

    // Check for complexity keywords
    const complexityKeywords = {
      extreme: [
        'refactor',
        'redesign',
        'rewrite',
        'migrate',
        'restructure',
        'overhaul',
      ],
      high: [
        'multiple files',
        'cross-system',
        'integration',
        'architecture',
        'performance',
      ],
      medium: ['update', 'modify', 'enhance', 'improve', 'add feature'],
    };

    // Score based on keywords
    let score = 0;
    for (const keyword of complexityKeywords.extreme) {
      if (contentLower.includes(keyword)) {
        score += 3;
      }
    }
    for (const keyword of complexityKeywords.high) {
      if (contentLower.includes(keyword)) {
        score += 2;
      }
    }
    for (const keyword of complexityKeywords.medium) {
      if (contentLower.includes(keyword)) {
        score += 1;
      }
    }

    // Check for file count indicators
    if (contentLower.includes('files') || contentLower.includes('file')) {
      score += 1;
    }

    // Check for testing requirements
    if (
      contentLower.includes('test') ||
      contentLower.includes('verify') ||
      contentLower.includes('validate')
    ) {
      metrics.riskFactors.push('Testing required');
      score += 1;
    }

    // Check for breaking changes
    if (
      contentLower.includes('breaking') ||
      contentLower.includes('breaking change')
    ) {
      metrics.riskFactors.push('Breaking changes');
      score += 2;
    }

    // Determine complexity level
    if (score >= 6) {
      metrics.estimatedComplexity = 'extreme';
      metrics.estimatedTimeMinutes = 90;
    } else if (score >= 4) {
      metrics.estimatedComplexity = 'high';
      metrics.estimatedTimeMinutes = 45;
    } else if (score >= 2) {
      metrics.estimatedComplexity = 'medium';
      metrics.estimatedTimeMinutes = 20;
    } else {
      metrics.estimatedComplexity = 'low';
      metrics.estimatedTimeMinutes = 5;
    }

    return metrics;
  }
}

// Export singleton instance
export const complexityAnalysisPlugin = new ComplexityAnalysisPlugin();
