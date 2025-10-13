import { useState } from 'react';
import type { AgentTask } from '../types/coordinator';

interface UsePromptNotesResult {
  isEditing: boolean;
  draftNotes: string;
  setDraftNotes: (notes: string) => void;
  handleEdit: () => void;
  handleSave: () => Promise<void>;
  handleCancel: () => void;
  handleClear: () => Promise<void>;
  isSaving: boolean;
  error: string | null;
  characterCount: number;
  isOverLimit: boolean;
}

interface UsePromptNotesParams {
  task: AgentTask;
  onUpdate: () => void;
}

const MAX_CHARACTERS = 5000;

export function usePromptNotes({ task, onUpdate }: UsePromptNotesParams): UsePromptNotesResult {
  const [isEditing, setIsEditing] = useState(false);
  const [draftNotes, setDraftNotes] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const characterCount = draftNotes.length;
  const isOverLimit = characterCount > MAX_CHARACTERS;

  const handleEdit = () => {
    setDraftNotes(task.humanPromptNotes || '');
    setIsEditing(true);
    setError(null);
  };

  const handleSave = async () => {
    if (isOverLimit) {
      setError('Notes exceed maximum length of 5000 characters');
      return;
    }

    setIsSaving(true);
    setError(null);

    // Store original state for rollback
    const originalNotes = task.humanPromptNotes;

    try {
      // Optimistic update - call onUpdate first to update UI
      onUpdate();

      // Then call MCP (this would be done by the parent component)
      // The actual MCP call is handled by the component using this hook

      setIsEditing(false);
    } catch (err) {
      // Rollback on error
      setError(err instanceof Error ? err.message : 'Failed to save notes');
      // Restore original notes
      if (originalNotes !== undefined) {
        setDraftNotes(originalNotes);
      }
      // Trigger UI rollback
      onUpdate();
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    setDraftNotes('');
    setIsEditing(false);
    setError(null);
  };

  const handleClear = async () => {
    setIsSaving(true);
    setError(null);

    // Store original state for rollback
    const originalNotes = task.humanPromptNotes;

    try {
      // Optimistic update
      onUpdate();

      setIsEditing(false);
      setDraftNotes('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear notes');
      // Restore original notes
      if (originalNotes !== undefined) {
        setDraftNotes(originalNotes);
      }
      onUpdate();
    } finally {
      setIsSaving(false);
    }
  };

  return {
    isEditing,
    draftNotes,
    setDraftNotes,
    handleEdit,
    handleSave,
    handleCancel,
    handleClear,
    isSaving,
    error,
    characterCount,
    isOverLimit
  };
}
