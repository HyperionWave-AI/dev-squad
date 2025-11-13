/**
 * Tool Result Size Helpers
 *
 * Utilities for analyzing and handling large tool results with progressive disclosure.
 * Implements Claude-style size limiting for better UX with large outputs.
 */

export interface ToolResultSizeInfo {
  size: number;
  tier: 'normal' | 'large' | 'truncated';
  displaySize: string;
}

// Size thresholds (must match backend config)
const MAX_NORMAL_SIZE = 50 * 1024;      // 50KB - display fully
const MAX_TRUNCATED_SIZE = 500 * 1024;  // 500KB - show preview with expand
const PREVIEW_SIZE = 10 * 1024;         // 10KB - preview size for truncation

/**
 * Analyze tool result size and determine display tier
 */
export function analyzeToolResultSize(result: any): ToolResultSizeInfo {
  const jsonStr = typeof result === 'string'
    ? result
    : JSON.stringify(result, null, 2);

  // Use Blob for accurate byte count (handles UTF-8 properly)
  const size = new Blob([jsonStr]).size;

  let tier: 'normal' | 'large' | 'truncated';
  if (size <= MAX_NORMAL_SIZE) {
    tier = 'normal';
  } else if (size <= MAX_TRUNCATED_SIZE) {
    tier = 'large';
  } else {
    tier = 'truncated';
  }

  return {
    size,
    tier,
    displaySize: formatSize(size),
  };
}

/**
 * Truncate tool result to specified byte limit
 */
export function truncateToolResult(result: any, maxBytes: number = PREVIEW_SIZE): string {
  const jsonStr = typeof result === 'string'
    ? result
    : JSON.stringify(result, null, 2);

  if (jsonStr.length <= maxBytes) {
    return jsonStr;
  }

  return jsonStr.substring(0, maxBytes) + '\n\n... [truncated]';
}

/**
 * Format byte size into human-readable string
 */
export function formatSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes}B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)}KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

/**
 * Check if result is likely from backend suppression
 * (Backend sends helpful message instead of actual result)
 */
export function isBackendSuppressedResult(result: any): boolean {
  if (typeof result !== 'string') {
    return false;
  }

  // Check for suppression message markers
  return result.includes('⚠️ Tool Result Too Large') ||
         result.includes('too large to display') ||
         result.includes('[Output truncated:');
}

/**
 * Get display content based on size and user preference
 */
export interface DisplayOptions {
  showFull: boolean; // User wants to see full result
}

export interface DisplayResult {
  content: string;
  showExpandButton: boolean;
  showWarning: boolean;
  warningMessage?: string;
}

export function getDisplayContent(
  result: any,
  options: DisplayOptions
): DisplayResult {
  // Check if backend already suppressed/truncated
  if (isBackendSuppressedResult(result)) {
    // Backend handled it, show as-is
    return {
      content: result as string,
      showExpandButton: false,
      showWarning: false,
    };
  }

  // Analyze size
  const sizeInfo = analyzeToolResultSize(result);

  if (sizeInfo.tier === 'normal') {
    // Small enough, display fully
    return {
      content: typeof result === 'string'
        ? result
        : JSON.stringify(result, null, 2),
      showExpandButton: false,
      showWarning: false,
    };
  }

  if (sizeInfo.tier === 'large') {
    // Show preview with expand option
    const content = options.showFull
      ? (typeof result === 'string'
          ? result
          : JSON.stringify(result, null, 2))
      : truncateToolResult(result, PREVIEW_SIZE);

    return {
      content,
      showExpandButton: true,
      showWarning: true,
      warningMessage: `Large result (${sizeInfo.displaySize})`,
    };
  }

  // Tier: truncated (very large)
  // This shouldn't normally happen if backend is working,
  // but provide fallback
  return {
    content: truncateToolResult(result, PREVIEW_SIZE),
    showExpandButton: false,
    showWarning: true,
    warningMessage: `Very large result (${sizeInfo.displaySize}) - showing preview only`,
  };
}
