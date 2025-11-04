import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Chip,
  Box,
  Alert,
  CircularProgress,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import CloseIcon from '@mui/icons-material/Close';
import { knowledgeApi } from '../../services/knowledgeApi';
import type { CreateCollectionRequest } from '../../types/knowledge';

interface CreateCollectionModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const CATEGORIES = ['Tech', 'Task', 'UI', 'Ops', 'Other'];

export const CreateCollectionModal: React.FC<CreateCollectionModalProps> = ({
  open,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<CreateCollectionRequest>({
    name: '',
    category: 'Other',
    description: '',
    tags: [],
  });
  const [tagInput, setTagInput] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const handleClose = () => {
    if (!loading) {
      setFormData({ name: '', category: 'Other', description: '', tags: [] });
      setTagInput('');
      setError(null);
      onClose();
    }
  };

  const handleAddTag = () => {
    const tag = tagInput.trim();
    if (tag && !formData.tags.includes(tag)) {
      setFormData({ ...formData, tags: [...formData.tags, tag] });
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((tag) => tag !== tagToRemove),
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.name.trim()) {
      setError('Collection name is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await knowledgeApi.createCollection(formData);
      onSuccess();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create collection');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Create New Collection</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {/* Collection Name */}
          <TextField
            label="Collection Name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            fullWidth
            required
            sx={{ mb: 2 }}
            disabled={loading}
            placeholder="e.g., Project Documentation"
          />

          {/* Category */}
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel id="category-label">Category</InputLabel>
            <Select
              labelId="category-label"
              value={formData.category}
              label="Category"
              onChange={(e) => setFormData({ ...formData, category: e.target.value })}
              disabled={loading}
            >
              {CATEGORIES.map((cat) => (
                <MenuItem key={cat} value={cat}>
                  {cat}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          {/* Description */}
          <TextField
            label="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            fullWidth
            multiline
            rows={3}
            sx={{ mb: 2 }}
            disabled={loading}
            placeholder="Describe what this collection contains..."
          />

          {/* Tags Input */}
          <Box sx={{ mb: 2 }}>
            <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <TextField
                label="Add Tags"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleAddTag();
                  }
                }}
                fullWidth
                disabled={loading}
                placeholder="Press Enter to add"
              />
              <Button
                onClick={handleAddTag}
                variant="outlined"
                disabled={loading || !tagInput.trim()}
                sx={{ minWidth: '80px' }}
              >
                Add
              </Button>
            </Box>

            {/* Display Tags */}
            {formData.tags.length > 0 && (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {formData.tags.map((tag) => (
                  <Chip
                    key={tag}
                    label={tag}
                    onDelete={() => handleRemoveTag(tag)}
                    disabled={loading}
                    size="small"
                  />
                ))}
              </Box>
            )}
          </Box>
        </DialogContent>

        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={handleClose} disabled={loading} startIcon={<CloseIcon />}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={loading || !formData.name.trim()}
            startIcon={loading ? <CircularProgress size={20} /> : <AddIcon />}
          >
            {loading ? 'Creating...' : 'Create Collection'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};
