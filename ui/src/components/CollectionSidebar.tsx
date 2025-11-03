import React, { useState } from 'react';
import {
  Box,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
  Badge,
  Divider,
  Chip,
  TextField,
  IconButton,
  InputAdornment,
  CircularProgress,
  Snackbar,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Settings as SettingsIcon,
  Clear as ClearIcon,
  DeleteOutline as DeleteIcon,
  VerifiedUser as VerifiedUserIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import type { CollectionInfo } from '../services/knowledgeService';
import { knowledgeService } from '../services/knowledgeService';
import { CollectionSettingsDialog } from './CollectionSettingsDialog';
import { DeleteCollectionDialog } from './DeleteCollectionDialog';

interface CollectionSidebarProps {
  collections: CollectionInfo[];
  selectedCollection: string | null;
  onSelectCollection: (collection: string) => void;
  onCollectionsChanged: () => void;
}

export const CollectionSidebar: React.FC<CollectionSidebarProps> = ({
  collections,
  selectedCollection,
  onSelectCollection,
  onCollectionsChanged,
}) => {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [settingsCollection, setSettingsCollection] = useState<CollectionInfo | null>(null);
  const [deleteCollection, setDeleteCollection] = useState<CollectionInfo | null>(null);
  const [verifyingCollection, setVerifyingCollection] = useState<string | null>(null);
  const [verifyError, setVerifyError] = useState<string | null>(null);

  // Filter collections by search query
  const filteredCollections = collections.filter((col) =>
    col.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Group collections by category
  const groupedCollections = filteredCollections.reduce((acc, col) => {
    if (!acc[col.category]) {
      acc[col.category] = [];
    }
    acc[col.category].push(col);
    return acc;
  }, {} as Record<string, CollectionInfo[]>);

  // Category display order
  const categoryOrder = ['Task', 'Tech', 'UI', 'Ops', 'Other'];

  // Category colors
  const categoryColors: Record<string, 'primary' | 'secondary' | 'success' | 'warning' | 'default'> = {
    Task: 'primary',
    Tech: 'secondary',
    UI: 'success',
    Ops: 'warning',
    Other: 'default',
  };

  const handleOpenSettings = (collection: CollectionInfo, event: React.MouseEvent) => {
    event.stopPropagation();
    setSettingsCollection(collection);
  };

  const handleCloseSettings = () => {
    setSettingsCollection(null);
  };

  const handleSettingsSaved = () => {
    onCollectionsChanged();
  };

  const handleOpenDelete = (collection: CollectionInfo, event: React.MouseEvent) => {
    event.stopPropagation();
    setDeleteCollection(collection);
  };

  const handleCloseDelete = () => {
    setDeleteCollection(null);
  };

  const handleConfirmDelete = async () => {
    if (!deleteCollection) return;

    try {
      await knowledgeService.deleteCollection(deleteCollection.id);
      setDeleteCollection(null);
      onCollectionsChanged();
    } catch (err) {
      // Dialog will handle the error display
      throw err;
    }
  };

  const handleVerify = async (collection: CollectionInfo, event: React.MouseEvent) => {
    event.stopPropagation();
    setVerifyingCollection(collection.id);
    setVerifyError(null);

    try {
      const { sessionId } = await knowledgeService.verifyKnowledgeArticle(collection.id);
      navigate(`/chat/${sessionId}`);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to verify knowledge article';
      setVerifyError(errorMessage);
    } finally {
      setVerifyingCollection(null);
    }
  };

  const truncateText = (text: string, maxLength: number) => {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  };

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', bgcolor: 'background.paper' }}>
      {/* Header with search */}
      <Box sx={{ p: 2 }}>
        <Typography variant="h6" gutterBottom>
          Collections
        </Typography>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {collections.length} collections
        </Typography>

        {/* Search field */}
        <TextField
          size="small"
          fullWidth
          placeholder="Search collections..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
            endAdornment: searchQuery && (
              <InputAdornment position="end">
                <IconButton
                  size="small"
                  onClick={() => setSearchQuery('')}
                  edge="end"
                >
                  <ClearIcon fontSize="small" />
                </IconButton>
              </InputAdornment>
            ),
          }}
          sx={{ mt: 1 }}
        />
      </Box>

      <Divider />

      {/* Collections list */}
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        {categoryOrder.map((category) => {
          const categoryCollections = groupedCollections[category];
          if (!categoryCollections || categoryCollections.length === 0) return null;

          return (
            <Box key={category} sx={{ mb: 2 }}>
              <Box sx={{ px: 2, py: 1 }}>
                <Chip
                  label={category}
                  size="small"
                  color={categoryColors[category]}
                  sx={{ fontWeight: 'bold' }}
                />
              </Box>

              <List dense>
                {categoryCollections.map((collection) => (
                  <ListItem
                    key={collection.name}
                    disablePadding
                    secondaryAction={
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        <IconButton
                          edge="end"
                          size="small"
                          onClick={(e) => handleVerify(collection, e)}
                          disabled={verifyingCollection === collection.id}
                          aria-label="Verify knowledge article"
                          sx={{
                            '&:hover': {
                              color: 'success.main',
                            }
                          }}
                        >
                          {verifyingCollection === collection.id ? (
                            <CircularProgress size={16} />
                          ) : (
                            <VerifiedUserIcon fontSize="small" />
                          )}
                        </IconButton>
                        <IconButton
                          edge="end"
                          size="small"
                          onClick={(e) => handleOpenDelete(collection, e)}
                          sx={{
                            '&:hover': {
                              color: 'error.main',
                            }
                          }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                        <IconButton
                          edge="end"
                          size="small"
                          onClick={(e) => handleOpenSettings(collection, e)}
                          sx={{ mr: 1 }}
                        >
                          <SettingsIcon fontSize="small" />
                        </IconButton>
                      </Box>
                    }
                  >
                    <ListItemButton
                      selected={selectedCollection === collection.name}
                      onClick={() => onSelectCollection(collection.name)}
                    >
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, pr: 4 }}>
                            <Typography variant="body2" sx={{ flex: 1, fontSize: '0.875rem' }}>
                              {collection.name}
                            </Typography>
                            <Badge
                              badgeContent={collection.count}
                              color="primary"
                              max={999}
                            />
                          </Box>
                        }
                        secondary={
                          <Box>
                            {collection.description && (
                              <Typography
                                variant="caption"
                                color="text.secondary"
                                sx={{
                                  display: 'block',
                                  fontSize: '0.7rem',
                                  mt: 0.5,
                                }}
                              >
                                {truncateText(collection.description, 40)}
                              </Typography>
                            )}
                            {collection.tags && collection.tags.length > 0 && (
                              <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 0.5 }}>
                                {collection.tags.slice(0, 2).map((tag) => (
                                  <Chip
                                    key={tag}
                                    label={tag}
                                    size="small"
                                    sx={{
                                      height: 18,
                                      fontSize: '0.65rem',
                                      '& .MuiChip-label': {
                                        px: 0.75,
                                      },
                                    }}
                                  />
                                ))}
                                {collection.tags.length > 2 && (
                                  <Chip
                                    label={`+${collection.tags.length - 2}`}
                                    size="small"
                                    sx={{
                                      height: 18,
                                      fontSize: '0.65rem',
                                      '& .MuiChip-label': {
                                        px: 0.75,
                                      },
                                    }}
                                  />
                                )}
                              </Box>
                            )}
                          </Box>
                        }
                      />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>
            </Box>
          );
        })}

        {filteredCollections.length === 0 && (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No collections found
            </Typography>
          </Box>
        )}
      </Box>

      {/* Settings Dialog */}
      <CollectionSettingsDialog
        collection={settingsCollection}
        open={settingsCollection !== null}
        onClose={handleCloseSettings}
        onSaved={handleSettingsSaved}
      />

      {/* Delete Confirmation Dialog */}
      {deleteCollection && (
        <DeleteCollectionDialog
          open={deleteCollection !== null}
          collectionName={deleteCollection.name}
          entryCount={deleteCollection.count}
          onConfirm={handleConfirmDelete}
          onCancel={handleCloseDelete}
        />
      )}

      {/* Error Snackbar */}
      <Snackbar
        open={verifyError !== null}
        autoHideDuration={6000}
        onClose={() => setVerifyError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setVerifyError(null)} severity="error" sx={{ width: '100%' }}>
          {verifyError}
        </Alert>
      </Snackbar>
    </Box>
  );
};
