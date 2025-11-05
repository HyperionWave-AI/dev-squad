/**
 * MetricsDashboard Component
 *
 * Displays task metrics in a beautiful 4-card grid layout
 * Features: real-time updates, responsive design, dark mode support
 */

import React from 'react';
import { Card, CardContent } from '@/components/molecules/Card';
import {
  CheckCircle,
  Clock,
  TrendingUp,
  BarChart3
} from 'lucide-react';
import { cn } from '@/utils';

export interface TaskMetrics {
  totalTasks: number;
  completedTasks: number;
  averageExecutionTime: string; // formatted string like "2.5h" or "30m"
  successRate: number; // percentage 0-100
}

interface MetricsDashboardProps {
  metrics: TaskMetrics;
  className?: string;
}

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  iconColor: string;
  iconBgColor: string;
  subtitle?: string;
}

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  icon,
  iconColor,
  iconBgColor,
  subtitle
}) => (
  <Card className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border-white/30 dark:border-gray-700/30 hover:shadow-xl transition-all duration-300 hover:-translate-y-1">
    <CardContent className="p-6">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-2">
            {title}
          </p>
          <p className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
            {value}
          </p>
          {subtitle && (
            <p className="text-xs text-gray-500 dark:text-gray-500">
              {subtitle}
            </p>
          )}
        </div>
        <div className={cn(
          'p-3 rounded-lg',
          iconBgColor
        )}>
          <div className={iconColor}>
            {icon}
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
);

export const MetricsDashboard: React.FC<MetricsDashboardProps> = ({
  metrics,
  className
}) => {
  const { totalTasks, completedTasks, averageExecutionTime, successRate } = metrics;

  const metricCards: MetricCardProps[] = [
    {
      title: 'Total Tasks',
      value: totalTasks,
      icon: <BarChart3 className="w-6 h-6" />,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBgColor: 'bg-blue-100 dark:bg-blue-900/30',
      subtitle: 'All tasks in system',
    },
    {
      title: 'Completed',
      value: completedTasks,
      icon: <CheckCircle className="w-6 h-6" />,
      iconColor: 'text-green-600 dark:text-green-400',
      iconBgColor: 'bg-green-100 dark:bg-green-900/30',
      subtitle: `${totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0}% of total`,
    },
    {
      title: 'Avg. Execution Time',
      value: averageExecutionTime,
      icon: <Clock className="w-6 h-6" />,
      iconColor: 'text-purple-600 dark:text-purple-400',
      iconBgColor: 'bg-purple-100 dark:bg-purple-900/30',
      subtitle: 'Time to complete',
    },
    {
      title: 'Success Rate',
      value: `${successRate.toFixed(1)}%`,
      icon: <TrendingUp className="w-6 h-6" />,
      iconColor: successRate >= 80
        ? 'text-green-600 dark:text-green-400'
        : successRate >= 50
        ? 'text-yellow-600 dark:text-yellow-400'
        : 'text-red-600 dark:text-red-400',
      iconBgColor: successRate >= 80
        ? 'bg-green-100 dark:bg-green-900/30'
        : successRate >= 50
        ? 'bg-yellow-100 dark:bg-yellow-900/30'
        : 'bg-red-100 dark:bg-red-900/30',
      subtitle: 'Completion ratio',
    },
  ];

  return (
    <div className={cn('w-full', className)}>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {metricCards.map((card, index) => (
          <MetricCard key={index} {...card} />
        ))}
      </div>
    </div>
  );
};
