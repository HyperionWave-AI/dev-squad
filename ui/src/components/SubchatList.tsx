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
  Container,
  Divider,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { subchatService, type Subchat } from '../services/subchatService';
import SubchatCard from './SubchatCard';
import SubchatCreationDialog from './SubchatCreationDialog';
import SubchatDetailView from './SubchatDetailView';

interface SubchatListProps {
  parentChatId: string;
  onSubchatClick?: (subchatId: string) => void;
  onSubchatCreated?: () => void | Promise<void>; // Callback to refresh parent sessions list
}

export const SubchatList: React.FC<SubchatListProps> = ({
  parentChatId,
  onSubchatClick,
  onSubchatCreated: onSubchatCreatedCallback,
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

  const handleSubchatCreated = async (subchatId: string) => {
    console.log('[SubchatList] handleSubchatCreated called with subchatId:', subchatId);
    console.log('[SubchatList] onSubchatCreatedCallback exists?', !!onSubchatCreatedCallback);

    setDialogOpen(false);
    loadSubchats(); // Refresh subchats list in drawer

    // Notify parent to refresh sessions list (for tree structure in sidebar)
    if (onSubchatCreatedCallback) {
      console.log('[SubchatList] Calling onSubchatCreatedCallback...');
      await onSubchatCreatedCallback();
      console.log('[SubchatList] onSubchatCreatedCallback completed');
    } else {
      console.warn('[SubchatList] onSubchatCreatedCallback is not defined!');
    }

    if (onSubchatClick) {
      console.log('[SubchatList] Navigating to subchat:', subchatId);
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
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Box 
          display="flex" 
          justifyContent="center" 
          alignItems="center" 
          minHeight={300}
          sx={{
            backgroundColor: 'background.paper',
            borderRadius: 2,
            boxShadow: 1,
          }}
        >
          <Box textAlign="center">
            <CircularProgress size={48} sx={{ mb: 2 }} />
            <Typography variant="body2" color="text.secondary">
              Loading subchats...
            </Typography>
          </Box>
        </Box>
      </Container>
    );
  }

  // Separate running/active subchats from completed/failed ones
  const runningSubchats = subchats.filter((s) => s.status === 'active');
  const completedSubchats = subchats.filter((s) => s.status !== 'active');

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Box sx={{ backgroundColor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 3 }}>
        {/* Header with Create button */}
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={4}
          sx={{
            flexDirection: { xs: 'column', sm: 'row' },
            gap: { xs: 2, sm: 0 },
          }}
        >
          <Box>
            <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
              Subchats
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {subchats.length} {subchats.length === 1 ? 'subchat' : 'subchats'} total
            </Typography>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setDialogOpen(true)}
            sx={{
              minWidth: { xs: '100%', sm: 'auto' },
              py: 1.5,
              px: 3,
              borderRadius: 2,
              textTransform: 'none',
              fontWeight: 600,
            }}
          >
            Create Subchat
          </Button>
        </Box>

        {/* Error message */}
        {error && (
          <Alert 
            severity="error" 
            sx={{ 
              mb: 3, 
              borderRadius: 2,
              '& .MuiAlert-message': {
                width: '100%',
              },
            }}
          >
            {error}
          </Alert>
        )}

        {/* Empty state */}
        {!loading && !error && subchats.length === 0 && (
          <Paper
            variant="outlined"
            sx={{
              p: 6,
              textAlign: 'center',
              backgroundColor: 'background.default',
              borderRadius: 3,
              border: '2px dashed',
              borderColor: 'divider',
            }}
          >
            <Box sx={{ maxWidth: 400, mx: 'auto' }}>
              <Typography variant="h6" color="text.primary" gutterBottom sx={{ fontWeight: 600 }}>
                No subchats yet
              </Typography>
              <Typography variant="body1" color="text.secondary" paragraph sx={{ mb: 3 }}>
                Create a subchat to delegate work to a specialist agent and organize your workflow
              </Typography>
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => setDialogOpen(true)}
                sx={{
                  py: 1.5,
                  px: 4,
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                }}
              >
                Create First Subchat
              </Button>
            </Box>
          </Paper>
        )}

        {/* Running Subchats Section */}
        {runningSubchats.length > 0 && (
          <Box mb={5}>
            <Box display="flex" alignItems="center" mb={3}>
              <Typography 
                variant="h6" 
                sx={{ 
                  fontWeight: 600, 
                  color: 'primary.main',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                }}
              >
                Running
                <Box
                  component="span"
                  sx={{
                    backgroundColor: 'primary.main',
                    color: 'primary.contrastText',
                    borderRadius: '12px',
                    px: 1.5,
                    py: 0.5,
                    fontSize: '0.75rem',
                    fontWeight: 600,
                  }}
                >
                  {runningSubchats.length}
                </Box>
              </Typography>
            </Box>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: {
                  xs: '1fr',
                  sm: 'repeat(auto-fit, minmax(320px, 1fr))',
                  lg: 'repeat(auto-fit, minmax(350px, 1fr))',
                },
                gap: 3,
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
                    <Box sx={{ mt: 2, ml: 1 }}>
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

        {/* Divider between sections */}
        {runningSubchats.length > 0 && completedSubchats.length > 0 && (
          <Divider sx={{ my: 4 }} />
        )}

        {/* Completed/Failed Subchats Section */}
        {completedSubchats.length > 0 && (
          <Box>
            <Box display="flex" alignItems="center" mb={3}>
              <Typography 
                variant="h6" 
                sx={{ 
                  fontWeight: 600, 
                  color: 'text.secondary',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                }}
              >
                Completed
                <Box
                  component="span"
                  sx={{
                    backgroundColor: 'grey.200',
                    color: 'text.secondary',
                    borderRadius: '12px',
                    px: 1.5,
                    py: 0.5,
                    fontSize: '0.75rem',
                    fontWeight: 600,
                  }}
                >
                  {completedSubchats.length}
                </Box>
              </Typography>
            </Box>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: {
                  xs: '1fr',
                  sm: 'repeat(auto-fit, minmax(320px, 1fr))',
                  lg: 'repeat(auto-fit, minmax(350px, 1fr))',
                },
                gap: 3,
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
                    <Box sx={{ mt: 2, ml: 1 }}>
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
    </Container>
  );
};

export default SubchatList;