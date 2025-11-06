/**
 * Prometheus Metrics Parser
 *
 * Parses Prometheus text format metrics into structured data.
 * Used by MetricsDashboard to display system metrics.
 */

export interface ParsedMetrics {
  // WebSocket metrics
  wsConnectionsActive: number;
  wsConnectionsTotal: number;

  // Message validation metrics
  messageValidationSuccess: number;
  messageValidationFailed: number;
  messageValidationSuccessRate: number;

  // Chat message metrics
  chatMessagesTotal: number;
  chatMessagesSuccess: number;
  chatMessagesFailed: number;
  chatMessagesSuccessRate: number;

  // AI streaming metrics
  aiStreamTokensTotal: number;
  aiStreamDurationMs: number;
  aiStreamChunksTotal: number;

  // HTTP request metrics
  httpRequestsTotal: number;
  httpRequestsSuccess: number;
  httpRequestsFailed: number;
  httpRequestsErrorRate: number;

  // MongoDB metrics
  mongoReadsTotal: number;
  mongoWritesTotal: number;
  mongoErrorsTotal: number;

  // Response time metrics (percentiles in ms)
  responseTimeP50: number;
  responseTimeP95: number;
  responseTimeP99: number;

  // System health
  systemHealth: 'healthy' | 'degraded' | 'unhealthy';
}

interface MetricLine {
  name: string;
  value: number;
  labels?: Record<string, string>;
}

/**
 * Parse Prometheus text format metrics
 * @param text Raw Prometheus metrics text
 * @returns Structured metrics object
 */
export function parsePrometheusMetrics(text: string): ParsedMetrics {
  const lines = text.split('\n');
  const metrics = new Map<string, MetricLine[]>();

  // Parse each line
  for (const line of lines) {
    // Skip comments and empty lines
    if (line.startsWith('#') || line.trim() === '') {
      continue;
    }

    // Parse metric line: metric_name{label="value"} value
    const parsed = parseMetricLine(line);
    if (parsed) {
      if (!metrics.has(parsed.name)) {
        metrics.set(parsed.name, []);
      }
      metrics.get(parsed.name)!.push(parsed);
    }
  }

  // Extract specific metrics
  const wsActive = getMetricValue(metrics, 'hyperion_ws_connections_active', {}) || 0;
  const wsTotal = getMetricValue(metrics, 'hyperion_ws_connections_total', {}) || 0;

  const validationSuccess = getMetricValue(metrics, 'hyperion_message_validation_total', { status: 'success' }) || 0;
  const validationFailed = getMetricValue(metrics, 'hyperion_message_validation_total', { status: 'failed' }) || 0;
  const validationTotal = validationSuccess + validationFailed;
  const validationSuccessRate = validationTotal > 0 ? validationSuccess / validationTotal : 1;

  const chatSuccess = getMetricValue(metrics, 'hyperion_chat_messages_total', { status: 'success' }) || 0;
  const chatFailed = getMetricValue(metrics, 'hyperion_chat_messages_total', { status: 'failed' }) || 0;
  const chatTotal = chatSuccess + chatFailed;
  const chatSuccessRate = chatTotal > 0 ? chatSuccess / chatTotal : 1;

  const aiTokens = getMetricValue(metrics, 'hyperion_ai_stream_tokens_total', {}) || 0;
  const aiDuration = getMetricValue(metrics, 'hyperion_ai_stream_duration_ms', {}) || 0;
  const aiChunks = getMetricValue(metrics, 'hyperion_ai_stream_chunks_total', {}) || 0;

  const httpSuccess = getMetricValue(metrics, 'hyperion_http_requests_total', { status: '2xx' }) || 0;
  const httpClientError = getMetricValue(metrics, 'hyperion_http_requests_total', { status: '4xx' }) || 0;
  const httpServerError = getMetricValue(metrics, 'hyperion_http_requests_total', { status: '5xx' }) || 0;
  const httpTotal = httpSuccess + httpClientError + httpServerError;
  const httpFailed = httpClientError + httpServerError;
  const httpErrorRate = httpTotal > 0 ? httpFailed / httpTotal : 0;

  const mongoReads = getMetricValue(metrics, 'hyperion_mongo_operations_total', { operation: 'read' }) || 0;
  const mongoWrites = getMetricValue(metrics, 'hyperion_mongo_operations_total', { operation: 'write' }) || 0;
  const mongoErrors = getMetricValue(metrics, 'hyperion_mongo_operations_total', { status: 'error' }) || 0;

  const p50 = getMetricValue(metrics, 'hyperion_response_time_ms', { quantile: '0.5' }) || 0;
  const p95 = getMetricValue(metrics, 'hyperion_response_time_ms', { quantile: '0.95' }) || 0;
  const p99 = getMetricValue(metrics, 'hyperion_response_time_ms', { quantile: '0.99' }) || 0;

  // Calculate system health
  let systemHealth: 'healthy' | 'degraded' | 'unhealthy' = 'healthy';
  if (chatSuccessRate < 0.95 || httpErrorRate > 0.05 || mongoErrors > 10) {
    systemHealth = 'degraded';
  }
  if (chatSuccessRate < 0.8 || httpErrorRate > 0.2 || mongoErrors > 50) {
    systemHealth = 'unhealthy';
  }

  return {
    wsConnectionsActive: wsActive,
    wsConnectionsTotal: wsTotal,
    messageValidationSuccess: validationSuccess,
    messageValidationFailed: validationFailed,
    messageValidationSuccessRate: validationSuccessRate,
    chatMessagesTotal: chatTotal,
    chatMessagesSuccess: chatSuccess,
    chatMessagesFailed: chatFailed,
    chatMessagesSuccessRate: chatSuccessRate,
    aiStreamTokensTotal: aiTokens,
    aiStreamDurationMs: aiDuration,
    aiStreamChunksTotal: aiChunks,
    httpRequestsTotal: httpTotal,
    httpRequestsSuccess: httpSuccess,
    httpRequestsFailed: httpFailed,
    httpRequestsErrorRate: httpErrorRate,
    mongoReadsTotal: mongoReads,
    mongoWritesTotal: mongoWrites,
    mongoErrorsTotal: mongoErrors,
    responseTimeP50: p50,
    responseTimeP95: p95,
    responseTimeP99: p99,
    systemHealth,
  };
}

/**
 * Parse a single metric line
 * @param line Metric line from Prometheus format
 * @returns Parsed metric or null if invalid
 */
function parseMetricLine(line: string): MetricLine | null {
  try {
    // Match: metric_name{label="value",label2="value2"} 123.45
    const match = line.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\{([^}]*)\}\s+([0-9.]+)$/);
    if (match) {
      const name = match[1];
      const labelsStr = match[2];
      const value = parseFloat(match[3]);
      if (isNaN(value)) return null;

      // Parse labels
      const labels: Record<string, string> = {};
      if (labelsStr) {
        const labelMatches = labelsStr.matchAll(/([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"/g);
        for (const labelMatch of labelMatches) {
          labels[labelMatch[1]] = labelMatch[2];
        }
      }

      return { name, value, labels };
    }

    // Match: metric_name 123.45 (no labels)
    const simpleMatch = line.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s+([0-9.]+)$/);
    if (simpleMatch) {
      const value = parseFloat(simpleMatch[2]);
      if (isNaN(value)) return null;
      return {
        name: simpleMatch[1],
        value: value,
        labels: {},
      };
    }

    return null;
  } catch (err) {
    console.error('Failed to parse metric line:', line, err);
    return null;
  }
}

/**
 * Get metric value by name and labels
 * @param metrics Parsed metrics map
 * @param name Metric name
 * @param matchLabels Labels to match
 * @returns Metric value or null
 */
function getMetricValue(
  metrics: Map<string, MetricLine[]>,
  name: string,
  matchLabels: Record<string, string>
): number | null {
  const metricLines = metrics.get(name);
  if (!metricLines) {
    return null;
  }

  // Find metric with matching labels
  for (const line of metricLines) {
    const labels = line.labels || {};
    const matches = Object.entries(matchLabels).every(
      ([key, value]) => labels[key] === value
    );
    if (matches) {
      return line.value;
    }
  }

  // If no specific match, return first metric (for metrics without labels)
  if (Object.keys(matchLabels).length === 0 && metricLines.length > 0) {
    return metricLines[0].value;
  }

  return null;
}

/**
 * Format metric value for display
 * @param value Raw numeric value
 * @param type Value type (number, duration, percentage, bytes)
 * @returns Formatted string
 */
export function formatMetricValue(
    value: number | undefined | null, 
    type: 'number' | 'duration' | 'percentage' | 'bytes'
): string {
  // ⭐️ FIX: This guard clause prevents crashes from null, undefined, or NaN values.
  if (value === null || typeof value === 'undefined' || isNaN(value)) {
    return 'N/A';
  }

  switch (type) {
    case 'number':
      if (value >= 1_000_000) {
        return `${(value / 1_000_000).toFixed(2)}M`;
      }
      if (value >= 1_000) {
        return `${(value / 1_000).toFixed(2)}k`;
      }
      return value.toFixed(0);

    case 'duration':
      if (value >= 1000) {
        return `${(value / 1000).toFixed(2)}s`;
      }
      return `${value.toFixed(0)}ms`;

    case 'percentage':
      return `${(value * 100).toFixed(1)}%`;

    case 'bytes':
      if (value >= 1_073_741_824) {
        return `${(value / 1_073_741_824).toFixed(2)}GB`;
      }
      if (value >= 1_048_576) {
        return `${(value / 1_048_576).toFixed(2)}MB`;
      }
      if (value >= 1_024) {
        return `${(value / 1_024).toFixed(2)}KB`;
      }
      return `${value}B`;

    default:
      return value.toString();
  }
}

/**
 * Calculate rate between two values over time
 * @param current Current value
 * @param previous Previous value
 * @param timeDeltaMs Time delta in milliseconds
 * @returns Rate per second
 */
export function calculateRate(
  current: number,
  previous: number,
  timeDeltaMs: number
): number {
  if (timeDeltaMs <= 0) {
    return 0;
  }
  const delta = current - previous;
  const timeSeconds = timeDeltaMs / 1000;
  return delta / timeSeconds;
}

/**
 * Get trend indicator for metric
 * @param current Current value
 * @param previous Previous value
 * @returns Trend direction
 */
export function getMetricTrend(
  current: number,
  previous: number
): 'up' | 'down' | 'neutral' {
  const threshold = 0.05; // 5% change threshold
  const percentChange = previous > 0 ? (current - previous) / previous : 0;

  if (Math.abs(percentChange) < threshold) {
    return 'neutral';
  }

  return percentChange > 0 ? 'up' : 'down';
}