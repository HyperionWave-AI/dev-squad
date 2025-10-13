/**
 * SubchatList Component
 *
 * Displays list of child subchats for a parent chat with create button.
 * Fetches subchats on mount and handles loading/empty states.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Typography,
  CircularProgress,
  Alert,
  Paper,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { subchatService, type Subchat } from '../services/subchatService';
import SubchatCard from './SubchatCard';
import SubchatCreationDialog from './SubchatCreationDialog';

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

  useEffect(() => {
    loadSubchats();
  }, [parentChatId]);

  const loadSubchats = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await subchatService.getSubchatsByParent(parentChatId);
      setSubchats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subchats');
    } finally {
      setLoading(false);
    }
  };

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

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
        <CircularProgress />
      </Box>
    );
  }

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

      {/* Subchat grid */}
      {subchats.length > 0 && (
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
          {subchats.map((subchat) => (
            <Box key={subchat.id}>
              <SubchatCard
                subchat={subchat}
                onClick={handleCardClick}
              />
            </Box>
          ))}
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
