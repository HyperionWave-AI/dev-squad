/**
 * Prometheus Metrics Parser
 *
 * Parses Prometheus text format metrics into structured data
 */

export interface MetricValue {
  name: string;
  value: number;
  labels?: Record<string, string>;
  timestamp?: number;
}

export interface HistogramData {
  count: number;
  sum: number;
  buckets: Array<{ le: string; count: number }>;
  p95?: number;
  p99?: number;
  avg?: number;
}

export interface ParsedMetrics {
  gauges: Map<string, MetricValue>;
  counters: Map<string, MetricValue>;
  histograms: Map<string, HistogramData>;
}

/**
 * Parse a single Prometheus metric line
 */
function parseMetricLine(line: string): MetricValue | null {
  // Skip comments and empty lines
  if (line.startsWith('#') || !line.trim()) {
    return null;
  }

  // Parse metric line: metric_name{label1="value1",label2="value2"} value timestamp
  // Or simple: metric_name value timestamp

  const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*?)(?:\{([^}]+)\})?\s+([\d.eE+-]+)(?:\s+(\d+))?$/);

  if (!match) {
    return null;
  }

  const [, name, labelsStr, valueStr, timestampStr] = match;

  const labels: Record<string, string> = {};
  if (labelsStr) {
    // Parse labels: label1="value1",label2="value2"
    const labelPairs = labelsStr.match(/([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"/g);
    if (labelPairs) {
      labelPairs.forEach(pair => {
        const [key, value] = pair.split('=');
        labels[key] = value.replace(/"/g, '');
      });
    }
  }

  return {
    name,
    value: parseFloat(valueStr),
    labels: Object.keys(labels).length > 0 ? labels : undefined,
    timestamp: timestampStr ? parseInt(timestampStr, 10) : undefined,
  };
}

/**
 * Determine metric type from Prometheus TYPE comment
 */
function getMetricType(lines: string[], currentIndex: number): 'counter' | 'gauge' | 'histogram' | 'summary' | 'unknown' {
  // Look backwards for TYPE comment
  for (let i = currentIndex - 1; i >= 0; i--) {
    const line = lines[i].trim();

    if (!line.startsWith('#')) {
      break; // Stop at first non-comment line
    }

    if (line.startsWith('# TYPE')) {
      const typeMatch = line.match(/# TYPE\s+([a-zA-Z_:][a-zA-Z0-9_:]*)\s+(counter|gauge|histogram|summary)/);
      if (typeMatch) {
        return typeMatch[2] as 'counter' | 'gauge' | 'histogram' | 'summary';
      }
    }
  }

  return 'unknown';
}

/**
 * Calculate percentiles from histogram buckets
 */
function calculatePercentile(buckets: Array<{ le: string; count: number }>, percentile: number, totalCount: number): number {
  const targetCount = (percentile / 100) * totalCount;

  for (let i = 0; i < buckets.length; i++) {
    if (buckets[i].count >= targetCount) {
      const le = buckets[i].le;
      if (le === '+Inf') {
        // Use previous bucket's upper bound
        return i > 0 ? parseFloat(buckets[i - 1].le) : 0;
      }
      return parseFloat(le);
    }
  }

  return 0;
}

/**
 * Parse Prometheus text format metrics
 */
export function parsePrometheusMetrics(text: string): ParsedMetrics {
  const lines = text.split('\n');
  const gauges = new Map<string, MetricValue>();
  const counters = new Map<string, MetricValue>();
  const histograms = new Map<string, HistogramData>();

  const histogramBuilders = new Map<string, {
    buckets: Array<{ le: string; count: number }>;
    sum?: number;
    count?: number;
  }>();

  lines.forEach((line, index) => {
    const metric = parseMetricLine(line);
    if (!metric) return;

    const type = getMetricType(lines, index);

    // Handle histogram buckets
    if (metric.name.endsWith('_bucket') && type === 'histogram') {
      const baseName = metric.name.replace(/_bucket$/, '');
      const le = metric.labels?.le || '';

      if (!histogramBuilders.has(baseName)) {
        histogramBuilders.set(baseName, { buckets: [] });
      }

      histogramBuilders.get(baseName)!.buckets.push({ le, count: metric.value });
      return;
    }

    // Handle histogram sum
    if (metric.name.endsWith('_sum') && type === 'histogram') {
      const baseName = metric.name.replace(/_sum$/, '');

      if (!histogramBuilders.has(baseName)) {
        histogramBuilders.set(baseName, { buckets: [] });
      }

      histogramBuilders.get(baseName)!.sum = metric.value;
      return;
    }

    // Handle histogram count
    if (metric.name.endsWith('_count') && type === 'histogram') {
      const baseName = metric.name.replace(/_count$/, '');

      if (!histogramBuilders.has(baseName)) {
        histogramBuilders.set(baseName, { buckets: [] });
      }

      histogramBuilders.get(baseName)!.count = metric.value;
      return;
    }

    // Handle regular metrics
    if (type === 'gauge') {
      gauges.set(metric.name, metric);
    } else if (type === 'counter') {
      counters.set(metric.name, metric);
    }
  });

  // Build complete histogram data
  histogramBuilders.forEach((builder, name) => {
    const count = builder.count || 0;
    const sum = builder.sum || 0;
    const buckets = builder.buckets.sort((a, b) => {
      const aVal = a.le === '+Inf' ? Infinity : parseFloat(a.le);
      const bVal = b.le === '+Inf' ? Infinity : parseFloat(b.le);
      return aVal - bVal;
    });

    const histogramData: HistogramData = {
      count,
      sum,
      buckets,
      avg: count > 0 ? sum / count : 0,
      p95: count > 0 ? calculatePercentile(buckets, 95, count) : 0,
      p99: count > 0 ? calculatePercentile(buckets, 99, count) : 0,
    };

    histograms.set(name, histogramData);
  });

  return { gauges, counters, histograms };
}

/**
 * Format metric value with appropriate units
 */
export function formatMetricValue(value: number, unit?: 'bytes' | 'seconds' | 'ms' | 'rate'): string {
  if (unit === 'bytes') {
    if (value >= 1024 * 1024 * 1024) {
      return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`;
    }
    if (value >= 1024 * 1024) {
      return `${(value / (1024 * 1024)).toFixed(2)} MB`;
    }
    if (value >= 1024) {
      return `${(value / 1024).toFixed(2)} KB`;
    }
    return `${value.toFixed(0)} B`;
  }

  if (unit === 'seconds') {
    if (value >= 60) {
      return `${(value / 60).toFixed(1)}m`;
    }
    return `${value.toFixed(2)}s`;
  }

  if (unit === 'ms') {
    return `${value.toFixed(2)}ms`;
  }

  if (unit === 'rate') {
    return `${value.toFixed(1)}/min`;
  }

  // Default: format with appropriate decimal places
  if (value >= 1000) {
    return value.toLocaleString('en-US', { maximumFractionDigits: 0 });
  }

  return value.toLocaleString('en-US', { maximumFractionDigits: 2 });
}

/**
 * Calculate rate from counter (per minute)
 */
export function calculateRate(currentValue: number, previousValue: number, intervalSeconds: number): number {
  const delta = currentValue - previousValue;
  const ratePerSecond = delta / intervalSeconds;
  return ratePerSecond * 60; // Convert to per-minute rate
}
