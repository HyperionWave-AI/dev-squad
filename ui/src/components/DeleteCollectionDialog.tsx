import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  CircularProgress,
} from '@mui/material';
import { WarningAmber } from '@mui/icons-material';

interface DeleteCollectionDialogProps {
  open: boolean;
  collectionName: string;
  entryCount: number;
  onConfirm: () => Promise<void>;
  onCancel: () => void;
}

export const DeleteCollectionDialog: React.FC<DeleteCollectionDialogProps> = ({
  open,
  collectionName,
  entryCount,
  onConfirm,
  onCancel,
}) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setLoading(true);
    setError(null);
    try {
      await onConfirm();
      // onConfirm should close the dialog on success
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete collection');
      setLoading(false);
    }
  };

  const handleCancel = () => {
    if (!loading) {
      setError(null);
      onCancel();
    }
  };

  return (
    <Dialog
      open={open}
      onClose={handleCancel}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningAmber color="error" />
          <Typography variant="h6">Delete Collection</Typography>
        </Box>
      </DialogTitle>
      <DialogContent>
        <Typography variant="body1" gutterBottom>
          Are you sure you want to delete the collection <strong>{collectionName}</strong>?
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
          This will permanently delete:
        </Typography>
        <Box component="ul" sx={{ mt: 1, mb: 2 }}>
          <li>
            <Typography variant="body2" color="text.secondary">
              {entryCount} knowledge {entryCount === 1 ? 'entry' : 'entries'} from MongoDB
            </Typography>
          </li>
          <li>
            <Typography variant="body2" color="text.secondary">
              All vector embeddings from Qdrant
            </Typography>
          </li>
          <li>
            <Typography variant="body2" color="text.secondary">
              Collection metadata and settings
            </Typography>
          </li>
        </Box>
        <Typography variant="body2" color="error" sx={{ fontWeight: 'bold' }}>
          This action cannot be undone.
        </Typography>
        {error && (
          <Typography variant="body2" color="error" sx={{ mt: 2 }}>
            {error}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleCancel} disabled={loading}>
          Cancel
        </Button>
        <Button
          onClick={handleConfirm}
          color="error"
          variant="contained"
          disabled={loading}
          startIcon={loading ? <CircularProgress size={16} /> : null}
        >
          {loading ? 'Deleting...' : 'Delete Collection'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
