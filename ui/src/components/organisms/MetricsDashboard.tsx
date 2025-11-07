/**
 * Prometheus MetricsDashboard Component
 *
 * Displays system-wide Prometheus metrics in a responsive grid layout.
 * Features:
 * - Auto-refresh every 5 seconds
 * - 12+ metric cards with color-coded status
 * - Trend indicators (up/down/neutral)
 * - WebSocket, HTTP, MongoDB, AI streaming metrics
 * - Response time percentiles (P50, P95, P99)
 * - System health indicator
 */

import React, { useEffect, useState, useCallback } from 'react';
import { Card, CardContent } from '@/components/molecules/Card';
import {
  Activity,
  CheckCircle,
  XCircle,
  MessageSquare,
  Zap,
  Globe,
  Database,
  Clock,
  TrendingUp,
  TrendingDown,
  Minus,
  AlertCircle,
} from 'lucide-react';
import { cn } from '@/utils';
import {
  parsePrometheusMetrics,
  formatMetricValue,
  type ParsedMetrics,
} from '@/utils/metricsParser';

interface MetricsDashboardProps {
  className?: string;
}

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  iconColor: string;
  iconBgColor: string;
  subtitle?: string;
  trend?: 'up' | 'down' | 'neutral';
  status?: 'success' | 'warning' | 'error';
}

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  icon,
  iconColor,
  iconBgColor,
  subtitle,
  trend,
  status,
}) => (
  <Card
    className={cn(
      'backdrop-blur-md border-white/30 dark:border-gray-700/30',
      'hover:shadow-xl transition-all duration-300 hover:-translate-y-1',
      status === 'success'
        ? 'bg-green-50/70 dark:bg-green-900/10'
        : status === 'warning'
        ? 'bg-yellow-50/70 dark:bg-yellow-900/10'
        : status === 'error'
        ? 'bg-red-50/70 dark:bg-red-900/10'
        : 'bg-white/70 dark:bg-gray-800/70'
    )}
  >
    <CardContent className="p-6">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
              {title}
            </p>
            {trend && (
              <div className="text-gray-500 dark:text-gray-400">
                {trend === 'up' && <TrendingUp className="w-3 h-3" />}
                {trend === 'down' && <TrendingDown className="w-3 h-3" />}
                {trend === 'neutral' && <Minus className="w-3 h-3" />}
              </div>
            )}
          </div>
          <p className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
            {value}
          </p>
          {subtitle && (
            <p className="text-xs text-gray-500 dark:text-gray-500">
              {subtitle}
            </p>
          )}
        </div>
        <div className={cn('p-3 rounded-lg', iconBgColor)}>
          <div className={iconColor}>{icon}</div>
        </div>
      </div>
    </CardContent>
  </Card>
);

export const MetricsDashboard: React.FC<MetricsDashboardProps> = ({
  className,
}) => {
  const [metrics, setMetrics] = useState<ParsedMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  // Fetch metrics from /metrics endpoint
  const fetchMetrics = useCallback(async () => {
    try {
      const response = await fetch('/metrics');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const text = await response.text();
      const parsed = parsePrometheusMetrics(text);
      setMetrics(parsed);
      setError(null);
      setLastUpdate(new Date());
    } catch (err) {
      console.error('Failed to fetch metrics:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch and auto-refresh every 5 seconds
  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000);
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  if (loading) {
    return (
      <div className={cn('w-full', className)}>
        <div className="flex items-center justify-center p-12">
          <Activity className="w-8 h-8 text-blue-500 animate-spin" />
          <span className="ml-3 text-gray-600 dark:text-gray-400">
            Loading metrics...
          </span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={cn('w-full', className)}>
        <Card className="bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-700">
          <CardContent className="p-6">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-6 h-6 text-red-600 dark:text-red-400" />
              <div>
                <p className="font-semibold text-red-900 dark:text-red-100">
                  Failed to load metrics
                </p>
                <p className="text-sm text-red-700 dark:text-red-300">
                  {error}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!metrics) {
    return null;
  }

  // Build metric cards
  const metricCards: MetricCardProps[] = [
    // System Health
    {
      title: 'System Health',
      value:
        metrics.systemHealth === 'healthy'
          ? 'Healthy'
          : metrics.systemHealth === 'degraded'
          ? 'Degraded'
          : 'Unhealthy',
      icon: <Activity className="w-6 h-6" />,
      iconColor:
        metrics.systemHealth === 'healthy'
          ? 'text-green-600 dark:text-green-400'
          : metrics.systemHealth === 'degraded'
          ? 'text-yellow-600 dark:text-yellow-400'
          : 'text-red-600 dark:text-red-400',
      iconBgColor:
        metrics.systemHealth === 'healthy'
          ? 'bg-green-100 dark:bg-green-900/30'
          : metrics.systemHealth === 'degraded'
          ? 'bg-yellow-100 dark:bg-yellow-900/30'
          : 'bg-red-100 dark:bg-red-900/30',
      subtitle: 'Overall system status',
      status:
        metrics.systemHealth === 'healthy'
          ? 'success'
          : metrics.systemHealth === 'degraded'
          ? 'warning'
          : 'error',
    },

    // WebSocket Connections
    {
      title: 'WebSocket Connections',
      value: `${metrics.wsConnectionsActive}/${metrics.wsConnectionsTotal}`,
      icon: <Activity className="w-6 h-6" />,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBgColor: 'bg-blue-100 dark:bg-blue-900/30',
      subtitle: 'Active / Total',
    },

    // Message Validation
    {
      title: 'Message Validation',
      value: formatMetricValue(metrics.messageValidationSuccessRate, 'percentage'),
      icon: <CheckCircle className="w-6 h-6" />,
      iconColor:
        metrics.messageValidationSuccessRate >= 0.95
          ? 'text-green-600 dark:text-green-400'
          : metrics.messageValidationSuccessRate >= 0.8
          ? 'text-yellow-600 dark:text-yellow-400'
          : 'text-red-600 dark:text-red-400',
      iconBgColor:
        metrics.messageValidationSuccessRate >= 0.95
          ? 'bg-green-100 dark:bg-green-900/30'
          : metrics.messageValidationSuccessRate >= 0.8
          ? 'bg-yellow-100 dark:bg-yellow-900/30'
          : 'bg-red-100 dark:bg-red-900/30',
      subtitle: `${metrics.messageValidationFailed} failed`,
      status:
        metrics.messageValidationSuccessRate >= 0.95
          ? 'success'
          : metrics.messageValidationSuccessRate >= 0.8
          ? 'warning'
          : 'error',
    },

    // Chat Messages
    {
      title: 'Chat Messages',
      value: formatMetricValue(metrics.chatMessagesTotal, 'number'),
      icon: <MessageSquare className="w-6 h-6" />,
      iconColor: 'text-purple-600 dark:text-purple-400',
      iconBgColor: 'bg-purple-100 dark:bg-purple-900/30',
      subtitle: `${formatMetricValue(
        metrics.chatMessagesSuccessRate,
        'percentage'
      )} success rate`,
      status:
        metrics.chatMessagesSuccessRate >= 0.95
          ? 'success'
          : metrics.chatMessagesSuccessRate >= 0.8
          ? 'warning'
          : 'error',
    },

    // AI Streaming Tokens
    {
      title: 'AI Stream Tokens',
      value: formatMetricValue(metrics.aiStreamTokensTotal, 'number'),
      icon: <Zap className="w-6 h-6" />,
      iconColor: 'text-yellow-600 dark:text-yellow-400',
      iconBgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
      subtitle: `${metrics.aiStreamChunksTotal} chunks`,
    },

    // AI Streaming Duration
    {
      title: 'AI Stream Duration',
      value: formatMetricValue(metrics.aiStreamDurationMs, 'duration'),
      icon: <Clock className="w-6 h-6" />,
      iconColor: 'text-orange-600 dark:text-orange-400',
      iconBgColor: 'bg-orange-100 dark:bg-orange-900/30',
      subtitle: 'Total streaming time',
    },

    // HTTP Requests
    {
      title: 'HTTP Requests',
      value: formatMetricValue(metrics.httpRequestsTotal, 'number'),
      icon: <Globe className="w-6 h-6" />,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBgColor: 'bg-blue-100 dark:bg-blue-900/30',
      subtitle: `${formatMetricValue(
        metrics.httpRequestsErrorRate,
        'percentage'
      )} error rate`,
      status:
        metrics.httpRequestsErrorRate <= 0.05
          ? 'success'
          : metrics.httpRequestsErrorRate <= 0.2
          ? 'warning'
          : 'error',
    },

    // MongoDB Reads
    {
      title: 'MongoDB Reads',
      value: formatMetricValue(metrics.mongoReadsTotal, 'number'),
      icon: <Database className="w-6 h-6" />,
      iconColor: 'text-green-600 dark:text-green-400',
      iconBgColor: 'bg-green-100 dark:bg-green-900/30',
      subtitle: 'Total read operations',
    },

    // MongoDB Writes
    {
      title: 'MongoDB Writes',
      value: formatMetricValue(metrics.mongoWritesTotal, 'number'),
      icon: <Database className="w-6 h-6" />,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBgColor: 'bg-blue-100 dark:bg-blue-900/30',
      subtitle: 'Total write operations',
    },

    // MongoDB Errors
    {
      title: 'MongoDB Errors',
      value: metrics.mongoErrorsTotal,
      icon: <XCircle className="w-6 h-6" />,
      iconColor:
        metrics.mongoErrorsTotal === 0
          ? 'text-green-600 dark:text-green-400'
          : metrics.mongoErrorsTotal < 10
          ? 'text-yellow-600 dark:text-yellow-400'
          : 'text-red-600 dark:text-red-400',
      iconBgColor:
        metrics.mongoErrorsTotal === 0
          ? 'bg-green-100 dark:bg-green-900/30'
          : metrics.mongoErrorsTotal < 10
          ? 'bg-yellow-100 dark:bg-yellow-900/30'
          : 'bg-red-100 dark:bg-red-900/30',
      subtitle: 'Database errors',
      status:
        metrics.mongoErrorsTotal === 0
          ? 'success'
          : metrics.mongoErrorsTotal < 10
          ? 'warning'
          : 'error',
    },

    // Response Time P50
    {
      title: 'Response Time (P50)',
      value: formatMetricValue(metrics.responseTimeP50, 'duration'),
      icon: <Clock className="w-6 h-6" />,
      iconColor: 'text-cyan-600 dark:text-cyan-400',
      iconBgColor: 'bg-cyan-100 dark:bg-cyan-900/30',
      subtitle: '50th percentile',
    },

    // Response Time P95
    {
      title: 'Response Time (P95)',
      value: formatMetricValue(metrics.responseTimeP95, 'duration'),
      icon: <Clock className="w-6 h-6" />,
      iconColor: 'text-indigo-600 dark:text-indigo-400',
      iconBgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
      subtitle: '95th percentile',
    },

    // Response Time P99
    {
      title: 'Response Time (P99)',
      value: formatMetricValue(metrics.responseTimeP99, 'duration'),
      icon: <Clock className="w-6 h-6" />,
      iconColor: 'text-violet-600 dark:text-violet-400',
      iconBgColor: 'bg-violet-100 dark:bg-violet-900/30',
      subtitle: '99th percentile',
    },
  ];

  return (
    <div className={cn('w-full', className)}>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            System Metrics
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Real-time Prometheus metrics • Auto-refresh every 5s
          </p>
        </div>
        <div className="text-right">
          <p className="text-xs text-gray-500 dark:text-gray-500">
            Last updated
          </p>
          <p className="text-sm font-mono text-gray-700 dark:text-gray-300">
            {lastUpdate.toLocaleTimeString()}
          </p>
        </div>
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {metricCards.map((card, index) => (
          <MetricCard key={index} {...card} />
        ))}
      </div>
    </div>
  );
};
