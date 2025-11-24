/**
 * Connection Health Monitor
 * 
 * Monitors WebSocket connection health and provides metrics.
 * Tracks latency, packet loss, uptime, and reconnection frequency.
 * 
 * Features:
 * - Real-time latency measurement (ping/pong)
 * - Packet loss detection
 * - Uptime tracking
 * - Reconnection frequency monitoring
 * - Health status reporting
 */

export interface ConnectionMetrics {
  latencyMs: number;
  packetLoss: number; // 0-1 (0% - 100%)
  uptimeMs: number;
  reconnectionCount: number;
  lastPingTime: number | null;
  lastPongTime: number | null;
  isHealthy: boolean;
}

export class ConnectionHealthMonitor {
  private metrics: ConnectionMetrics = {
    latencyMs: 0,
    packetLoss: 0,
    uptimeMs: 0,
    reconnectionCount: 0,
    lastPingTime: null,
    lastPongTime: null,
    isHealthy: true,
  };

  private connectionStartTime: number | null = null;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private pendingPings: Map<number, number> = new Map(); // pingId -> timestamp
  private nextPingId: number = 0;

  // Thresholds for health status
  private readonly LATENCY_THRESHOLD_MS = 5000; // 5 seconds
  private readonly PACKET_LOSS_THRESHOLD = 0.1; // 10%
  private readonly PING_INTERVAL_MS = 30000; // 30 seconds

  /**
   * Start monitoring connection health
   */
  start(): void {
    if (this.connectionStartTime === null) {
      this.connectionStartTime = Date.now();
    }

    // Start periodic ping
    this.pingInterval = setInterval(() => {
      this.sendPing();
    }, this.PING_INTERVAL_MS);
  }

  /**
   * Stop monitoring connection health
   */
  stop(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }

    this.connectionStartTime = null;
    this.pendingPings.clear();
  }

  /**
   * Record a ping sent
   */
  recordPing(): number {
    const pingId = this.nextPingId++;
    const timestamp = Date.now();
    this.pendingPings.set(pingId, timestamp);
    this.metrics.lastPingTime = timestamp;
    return pingId;
  }

  /**
   * Record a pong received
   */
  recordPong(pingId: number): void {
    const pingTime = this.pendingPings.get(pingId);
    if (pingTime !== undefined) {
      const latency = Date.now() - pingTime;
      this.metrics.latencyMs = latency;
      this.metrics.lastPongTime = Date.now();
      this.pendingPings.delete(pingId);

      // Update health status
      this.updateHealthStatus();
    }
  }

  /**
   * Record a reconnection
   */
  recordReconnection(): void {
    this.metrics.reconnectionCount++;
    this.connectionStartTime = Date.now();
    this.pendingPings.clear();
  }

  /**
   * Get current metrics
   */
  getMetrics(): Readonly<ConnectionMetrics> {
    // Update uptime
    if (this.connectionStartTime !== null) {
      this.metrics.uptimeMs = Date.now() - this.connectionStartTime;
    }

    // Calculate packet loss
    this.metrics.packetLoss = this.calculatePacketLoss();

    return Object.freeze({ ...this.metrics });
  }

  /**
   * Check if connection is healthy
   */
  isHealthy(): boolean {
    return this.metrics.isHealthy;
  }

  /**
   * Get health status message
   */
  getHealthStatus(): string {
    if (!this.metrics.isHealthy) {
      if (this.metrics.latencyMs > this.LATENCY_THRESHOLD_MS) {
        return `High latency: ${this.metrics.latencyMs}ms`;
      }
      if (this.metrics.packetLoss > this.PACKET_LOSS_THRESHOLD) {
        return `High packet loss: ${(this.metrics.packetLoss * 100).toFixed(1)}%`;
      }
    }
    return 'Connection healthy';
  }

  // Private helper methods

  private sendPing(): void {
    // This would be called by the WebSocket manager
    // to send a ping message
  }

  private calculatePacketLoss(): number {
    if (this.nextPingId === 0) {
      return 0;
    }

    const totalPings = this.nextPingId;

    // Consider pings older than 2x the ping interval as lost
    const now = Date.now();
    let lostPings = 0;

    for (const [, timestamp] of this.pendingPings) {
      if (now - timestamp > this.PING_INTERVAL_MS * 2) {
        lostPings++;
      }
    }

    return lostPings / totalPings;
  }

  private updateHealthStatus(): void {
    this.metrics.isHealthy =
      this.metrics.latencyMs <= this.LATENCY_THRESHOLD_MS &&
      this.metrics.packetLoss <= this.PACKET_LOSS_THRESHOLD;
  }
}
