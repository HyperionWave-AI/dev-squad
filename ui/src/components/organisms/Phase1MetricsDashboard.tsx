/**
 * Phase 1 Metrics Dashboard Component
 *
 * Displays Phase 1 metrics (costs, tokens, cache performance) from the backend API.
 * Features:
 * - Cost by provider (pie chart)
 * - Cost by model (bar chart)
 * - Daily cost trends (line chart)
 * - Cache hit rate (gauge)
 * - Tool execution stats (table)
 * - Auto-refresh every 10 seconds
 */

import React, { useEffect, useState, useCallback } from 'react';
import { Card, CardContent } from '@/components/molecules/Card';
import {
  Activity,
  TrendingUp,
  TrendingDown,
  Minus,
  AlertCircle,
  DollarSign,
  Zap,
  Target,
  BarChart3,
} from 'lucide-react';
import { cn } from '@/utils';
import {
  parsePhase1Metrics,
  formatMetricValue,
  type Phase1Metrics,
  type ToolMetrics,
} from '@/utils/metricsParser';

interface Phase1MetricsDashboardProps {
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

/**
 * Pie Chart Component (simple SVG-based)
 */
const PieChart: React.FC<{
  data: Array<{ label: string; value: number }>;
  title: string;
}> = ({ data, title }) => {
  const total = data.reduce((sum, item) => sum + item.value, 0);
  const colors = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'];

  let currentAngle = 0;
  const slices = data.map((item, index) => {
    const sliceAngle = (item.value / total) * 360;
    const startAngle = currentAngle;
    const endAngle = currentAngle + sliceAngle;
    currentAngle = endAngle;

    const startRad = (startAngle * Math.PI) / 180;
    const endRad = (endAngle * Math.PI) / 180;
    const x1 = 100 + 80 * Math.cos(startRad);
    const y1 = 100 + 80 * Math.sin(startRad);
    const x2 = 100 + 80 * Math.cos(endRad);
    const y2 = 100 + 80 * Math.sin(endRad);

    const largeArc = sliceAngle > 180 ? 1 : 0;
    const pathData = [
      `M 100 100`,
      `L ${x1} ${y1}`,
      `A 80 80 0 ${largeArc} 1 ${x2} ${y2}`,
      'Z',
    ].join(' ');

    return (
      <path
        key={index}
        d={pathData}
        fill={colors[index % colors.length]}
        stroke="white"
        strokeWidth="2"
      />
    );
  });

  return (
    <Card className="backdrop-blur-md border-white/30 dark:border-gray-700/30 bg-white/70 dark:bg-gray-800/70">
      <CardContent className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {title}
        </h3>
        <div className="flex items-center justify-between">
          <svg width="200" height="200" viewBox="0 0 200 200">
            {slices}
            <circle cx="100" cy="100" r="50" fill="white" />
            <circle cx="100" cy="100" r="50" fill="currentColor" className="dark:fill-gray-800" />
          </svg>
          <div className="flex-1 ml-6 space-y-2">
            {data.map((item, index) => (
              <div key={index} className="flex items-center gap-2">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: colors[index % colors.length] }}
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  {item.label}: {formatMetricValue(item.value, 'currency')}
                </span>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

/**
 * Bar Chart Component (simple SVG-based)
 */
const BarChart: React.FC<{
  data: Array<{ label: string; value: number }>;
  title: string;
}> = ({ data, title }) => {
  const maxValue = Math.max(...data.map((d) => d.value), 1);
  const barHeight = 150;

  return (
    <Card className="backdrop-blur-md border-white/30 dark:border-gray-700/30 bg-white/70 dark:bg-gray-800/70">
      <CardContent className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {title}
        </h3>
        <div className="flex items-end justify-around h-48 gap-2">
          {data.map((item, index) => (
            <div key={index} className="flex flex-col items-center gap-2">
              <div
                className="bg-blue-500 rounded-t transition-all hover:bg-blue-600"
                style={{
                  width: '40px',
                  height: `${(item.value / maxValue) * barHeight}px`,
                }}
              />
              <span className="text-xs text-gray-600 dark:text-gray-400 text-center max-w-[50px]">
                {item.label}
              </span>
              <span className="text-xs font-semibold text-gray-900 dark:text-white">
                {formatMetricValue(item.value, 'currency')}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};

/**
 * Line Chart Component (simple SVG-based)
 */
const LineChart: React.FC<{
  data: Array<{ label: string; value: number }>;
  title: string;
}> = ({ data, title }) => {
  const maxValue = Math.max(...data.map((d) => d.value), 1);
  const width = 400;
  const height = 200;
  const padding = 40;

  const points = data.map((item, index) => {
    const x = padding + (index / (data.length - 1 || 1)) * (width - 2 * padding);
    const y = height - padding - (item.value / maxValue) * (height - 2 * padding);
    return { x, y, ...item };
  });

  const pathData = points
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`)
    .join(' ');

  return (
    <Card className="backdrop-blur-md border-white/30 dark:border-gray-700/30 bg-white/70 dark:bg-gray-800/70">
      <CardContent className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {title}
        </h3>
        <svg width={width} height={height} className="w-full">
          {/* Grid lines */}
          {[0, 0.25, 0.5, 0.75, 1].map((ratio, i) => (
            <line
              key={i}
              x1={padding}
              y1={height - padding - ratio * (height - 2 * padding)}
              x2={width - padding}
              y2={height - padding - ratio * (height - 2 * padding)}
              stroke="#e5e7eb"
              strokeDasharray="4"
              className="dark:stroke-gray-700"
            />
          ))}

          {/* Line */}
          <path
            d={pathData}
            fill="none"
            stroke="#3b82f6"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Points */}
          {points.map((p, i) => (
            <circle
              key={i}
              cx={p.x}
              cy={p.y}
              r="4"
              fill="#3b82f6"
              stroke="white"
              strokeWidth="2"
            />
          ))}

          {/* X-axis labels */}
          {points.map((p, i) => (
            <text
              key={`label-${i}`}
              x={p.x}
              y={height - 10}
              textAnchor="middle"
              className="text-xs fill-gray-600 dark:fill-gray-400"
            >
              {p.label}
            </text>
          ))}
        </svg>
      </CardContent>
    </Card>
  );
};

/**
 * Gauge Component (simple SVG-based)
 */
const Gauge: React.FC<{
  value: number;
  title: string;
  max?: number;
}> = ({ value, title, max = 1 }) => {
  const percentage = Math.min((value / max) * 100, 100);
  const angle = (percentage / 100) * 180 - 90;

  return (
    <Card className="backdrop-blur-md border-white/30 dark:border-gray-700/30 bg-white/70 dark:bg-gray-800/70">
      <CardContent className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {title}
        </h3>
        <div className="flex flex-col items-center">
          <svg width="150" height="100" viewBox="0 0 150 100">
            {/* Background arc */}
            <path
              d="M 30 80 A 50 50 0 0 1 120 80"
              fill="none"
              stroke="#e5e7eb"
              strokeWidth="8"
              className="dark:stroke-gray-700"
            />

            {/* Value arc */}
            <path
              d="M 30 80 A 50 50 0 0 1 120 80"
              fill="none"
              stroke="#10b981"
              strokeWidth="8"
              strokeDasharray={`${(percentage / 100) * 157} 157`}
              strokeLinecap="round"
            />

            {/* Needle */}
            <line
              x1="75"
              y1="80"
              x2={75 + 40 * Math.cos((angle * Math.PI) / 180)}
              y2={80 + 40 * Math.sin((angle * Math.PI) / 180)}
              stroke="#374151"
              strokeWidth="3"
              strokeLinecap="round"
              className="dark:stroke-gray-300"
            />

            {/* Center circle */}
            <circle cx="75" cy="80" r="5" fill="#374151" className="dark:fill-gray-300" />
          </svg>

          <div className="mt-4 text-center">
            <p className="text-3xl font-bold text-gray-900 dark:text-white">
              {formatMetricValue(percentage, 'percentage')}
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Cache Hit Rate
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

/**
 * Tool Stats Table Component
 */
const ToolStatsTable: React.FC<{
  tools: ToolMetrics[];
}> = ({ tools }) => {
  return (
    <Card className="backdrop-blur-md border-white/30 dark:border-gray-700/30 bg-white/70 dark:bg-gray-800/70">
      <CardContent className="p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Tool Execution Stats
        </h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700">
                <th className="text-left py-2 px-3 font-semibold text-gray-700 dark:text-gray-300">
                  Tool
                </th>
                <th className="text-right py-2 px-3 font-semibold text-gray-700 dark:text-gray-300">
                  Executions
                </th>
                <th className="text-right py-2 px-3 font-semibold text-gray-700 dark:text-gray-300">
                  Success
                </th>
                <th className="text-right py-2 px-3 font-semibold text-gray-700 dark:text-gray-300">
                  Failures
                </th>
                <th className="text-right py-2 px-3 font-semibold text-gray-700 dark:text-gray-300">
                  Avg Time (ms)
                </th>
              </tr>
            </thead>
            <tbody>
              {tools.map((tool, index) => (
                <tr
                  key={index}
                  className="border-b border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50"
                >
                  <td className="py-3 px-3 text-gray-900 dark:text-white font-medium">
                    {tool.toolName}
                  </td>
                  <td className="py-3 px-3 text-right text-gray-700 dark:text-gray-300">
                    {tool.executionCount}
                  </td>
                  <td className="py-3 px-3 text-right text-green-600 dark:text-green-400 font-semibold">
                    {tool.successCount}
                  </td>
                  <td className="py-3 px-3 text-right text-red-600 dark:text-red-400 font-semibold">
                    {tool.failureCount}
                  </td>
                  <td className="py-3 px-3 text-right text-gray-700 dark:text-gray-300">
                    {tool.averageExecutionTimeMs.toFixed(2)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
};

export const Phase1MetricsDashboard: React.FC<Phase1MetricsDashboardProps> = ({
  className,
}) => {
  const [metrics, setMetrics] = useState<Phase1Metrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  // Fetch Phase 1 metrics from /api/v1/metrics/phase1 endpoint
  const fetchMetrics = useCallback(async () => {
    try {
      const response = await fetch('/api/v1/metrics/phase1');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = await response.json();
      const parsed = parsePhase1Metrics(data);
      setMetrics(parsed);
      setError(null);
      setLastUpdate(new Date());
    } catch (err) {
      console.error('Failed to fetch Phase 1 metrics:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch and auto-refresh every 10 seconds
  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 10000);
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  if (loading) {
    return (
      <div className={cn('w-full', className)}>
        <div className="flex items-center justify-center p-12">
          <Activity className="w-8 h-8 text-blue-500 animate-spin" />
          <span className="ml-3 text-gray-600 dark:text-gray-400">
            Loading Phase 1 metrics...
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
                  Failed to load Phase 1 metrics
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

  // Prepare data for charts
  const costByProvider = metrics.providers.map((p) => ({
    label: p.provider,
    value: p.totalCost,
  }));

  const costByModel = Object.entries(
    metrics.providers.reduce(
      (acc, p) => {
        Object.entries(p.models).forEach(([model, stats]) => {
          acc[model] = (acc[model] || 0) + stats.cost;
        });
        return acc;
      },
      {} as Record<string, number>
    )
  )
    .map(([label, value]) => ({ label, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 5);

  const dailyCostTrends = metrics.dailyCosts
    .slice(-7)
    .map((d) => ({
      label: d.date.split('-')[2], // Just the day
      value: d.totalCost,
    }));

  const cacheMetrics = metrics.cache.length > 0
    ? metrics.cache.reduce((sum, c) => sum + c.hitRate, 0) / metrics.cache.length
    : 0;

  // Build summary metric cards
  const summaryCards: MetricCardProps[] = [
    {
      title: 'Total Cost',
      value: formatMetricValue(metrics.totalCost, 'currency'),
      icon: <DollarSign className="w-6 h-6" />,
      iconColor: 'text-green-600 dark:text-green-400',
      iconBgColor: 'bg-green-100 dark:bg-green-900/30',
      subtitle: `${metrics.totalCalls} total calls`,
    },
    {
      title: 'Total Tokens',
      value: formatMetricValue(metrics.totalTokens, 'number'),
      icon: <Zap className="w-6 h-6" />,
      iconColor: 'text-yellow-600 dark:text-yellow-400',
      iconBgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
      subtitle: `${formatMetricValue(metrics.averageCostPerCall, 'currency')} per call`,
    },
    {
      title: 'Avg Cost Per Call',
      value: formatMetricValue(metrics.averageCostPerCall, 'currency'),
      icon: <Target className="w-6 h-6" />,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBgColor: 'bg-blue-100 dark:bg-blue-900/30',
      subtitle: 'Average cost',
    },
    {
      title: 'Cache Hit Rate',
      value: formatMetricValue(cacheMetrics, 'percentage'),
      icon: <BarChart3 className="w-6 h-6" />,
      iconColor: 'text-purple-600 dark:text-purple-400',
      iconBgColor: 'bg-purple-100 dark:bg-purple-900/30',
      subtitle: `${metrics.cache.length} tools tracked`,
    },
  ];

  return (
    <div className={cn('w-full', className)}>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Phase 1 Metrics
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Cost tracking and performance analytics • Auto-refresh every 10s
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

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {summaryCards.map((card, index) => (
          <MetricCard key={index} {...card} />
        ))}
      </div>

      {/* Charts Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Cost by Provider */}
        {costByProvider.length > 0 && (
          <PieChart data={costByProvider} title="Cost by Provider" />
        )}

        {/* Cost by Model */}
        {costByModel.length > 0 && (
          <BarChart data={costByModel} title="Cost by Model (Top 5)" />
        )}

        {/* Daily Cost Trends */}
        {dailyCostTrends.length > 0 && (
          <LineChart data={dailyCostTrends} title="Daily Cost Trends (Last 7 Days)" />
        )}

        {/* Cache Hit Rate Gauge */}
        <Gauge value={cacheMetrics} title="Overall Cache Performance" max={1} />
      </div>

      {/* Tool Stats Table */}
      {metrics.tools.length > 0 && (
        <ToolStatsTable tools={metrics.tools} />
      )}
    </div>
  );
};
