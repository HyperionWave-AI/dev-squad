/**
 * SubchatList Component
 *
 * Displays list of child subchats for a parent chat with create button.
 * Fetches subchats on mount and handles loading/empty states.
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Typography,
  CircularProgress,
  Alert,
  Paper,
  Collapse,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { subchatService, type Subchat } from '../services/subchatService';
import SubchatCard from './SubchatCard';
import SubchatCreationDialog from './SubchatCreationDialog';
import SubchatDetailView from './SubchatDetailView';

interface SubchatListProps {
  parentChatId: string;
  onSubchatClick?: (subchatId: string) => void;
}

export const SubchatList: React.FC<SubchatListProps> = ({
  parentChatId,
  onSubchatClick,
}) => {
  const [subchats, setSubchats] = useState<Subchat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [expandedSubchatId, setExpandedSubchatId] = useState<string | null>(null);

  const loadSubchats = useCallback(async (isBackgroundRefresh = false) => {
    // Don't show loading spinner for background refreshes
    if (!isBackgroundRefresh) {
      setLoading(true);
    }
    setError(null);
    try {
      const data = await subchatService.getSubchatsByParent(parentChatId);
      setSubchats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subchats');
    } finally {
      if (!isBackgroundRefresh) {
        setLoading(false);
      }
    }
  }, [parentChatId]);

  useEffect(() => {
    loadSubchats();

    // Set up auto-refresh polling every 5 seconds for real-time updates
    const intervalId = setInterval(() => {
      loadSubchats(true); // Pass true to indicate background refresh
    }, 5000);

    // Clean up interval on unmount or when parentChatId changes
    return () => clearInterval(intervalId);
  }, [loadSubchats]);

  const handleSubchatCreated = (subchatId: string) => {
    setDialogOpen(false);
    loadSubchats(); // Refresh list
    if (onSubchatClick) {
      onSubchatClick(subchatId); // Navigate to new subchat
    }
  };

  const handleCardClick = (subchatId: string) => {
    if (onSubchatClick) {
      onSubchatClick(subchatId);
    }
  };

  const handleToggleDetails = (subchatId: string) => {
    setExpandedSubchatId((prev) => (prev === subchatId ? null : subchatId));
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
        <CircularProgress />
      </Box>
    );
  }

  // Separate running/active subchats from completed/failed ones
  const runningSubchats = subchats.filter((s) => s.status === 'active');
  const completedSubchats = subchats.filter((s) => s.status !== 'active');

  return (
    <Box>
      {/* Header with Create button */}
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h6" component="h2">
          Subchats ({subchats.length})
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
        >
          Create Subchat
        </Button>
      </Box>

      {/* Error message */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {/* Empty state */}
      {!loading && !error && subchats.length === 0 && (
        <Paper
          variant="outlined"
          sx={{
            p: 4,
            textAlign: 'center',
            backgroundColor: 'background.default',
          }}
        >
          <Typography variant="body1" color="text.secondary" gutterBottom>
            No subchats yet
          </Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            Create a subchat to delegate work to a specialist agent
          </Typography>
          <Button
            variant="outlined"
            startIcon={<AddIcon />}
            onClick={() => setDialogOpen(true)}
          >
            Create First Subchat
          </Button>
        </Paper>
      )}

      {/* Running Subchats Section */}
      {runningSubchats.length > 0 && (
        <Box mb={4}>
          <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 'bold', color: 'primary.main' }}>
            Running ({runningSubchats.length})
          </Typography>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: {
                xs: '1fr',
                sm: 'repeat(2, 1fr)',
                md: 'repeat(3, 1fr)',
              },
              gap: 2,
            }}
          >
            {runningSubchats.map((subchat) => (
              <Box key={subchat.id}>
                <SubchatCard
                  subchat={subchat}
                  onClick={handleCardClick}
                  isExpanded={expandedSubchatId === subchat.id}
                  onToggleDetails={handleToggleDetails}
                />
                <Collapse in={expandedSubchatId === subchat.id}>
                  <Box sx={{ mt: 2, ml: 2 }}>
                    <SubchatDetailView
                      subchatId={subchat.id}
                      onClose={() => setExpandedSubchatId(null)}
                    />
                  </Box>
                </Collapse>
              </Box>
            ))}
          </Box>
        </Box>
      )}

      {/* Completed/Failed Subchats Section */}
      {completedSubchats.length > 0 && (
        <Box>
          <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 'bold', color: 'text.secondary' }}>
            Completed ({completedSubchats.length})
          </Typography>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: {
                xs: '1fr',
                sm: 'repeat(2, 1fr)',
                md: 'repeat(3, 1fr)',
              },
              gap: 2,
            }}
          >
            {completedSubchats.map((subchat) => (
              <Box key={subchat.id}>
                <SubchatCard
                  subchat={subchat}
                  onClick={handleCardClick}
                  isExpanded={expandedSubchatId === subchat.id}
                  onToggleDetails={handleToggleDetails}
                />
                <Collapse in={expandedSubchatId === subchat.id}>
                  <Box sx={{ mt: 2, ml: 2 }}>
                    <SubchatDetailView
                      subchatId={subchat.id}
                      onClose={() => setExpandedSubchatId(null)}
                    />
                  </Box>
                </Collapse>
              </Box>
            ))}
          </Box>
        </Box>
      )}

      {/* Creation dialog */}
      <SubchatCreationDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        parentChatId={parentChatId}
        onSubchatCreated={handleSubchatCreated}
      />
    </Box>
  );
};

export default SubchatList;
