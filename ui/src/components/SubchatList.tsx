/**
 * SubchatList Component
 *
 * Displays list of child subchats for a parent chat with create button.
 * Fetches subchats on mount and handles loading/empty states.
 * Responsive design optimized for mobile, tablet, and desktop viewports.
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Button,
  Typography,
  CircularProgress,
  Alert,
  Paper,
  Collapse,
  Container,
  useTheme,
  useMediaQuery,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon } from '@mui/icons-material';
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
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  const [subchats, setSubchats] = useState<Subchat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [expandedSubchatId, setExpandedSubchatId] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [subchatToDelete, setSubchatToDelete] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Calculate running subchats (memoized to avoid recalculation on every render)
  const runningSubchats = useMemo(
    () => subchats.filter((s) => s.status === 'active'),
    [subchats]
  );

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

  // Initial load effect
  useEffect(() => {
    loadSubchats();
  }, [loadSubchats]);

  // Page Visibility API - immediate refresh when tab becomes visible
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (!document.hidden) {
        loadSubchats(true); // Background refresh when tab becomes visible
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [loadSubchats]);

  // Conditional polling - only poll when there are active subchats
  useEffect(() => {
    // Only set up polling if there are running subchats AND tab is visible
    if (runningSubchats.length === 0 || document.hidden) {
      return; // No polling needed
    }

    // Poll every 15 seconds for active subchats (reduced from 5s)
    const intervalId = setInterval(() => {
      if (!document.hidden) {
        loadSubchats(true); // Pass true to indicate background refresh
      }
    }, 15000);

    // Clean up interval on unmount or when dependencies change
    return () => clearInterval(intervalId);
  }, [loadSubchats, runningSubchats.length]);

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

  const handleDeleteClick = (subchatId: string) => {
    setSubchatToDelete(subchatId);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!subchatToDelete) return;

    setIsDeleting(true);
    try {
      await subchatService.deleteSubchat(subchatToDelete);
      // Remove from UI
      setSubchats((prev) => prev.filter((s) => s.id !== subchatToDelete));
      setDeleteDialogOpen(false);
      setSubchatToDelete(null);
      // Optionally refresh the list to be sure
      loadSubchats(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete subchat');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
    setSubchatToDelete(null);
  };

  // Responsive typography scaling
  const getResponsiveTypography = () => ({
    h5: {
      fontSize: {
        xs: '1.25rem', // 20px on mobile
        sm: '1.5rem',  // 24px on tablet
        md: '1.75rem', // 28px on desktop
      },
      lineHeight: {
        xs: 1.3,
        sm: 1.4,
        md: 1.5,
      },
    },
    h6: {
      fontSize: {
        xs: '1rem',    // 16px on mobile
        sm: '1.125rem', // 18px on tablet
        md: '1.25rem',  // 20px on desktop
      },
    },
    body1: {
      fontSize: {
        xs: '0.875rem', // 14px on mobile
        sm: '1rem',     // 16px on tablet+
      },
    },
    body2: {
      fontSize: {
        xs: '0.75rem',  // 12px on mobile
        sm: '0.875rem', // 14px on tablet+
      },
    },
  });

  // Responsive spacing and sizing
  const getResponsiveSpacing = () => ({
    containerPadding: {
      xs: 2, // 16px on mobile
      sm: 3, // 24px on tablet
      md: 4, // 32px on desktop
    },
    cardPadding: {
      xs: 2, // 16px on mobile
      sm: 2.5, // 20px on tablet
      md: 3, // 24px on desktop
    },
    buttonMinHeight: {
      xs: 44, // Touch-friendly 44px minimum
      sm: 40,
      md: 36,
    },
    gridSpacing: {
      xs: 2, // 16px on mobile
      sm: 2.5, // 20px on tablet
      md: 3, // 24px on desktop
    },
  });

  const responsiveStyles = getResponsiveTypography();
  const spacing = getResponsiveSpacing();

  if (loading) {
    return (
      <Container 
        maxWidth="lg" 
        sx={{ 
          py: spacing.containerPadding,
          px: { xs: 2, sm: 3 }
        }}
      >
        <Box 
          display="flex" 
          justifyContent="center" 
          alignItems="center" 
          minHeight={{ xs: 200, sm: 250, md: 300 }}
          sx={{
            backgroundColor: 'background.paper',
            borderRadius: { xs: 1, sm: 2 },
            boxShadow: 1,
          }}
        >
          <Box textAlign="center">
            <CircularProgress
              size={44}
              sx={{ mb: 2 }}
            />
            <Typography 
              variant="body2" 
              color="text.secondary"
              sx={{ fontSize: responsiveStyles.body2.fontSize }}
            >
              Loading subchats...
            </Typography>
          </Box>
        </Box>
      </Container>
    );
  }

  // Separate completed/failed subchats (runningSubchats already calculated at top)
  const completedSubchats = useMemo(
    () => subchats.filter((s) => s.status !== 'active'),
    [subchats]
  );

  return (
    <Container 
      maxWidth="lg" 
      sx={{ 
        py: spacing.containerPadding,
        px: { xs: 2, sm: 3 }
      }}
    >
      <Box sx={{ 
        backgroundColor: 'background.paper', 
        borderRadius: { xs: 1, sm: 2 }, 
        boxShadow: 1, 
        p: spacing.cardPadding 
      }}>
        {/* Header with Create button */}
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={{ xs: 3, sm: 4 }}
          sx={{
            flexDirection: { xs: 'column', sm: 'row' },
            gap: { xs: 2, sm: 0 },
            alignItems: { xs: 'stretch', sm: 'center' },
          }}
        >
          <Box sx={{ textAlign: { xs: 'center', sm: 'left' } }}>
            <Typography 
              variant="h5" 
              component="h2" 
              sx={{ 
                fontWeight: 600, 
                mb: 1,
                fontSize: responsiveStyles.h5.fontSize,
                lineHeight: responsiveStyles.h5.lineHeight,
              }}
            >
              Subchats
            </Typography>
            <Typography 
              variant="body2" 
              color="text.secondary"
              sx={{ fontSize: responsiveStyles.body2.fontSize }}
            >
              {subchats.length} {subchats.length === 1 ? 'subchat' : 'subchats'} total
            </Typography>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setDialogOpen(true)}
            sx={{
              minWidth: { xs: '100%', sm: 'auto' },
              minHeight: spacing.buttonMinHeight,
              py: { xs: 1.5, sm: 1.25 },
              px: { xs: 3, sm: 2.5, md: 3 },
              borderRadius: { xs: 1.5, sm: 2 },
              textTransform: 'none',
              fontWeight: 600,
              fontSize: { xs: '0.875rem', sm: '1rem' },
              // Touch-friendly spacing on mobile
              '&:active': {
                transform: isMobile ? 'scale(0.98)' : 'none',
              },
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
              borderRadius: { xs: 1, sm: 2 },
              fontSize: responsiveStyles.body2.fontSize,
              '& .MuiAlert-message': {
                width: '100%',
              },
            }}
          >
            {error}
          </Alert>
        )}

        {/* Empty state */}
        {subchats.length === 0 && !error && (
          <Paper
            variant="outlined"
            sx={{
              p: { xs: 4, sm: 6 },
              textAlign: 'center',
              backgroundColor: 'background.default',
              borderRadius: { xs: 2, sm: 3 },
              border: '2px dashed',
              borderColor: 'divider',
              minHeight: { xs: 200, sm: 250, md: 300 },
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography 
              variant="h6" 
              color="text.secondary" 
              gutterBottom
              sx={{ 
                fontSize: responsiveStyles.h6.fontSize,
                mb: 2,
              }}
            >
              No subchats yet
            </Typography>
            <Typography 
              variant="body1" 
              color="text.secondary" 
              paragraph
              sx={{ 
                fontSize: responsiveStyles.body1.fontSize,
                maxWidth: { xs: '100%', sm: 400 },
                lineHeight: 1.6,
              }}
            >
              Create your first subchat to start collaborating with specialist agents on specific topics or tasks.
            </Typography>
            <Button
              variant="outlined"
              startIcon={<AddIcon />}
              onClick={() => setDialogOpen(true)}
              sx={{
                mt: 2,
                minHeight: spacing.buttonMinHeight,
                px: { xs: 3, sm: 4 },
                borderRadius: { xs: 1.5, sm: 2 },
                textTransform: 'none',
                fontWeight: 500,
                fontSize: responsiveStyles.body1.fontSize,
              }}
            >
              Create First Subchat
            </Button>
          </Paper>
        )}

        {/* Running/Active Subchats Section */}
        {runningSubchats.length > 0 && (
          <Box mb={4}>
            <Box 
              display="flex" 
              alignItems="center" 
              mb={3}
              sx={{
                borderBottom: '1px solid',
                borderColor: 'divider',
                pb: 2,
              }}
            >
              <Typography 
                variant="h6" 
                sx={{ 
                  fontWeight: 600,
                  fontSize: responsiveStyles.h6.fontSize,
                  mr: 2,
                }}
              >
                Active Subchats
              </Typography>
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{
                  backgroundColor: 'primary.main',
                  color: 'primary.contrastText',
                  px: 1.5,
                  py: 0.5,
                  borderRadius: 1,
                  fontSize: responsiveStyles.body2.fontSize,
                  fontWeight: 500,
                }}
              >
                {runningSubchats.length}
              </Typography>
            </Box>
            <Box>
              {runningSubchats.map((subchat) => (
                <Box 
                  key={subchat.id}
                  sx={{
                    mb: 2,
                    '&:last-child': {
                      mb: 0,
                    },
                  }}
                >
                  <SubchatCard
                    subchat={subchat}
                    onClick={() => handleCardClick(subchat.id)}
                    onDelete={handleDeleteClick}
                    onToggleDetails={() => handleToggleDetails(subchat.id)}
                    isExpanded={expandedSubchatId === subchat.id}
                  />
                  <Collapse in={expandedSubchatId === subchat.id}>
                    <Box mt={2}>
                      <SubchatDetailView subchatId={subchat.id} />
                    </Box>
                  </Collapse>
                </Box>
              ))}
            </Box>
          </Box>
        )}

        {/* Completed Subchats Section */}
        {completedSubchats.length > 0 && (
          <Box>
            <Box 
              display="flex" 
              alignItems="center" 
              mb={3}
              sx={{
                borderBottom: '1px solid',
                borderColor: 'divider',
                pb: 2,
              }}
            >
              <Typography 
                variant="h6" 
                sx={{ 
                  fontWeight: 600,
                  fontSize: responsiveStyles.h6.fontSize,
                  mr: 2,
                }}
              >
                Completed Subchats
              </Typography>
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{
                  backgroundColor: 'success.main',
                  color: 'success.contrastText',
                  px: 1.5,
                  py: 0.5,
                  borderRadius: 1,
                  fontSize: responsiveStyles.body2.fontSize,
                  fontWeight: 500,
                }}
              >
                {completedSubchats.length}
              </Typography>
            </Box>
            <Box 
              sx={{
                display: 'grid',
                gridTemplateColumns: {
                  xs: '1fr',
                  sm: 'repeat(2, 1fr)',
                  lg: 'repeat(3, 1fr)',
                },
                gap: spacing.gridSpacing,
              }}
            >
              {completedSubchats.map((subchat) => (
                <Box key={subchat.id}>
                  <SubchatCard
                    subchat={subchat}
                    onClick={() => handleCardClick(subchat.id)}
                    onDelete={handleDeleteClick}
                    onToggleDetails={() => handleToggleDetails(subchat.id)}
                    isExpanded={expandedSubchatId === subchat.id}
                  />
                  <Collapse in={expandedSubchatId === subchat.id}>
                    <Box mt={2}>
                      <SubchatDetailView subchatId={subchat.id} />
                    </Box>
                  </Collapse>
                </Box>
              ))}
            </Box>
          </Box>
        )}
      </Box>

      {/* Creation Dialog */}
      <SubchatCreationDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        parentChatId={parentChatId}
        onSubchatCreated={handleSubchatCreated}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={handleDeleteCancel}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>Delete Subchat?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete this subchat? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ p: 2, gap: 1 }}>
          <Button onClick={handleDeleteCancel} disabled={isDeleting} variant="outlined">
            Cancel
          </Button>
          <Button
            onClick={handleDeleteConfirm}
            color="error"
            variant="contained"
            disabled={isDeleting}
            startIcon={isDeleting ? <CircularProgress size={16} sx={{ color: 'inherit' }} /> : <DeleteIcon />}
          >
            {isDeleting ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default SubchatList;