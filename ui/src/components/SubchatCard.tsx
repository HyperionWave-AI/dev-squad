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
} from '@mui/icons-material';
import type { Subchat } from '../services/subchatService';

interface SubchatCardProps {
  subchat: Subchat;
  onClick?: (subchatId: string) => void;
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
  isExpanded = false,
  onToggleDetails
}) => {
  const handleClick = () => {
    if (onClick) {
      onClick(subchat.id);
    }
  };

  const handleToggleDetails = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    if (onToggleDetails) {
      onToggleDetails(subchat.id);
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
        transition: 'all 0.2s ease-in-out',
        '&:hover': {
          boxShadow: 3,
          transform: 'translateY(-2px)',
        },
        // Highlight when expanded
        ...(isExpanded && {
          borderColor: 'primary.main',
          borderWidth: 2,
          boxShadow: 4,
        }),
        // Add subtle pulse animation for running subchats
        ...(isRunning && !isExpanded && {
          animation: 'pulse 2s ease-in-out infinite',
          '@keyframes pulse': {
            '0%, 100%': { borderColor: 'divider' },
            '50%': { borderColor: 'primary.main', borderWidth: 2 },
          },
        }),
      }}
    >
      <CardActionArea onClick={handleClick} disabled={!onClick}>
        <CardContent>
          <Stack spacing={2}>
            {/* Subagent name with category badge and expand button */}
            <Box display="flex" alignItems="center" gap={1} flexWrap="wrap">
              <AgentIcon fontSize="small" color="action" />
              <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
                {subchat.subagentName}
              </Typography>
              <Chip
                label={category}
                color={categoryColor}
                size="small"
              />
              {onToggleDetails && (
                <IconButton
                  onClick={handleToggleDetails}
                  size="small"
                  sx={{
                    transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                    transition: 'transform 0.3s',
                  }}
                  aria-label={isExpanded ? 'Hide details' : 'Show details'}
                >
                  <ExpandMoreIcon />
                </IconButton>
              )}
            </Box>

            {/* Status badge with icon and progress indicator */}
            <Box>
              <Box display="flex" alignItems="center" gap={1} mb={isRunning ? 1 : 0}>
                <Chip
                  icon={<StatusIcon />}
                  label={subchat.status}
                  color={statusColor}
                  size="small"
                  sx={{ textTransform: 'capitalize' }}
                />
                {isRunning && (
                  <CircularProgress size={16} thickness={5} />
                )}
              </Box>
              {/* Progress bar for running subchats */}
              {isRunning && (
                <LinearProgress color="primary" />
              )}
            </Box>

            {/* Assigned Task ID */}
            {subchat.assignedTaskId && (
              <Box display="flex" alignItems="center" gap={1}>
                <TaskIcon fontSize="small" color="primary" />
                <Typography variant="body2" color="text.secondary">
                  Task: {subchat.assignedTaskId.substring(0, 8)}...
                </Typography>
              </Box>
            )}

            {/* Assigned TODO ID */}
            {subchat.assignedTodoId && (
              <Box display="flex" alignItems="center" gap={1}>
                <TodoIcon fontSize="small" color="success" />
                <Typography variant="body2" color="text.secondary">
                  TODO: {subchat.assignedTodoId.substring(0, 8)}...
                </Typography>
              </Box>
            )}

            {/* Creation timestamp */}
            <Box display="flex" alignItems="center" gap={1}>
              <TimeIcon fontSize="small" color="action" />
              <Typography variant="caption" color="text.secondary">
                Created {formatDate(subchat.createdAt)}
              </Typography>
            </Box>
          </Stack>
        </CardContent>
      </CardActionArea>
    </Card>
  );
};

export default SubchatCard;
