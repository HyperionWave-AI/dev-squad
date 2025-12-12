import React, { useEffect, useState } from 'react';
import { ConnectionState } from '../services/WebSocketManager';

interface ConnectionStatusProps {
  state: ConnectionState;
  metrics?: {
    uptime: number;
    reconnectCount: number;
    messagesSent: number;
    messagesReceived: number;
    bufferUsage: number;
  };
  showDetails?: boolean;
}

/**
 * ConnectionStatus Component
 * 
 * Displays WebSocket connection status with visual indicators and optional metrics.
 * Shows connection state, reconnection attempts, and buffer usage.
 */
export const ConnectionStatus: React.FC<ConnectionStatusProps> = ({
  state,
  metrics,
  showDetails = false,
}) => {
  const [displayUptime, setDisplayUptime] = useState<string>('0s');

  // Update uptime display every second
  useEffect(() => {
    if (!metrics) return;

    const updateUptime = () => {
      const seconds = Math.floor(metrics.uptime / 1000);
      const minutes = Math.floor(seconds / 60);
      const hours = Math.floor(minutes / 60);

      if (hours > 0) {
        setDisplayUptime(`${hours}h ${minutes % 60}m`);
      } else if (minutes > 0) {
        setDisplayUptime(`${minutes}m ${seconds % 60}s`);
      } else {
        setDisplayUptime(`${seconds}s`);
      }
    };

    updateUptime();
    const interval = setInterval(updateUptime, 1000);
    return () => clearInterval(interval);
  }, [metrics]);

  // Determine status color and icon based on connection state
  const getStatusStyle = () => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return {
          color: '#10b981', // green
          icon: '●',
          label: 'Connected',
        };
      case ConnectionState.CONNECTING:
        return {
          color: '#f59e0b', // amber
          icon: '◐',
          label: 'Connecting...',
        };
      case ConnectionState.RECONNECTING:
        return {
          color: '#f59e0b', // amber
          icon: '◑',
          label: 'Reconnecting...',
        };
      case ConnectionState.ERROR:
        return {
          color: '#ef4444', // red
          icon: '✕',
          label: 'Connection Error',
        };
      case ConnectionState.DISCONNECTED:
        return {
          color: '#6b7280', // gray
          icon: '○',
          label: 'Disconnected',
        };
      case ConnectionState.DISCONNECTING:
        return {
          color: '#6b7280', // gray
          icon: '◔',
          label: 'Disconnecting...',
        };
      default:
        return {
          color: '#6b7280',
          icon: '?',
          label: 'Unknown',
        };
    }
  };

  const status = getStatusStyle();

  // Determine buffer warning
  const bufferWarning = metrics && metrics.bufferUsage > 80;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 12px',
        borderRadius: '6px',
        backgroundColor: 'rgba(0, 0, 0, 0.05)',
        fontSize: '12px',
        fontFamily: 'system-ui, -apple-system, sans-serif',
      }}
    >
      {/* Status Indicator */}
      <span
        style={{
          fontSize: '16px',
          color: status.color,
          fontWeight: 'bold',
          animation: state === ConnectionState.CONNECTING || state === ConnectionState.RECONNECTING
            ? 'pulse 1s infinite'
            : 'none',
        }}
      >
        {status.icon}
      </span>

      {/* Status Label */}
      <span style={{ color: status.color, fontWeight: '500' }}>
        {status.label}
      </span>

      {/* Reconnect Count */}
      {metrics && metrics.reconnectCount > 0 && (
        <span style={{ color: '#6b7280', marginLeft: '4px' }}>
          (reconnected {metrics.reconnectCount}x)
        </span>
      )}

      {/* Buffer Warning */}
      {bufferWarning && (
        <span
          style={{
            color: '#ef4444',
            fontWeight: '600',
            marginLeft: '4px',
            animation: 'pulse 1s infinite',
          }}
        >
          ⚠ Buffer {metrics!.bufferUsage}%
        </span>
      )}

      {/* Detailed Metrics */}
      {showDetails && metrics && (
        <div
          style={{
            marginLeft: 'auto',
            display: 'flex',
            gap: '16px',
            fontSize: '11px',
            color: '#6b7280',
          }}
        >
          <span>↑ {metrics.messagesSent}</span>
          <span>↓ {metrics.messagesReceived}</span>
          <span>⏱ {displayUptime}</span>
        </div>
      )}

      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }
      `}</style>
    </div>
  );
};

/**
 * ConnectionStatusBadge Component
 * 
 * Compact badge version for displaying in headers or toolbars.
 */
export const ConnectionStatusBadge: React.FC<{ state: ConnectionState }> = ({ state }) => {
  const getStatusStyle = () => {
    switch (state) {
      case ConnectionState.CONNECTED:
        return { bg: '#d1fae5', text: '#065f46' };
      case ConnectionState.CONNECTING:
      case ConnectionState.RECONNECTING:
        return { bg: '#fef3c7', text: '#92400e' };
      case ConnectionState.ERROR:
        return { bg: '#fee2e2', text: '#991b1b' };
      default:
        return { bg: '#f3f4f6', text: '#374151' };
    }
  };

  const style = getStatusStyle();
  const label = state.charAt(0) + state.slice(1).toLowerCase();

  return (
    <span
      style={{
        display: 'inline-block',
        padding: '4px 8px',
        borderRadius: '4px',
        backgroundColor: style.bg,
        color: style.text,
        fontSize: '11px',
        fontWeight: '600',
        fontFamily: 'system-ui, -apple-system, sans-serif',
      }}
    >
      {label}
    </span>
  );
};

export default ConnectionStatus;
