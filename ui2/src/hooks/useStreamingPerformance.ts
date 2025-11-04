/**
 * useStreamingPerformance Hook
 *
 * Monitors streaming performance with FPS tracking and network statistics.
 * Provides real-time metrics for debugging and optimization.
 */

import { useState, useEffect, useRef, useCallback } from 'react';

export interface StreamingStats {
  bytesReceived: number;
  chunksReceived: number;
  tokensReceived: number;
  duration: number;
  currentFps: number;
  averageFps: number;
}

export interface PerformanceMetrics {
  fps: number;
  timestamp: number;
}

/**
 * Hook for tracking streaming performance
 */
export function useStreamingPerformance() {
  const [stats, setStats] = useState<StreamingStats>({
    bytesReceived: 0,
    chunksReceived: 0,
    tokensReceived: 0,
    duration: 0,
    currentFps: 0,
    averageFps: 0,
  });

  const [fpsHistory, setFpsHistory] = useState<number[]>([]);
  const [isMonitoring, setIsMonitoring] = useState(false);

  // Track internal metrics
  const metricsRef = useRef({
    bytesReceived: 0,
    chunksReceived: 0,
    tokensReceived: 0,
    startTime: 0,
    lastUpdateTime: 0,
    lastChunkTime: 0,
    fpsReadings: [] as number[],
  });

  const fpsIntervalRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Start monitoring
   */
  const startMonitoring = useCallback(() => {
    metricsRef.current = {
      bytesReceived: 0,
      chunksReceived: 0,
      tokensReceived: 0,
      startTime: Date.now(),
      lastUpdateTime: Date.now(),
      lastChunkTime: Date.now(),
      fpsReadings: [],
    };

    setIsMonitoring(true);

    // Update FPS every 100ms
    fpsIntervalRef.current = setInterval(() => {
      const now = Date.now();
      const timeSinceLastChunk = now - metricsRef.current.lastChunkTime;

      // Calculate instantaneous FPS (only if recently received chunks)
      const instantFps = timeSinceLastChunk < 1000
        ? Math.round(1000 / Math.max(timeSinceLastChunk, 16))
        : 0;

      // Add to FPS history
      if (instantFps > 0) {
        metricsRef.current.fpsReadings.push(instantFps);
        // Keep only last 50 readings
        if (metricsRef.current.fpsReadings.length > 50) {
          metricsRef.current.fpsReadings.shift();
        }
      }

      // Calculate average FPS
      const avgFps = metricsRef.current.fpsReadings.length > 0
        ? Math.round(
            metricsRef.current.fpsReadings.reduce((a, b) => a + b, 0) /
            metricsRef.current.fpsReadings.length
          )
        : 0;

      const duration = now - metricsRef.current.startTime;

      setStats({
        bytesReceived: metricsRef.current.bytesReceived,
        chunksReceived: metricsRef.current.chunksReceived,
        tokensReceived: metricsRef.current.tokensReceived,
        duration,
        currentFps: instantFps,
        averageFps: avgFps,
      });

      setFpsHistory(prev => {
        const newHistory = [...prev, instantFps];
        return newHistory.slice(-100); // Keep last 100 readings for visualization
      });
    }, 100);
  }, []);

  /**
   * Stop monitoring
   */
  const stopMonitoring = useCallback(() => {
    if (fpsIntervalRef.current) {
      clearInterval(fpsIntervalRef.current);
      fpsIntervalRef.current = null;
    }
    setIsMonitoring(false);
  }, []);

  /**
   * Record a chunk received
   */
  const recordChunk = useCallback((content: string) => {
    if (!isMonitoring) return;

    const now = Date.now();
    const bytes = new Blob([content]).size;

    metricsRef.current.bytesReceived += bytes;
    metricsRef.current.chunksReceived += 1;
    metricsRef.current.tokensReceived += content.length;
    metricsRef.current.lastChunkTime = now;
    metricsRef.current.lastUpdateTime = now;
  }, [isMonitoring]);

  /**
   * Reset metrics
   */
  const resetMetrics = useCallback(() => {
    metricsRef.current = {
      bytesReceived: 0,
      chunksReceived: 0,
      tokensReceived: 0,
      startTime: Date.now(),
      lastUpdateTime: Date.now(),
      lastChunkTime: Date.now(),
      fpsReadings: [],
    };

    setStats({
      bytesReceived: 0,
      chunksReceived: 0,
      tokensReceived: 0,
      duration: 0,
      currentFps: 0,
      averageFps: 0,
    });

    setFpsHistory([]);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (fpsIntervalRef.current) {
        clearInterval(fpsIntervalRef.current);
      }
    };
  }, []);

  // Performance health indicator
  const isPerformanceGood = stats.currentFps >= 20 && stats.currentFps <= 60;

  return {
    stats,
    fpsHistory,
    isMonitoring,
    isPerformanceGood,
    startMonitoring,
    stopMonitoring,
    recordChunk,
    resetMetrics,
  };
}
