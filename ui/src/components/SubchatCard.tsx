/**
 * SubchatCard Component
 *
 * Displays individual subchat information with subagent badge,
 * assigned task/todo, and navigation capability.
 */

import React from 'react';
import {
  Card,
  CardContent,
  CardActionArea,
  Typography,
  Chip,
  Box,
  Stack,
  CircularProgress,
  LinearProgress,
  IconButton,
  Tooltip,
  Divider,
} from '@mui/material';
import {
  AccountTree as TaskIcon,
  CheckCircle as TodoIcon,
  Person as AgentIcon,
  Schedule as TimeIcon,
  PlayArrow as RunningIcon,
  CheckCircleOutline as CompletedIcon,
  Error as ErrorIcon,
  ExpandMore as ExpandMoreIcon,
  Launch as LaunchIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import type { Subchat } from '../services/subchatService';

interface SubchatCardProps {
  subchat: Subchat;
  onClick?: (subchatId: string) => void;
  onDelete?: (subchatId: string) => void;
  isExpanded?: boolean;
  onToggleDetails?: (subchatId: string) => void;
}

// Category colors for subagent badges
const CATEGORY_COLORS: Record<string, 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info'> = {
  'Backend Infrastructure': 'primary',
  'Frontend & Experience': 'secondary',
  'Platform & Operations': 'success',
  'Testing & Quality': 'warning',
};

// Status colors
const STATUS_COLORS: Record<string, 'default' | 'primary' | 'success' | 'error'> = {
  active: 'primary',
  completed: 'success',
  failed: 'error',
};

// Status icons
const STATUS_ICONS: Record<string, React.ElementType> = {
  active: RunningIcon,
  completed: CompletedIcon,
  failed: ErrorIcon,
};

export const SubchatCard: React.FC<SubchatCardProps> = ({
  subchat,
  onClick,
  onDelete,
  isExpanded = false,
  onToggleDetails
}) => {
  const handleClick = () => {
    if (onClick && subchat.sessionId) {
      // Pass the session ID (not subchat ID) for navigation
      onClick(subchat.sessionId);
    }
  };

  const handleToggleDetails = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    if (onToggleDetails) {
      onToggleDetails(subchat.id);
    }
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    if (onDelete) {
      onDelete(subchat.id);
    }
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) {
      return `${diffMins}m ago`;
    } else if (diffHours < 24) {
      return `${diffHours}h ago`;
    } else if (diffDays < 7) {
      return `${diffDays}d ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  // Extract category from subagent name (if it follows naming convention)
  const getSubagentCategory = (name: string): string => {
    if (name.includes('go-') || name.includes('Backend')) return 'Backend Infrastructure';
    if (name.includes('ui-') || name.includes('Frontend')) return 'Frontend & Experience';
    if (name.includes('sre') || name.includes('k8s-')) return 'Platform & Operations';
    if (name.includes('tester') || name.includes('Testing')) return 'Testing & Quality';
    return 'Backend Infrastructure'; // default
  };

  const category = getSubagentCategory(subchat.subagentName);
  const categoryColor = CATEGORY_COLORS[category] || 'default';
  const statusColor = STATUS_COLORS[subchat.status] || 'default';
  const StatusIcon = STATUS_ICONS[subchat.status] || RunningIcon;
  const isRunning = subchat.status === 'active';

  return (
    <Card
      variant="outlined"
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        borderRadius: 3,
        overflow: 'hidden',
        position: 'relative',
        '&:hover': {
          boxShadow: (theme) => theme.shadows[8],
          transform: 'translateY(-4px)',
          borderColor: 'primary.main',
        },
        // Highlight when expanded
        ...(isExpanded && {
          borderColor: 'primary.main',
          borderWidth: 2,
          boxShadow: (theme) => theme.shadows[12],
          transform: 'translateY(-2px)',
        }),
        // Add subtle pulse animation for running subchats
        ...(isRunning && !isExpanded && {
          '&::before': {
            content: '""',
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            height: '4px',
            background: 'linear-gradient(90deg, transparent, primary.main, transparent)',
            animation: 'shimmer 2s ease-in-out infinite',
          },
          '@keyframes shimmer': {
            '0%': { transform: 'translateX(-100%)' },
            '100%': { transform: 'translateX(100%)' },
          },
        }),
      }}
    >
      <CardActionArea 
        onClick={handleClick} 
        disabled={!onClick}
        sx={{ 
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'stretch',
          '&:hover .launch-icon': {
            opacity: 1,
            transform: 'translateX(0)',
          },
        }}
      >
        <CardContent sx={{ flexGrow: 1, p: 3 }}>
          <Stack spacing={2.5} height="100%">
            {/* Header with subagent name and category */}
            <Box>
              <Box display="flex" alignItems="flex-start" gap={1.5} mb={1.5}>
                <Box
                  sx={{
                    p: 1,
                    borderRadius: 2,
                    backgroundColor: `${categoryColor}.50`,
                    color: `${categoryColor}.main`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <AgentIcon fontSize="small" />
                </Box>
                <Box flexGrow={1} minWidth={0}>
                  <Typography 
                    variant="h6" 
                    component="div" 
                    sx={{ 
                      fontWeight: 600,
                      fontSize: '1.1rem',
                      lineHeight: 1.3,
                      mb: 0.5,
                      wordBreak: 'break-word',
                    }}
                  >
                    {subchat.subagentName}
                  </Typography>
                  <Chip
                    label={category}
                    color={categoryColor}
                    size="small"
                    sx={{
                      fontSize: '0.75rem',
                      height: 24,
                      fontWeight: 500,
                    }}
                  />
                </Box>
                {onClick && (
                  <Tooltip title="Open subchat">
                    <Box
                      className="launch-icon"
                      sx={{
                        opacity: 0,
                        transform: 'translateX(8px)',
                        transition: 'all 0.2s ease',
                        color: 'primary.main',
                      }}
                    >
                      <LaunchIcon fontSize="small" />
                    </Box>
                  </Tooltip>
                )}
              </Box>
            </Box>

            <Divider />

            {/* Status section with enhanced visual feedback */}
            <Box>
              <Box display="flex" alignItems="center" gap={1.5} mb={isRunning ? 1.5 : 0}>
                <Chip
                  icon={<StatusIcon />}
                  label={subchat.status}
                  color={statusColor}
                  size="small"
                  sx={{ 
                    textTransform: 'capitalize',
                    fontWeight: 600,
                    '& .MuiChip-icon': {
                      fontSize: '1rem',
                    },
                  }}
                />
                {isRunning && (
                  <Box display="flex" alignItems="center" gap={0.5}>
                    <CircularProgress size={16} thickness={5} />
                    <Typography variant="caption" color="primary.main" sx={{ fontWeight: 500 }}>
                      Processing...
                    </Typography>
                  </Box>
                )}
              </Box>
              {/* Enhanced progress bar for running subchats */}
              {isRunning && (
                <Box sx={{ position: 'relative' }}>
                  <LinearProgress 
                    color="primary" 
                    sx={{
                      height: 6,
                      borderRadius: 3,
                      backgroundColor: 'primary.50',
                      '& .MuiLinearProgress-bar': {
                        borderRadius: 3,
                      },
                    }}
                  />
                </Box>
              )}
            </Box>

            {/* Assignment information */}
            <Stack spacing={1.5} sx={{ flexGrow: 1 }}>
              {/* Assigned Task ID */}
              {subchat.assignedTaskId && (
                <Box display="flex" alignItems="center" gap={1.5}>
                  <Box
                    sx={{
                      p: 0.5,
                      borderRadius: 1,
                      backgroundColor: 'primary.50',
                      color: 'primary.main',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <TaskIcon fontSize="small" />
                  </Box>
                  <Box>
                    <Typography variant="caption" color="text.secondary" display="block">
                      Assigned Task
                    </Typography>
                    <Typography variant="body2" sx={{ fontWeight: 500, fontFamily: 'monospace' }}>
                      {subchat.assignedTaskId.substring(0, 8)}...
                    </Typography>
                  </Box>
                </Box>
              )}

              {/* Assigned TODO ID */}
              {subchat.assignedTodoId && (
                <Box display="flex" alignItems="center" gap={1.5}>
                  <Box
                    sx={{
                      p: 0.5,
                      borderRadius: 1,
                      backgroundColor: 'success.50',
                      color: 'success.main',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <TodoIcon fontSize="small" />
                  </Box>
                  <Box>
                    <Typography variant="caption" color="text.secondary" display="block">
                      Assigned TODO
                    </Typography>
                    <Typography variant="body2" sx={{ fontWeight: 500, fontFamily: 'monospace' }}>
                      {subchat.assignedTodoId.substring(0, 8)}...
                    </Typography>
                  </Box>
                </Box>
              )}
            </Stack>

            {/* Footer with timestamp, delete button, and expand button */}
            <Box display="flex" alignItems="center" justifyContent="space-between" pt={1}>
              <Box display="flex" alignItems="center" gap={1}>
                <TimeIcon fontSize="small" color="action" />
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }}>
                  {formatDate(subchat.createdAt)}
                </Typography>
              </Box>
              <Box display="flex" gap={1}>
                {onDelete && (
                  <Tooltip title="Delete subchat">
                    <IconButton
                      onClick={handleDelete}
                      size="small"
                      sx={{
                        backgroundColor: 'error.50',
                        color: 'error.main',
                        '&:hover': {
                          backgroundColor: 'error.100',
                          color: 'error.dark',
                        },
                      }}
                      aria-label="Delete subchat"
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {onToggleDetails && (
                  <Tooltip title={isExpanded ? 'Hide details' : 'Show details'}>
                    <IconButton
                      onClick={handleToggleDetails}
                      size="small"
                      sx={{
                        transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                        transition: 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                        backgroundColor: 'action.hover',
                        '&:hover': {
                          backgroundColor: 'primary.50',
                          color: 'primary.main',
                        },
                      }}
                      aria-label={isExpanded ? 'Hide details' : 'Show details'}
                    >
                      <ExpandMoreIcon />
                    </IconButton>
                  </Tooltip>
                )}
              </Box>
            </Box>
          </Stack>
        </CardContent>
      </CardActionArea>
    </Card>
  );
};

export default SubchatCard;