import React, { useEffect } from 'react';
import { FileText, FolderOpen, Clock, Activity, RefreshCw } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Badge } from '../atoms/Badge';
import type { IndexStatus } from '../../types/codeSearch';

interface IndexStatusDisplayProps {
  status: IndexStatus;
  onRefresh: () => void;
}

export const IndexStatusDisplay: React.FC<IndexStatusDisplayProps> = ({
  status,
  onRefresh,
}) => {
  // Auto-refresh every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      onRefresh();
    }, 5000);

    return () => clearInterval(interval);
  }, [onRefresh]);

  const formatLastScan = (timestamp?: string): string => {
    if (!timestamp) return 'Never';

    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  };

  const statusCards = [
    {
      icon: FileText,
      label: 'Total Files',
      value: status.totalFiles.toLocaleString(),
      color: 'text-blue-600 dark:text-blue-400',
      bgColor: 'bg-blue-50 dark:bg-blue-900/20',
    },
    {
      icon: FolderOpen,
      label: 'Total Folders',
      value: status.totalFolders.toLocaleString(),
      color: 'text-purple-600 dark:text-purple-400',
      bgColor: 'bg-purple-50 dark:bg-purple-900/20',
    },
    {
      icon: Clock,
      label: 'Last Scan',
      value: formatLastScan(status.lastScanTime),
      color: 'text-green-600 dark:text-green-400',
      bgColor: 'bg-green-50 dark:bg-green-900/20',
    },
    {
      icon: Activity,
      label: 'Status',
      value: status.isRunning ? 'Running' : 'Idle',
      color: status.isRunning ? 'text-orange-600 dark:text-orange-400' : 'text-gray-600 dark:text-gray-400',
      bgColor: status.isRunning ? 'bg-orange-50 dark:bg-orange-900/20' : 'bg-gray-50 dark:bg-gray-900/20',
    },
  ];

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Index Status
          </h2>
          <Button
            variant="ghost"
            size="icon"
            onClick={onRefresh}
            title="Refresh status"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Status Cards */}
      <div className="p-4 space-y-3">
        {statusCards.map((card, index) => {
          const Icon = card.icon;
          return (
            <div
              key={index}
              className={`${card.bgColor} rounded-lg p-3 border border-gray-200 dark:border-gray-700`}
            >
              <div className="flex items-center gap-3">
                <div className={`${card.color} flex-shrink-0`}>
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-gray-600 dark:text-gray-400 mb-0.5">
                    {card.label}
                  </p>
                  <p className={`text-lg font-semibold ${card.color}`}>
                    {card.value}
                  </p>
                </div>
                {card.label === 'Status' && status.isRunning && (
                  <Badge variant="warning" className="animate-pulse">
                    Active
                  </Badge>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Auto-refresh indicator */}
      <div className="px-4 pb-4">
        <div className="flex items-center justify-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <div className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
          Auto-refreshing every 5s
        </div>
      </div>
    </div>
  );
};
