import React from 'react';
import { AlertCircle, Wifi, WifiOff } from 'lucide-react';

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error';

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  showLabel?: boolean;
}

export const ConnectionStatusIndicator: React.FC<ConnectionStatusIndicatorProps> = ({
  status,
  showLabel = false,
}) => {
  const getStatusConfig = (status: ConnectionStatus) => {
    switch (status) {
      case 'connected':
        return {
          icon: Wifi,
          color: 'text-green-600 dark:text-green-400',
          bgColor: 'bg-green-100 dark:bg-green-900/30',
          label: 'Connected',
          title: 'WebSocket connection is stable',
          animate: false,
        };
      case 'connecting':
        return {
          icon: Wifi,
          color: 'text-blue-600 dark:text-blue-400',
          bgColor: 'bg-blue-100 dark:bg-blue-900/30',
          label: 'Connecting...',
          title: 'Attempting to establish connection',
          animate: true,
        };
      case 'disconnected':
        return {
          icon: WifiOff,
          color: 'text-gray-600 dark:text-gray-400',
          bgColor: 'bg-gray-100 dark:bg-gray-900/30',
          label: 'Disconnected',
          title: 'WebSocket connection lost',
          animate: false,
        };
      case 'error':
        return {
          icon: AlertCircle,
          color: 'text-red-600 dark:text-red-400',
          bgColor: 'bg-red-100 dark:bg-red-900/30',
          label: 'Connection Error',
          title: 'WebSocket connection failed - will retry',
          animate: true,
        };
    }
  };

  const config = getStatusConfig(status);
  const Icon = config.icon;

  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg ${config.bgColor} transition-all ${
        config.animate ? 'animate-pulse' : ''
      }`}
      title={config.title}
    >
      <Icon className={`w-4 h-4 ${config.color} ${config.animate ? 'animate-spin' : ''}`} />
      {showLabel && (
        <span className={`text-xs font-medium ${config.color}`}>
          {config.label}
        </span>
      )}
    </div>
  );
};

export default ConnectionStatusIndicator;
