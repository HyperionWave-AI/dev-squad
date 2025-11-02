import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  LinearProgress,
  Chip,
  List,
  ListItem,
  ListItemText,
  Divider,
  Alert,
} from '@mui/material';
import {
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import type { ReviewResult } from '../services/knowledgeService';

interface ReviewResultDialogProps {
  open: boolean;
  onClose: () => void;
  result: ReviewResult | null;
}

const getHealthColor = (score: number): string => {
  if (score >= 90) return '#4caf50'; // green
  if (score >= 70) return '#ffeb3b'; // yellow
  if (score >= 40) return '#ff9800'; // orange
  return '#f44336'; // red
};

const getHealthLabel = (score: number): string => {
  if (score >= 90) return 'Excellent';
  if (score >= 70) return 'Good';
  if (score >= 40) return 'Fair';
  return 'Poor';
};

export const ReviewResultDialog: React.FC<ReviewResultDialogProps> = ({
  open,
  onClose,
  result,
}) => {
  if (!result) {
    return null;
  }

  const { scores, verification, actions } = result;
  const healthColor = getHealthColor(scores.health);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Typography variant="h6">Entry Review Results</Typography>
          <Chip
            label={`${getHealthLabel(scores.health)} (${Math.round(scores.health)})`}
            sx={{
              backgroundColor: healthColor,
              color: scores.health >= 70 ? '#000' : '#fff',
              fontWeight: 'bold',
            }}
          />
        </Box>
      </DialogTitle>
      <DialogContent>
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Overall Health Score
          </Typography>
          <Box display="flex" alignItems="center" gap={2}>
            <Box flex={1}>
              <LinearProgress
                variant="determinate"
                value={scores.health}
                sx={{
                  height: 10,
                  borderRadius: 5,
                  backgroundColor: '#e0e0e0',
                  '& .MuiLinearProgress-bar': {
                    backgroundColor: healthColor,
                  },
                }}
              />
            </Box>
            <Typography variant="body2" sx={{ minWidth: 40 }}>
              {Math.round(scores.health)}%
            </Typography>
          </Box>
        </Box>

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" gutterBottom>
          Component Scores
        </Typography>
        <Box sx={{ mb: 3 }}>
          {[
            { label: 'Alignment', value: scores.alignment },
            { label: 'Freshness', value: scores.freshness },
            { label: 'Verbosity', value: scores.verbosity },
            { label: 'Uniqueness', value: scores.uniqueness },
          ].map((item) => (
            <Box key={item.label} sx={{ mb: 1.5 }}>
              <Box display="flex" justifyContent="space-between" mb={0.5}>
                <Typography variant="body2">{item.label}</Typography>
                <Typography variant="body2" fontWeight="bold">
                  {Math.round(item.value)}%
                </Typography>
              </Box>
              <LinearProgress
                variant="determinate"
                value={item.value}
                sx={{
                  height: 6,
                  borderRadius: 3,
                  backgroundColor: '#e0e0e0',
                  '& .MuiLinearProgress-bar': {
                    backgroundColor: getHealthColor(item.value),
                  },
                }}
              />
            </Box>
          ))}
        </Box>

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" gutterBottom>
          Reference Verification
        </Typography>
        <Box sx={{ mb: 2 }}>
          <Box display="flex" gap={2} mb={1}>
            <Chip
              icon={<CheckCircleIcon />}
              label={`${verification.validReferences} Valid`}
              color="success"
              size="small"
            />
            <Chip
              icon={<ErrorIcon />}
              label={`${verification.brokenReferences.length} Broken`}
              color="error"
              size="small"
            />
          </Box>
        </Box>

        {verification.brokenReferences.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Alert severity="warning" sx={{ mb: 1 }}>
              Found {verification.brokenReferences.length} broken reference(s)
            </Alert>
            <List dense>
              {verification.brokenReferences.map((ref, index) => (
                <ListItem key={index} sx={{ px: 0 }}>
                  <ListItemText
                    primary={
                      <Typography variant="body2" component="span">
                        <strong>{ref.type}:</strong> {ref.value}
                      </Typography>
                    }
                    secondary={
                      <Typography variant="caption" color="error">
                        {ref.error}
                      </Typography>
                    }
                  />
                </ListItem>
              ))}
            </List>
          </Box>
        )}

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" gutterBottom>
          Suggested Actions
        </Typography>
        {actions.length > 0 ? (
          <List dense>
            {actions.map((action, index) => (
              <ListItem key={index} sx={{ px: 0 }}>
                <Box display="flex" alignItems="center" gap={1} width="100%">
                  {action.applied ? (
                    <CheckCircleIcon color="success" fontSize="small" />
                  ) : (
                    <WarningIcon color="warning" fontSize="small" />
                  )}
                  <ListItemText
                    primary={
                      <Typography variant="body2">
                        <strong>{action.type}</strong>: {action.description}
                      </Typography>
                    }
                  />
                  {action.applied && (
                    <Chip label="Applied" color="success" size="small" />
                  )}
                </Box>
              </ListItem>
            ))}
          </List>
        ) : (
          <Typography variant="body2" color="text.secondary">
            No actions suggested
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} variant="contained">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};
