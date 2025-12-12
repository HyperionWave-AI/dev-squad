/**
 * ConnectionStatusIndicator
 * 
 * Displays real-time WebSocket connection status with visual feedback.
 * Shows connection state: connected, connecting, disconnected, error
 */

import React from 'react';
import { Wifi, WifiOff, AlertCircle } from 'lucide-react';

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error';

export const ConnectionStatusValues = {
  CONNECTED: 'connected' as const,
  CONNECTING: 'connecting' as const,
  DISCONNECTED: 'disconnected' as const,
  ERROR: 'error' as const,
};

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  showLabel?: boolean;
  className?: string;
}

export const ConnectionStatusIndicator: React.FC<ConnectionStatusIndicatorProps> = ({
  status,
  showLabel = false,
  className = '',
}) => {
  const getStatusConfig = () => {
    switch (status) {
      case 'connected':
        return {
          icon: Wifi,
          color: 'text-green-600 dark:text-green-400',
          bgColor: 'bg-green-100 dark:bg-green-900/30',
          label: 'Connected',
          pulse: true,
        };
      case 'connecting':
        return {
          icon: Wifi,
          color: 'text-yellow-600 dark:text-yellow-400',
          bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
          label: 'Connecting...',
          pulse: true,
        };
      case 'disconnected':
        return {
          icon: WifiOff,
          color: 'text-gray-600 dark:text-gray-400',
          bgColor: 'bg-gray-100 dark:bg-gray-900/30',
          label: 'Disconnected',
          pulse: false,
        };
      case 'error':
        return {
          icon: AlertCircle,
          color: 'text-red-600 dark:text-red-400',
          bgColor: 'bg-red-100 dark:bg-red-900/30',
          label: 'Connection Error',
          pulse: false,
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;

  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg ${config.bgColor} ${className}`}
      title={`WebSocket: ${config.label}`}
    >
      <div className="relative">
        <Icon className={`w-4 h-4 ${config.color}`} />
        {config.pulse && (
          <div className="absolute inset-0 w-4 h-4 rounded-full animate-pulse opacity-50" />
        )}
      </div>
      {showLabel && (
        <span className={`text-xs font-medium ${config.color}`}>
          {config.label}
        </span>
      )}
    </div>
  );
};

export default ConnectionStatusIndicator;
