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
import type { FlattenedTask, Priority, TaskStatus } from '../types/coordinator';

interface KanbanTaskCardProps {
  task: FlattenedTask;
  index: number;
  onClick?: (task: FlattenedTask) => void;
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

  // Visual styling based on task type
  const getTaskTypeColor = () => {
    switch (task.taskType) {
      case 'human': return '#3b82f6'; // Blue
      case 'agent': return '#8b5cf6'; // Purple
      case 'todo': return '#10b981'; // Green
      default: return '#64748b'; // Gray
    }
  };

  const getTaskTypeLabel = () => {
    switch (task.taskType) {
      case 'human': return '👤 Human';
      case 'agent': return '🤖 Agent';
      case 'todo': return '📋 Todo';
      default: return '';
    }
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
            borderLeft: `4px solid ${getTaskTypeColor()}`,
            '&:hover': {
              transform: 'translateY(-2px)',
              boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
            },
          }}
        >
          <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
            {/* Header with Task Type and Priority */}
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1, gap: 1 }}>
              <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                <Chip
                  label={getTaskTypeLabel()}
                  size="small"
                  sx={{
                    height: 20,
                    fontSize: '0.65rem',
                    fontWeight: 600,
                    backgroundColor: getTaskTypeColor(),
                    color: 'white',
                  }}
                />
                {task.priority && (
                  <Chip
                    label={task.priority.toUpperCase()}
                    size="small"
                    color={getPriorityColor(task.priority)}
                    sx={{ height: 20, fontSize: '0.65rem', fontWeight: 600 }}
                  />
                )}
              </Box>
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

            {/* Context Information */}
            {task.contextSummary && (
              <Box sx={{ mb: 1, p: 1, backgroundColor: '#eff6ff', border: '1px solid #bfdbfe', borderRadius: 1 }}>
                <Typography sx={{ fontSize: '0.65rem', fontWeight: 600, color: '#1e3a8a', mb: 0.5 }}>
                  📋 Context
                </Typography>
                <Typography sx={{ fontSize: '0.65rem', color: '#1e40af' }}>
                  {task.contextSummary}
                </Typography>
              </Box>
            )}

            {task.filesModified && task.filesModified.length > 0 && (
              <Box sx={{ mb: 1, p: 1, backgroundColor: '#faf5ff', border: '1px solid #e9d5ff', borderRadius: 1 }}>
                <Typography sx={{ fontSize: '0.65rem', fontWeight: 600, color: '#581c87', mb: 0.5 }}>
                  📁 Files ({task.filesModified.length})
                </Typography>
                <Box component="ul" sx={{ fontSize: '0.65rem', color: '#6b21a8', pl: 2, m: 0 }}>
                  {task.filesModified.slice(0, 3).map((file, idx) => (
                    <li key={idx} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {file}
                    </li>
                  ))}
                  {task.filesModified.length > 3 && (
                    <li style={{ fontStyle: 'italic' }}>+ {task.filesModified.length - 3} more</li>
                  )}
                </Box>
              </Box>
            )}

            {task.qdrantCollections && task.qdrantCollections.length > 0 && (
              <Box sx={{ mb: 1, p: 1, backgroundColor: '#f0fdf4', border: '1px solid #bbf7d0', borderRadius: 1 }}>
                <Typography sx={{ fontSize: '0.65rem', fontWeight: 600, color: '#14532d', mb: 0.5 }}>
                  🔍 Knowledge
                </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {task.qdrantCollections.map((collection, idx) => (
                    <Chip
                      key={idx}
                      label={collection}
                      size="small"
                      sx={{
                        height: 16,
                        fontSize: '0.6rem',
                        backgroundColor: '#bbf7d0',
                        color: '#15803d',
                        '& .MuiChip-label': { px: 0.75 }
                      }}
                    />
                  ))}
                </Box>
              </Box>
            )}

            {task.priorWorkSummary && (
              <Box sx={{ mb: 1, p: 1, backgroundColor: '#fffbeb', border: '1px solid #fde68a', borderRadius: 1 }}>
                <Typography sx={{ fontSize: '0.65rem', fontWeight: 600, color: '#78350f', mb: 0.5 }}>
                  🔗 Prior Work
                </Typography>
                <Typography sx={{ fontSize: '0.65rem', color: '#92400e' }}>
                  {task.priorWorkSummary}
                </Typography>
              </Box>
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