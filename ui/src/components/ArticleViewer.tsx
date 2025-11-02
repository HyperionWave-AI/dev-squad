import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Divider,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  Assessment as AssessmentIcon,
  Compress as CompressIcon,
} from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { knowledgeService, type KnowledgeEntry, type ReviewResult, type CompactionResult } from '../services/knowledgeService';
import { ReviewResultDialog } from './ReviewResultDialog';
import { CompactionDialog } from './CompactionDialog';

interface ArticleViewerProps {
  entry: KnowledgeEntry | null;
  onEdit: () => void;
  onDelete: () => void;
}

export const ArticleViewer: React.FC<ArticleViewerProps> = ({
  entry,
  onEdit,
  onDelete,
}) => {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [reviewResult, setReviewResult] = useState<ReviewResult | null>(null);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [compacting, setCompacting] = useState(false);
  const [compactionResult, setCompactionResult] = useState<CompactionResult | null>(null);
  const [compactionDialogOpen, setCompactionDialogOpen] = useState(false);

  if (!entry) {
    return (
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          p: 4,
        }}
      >
        <Typography variant="body1" color="text.secondary">
          Select an entry to view
        </Typography>
      </Box>
    );
  }

  const handleDeleteClick = () => {
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    setDeleteDialogOpen(false);
    onDelete();
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
  };

  const handleReview = async () => {
    if (!entry) return;

    setReviewing(true);
    try {
      const result = await knowledgeService.reviewEntry(entry.id, 'full', false);
      setReviewResult(result);
      setReviewDialogOpen(true);
    } catch (err) {
      console.error('Failed to review entry:', err);
    } finally {
      setReviewing(false);
    }
  };

  const handleCompact = async () => {
    if (!entry) return;

    setCompacting(true);
    try {
      const result = await knowledgeService.compactEntry(entry.id, 500, true);
      setCompactionResult(result);
      setCompactionDialogOpen(true);
    } catch (err) {
      console.error('Failed to compact entry:', err);
    } finally {
      setCompacting(false);
    }
  };

  const handleApplyCompaction = async () => {
    if (!entry) return;

    try {
      await knowledgeService.compactEntry(entry.id, 500, false);
      setCompactionDialogOpen(false);
      // Refresh the entry by triggering onEdit or similar
      window.location.reload(); // Temporary: ideally trigger a refresh callback
    } catch (err) {
      console.error('Failed to apply compaction:', err);
    }
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return '';
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return '';
    }
  };

  return (
    <Box sx={{ height: '100%', overflow: 'auto', p: 3 }}>
      <Paper elevation={2} sx={{ p: 3 }}>
        {/* Header with actions */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <Chip label={entry.collection} color="primary" size="small" />
            {entry.createdAt && (
              <Typography variant="caption" color="text.secondary">
                Created: {formatDate(entry.createdAt)}
              </Typography>
            )}
          </Box>
          <Box>
            <Tooltip title="Review Entry">
              <span>
                <IconButton onClick={handleReview} color="info" disabled={reviewing || compacting}>
                  {reviewing ? <CircularProgress size={20} /> : <AssessmentIcon />}
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title="Compact Entry">
              <span>
                <IconButton onClick={handleCompact} color="secondary" disabled={reviewing || compacting}>
                  {compacting ? <CircularProgress size={20} /> : <CompressIcon />}
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title="Edit">
              <IconButton onClick={onEdit} color="primary">
                <EditIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title="Delete">
              <IconButton onClick={handleDeleteClick} color="error">
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Divider sx={{ mb: 3 }} />

        {/* Markdown content */}
        <Box
          sx={{
            '& h1': { fontSize: '2rem', fontWeight: 600, mb: 2, mt: 3 },
            '& h2': { fontSize: '1.5rem', fontWeight: 600, mb: 2, mt: 2 },
            '& h3': { fontSize: '1.25rem', fontWeight: 600, mb: 1.5, mt: 2 },
            '& h4': { fontSize: '1.1rem', fontWeight: 600, mb: 1, mt: 1.5 },
            '& p': { mb: 2, lineHeight: 1.7 },
            '& code': {
              bgcolor: 'grey.100',
              color: 'error.dark',
              px: 0.5,
              py: 0.25,
              borderRadius: 0.5,
              fontFamily: 'monospace',
              fontSize: '0.875rem',
            },
            '& pre': {
              bgcolor: 'grey.100',
              p: 2,
              borderRadius: 1,
              overflow: 'auto',
              mb: 2,
            },
            '& pre code': {
              bgcolor: 'transparent',
              color: 'inherit',
              p: 0,
            },
            '& ul, & ol': { mb: 2, pl: 3 },
            '& li': { mb: 0.5 },
            '& blockquote': {
              borderLeft: 4,
              borderColor: 'primary.main',
              pl: 2,
              ml: 0,
              fontStyle: 'italic',
              color: 'text.secondary',
            },
            '& table': {
              borderCollapse: 'collapse',
              width: '100%',
              mb: 2,
            },
            '& th, & td': {
              border: 1,
              borderColor: 'divider',
              p: 1,
              textAlign: 'left',
            },
            '& th': {
              bgcolor: 'grey.100',
              fontWeight: 600,
            },
          }}
        >
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {entry.text}
          </ReactMarkdown>
        </Box>

        {/* Metadata section */}
        {entry.metadata && Object.keys(entry.metadata).length > 0 && (
          <Accordion sx={{ mt: 3 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle2">Metadata</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                {Object.entries(entry.metadata).map(([key, value]) => (
                  <Chip
                    key={key}
                    label={`${key}: ${JSON.stringify(value)}`}
                    variant="outlined"
                    size="small"
                  />
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>
        )}
      </Paper>

      {/* Delete confirmation dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={handleDeleteCancel}
      >
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete this entry? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteCancel} color="primary">
            Cancel
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Review Result Dialog */}
      <ReviewResultDialog
        open={reviewDialogOpen}
        onClose={() => setReviewDialogOpen(false)}
        result={reviewResult}
      />

      {/* Compaction Dialog */}
      <CompactionDialog
        open={compactionDialogOpen}
        onClose={() => setCompactionDialogOpen(false)}
        result={compactionResult}
        onApprove={handleApplyCompaction}
        loading={compacting}
      />
    </Box>
  );
};
