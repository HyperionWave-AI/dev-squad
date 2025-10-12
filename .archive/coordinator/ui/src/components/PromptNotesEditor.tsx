import { useState } from 'react';
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  TextField,
  Button,
  Box,
  Typography,
} from '@mui/material';
import {
  ExpandMore,
  Edit,
  Save,
  Cancel,
  Delete,
} from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';

interface PromptNotesEditorProps {
  notes?: string;
  notesAddedAt?: string;
  isEditable: boolean;
  onSave: (notes: string) => Promise<void>;
  onClear: () => Promise<void>;
  placeholder: string;
}

const MAX_CHARACTERS = 5000;
const WARNING_THRESHOLD = 4500;
const CRITICAL_THRESHOLD = 4900;

export function PromptNotesEditor({
  notes,
  notesAddedAt,
  isEditable,
  onSave,
  onClear,
  placeholder,
}: PromptNotesEditorProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [draftNotes, setDraftNotes] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const characterCount = draftNotes.length;
  const isOverLimit = characterCount > MAX_CHARACTERS;

  const getCounterColor = () => {
    if (characterCount > CRITICAL_THRESHOLD) return 'error.main';
    if (characterCount > WARNING_THRESHOLD) return 'warning.main';
    return 'success.main';
  };

  const handleEdit = () => {
    setDraftNotes(notes || '');
    setIsEditing(true);
  };

  const handleSave = async () => {
    if (isOverLimit) return;

    setIsSaving(true);
    try {
      await onSave(draftNotes);
      setIsEditing(false);
    } catch (error) {
      console.error('Failed to save notes:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    setDraftNotes('');
    setIsEditing(false);
  };

  const handleClear = async () => {
    if (!window.confirm('Are you sure you want to clear these notes?')) {
      return;
    }

    setIsSaving(true);
    try {
      await onClear();
      setIsEditing(false);
      setDraftNotes('');
    } catch (error) {
      console.error('Failed to clear notes:', error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Accordion defaultExpanded={!!notes}>
      <AccordionSummary expandIcon={<ExpandMore />}>
        <Typography variant="subtitle2">
          {notes ? 'Human Guidance Notes' : 'Add Human Guidance Notes'}
        </Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Box sx={{ width: '100%' }}>
          {isEditing ? (
            <>
              <TextField
                fullWidth
                multiline
                rows={6}
                value={draftNotes}
                onChange={(e) => setDraftNotes(e.target.value)}
                placeholder={placeholder}
                disabled={isSaving}
                error={isOverLimit}
                helperText={
                  isOverLimit
                    ? `Exceeds maximum length by ${characterCount - MAX_CHARACTERS} characters`
                    : undefined
                }
              />
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  mt: 1,
                }}
              >
                <Typography
                  variant="caption"
                  sx={{
                    color: getCounterColor(),
                    fontWeight: characterCount > WARNING_THRESHOLD ? 'bold' : 'normal',
                  }}
                >
                  {characterCount}/{MAX_CHARACTERS} characters
                </Typography>
                <Box>
                  <Button
                    startIcon={<Cancel />}
                    onClick={handleCancel}
                    disabled={isSaving}
                    size="small"
                    aria-label="Cancel editing notes"
                  >
                    Cancel
                  </Button>
                  <Button
                    startIcon={<Save />}
                    onClick={handleSave}
                    disabled={isSaving || isOverLimit}
                    variant="contained"
                    size="small"
                    sx={{ ml: 1 }}
                    aria-label="Save notes"
                  >
                    {isSaving ? 'Saving...' : 'Save'}
                  </Button>
                </Box>
              </Box>
            </>
          ) : (
            <>
              {notes ? (
                <Box>
                  <Box sx={{ mb: 2 }}>
                    <ReactMarkdown>{notes}</ReactMarkdown>
                  </Box>
                  {notesAddedAt && (
                    <Typography variant="caption" color="text.secondary">
                      Added: {new Date(notesAddedAt).toLocaleString()}
                    </Typography>
                  )}
                  {isEditable && (
                    <Box sx={{ mt: 2 }}>
                      <Button
                        startIcon={<Edit />}
                        onClick={handleEdit}
                        size="small"
                        aria-label="Edit notes"
                      >
                        Edit
                      </Button>
                      <Button
                        startIcon={<Delete />}
                        onClick={handleClear}
                        size="small"
                        color="error"
                        sx={{ ml: 1 }}
                        aria-label="Clear notes"
                      >
                        Clear
                      </Button>
                    </Box>
                  )}
                </Box>
              ) : (
                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    No notes added yet.
                  </Typography>
                  {isEditable && (
                    <Button
                      startIcon={<Edit />}
                      onClick={handleEdit}
                      variant="outlined"
                      size="small"
                      aria-label="Add notes"
                    >
                      Add Notes
                    </Button>
                  )}
                </Box>
              )}
              {!isEditable && notes && (
                <Typography variant="caption" color="warning.main" sx={{ mt: 1, display: 'block' }}>
                  Notes locked - task in progress
                </Typography>
              )}
            </>
          )}
        </Box>
      </AccordionDetails>
    </Accordion>
  );
}
