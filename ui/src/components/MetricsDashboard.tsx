/**
 * MetricsDashboard Component
 *
 * Displays real-time Prometheus metrics in a beautiful card-based layout
 * Features: auto-refresh, color-coded status, responsive grid layout
 */

import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  CircularProgress,
  Alert,
  LinearProgress,
  Chip,
} from '@mui/material';
import {
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  Remove as RemoveIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Speed as SpeedIcon,
  Memory as MemoryIcon,
  Storage as StorageIcon,
  NetworkCheck as NetworkCheckIcon,
  Message as MessageIcon,
  Security as SecurityIcon,
  Timer as TimerIcon,
} from '@mui/icons-material';
import { parsePrometheusMetrics, formatMetricValue, calculateRate, type ParsedMetrics } from '../utils/metricsParser';

interface MetricSnapshot {
  metrics: ParsedMetrics;
  timestamp: number;
}

interface MetricCardData {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  color: 'success' | 'warning' | 'error' | 'info' | 'primary';
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  subtitle?: string;
}

const METRICS_URL = 'http://localhost:5555/metrics';
const REFRESH_INTERVAL = 5000; // 5 seconds

export function MetricsDashboard() {
  const [currentSnapshot, setCurrentSnapshot] = useState<MetricSnapshot | null>(null);
  const [previousSnapshot, setPreviousSnapshot] = useState<MetricSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  const fetchMetrics = useCallback(async () => {
    try {
      const response = await fetch(METRICS_URL);
      if (!response.ok) {
        throw new Error(`Failed to fetch metrics: ${response.statusText}`);
      }

      const text = await response.text();
      const metrics = parsePrometheusMetrics(text);
      const now = Date.now();

      setPreviousSnapshot(currentSnapshot);
      setCurrentSnapshot({ metrics, timestamp: now });
      setLastUpdate(new Date());
      setError(null);
      setLoading(false);
    } catch (err) {
      console.error('[MetricsDashboard] Error fetching metrics:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      setLoading(false);
    }
  }, [currentSnapshot]);

  useEffect(() => {
    // Initial fetch
    fetchMetrics();

    // Set up auto-refresh
    const intervalId = setInterval(fetchMetrics, REFRESH_INTERVAL);

    return () => clearInterval(intervalId);
  }, [fetchMetrics]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error" variant="filled">
          {error}
        </Alert>
      </Box>
    );
  }

  if (!currentSnapshot) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="info">No metrics data available</Alert>
      </Box>
    );
  }

  const { gauges, counters, histograms } = currentSnapshot.metrics;

  // Calculate rates if we have previous snapshot
  const intervalSeconds = previousSnapshot
    ? (currentSnapshot.timestamp - previousSnapshot.timestamp) / 1000
    : 0;

  // Extract key metrics
  const activeConnections = gauges.get('chat_websocket_active_connections')?.value || 0;
  const connectionsTotal = counters.get('chat_websocket_connections_total')?.value || 0;
  const messagesSent = counters.get('chat_websocket_messages_sent_total')?.value || 0;
  const messagesReceived = counters.get('chat_websocket_messages_received_total')?.value || 0;
  const sessionsCreated = counters.get('chat_sessions_created_total')?.value || 0;
  const validationRejections = counters.get('chat_message_validation_rejections_total')?.value || 0;
  const aiTokens = counters.get('chat_ai_stream_tokens_total')?.value || 0;
  const messageSizeExceeded = counters.get('chat_message_size_exceeded_total')?.value || 0;
  const goroutines = gauges.get('go_goroutines')?.value || 0;
  const memoryBytes = gauges.get('go_memstats_alloc_bytes')?.value || 0;

  // Calculate rates
  let connectionRate = 0;
  let messagesSentRate = 0;
  let messagesReceivedRate = 0;

  if (previousSnapshot && intervalSeconds > 0) {
    const prevConnections = previousSnapshot.metrics.counters.get('chat_websocket_connections_total')?.value || 0;
    const prevMessagesSent = previousSnapshot.metrics.counters.get('chat_websocket_messages_sent_total')?.value || 0;
    const prevMessagesReceived = previousSnapshot.metrics.counters.get('chat_websocket_messages_received_total')?.value || 0;

    connectionRate = calculateRate(connectionsTotal, prevConnections, intervalSeconds);
    messagesSentRate = calculateRate(messagesSent, prevMessagesSent, intervalSeconds);
    messagesReceivedRate = calculateRate(messagesReceived, prevMessagesReceived, intervalSeconds);
  }

  // Get histogram data
  const aiStreamDuration = histograms.get('chat_ai_stream_duration_seconds');
  const messageSaveDuration = histograms.get('chat_message_save_duration_seconds');

  // Prepare metric cards
  const metricCards: MetricCardData[] = [
    {
      title: 'Active Connections',
      value: activeConnections,
      icon: <NetworkCheckIcon sx={{ fontSize: 40 }} />,
      color: activeConnections > 0 ? 'success' : 'info',
      subtitle: 'WebSocket clients',
    },
    {
      title: 'Connection Rate',
      value: formatMetricValue(connectionRate, 'rate'),
      icon: <TrendingUpIcon sx={{ fontSize: 40 }} />,
      color: connectionRate > 10 ? 'warning' : 'info',
      subtitle: 'New connections/min',
    },
    {
      title: 'Total Sessions',
      value: sessionsCreated,
      icon: <MessageIcon sx={{ fontSize: 40 }} />,
      color: 'primary',
      subtitle: 'Chat sessions created',
    },
    {
      title: 'Messages Sent',
      value: messagesSent,
      icon: <TrendingUpIcon sx={{ fontSize: 40 }} />,
      color: 'success',
      trend: messagesSentRate > 0 ? 'up' : 'neutral',
      trendValue: formatMetricValue(messagesSentRate, 'rate'),
      subtitle: 'Outbound messages',
    },
    {
      title: 'Messages Received',
      value: messagesReceived,
      icon: <TrendingDownIcon sx={{ fontSize: 40 }} />,
      color: 'info',
      trend: messagesReceivedRate > 0 ? 'up' : 'neutral',
      trendValue: formatMetricValue(messagesReceivedRate, 'rate'),
      subtitle: 'Inbound messages',
    },
    {
      title: 'Validation Rejections',
      value: validationRejections,
      icon: <SecurityIcon sx={{ fontSize: 40 }} />,
      color: validationRejections > 0 ? 'error' : 'success',
      subtitle: 'Security blocks',
    },
    {
      title: 'AI Tokens Generated',
      value: aiTokens,
      icon: <SpeedIcon sx={{ fontSize: 40 }} />,
      color: 'primary',
      subtitle: 'Total AI output',
    },
    {
      title: 'Size Limit Exceeded',
      value: messageSizeExceeded,
      icon: <WarningIcon sx={{ fontSize: 40 }} />,
      color: messageSizeExceeded > 0 ? 'warning' : 'success',
      subtitle: 'Message size errors',
    },
    {
      title: 'Goroutines',
      value: goroutines,
      icon: <StorageIcon sx={{ fontSize: 40 }} />,
      color: goroutines > 1000 ? 'warning' : 'info',
      subtitle: 'Active Go routines',
    },
    {
      title: 'Memory Usage',
      value: formatMetricValue(memoryBytes, 'bytes'),
      icon: <MemoryIcon sx={{ fontSize: 40 }} />,
      color: memoryBytes > 1024 * 1024 * 500 ? 'warning' : 'info', // 500 MB threshold
      subtitle: 'Allocated memory',
    },
  ];

  // Performance metrics
  const performanceCards: MetricCardData[] = [];

  if (aiStreamDuration && aiStreamDuration.count > 0) {
    performanceCards.push({
      title: 'AI Stream P95',
      value: formatMetricValue(aiStreamDuration.p95 || 0, 'seconds'),
      icon: <TimerIcon sx={{ fontSize: 40 }} />,
      color: (aiStreamDuration.p95 || 0) > 10 ? 'warning' : 'success',
      subtitle: `Avg: ${formatMetricValue(aiStreamDuration.avg || 0, 'seconds')}`,
    });
  }

  if (messageSaveDuration && messageSaveDuration.count > 0) {
    performanceCards.push({
      title: 'DB Save P95',
      value: formatMetricValue((messageSaveDuration.p95 || 0) * 1000, 'ms'),
      icon: <StorageIcon sx={{ fontSize: 40 }} />,
      color: (messageSaveDuration.p95 || 0) > 0.1 ? 'warning' : 'success',
      subtitle: `Avg: ${formatMetricValue((messageSaveDuration.avg || 0) * 1000, 'ms')}`,
    });
  }

  return (
    <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
      {/* Header */}
      <Box sx={{ mb: 3, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="h5" fontWeight={600}>
          Metrics Dashboard
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Chip
            icon={<CheckCircleIcon />}
            label={`Updated ${lastUpdate?.toLocaleTimeString()}`}
            color="success"
            size="small"
          />
          <Chip label="Auto-refresh: 5s" size="small" variant="outlined" />
        </Box>
      </Box>

      {/* Loading indicator during refresh */}
      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Main Metrics Grid */}
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: {
            xs: '1fr',
            sm: 'repeat(2, 1fr)',
            md: 'repeat(3, 1fr)',
            lg: 'repeat(4, 1fr)',
          },
          gap: 2,
          mb: 3,
        }}
      >
        {metricCards.map((card, index) => (
          <Card key={index}
              sx={{
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                transition: 'transform 0.2s, box-shadow 0.2s',
                '&:hover': {
                  transform: 'translateY(-4px)',
                },
              }}
            >
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="body2" color="text.secondary" fontWeight={500}>
                    {card.title}
                  </Typography>
                  <Box sx={{ color: `${card.color}.main` }}>{card.icon}</Box>
                </Box>

                <Typography variant="h4" fontWeight={700} sx={{ mb: 0.5, color: `${card.color}.main` }}>
                  {card.value}
                </Typography>

                {card.trend && card.trendValue && (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                    {card.trend === 'up' && <TrendingUpIcon fontSize="small" color="success" />}
                    {card.trend === 'down' && <TrendingDownIcon fontSize="small" color="error" />}
                    {card.trend === 'neutral' && <RemoveIcon fontSize="small" color="disabled" />}
                    <Typography variant="caption" color={card.trend === 'up' ? 'success.main' : 'text.secondary'}>
                      {card.trendValue}
                    </Typography>
                  </Box>
                )}

                {card.subtitle && (
                  <Typography variant="caption" color="text.secondary">
                    {card.subtitle}
                  </Typography>
                )}
              </CardContent>
            </Card>
        ))}
      </Box>

      {/* Performance Metrics Section */}
      {performanceCards.length > 0 && (
        <>
          <Typography variant="h6" fontWeight={600} sx={{ mb: 2 }}>
            Performance Metrics
          </Typography>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: {
                xs: '1fr',
                sm: 'repeat(2, 1fr)',
                md: 'repeat(3, 1fr)',
              },
              gap: 2,
            }}
          >
            {performanceCards.map((card, index) => (
              <Card key={index}
                  sx={{
                    height: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    transition: 'transform 0.2s, box-shadow 0.2s',
                    '&:hover': {
                      transform: 'translateY(-4px)',
                    },
                  }}
                >
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1 }}>
                      <Typography variant="body2" color="text.secondary" fontWeight={500}>
                        {card.title}
                      </Typography>
                      <Box sx={{ color: `${card.color}.main` }}>{card.icon}</Box>
                    </Box>

                    <Typography variant="h4" fontWeight={700} sx={{ mb: 0.5, color: `${card.color}.main` }}>
                      {card.value}
                    </Typography>

                    {card.subtitle && (
                      <Typography variant="caption" color="text.secondary">
                        {card.subtitle}
                      </Typography>
                    )}
                  </CardContent>
                </Card>
            ))}
          </Box>
        </>
      )}
    </Box>
  );
}
