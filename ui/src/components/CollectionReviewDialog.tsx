import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Chip,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  LinearProgress,
  Checkbox,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Assessment as AssessmentIcon,
  Download as DownloadIcon,
  Compress as CompressIcon,
} from '@mui/icons-material';
import type { CollectionReviewResult } from '../services/knowledgeService';

interface CollectionReviewDialogProps {
  open: boolean;
  onClose: () => void;
  result: CollectionReviewResult | null;
  onCompactSelected?: (entryIds: string[]) => void;
  onViewEntry?: (entryId: string) => void;
}

const getHealthColor = (score: number): string => {
  if (score >= 90) return '#4caf50'; // green
  if (score >= 70) return '#ffeb3b'; // yellow
  if (score >= 40) return '#ff9800'; // orange
  return '#f44336'; // red
};

export const CollectionReviewDialog: React.FC<CollectionReviewDialogProps> = ({
  open,
  onClose,
  result,
  onCompactSelected,
  onViewEntry,
}) => {
  const [selectedEntries, setSelectedEntries] = useState<string[]>([]);

  if (!result) {
    return null;
  }

  const { summary, entries } = result;
  const avgHealthColor = getHealthColor(summary.averageHealth);

  const handleSelectAll = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      setSelectedEntries(entries.map((e) => e.entryId));
    } else {
      setSelectedEntries([]);
    }
  };

  const handleSelectEntry = (entryId: string) => {
    setSelectedEntries((prev) =>
      prev.includes(entryId)
        ? prev.filter((id) => id !== entryId)
        : [...prev, entryId]
    );
  };

  const handleCompactSelected = () => {
    if (onCompactSelected && selectedEntries.length > 0) {
      onCompactSelected(selectedEntries);
      setSelectedEntries([]);
    }
  };

  const handleExportReport = () => {
    const reportData = {
      collection: result.collection,
      summary,
      entries,
      generatedAt: new Date().toISOString(),
    };
    const blob = new Blob([JSON.stringify(reportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${result.collection}-review-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box display="flex" alignItems="center" gap={1}>
            <AssessmentIcon />
            <Typography variant="h6">Collection Review: {result.collection}</Typography>
          </Box>
          <Tooltip title="Export Report">
            <IconButton onClick={handleExportReport} size="small">
              <DownloadIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </DialogTitle>
      <DialogContent>
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Summary Statistics
          </Typography>
          <Box display="flex" gap={2} flexWrap="wrap" mb={2}>
            <Chip
              label={`Total Entries: ${summary.totalEntries}`}
              color="primary"
              variant="outlined"
            />
            <Chip
              label={`Reviewed: ${summary.entriesReviewed}`}
              color="secondary"
              variant="outlined"
            />
            <Chip
              label={`Low Score: ${summary.lowScoreCount}`}
              color="warning"
              variant="outlined"
            />
          </Box>
          <Box>
            <Typography variant="body2" gutterBottom>
              Average Health Score: {Math.round(summary.averageHealth)}%
            </Typography>
            <LinearProgress
              variant="determinate"
              value={summary.averageHealth}
              sx={{
                height: 8,
                borderRadius: 4,
                backgroundColor: '#e0e0e0',
                '& .MuiLinearProgress-bar': {
                  backgroundColor: avgHealthColor,
                },
              }}
            />
          </Box>
        </Box>

        {summary.lowScoreCount > 0 && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            Found {summary.lowScoreCount} entries with low health scores that may need attention.
          </Alert>
        )}

        {selectedEntries.length > 0 && (
          <Alert severity="info" sx={{ mb: 2 }}>
            {selectedEntries.length} entry(ies) selected
          </Alert>
        )}

        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox
                    indeterminate={
                      selectedEntries.length > 0 &&
                      selectedEntries.length < entries.length
                    }
                    checked={
                      entries.length > 0 &&
                      selectedEntries.length === entries.length
                    }
                    onChange={handleSelectAll}
                  />
                </TableCell>
                <TableCell>Entry ID</TableCell>
                <TableCell align="center">Health Score</TableCell>
                <TableCell>Issues</TableCell>
                <TableCell align="center">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {entries.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <Typography variant="body2" color="text.secondary" py={2}>
                      No entries to display
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                entries.map((entry) => {
                  const healthColor = getHealthColor(entry.healthScore);
                  return (
                    <TableRow
                      key={entry.entryId}
                      hover
                      sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
                    >
                      <TableCell padding="checkbox">
                        <Checkbox
                          checked={selectedEntries.includes(entry.entryId)}
                          onChange={() => handleSelectEntry(entry.entryId)}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography
                          variant="body2"
                          sx={{
                            fontFamily: 'monospace',
                            fontSize: '0.75rem',
                            maxWidth: 200,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                          }}
                        >
                          {entry.entryId}
                        </Typography>
                      </TableCell>
                      <TableCell align="center">
                        <Chip
                          label={Math.round(entry.healthScore)}
                          size="small"
                          sx={{
                            backgroundColor: healthColor,
                            color: entry.healthScore >= 70 ? '#000' : '#fff',
                            fontWeight: 'bold',
                            minWidth: 50,
                          }}
                        />
                      </TableCell>
                      <TableCell>
                        <Box display="flex" gap={0.5} flexWrap="wrap">
                          {entry.issues.length === 0 ? (
                            <Typography variant="caption" color="text.secondary">
                              No issues
                            </Typography>
                          ) : (
                            entry.issues.map((issue, idx) => (
                              <Chip
                                key={idx}
                                label={issue}
                                size="small"
                                variant="outlined"
                                sx={{ fontSize: '0.7rem' }}
                              />
                            ))
                          )}
                        </Box>
                      </TableCell>
                      <TableCell align="center">
                        <Button
                          size="small"
                          onClick={() => onViewEntry?.(entry.entryId)}
                          disabled={!onViewEntry}
                        >
                          View
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
        {onCompactSelected && (
          <Button
            onClick={handleCompactSelected}
            variant="contained"
            color="primary"
            disabled={selectedEntries.length === 0}
            startIcon={<CompressIcon />}
          >
            Compact Selected ({selectedEntries.length})
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
