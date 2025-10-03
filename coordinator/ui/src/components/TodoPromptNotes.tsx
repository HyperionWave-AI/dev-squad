import { useState } from 'react';
import {
  Box,
  IconButton,
  TextField,
  Button,
  Collapse,
  Typography,
  Chip,
} from '@mui/material';
import {
  Edit,
  Save,
  Cancel,
  ExpandMore,
  ExpandLess,
  Notes as NotesIcon,
} from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import type { TodoItem } from '../types/coordinator';
import { mcpClient } from '../services/mcpClient';

interface TodoPromptNotesProps {
  todo: TodoItem;
  agentTaskId: string;
  isTaskPending: boolean;
  onUpdate: () => void;
}

const MAX_PREVIEW_LENGTH = 50;

export function TodoPromptNotes({
  todo,
  agentTaskId,
  isTaskPending,
  onUpdate,
}: TodoPromptNotesProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [draftNotes, setDraftNotes] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const hasNotes = !!todo.humanPromptNotes;
  const preview = hasNotes && todo.humanPromptNotes!.length > MAX_PREVIEW_LENGTH
    ? `${todo.humanPromptNotes!.substring(0, MAX_PREVIEW_LENGTH)}...`
    : todo.humanPromptNotes || '';

  const handleEdit = () => {
    setDraftNotes(todo.humanPromptNotes || '');
    setIsEditing(true);
    setIsExpanded(true);
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      if (todo.humanPromptNotes) {
        await mcpClient.updateTodoPromptNotes(agentTaskId, todo.id, draftNotes);
      } else {
        await mcpClient.addTodoPromptNotes(agentTaskId, todo.id, draftNotes);
      }
      setIsEditing(false);
      onUpdate();
    } catch (error) {
      console.error('Failed to save TODO notes:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    setDraftNotes('');
    setIsEditing(false);
  };

  if (!hasNotes && !isTaskPending) {
    return null;
  }

  return (
    <Box sx={{ ml: 4, mt: 1, mb: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {hasNotes && (
          <Chip
            icon={<NotesIcon />}
            label="📝"
            size="small"
            variant="outlined"
            color="primary"
          />
        )}
        {!isExpanded && hasNotes && (
          <Typography variant="caption" color="text.secondary">
            {preview}
          </Typography>
        )}
        <IconButton
          size="small"
          onClick={() => setIsExpanded(!isExpanded)}
          aria-label={isExpanded ? 'Collapse notes' : 'Expand notes'}
        >
          {isExpanded ? <ExpandLess /> : <ExpandMore />}
        </IconButton>
      </Box>

      <Collapse in={isExpanded}>
        <Box sx={{ mt: 1, p: 1, bgcolor: 'background.paper', borderRadius: 1 }}>
          {isEditing ? (
            <>
              <TextField
                fullWidth
                multiline
                rows={3}
                value={draftNotes}
                onChange={(e) => setDraftNotes(e.target.value)}
                placeholder="Add guidance notes for this TODO..."
                disabled={isSaving}
                size="small"
              />
              <Box sx={{ mt: 1, display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                <Button
                  startIcon={<Cancel />}
                  onClick={handleCancel}
                  disabled={isSaving}
                  size="small"
                >
                  Cancel
                </Button>
                <Button
                  startIcon={<Save />}
                  onClick={handleSave}
                  disabled={isSaving}
                  variant="contained"
                  size="small"
                >
                  {isSaving ? 'Saving...' : 'Save'}
                </Button>
              </Box>
            </>
          ) : (
            <>
              {hasNotes && (
                <Box sx={{ mb: 1 }}>
                  <ReactMarkdown>{todo.humanPromptNotes}</ReactMarkdown>
                </Box>
              )}
              {isTaskPending && (
                <Button
                  startIcon={<Edit />}
                  onClick={handleEdit}
                  size="small"
                  variant="outlined"
                >
                  {hasNotes ? 'Edit' : 'Add Notes'}
                </Button>
              )}
            </>
          )}
        </Box>
      </Collapse>
    </Box>
  );
}
