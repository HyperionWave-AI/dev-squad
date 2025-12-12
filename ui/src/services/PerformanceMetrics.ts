/**
 * Performance Metrics Utility
 * 
 * Collects and analyzes performance metrics for the chat system.
 * Tracks message latency, streaming throughput, and resource usage.
 * 
 * Features:
 * - Message latency tracking
 * - Streaming throughput measurement
 * - Memory usage monitoring
 * - Performance statistics
 * - Performance alerts
 */

export interface PerformanceMetric {
  name: string;
  value: number;
  unit: string;
  timestamp: Date;
  context?: Record<string, any>;
}

export interface PerformanceStats {
  messageLatency: {
    min: number;
    max: number;
    average: number;
    p95: number;
    p99: number;
  };
  streamingThroughput: {
    bytesPerSecond: number;
    messagesPerSecond: number;
  };
  memoryUsage: {
    heapUsed: number;
    heapTotal: number;
    external: number;
  };
}

export class PerformanceMetrics {
  private metrics: PerformanceMetric[] = [];
  private readonly MAX_METRICS = 10000;
  private messageLatencies: number[] = [];
  private streamingStartTime: number | null = null;
  private streamingBytesReceived: number = 0;
  private streamingMessagesReceived: number = 0;

  /**
   * Record a message latency
   */
  recordMessageLatency(latencyMs: number, context?: Record<string, any>): void {
    this.messageLatencies.push(latencyMs);
    this.recordMetric('message_latency', latencyMs, 'ms', context);
  }

  /**
   * Record streaming data received
   */
  recordStreamingData(bytes: number, messageCount: number = 1): void {
    if (this.streamingStartTime === null) {
      this.streamingStartTime = Date.now();
    }

    this.streamingBytesReceived += bytes;
    this.streamingMessagesReceived += messageCount;
  }

  /**
   * End streaming session
   */
  endStreamingSession(): void {
    if (this.streamingStartTime !== null) {
      const durationMs = Date.now() - this.streamingStartTime;
      const throughputBytesPerSec = (this.streamingBytesReceived / durationMs) * 1000;
      const throughputMessagesPerSec = (this.streamingMessagesReceived / durationMs) * 1000;

      this.recordMetric('streaming_throughput_bytes_per_sec', throughputBytesPerSec, 'bytes/s');
      this.recordMetric('streaming_throughput_messages_per_sec', throughputMessagesPerSec, 'msg/s');

      // Reset
      this.streamingStartTime = null;
      this.streamingBytesReceived = 0;
      this.streamingMessagesReceived = 0;
    }
  }

  /**
   * Record a custom metric
   */
  recordMetric(
    name: string,
    value: number,
    unit: string = '',
    context?: Record<string, any>
  ): void {
    const metric: PerformanceMetric = {
      name,
      value,
      unit,
      timestamp: new Date(),
      context,
    };

    this.metrics.push(metric);

    // Keep metrics bounded
    if (this.metrics.length > this.MAX_METRICS) {
      this.metrics.shift();
    }
  }

  /**
   * Get performance statistics
   */
  getStats(): PerformanceStats {
    return {
      messageLatency: this.getMessageLatencyStats(),
      streamingThroughput: this.getStreamingThroughputStats(),
      memoryUsage: this.getMemoryUsage(),
    };
  }

  /**
   * Get all metrics
   */
  getMetrics(): PerformanceMetric[] {
    return [...this.metrics];
  }

  /**
   * Filter metrics by name
   */
  filterByName(name: string): PerformanceMetric[] {
    return this.metrics.filter(m => m.name === name);
  }

  /**
   * Filter metrics by time range
   */
  filterByTimeRange(startTime: Date, endTime: Date): PerformanceMetric[] {
    const startMs = startTime.getTime();
    const endMs = endTime.getTime();

    return this.metrics.filter(m => {
      const metricTime = m.timestamp.getTime();
      return metricTime >= startMs && metricTime <= endMs;
    });
  }

  /**
   * Get average metric value
   */
  getAverageMetric(name: string): number {
    const metrics = this.filterByName(name);
    if (metrics.length === 0) return 0;

    const sum = metrics.reduce((acc, m) => acc + m.value, 0);
    return sum / metrics.length;
  }

  /**
   * Get percentile metric value
   */
  getPercentileMetric(name: string, percentile: number): number {
    const metrics = this.filterByName(name);
    if (metrics.length === 0) return 0;

    const values = metrics.map(m => m.value).sort((a, b) => a - b);
    const index = Math.ceil((percentile / 100) * values.length) - 1;
    return values[Math.max(0, index)];
  }

  /**
   * Clear metrics
   */
  clear(): void {
    this.metrics = [];
    this.messageLatencies = [];
    this.streamingStartTime = null;
    this.streamingBytesReceived = 0;
    this.streamingMessagesReceived = 0;
  }

  /**
   * Export metrics as JSON
   */
  exportAsJSON(): string {
    return JSON.stringify({
      metrics: this.metrics,
      stats: this.getStats(),
    }, null, 2);
  }

  // Private helper methods

  private getMessageLatencyStats() {
    if (this.messageLatencies.length === 0) {
      return {
        min: 0,
        max: 0,
        average: 0,
        p95: 0,
        p99: 0,
      };
    }

    const sorted = [...this.messageLatencies].sort((a, b) => a - b);
    const sum = sorted.reduce((a, b) => a + b, 0);

    return {
      min: sorted[0],
      max: sorted[sorted.length - 1],
      average: sum / sorted.length,
      p95: sorted[Math.floor(sorted.length * 0.95)],
      p99: sorted[Math.floor(sorted.length * 0.99)],
    };
  }

  private getStreamingThroughputStats() {
    const throughputMetrics = this.filterByName('streaming_throughput_bytes_per_sec');
    const messageMetrics = this.filterByName('streaming_throughput_messages_per_sec');

    return {
      bytesPerSecond: throughputMetrics.length > 0
        ? throughputMetrics[throughputMetrics.length - 1].value
        : 0,
      messagesPerSecond: messageMetrics.length > 0
        ? messageMetrics[messageMetrics.length - 1].value
        : 0,
    };
  }

  private getMemoryUsage() {
    if (typeof performance !== 'undefined' && (performance as any).memory) {
      const memory = (performance as any).memory;
      return {
        heapUsed: memory.usedJSHeapSize,
        heapTotal: memory.totalJSHeapSize,
        external: memory.jsHeapSizeLimit,
      };
    }

    return {
      heapUsed: 0,
      heapTotal: 0,
      external: 0,
    };
  }
}

/**
 * Create a singleton instance
 */
let instance: PerformanceMetrics | null = null;

export function getPerformanceMetrics(): PerformanceMetrics {
  if (!instance) {
    instance = new PerformanceMetrics();
  }
  return instance;
}
