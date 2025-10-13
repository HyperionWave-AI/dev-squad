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
} from '@mui/material';
import {
  AccountTree as TaskIcon,
  CheckCircle as TodoIcon,
  Person as AgentIcon,
  Schedule as TimeIcon,
} from '@mui/icons-material';
import type { Subchat } from '../services/subchatService';

interface SubchatCardProps {
  subchat: Subchat;
  onClick?: (subchatId: string) => void;
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

export const SubchatCard: React.FC<SubchatCardProps> = ({ subchat, onClick }) => {
  const handleClick = () => {
    if (onClick) {
      onClick(subchat.id);
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

  return (
    <Card
      variant="outlined"
      sx={{
        transition: 'all 0.2s ease-in-out',
        '&:hover': {
          boxShadow: 3,
          transform: 'translateY(-2px)',
        },
      }}
    >
      <CardActionArea onClick={handleClick} disabled={!onClick}>
        <CardContent>
          <Stack spacing={2}>
            {/* Subagent name with category badge */}
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
            </Box>

            {/* Status badge */}
            <Box>
              <Chip
                label={subchat.status}
                color={statusColor}
                size="small"
                sx={{ textTransform: 'capitalize' }}
              />
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
