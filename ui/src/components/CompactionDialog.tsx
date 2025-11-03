import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Paper,
  Chip,
  Alert,
  CircularProgress,
} from '@mui/material';
import {
  Compress as CompressIcon,
  CheckCircle as CheckCircleIcon,
} from '@mui/icons-material';
import type { CompactionResult } from '../services/knowledgeService';

interface CompactionDialogProps {
  open: boolean;
  onClose: () => void;
  result: CompactionResult | null;
  onApprove: () => void;
  loading?: boolean;
}

export const CompactionDialog: React.FC<CompactionDialogProps> = ({
  open,
  onClose,
  result,
  onApprove,
  loading = false,
}) => {
  const [applying, setApplying] = useState(false);

  if (!result) {
    return null;
  }

  const handleApprove = async () => {
    setApplying(true);
    try {
      await onApprove();
    } finally {
      setApplying(false);
    }
  };

  const { original, compacted, compressionRatio, preserved } = result;
  const wordReduction = original.wordCount - compacted.wordCount;
  const percentReduction = Math.round((wordReduction / original.wordCount) * 100);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        <Box display="flex" alignItems="center" gap={1}>
          <CompressIcon />
          <Typography variant="h6">Compaction Preview</Typography>
        </Box>
      </DialogTitle>
      <DialogContent>
        {loading ? (
          <Box display="flex" justifyContent="center" alignItems="center" py={4}>
            <CircularProgress />
          </Box>
        ) : (
          <>
            <Box sx={{ mb: 3 }}>
              <Alert severity="info" sx={{ mb: 2 }}>
                This is a preview. Click "Apply Compaction" to save the changes.
              </Alert>
              <Box display="flex" gap={2} flexWrap="wrap">
                <Chip
                  label={`Word Count: ${original.wordCount} → ${compacted.wordCount}`}
                  color="primary"
                  variant="outlined"
                />
                <Chip
                  label={`Reduction: ${wordReduction} words (${percentReduction}%)`}
                  color="success"
                  variant="outlined"
                />
                <Chip
                  label={`Compression Ratio: ${Math.round(compressionRatio * 100)}%`}
                  color="secondary"
                  variant="outlined"
                />
              </Box>
            </Box>

            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle2" gutterBottom>
                Preserved Elements
              </Typography>
              <Box display="flex" gap={2}>
                <Chip
                  icon={<CheckCircleIcon />}
                  label={`${preserved.filePaths} File Paths`}
                  color="success"
                  size="small"
                />
                <Chip
                  icon={<CheckCircleIcon />}
                  label={`${preserved.functionNames} Function Names`}
                  color="success"
                  size="small"
                />
              </Box>
            </Box>

            <Box sx={{ display: 'flex', gap: 2, flexDirection: { xs: 'column', md: 'row' } }}>
              <Box sx={{ flex: 1 }}>
                <Paper
                  elevation={2}
                  sx={{
                    p: 2,
                    height: '400px',
                    overflow: 'auto',
                    backgroundColor: '#f5f5f5',
                  }}
                >
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight="bold">
                      Original
                    </Typography>
                    <Chip label={`${original.wordCount} words`} size="small" />
                  </Box>
                  <Typography
                    variant="body2"
                    sx={{
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                      lineHeight: 1.6,
                    }}
                  >
                    {original.text}
                  </Typography>
                </Paper>
              </Box>

              <Box sx={{ flex: 1 }}>
                <Paper
                  elevation={2}
                  sx={{
                    p: 2,
                    height: '400px',
                    overflow: 'auto',
                    backgroundColor: '#e8f5e9',
                  }}
                >
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight="bold">
                      Compacted
                    </Typography>
                    <Chip label={`${compacted.wordCount} words`} size="small" color="success" />
                  </Box>
                  <Typography
                    variant="body2"
                    sx={{
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                      lineHeight: 1.6,
                    }}
                  >
                    {compacted.text}
                  </Typography>
                </Paper>
              </Box>
            </Box>

            {compressionRatio < 0.5 && (
              <Alert severity="warning" sx={{ mt: 2 }}>
                This compaction achieves significant compression ({Math.round(compressionRatio * 100)}%).
                Please review carefully to ensure important information is preserved.
              </Alert>
            )}
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={applying}>
          Cancel
        </Button>
        <Button
          onClick={handleApprove}
          variant="contained"
          color="primary"
          disabled={applying || loading}
          startIcon={applying ? <CircularProgress size={20} /> : <CompressIcon />}
        >
          {applying ? 'Applying...' : 'Apply Compaction'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
