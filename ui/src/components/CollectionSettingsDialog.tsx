import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Autocomplete,
  Chip,
  Box,
  Alert,
  CircularProgress,
} from '@mui/material';
import {
  Assessment as AssessmentIcon,
  Compress as CompressIcon,
} from '@mui/icons-material';
import { knowledgeService, type CollectionInfo, type CollectionReviewResult } from '../services/knowledgeService';
import { CollectionReviewDialog } from './CollectionReviewDialog';

interface CollectionSettingsDialogProps {
  collection: CollectionInfo | null;
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}

export const CollectionSettingsDialog: React.FC<CollectionSettingsDialogProps> = ({
  collection,
  open,
  onClose,
  onSaved,
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [category, setCategory] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reviewing, setReviewing] = useState(false);
  const [reviewResult, setReviewResult] = useState<CollectionReviewResult | null>(null);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);

  // Initialize form when collection changes
  useEffect(() => {
    if (collection) {
      setName(collection.name);
      setDescription(collection.description || '');
      setTags(collection.tags || []);
      setCategory(collection.category);
    }
  }, [collection]);

  const handleSave = async () => {
    if (!collection) return;

    if (!name.trim()) {
      setError('Collection name cannot be empty');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      // First, update metadata
      await knowledgeService.updateCollectionMetadata(collection.name, {
        description,
        tags,
        category,
      });

      // Then, rename if name changed
      if (name !== collection.name) {
        await knowledgeService.renameCollection(collection.name, name);
      }

      setSaving(false);
      onSaved();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save collection settings');
      setSaving(false);
    }
  };

  const handleCancel = () => {
    setError(null);
    onClose();
  };

  const handleReviewCollection = async () => {
    if (!collection) return;

    setReviewing(true);
    setError(null);

    try {
      const result = await knowledgeService.reviewCollection(collection.name, 70, 100);
      setReviewResult(result);
      setReviewDialogOpen(true);
      setReviewing(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to review collection');
      setReviewing(false);
    }
  };

  const handleCompactCollection = async () => {
    if (!collection) return;

    if (!window.confirm(`Compact all low-scoring entries in "${collection.name}"? This action cannot be undone.`)) {
      return;
    }

    setError('Bulk compaction not yet implemented. Please use individual entry compaction.');
  };

  if (!collection) return null;

  return (
    <Dialog
      open={open}
      onClose={handleCancel}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>Collection Settings</DialogTitle>

      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {/* Collection Name */}
          <TextField
            label="Collection Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            fullWidth
            disabled={saving}
            helperText="Renaming will update all entries in this collection"
          />

          {/* Category */}
          <Autocomplete
            value={category}
            onChange={(_event, newValue) => setCategory(newValue || '')}
            options={['Task', 'Tech', 'UI', 'Ops', 'Other']}
            renderInput={(params) => (
              <TextField
                {...params}
                label="Category"
                helperText="Choose a category for organization"
              />
            )}
            disabled={saving}
            freeSolo
          />

          {/* Description */}
          <TextField
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            fullWidth
            multiline
            rows={3}
            disabled={saving}
            helperText="Brief description of what this collection contains"
          />

          {/* Tags */}
          <Autocomplete
            multiple
            value={tags}
            onChange={(_event, newValue) => setTags(newValue)}
            options={[]}
            freeSolo
            renderTags={(value: readonly string[], getTagProps) =>
              value.map((option: string, index: number) => (
                <Chip
                  variant="outlined"
                  label={option}
                  {...getTagProps({ index })}
                  key={option}
                />
              ))
            }
            renderInput={(params) => (
              <TextField
                {...params}
                label="Tags"
                placeholder="Type and press Enter to add tags"
                helperText="Add keywords to help find this collection"
              />
            )}
            disabled={saving}
          />

          {/* Entry count (read-only) */}
          <TextField
            label="Entry Count"
            value={collection.count}
            fullWidth
            disabled
            helperText="Number of entries in this collection"
          />
        </Box>
      </DialogContent>

      <DialogActions>
        <Box sx={{ flex: 1, display: 'flex', gap: 1 }}>
          <Button
            onClick={handleReviewCollection}
            disabled={saving || reviewing}
            startIcon={reviewing ? <CircularProgress size={20} /> : <AssessmentIcon />}
          >
            {reviewing ? 'Reviewing...' : 'Review Collection'}
          </Button>
          <Button
            onClick={handleCompactCollection}
            disabled={saving || reviewing}
            startIcon={<CompressIcon />}
          >
            Compact Collection
          </Button>
        </Box>
        <Button onClick={handleCancel} disabled={saving}>
          Cancel
        </Button>
        <Button
          onClick={handleSave}
          variant="contained"
          color="primary"
          disabled={saving || !name.trim()}
          startIcon={saving ? <CircularProgress size={20} /> : null}
        >
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </DialogActions>

      {/* Collection Review Dialog */}
      <CollectionReviewDialog
        open={reviewDialogOpen}
        onClose={() => setReviewDialogOpen(false)}
        result={reviewResult}
      />
    </Dialog>
  );
};
