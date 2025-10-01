import { Card, CardContent, Typography, Chip, Box, IconButton } from '@mui/material';
import { Draggable } from '@hello-pangea/dnd';
import {
  AccessTime,
  Person,
  CheckCircle,
  Schedule,
  Block,
  MoreVert
} from '@mui/icons-material';
import type { HumanTask, Priority, TaskStatus } from '../types/coordinator';

interface KanbanTaskCardProps {
  task: HumanTask;
  index: number;
  onClick?: (task: HumanTask) => void;
}

const getPriorityColor = (priority: Priority): 'default' | 'primary' | 'warning' | 'error' => {
  switch (priority) {
    case 'urgent':
      return 'error';
    case 'high':
      return 'warning';
    case 'medium':
      return 'primary';
    case 'low':
    default:
      return 'default';
  }
};

const getStatusIcon = (status: TaskStatus) => {
  switch (status) {
    case 'completed':
      return <CheckCircle fontSize="small" />;
    case 'in_progress':
      return <Schedule fontSize="small" />;
    case 'blocked':
      return <Block fontSize="small" />;
    case 'pending':
    default:
      return <AccessTime fontSize="small" />;
  }
};

const getStatusColor = (status: TaskStatus): string => {
  switch (status) {
    case 'completed':
      return '#16a34a'; // Green
    case 'in_progress':
      return '#2563eb'; // Blue
    case 'blocked':
      return '#dc2626'; // Red
    case 'pending':
    default:
      return '#64748b'; // Gray
  }
};

export function KanbanTaskCard({ task, index, onClick }: KanbanTaskCardProps) {
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now.getTime() - date.getTime());
    const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <Draggable draggableId={task.id} index={index}>
      {(provided, snapshot) => (
        <Card
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          onClick={() => onClick?.(task)}
          sx={{
            mb: 1.5,
            cursor: 'pointer',
            pointerEvents: 'auto',
            backgroundColor: snapshot.isDragging ? '#f1f5f9' : 'white',
            transform: snapshot.isDragging ? 'rotate(2deg)' : 'none',
            boxShadow: snapshot.isDragging
              ? '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)'
              : undefined,
            transition: 'all 0.2s ease',
            '&:hover': {
              transform: 'translateY(-2px)',
              boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
            },
          }}
        >
          <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
            {/* Header with Priority and Menu */}
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
              <Chip
                label={task.priority.toUpperCase()}
                size="small"
                color={getPriorityColor(task.priority)}
                sx={{ height: 20, fontSize: '0.65rem', fontWeight: 600 }}
              />
              <IconButton size="small" sx={{ mt: -0.5, mr: -0.5 }}>
                <MoreVert fontSize="small" />
              </IconButton>
            </Box>

            {/* Task Title */}
            <Typography
              variant="h6"
              sx={{
                fontSize: '0.875rem',
                fontWeight: 600,
                mb: 1,
                color: 'text.primary',
                lineHeight: 1.4,
              }}
            >
              {task.title}
            </Typography>

            {/* Task Description */}
            {task.description && (
              <Typography
                variant="body2"
                color="text.secondary"
                sx={{
                  mb: 1.5,
                  fontSize: '0.75rem',
                  lineHeight: 1.5,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical',
                }}
              >
                {task.description}
              </Typography>
            )}

            {/* Tags */}
            {task.tags && task.tags.length > 0 && (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 1.5 }}>
                {task.tags.map((tag, idx) => (
                  <Chip
                    key={idx}
                    label={tag}
                    size="small"
                    variant="outlined"
                    sx={{
                      height: 18,
                      fontSize: '0.625rem',
                      '& .MuiChip-label': { px: 1 }
                    }}
                  />
                ))}
              </Box>
            )}

            {/* Footer with Status and Date */}
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Box sx={{ color: getStatusColor(task.status), display: 'flex', alignItems: 'center' }}>
                  {getStatusIcon(task.status)}
                </Box>
                <Typography
                  variant="body2"
                  sx={{
                    fontSize: '0.7rem',
                    color: 'text.secondary',
                    fontWeight: 500
                  }}
                >
                  {task.status.replace('_', ' ')}
                </Typography>
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <AccessTime sx={{ fontSize: 12, color: 'text.secondary' }} />
                <Typography
                  variant="body2"
                  sx={{
                    fontSize: '0.7rem',
                    color: 'text.secondary'
                  }}
                >
                  {formatDate(task.createdAt)}
                </Typography>
              </Box>
            </Box>

            {/* Assigned User (if available) */}
            {task.createdBy && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mt: 1, pt: 1, borderTop: '1px solid #f1f5f9' }}>
                <Person sx={{ fontSize: 14, color: 'text.secondary' }} />
                <Typography
                  variant="body2"
                  sx={{
                    fontSize: '0.7rem',
                    color: 'text.secondary'
                  }}
                >
                  {task.createdBy}
                </Typography>
              </Box>
            )}
          </CardContent>
        </Card>
      )}
    </Draggable>
  );
}