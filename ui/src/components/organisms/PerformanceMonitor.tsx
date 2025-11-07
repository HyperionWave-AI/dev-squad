/**
 * PerformanceMonitor Component
 *
 * Collapsible performance monitoring panel for streaming chat.
 * Shows FPS, network stats, and performance health indicator.
 */

import React, { useState } from 'react';
import { Activity, ChevronDown, ChevronUp, Zap } from 'lucide-react';
import { Badge } from '@/components/atoms/Badge';
import type { StreamingStats } from '@/hooks/useStreamingPerformance';

export interface PerformanceMonitorProps {
  stats: StreamingStats;
  fpsHistory: number[];
  isPerformanceGood: boolean;
  isStreaming: boolean;
}

export const PerformanceMonitor: React.FC<PerformanceMonitorProps> = ({
  stats,
  fpsHistory,
  isPerformanceGood,
  isStreaming,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Get FPS status for color coding
  const getFpsStatus = () => {
    if (stats.currentFps >= 40) return { variant: 'success' as const, text: 'Excellent' };
    if (stats.currentFps >= 20) return { variant: 'warning' as const, text: 'Good' };
    return { variant: 'destructive' as const, text: 'Poor' };
  };

  const fpsStatus = getFpsStatus();

  return (
    <div className="fixed bottom-4 right-4 z-50">
      {/* Compact Badge - Always Visible */}
      <div
        className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg cursor-pointer hover:shadow-xl transition-all"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {/* Collapsed View */}
        {!isExpanded && (
          <div className="px-4 py-2 flex items-center gap-3">
            <Activity className="h-4 w-4 text-primary-600 dark:text-primary-400" />
            {isStreaming && (
              <Badge variant={isPerformanceGood ? 'success' : 'destructive'} className="text-xs">
                {stats.currentFps} FPS
              </Badge>
            )}
            {!isStreaming && (
              <span className="text-xs text-gray-600 dark:text-gray-400">Idle</span>
            )}
            <ChevronUp className="h-4 w-4 text-gray-400" />
          </div>
        )}

        {/* Expanded View */}
        {isExpanded && (
          <div className="px-4 py-3 min-w-[280px]">
            {/* Header */}
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Activity className="h-4 w-4 text-primary-600 dark:text-primary-400" />
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Performance
                </span>
              </div>
              <ChevronDown className="h-4 w-4 text-gray-400" />
            </div>

            {/* Streaming Status */}
            <div className="mb-3">
              {isStreaming ? (
                <Badge variant="default" className="text-xs animate-pulse">
                  <Zap className="h-3 w-3 mr-1" />
                  Streaming
                </Badge>
              ) : (
                <Badge variant="outline" className="text-xs">
                  Idle
                </Badge>
              )}
            </div>

            {/* FPS Metrics */}
            <div className="grid grid-cols-2 gap-3 mb-3">
              <div className="space-y-1">
                <div className="text-xs text-gray-600 dark:text-gray-400">Current FPS</div>
                <div className="flex items-center gap-2">
                  <Badge variant={fpsStatus.variant} className="text-sm font-bold">
                    {stats.currentFps}
                  </Badge>
                  <span className="text-xs text-gray-500">{fpsStatus.text}</span>
                </div>
              </div>
              <div className="space-y-1">
                <div className="text-xs text-gray-600 dark:text-gray-400">Average FPS</div>
                <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {stats.averageFps}
                </div>
              </div>
            </div>

            {/* Network Stats */}
            <div className="grid grid-cols-3 gap-2 mb-3 text-xs">
              <div>
                <div className="text-gray-600 dark:text-gray-400">Bytes</div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {Math.round(stats.bytesReceived / 1024)}KB
                </div>
              </div>
              <div>
                <div className="text-gray-600 dark:text-gray-400">Chunks</div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {stats.chunksReceived}
                </div>
              </div>
              <div>
                <div className="text-gray-600 dark:text-gray-400">Duration</div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {Math.round(stats.duration / 1000)}s
                </div>
              </div>
            </div>

            {/* FPS History Visualization */}
            {fpsHistory.length > 0 && (
              <div className="space-y-1">
                <div className="text-xs text-gray-600 dark:text-gray-400">FPS History</div>
                <div className="flex items-end space-x-0.5 h-12 bg-gray-100 dark:bg-gray-700 rounded px-1 py-1">
                  {fpsHistory.slice(-40).map((fps, index) => (
                    <div
                      key={index}
                      className={`w-1 rounded-t transition-all ${
                        fps >= 40
                          ? 'bg-green-500'
                          : fps >= 20
                          ? 'bg-yellow-500'
                          : 'bg-red-500'
                      }`}
                      style={{ height: `${Math.min((fps / 60) * 100, 100)}%` }}
                      title={`${fps} fps`}
                    />
                  ))}
                </div>
                <div className="flex justify-between text-xs text-gray-500">
                  <span>0 fps</span>
                  <span>Target: 20-60 fps</span>
                  <span>60 fps</span>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
