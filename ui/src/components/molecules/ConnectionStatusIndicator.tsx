/**
 * Connection Status Indicator Component
 * 
 * Displays WebSocket connection status in the UI.
 * Shows connection state, reconnection attempts, and last error.
 * 
 * Usage:
 * <ConnectionStatusIndicator 
 *   state={connectionState}
 *   isConnected={isConnected}
 *   lastError={lastError}
 *   reconnectAttempts={attempts}
 * />
 */

import React from 'react';
import { Wifi, WifiOff, AlertCircle, Loader } from 'lucide-react';
import { ConnectionState } from '@/services/chatService';

interface ConnectionStatusIndicatorProps {
  state: ConnectionState;
  isConnected: boolean;
  lastError?: string | null;
  reconnectAttempts?: number;
  maxReconnectAttempts?: number;
}

export const ConnectionStatusIndicator: React.FC<ConnectionStatusIndicatorProps> = ({
  state,
  lastError,
  reconnectAttempts = 0,
  maxReconnectAttempts = 5,
}) => {
  const getStatusColor = (): string => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return 'text-green-600 dark:text-green-400';
      case ConnectionState.CONNECTING:
        return 'text-yellow-600 dark:text-yellow-400';
      case ConnectionState.DISCONNECTED:
        return 'text-gray-600 dark:text-gray-400';
      case ConnectionState.ERROR:
        return 'text-red-600 dark:text-red-400';
      case ConnectionState.DISCONNECTING:
        return 'text-orange-600 dark:text-orange-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  const getStatusLabel = (): string => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return 'Connected';
      case ConnectionState.CONNECTING:
        return `Connecting${reconnectAttempts > 0 ? ` (${reconnectAttempts}/${maxReconnectAttempts})` : ''}`;
      case ConnectionState.DISCONNECTED:
        return 'Disconnected';
      case ConnectionState.ERROR:
        return 'Connection Error';
      case ConnectionState.DISCONNECTING:
        return 'Disconnecting';
      default:
        return 'Unknown';
    }
  };

  const getStatusIcon = (): React.ReactNode => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return <Wifi className="w-4 h-4" />;
      case ConnectionState.CONNECTING:
        return <Loader className="w-4 h-4 animate-spin" />;
      case ConnectionState.DISCONNECTED:
        return <WifiOff className="w-4 h-4" />;
      case ConnectionState.ERROR:
        return <AlertCircle className="w-4 h-4" />;
      case ConnectionState.DISCONNECTING:
        return <Loader className="w-4 h-4 animate-spin" />;
      default:
        return <WifiOff className="w-4 h-4" />;
    }
  };

  const getBgColor = (): string => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800';
      case ConnectionState.CONNECTING:
        return 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800';
      case ConnectionState.DISCONNECTED:
        return 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700';
      case ConnectionState.ERROR:
        return 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800';
      case ConnectionState.DISCONNECTING:
        return 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800';
      default:
        return 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700';
    }
  };

  return (
    <div className={`flex items-center gap-2 px-3 py-2 rounded-lg border ${getBgColor()} transition-all`}>
      <div className={`${getStatusColor()}`}>
        {getStatusIcon()}
      </div>
      <div className="flex flex-col gap-0.5">
        <span className={`text-xs font-semibold ${getStatusColor()}`}>
          {getStatusLabel()}
        </span>
        {lastError && state === ConnectionState.ERROR && (
          <span className="text-xs text-red-600 dark:text-red-400 truncate max-w-xs">
            {lastError}
          </span>
        )}
      </div>
    </div>
  );
};
