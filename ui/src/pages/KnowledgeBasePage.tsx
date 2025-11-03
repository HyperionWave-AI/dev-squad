import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Alert,
  Snackbar,
  CircularProgress,
  Typography,
} from '@mui/material';
import { CollectionSidebar } from '../components/CollectionSidebar';
import { ArticleList } from '../components/ArticleList';
import { ArticleViewer } from '../components/ArticleViewer';
import { ArticleEditor } from '../components/ArticleEditor';
import {
  knowledgeService,
  type KnowledgeEntry,
  type CollectionInfo,
} from '../services/knowledgeService';

export const KnowledgeBasePage: React.FC = () => {
  const [collections, setCollections] = useState<CollectionInfo[]>([]);
  const [selectedCollection, setSelectedCollection] = useState<string | null>(null);
  const [entries, setEntries] = useState<KnowledgeEntry[]>([]);
  const [selectedEntry, setSelectedEntry] = useState<KnowledgeEntry | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load collections on mount
  useEffect(() => {
    loadCollections();
  }, []);

  // Load entries when collection is selected
  useEffect(() => {
    if (selectedCollection) {
      loadEntries(selectedCollection);
    } else {
      setEntries([]);
      setSelectedEntry(null);
    }
  }, [selectedCollection]);

  const loadCollections = async () => {
    setLoading(true);
    try {
      const response = await knowledgeService.getCollections();
      setCollections(response.collections || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load collections');
    } finally {
      setLoading(false);
    }
  };

  const loadEntries = async (collection: string) => {
    setLoading(true);
    try {
      const response = await knowledgeService.getEntries(collection, 100);
      setEntries(response.entries || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load entries');
      setEntries([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectCollection = (collection: string) => {
    setSelectedCollection(collection);
    setSelectedEntry(null);
    setIsEditing(false);
  };

  const handleSelectEntry = (entry: KnowledgeEntry) => {
    setSelectedEntry(entry);
    setIsEditing(false);
  };

  const handleEdit = () => {
    setIsEditing(true);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
  };

  const handleSave = async (text: string, metadata: Record<string, any>) => {
    if (!selectedEntry) return;

    try {
      const response = await knowledgeService.updateEntry(selectedEntry.id, {
        text,
        metadata,
      });

      // Update the entry in the list
      setEntries(entries.map(e =>
        e.id === selectedEntry.id ? response.entry : e
      ));

      // Update selected entry
      setSelectedEntry(response.entry);

      setIsEditing(false);
      setSuccessMessage('Entry updated successfully');
      setError(null);
    } catch (err) {
      throw err; // Let the editor handle the error
    }
  };

  const handleDelete = async () => {
    if (!selectedEntry) return;

    try {
      await knowledgeService.deleteEntry(selectedEntry.id);

      // Remove entry from list
      setEntries(entries.filter(e => e.id !== selectedEntry.id));

      // Clear selection
      setSelectedEntry(null);
      setIsEditing(false);

      setSuccessMessage('Entry deleted successfully');
      setError(null);

      // Reload collections to update counts
      loadCollections();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete entry');
    }
  };

  const handleCloseError = () => {
    setError(null);
  };

  const handleCloseSuccess = () => {
    setSuccessMessage(null);
  };

  return (
    <Box sx={{ height: 'calc(100vh - 64px)', overflow: 'hidden', p: 2 }}>
      {/* Main layout with 3 columns */}
      <Box sx={{ display: 'flex', gap: 2, height: '100%' }}>
        {/* Left sidebar: Collections */}
        <Box sx={{ width: '25%', minWidth: 200, height: '100%' }}>
          <Paper elevation={1} sx={{ height: '100%', overflow: 'hidden' }}>
            {loading && !collections.length ? (
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: '100%',
                }}
              >
                <CircularProgress />
              </Box>
            ) : (
              <CollectionSidebar
                collections={collections}
                selectedCollection={selectedCollection}
                onSelectCollection={handleSelectCollection}
                onCollectionsChanged={loadCollections}
              />
            )}
          </Paper>
        </Box>

        {/* Middle: Article List */}
        <Box sx={{ width: '25%', minWidth: 200, height: '100%' }}>
          <Paper elevation={1} sx={{ height: '100%', overflow: 'hidden' }}>
            {!selectedCollection ? (
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
                  Select a collection to view entries
                </Typography>
              </Box>
            ) : loading ? (
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: '100%',
                }}
              >
                <CircularProgress />
              </Box>
            ) : (
              <ArticleList
                entries={entries}
                selectedEntryId={selectedEntry?.id || null}
                onSelectEntry={handleSelectEntry}
              />
            )}
          </Paper>
        </Box>

        {/* Right: Article Viewer or Editor */}
        <Box sx={{ flex: 1, minWidth: 300, height: '100%' }}>
          <Paper elevation={1} sx={{ height: '100%', overflow: 'hidden' }}>
            {isEditing && selectedEntry ? (
              <ArticleEditor
                entry={selectedEntry}
                onSave={handleSave}
                onCancel={handleCancelEdit}
              />
            ) : (
              <ArticleViewer
                entry={selectedEntry}
                onEdit={handleEdit}
                onDelete={handleDelete}
              />
            )}
          </Paper>
        </Box>
      </Box>

      {/* Error snackbar */}
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={handleCloseError}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseError} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>

      {/* Success snackbar */}
      <Snackbar
        open={!!successMessage}
        autoHideDuration={3000}
        onClose={handleCloseSuccess}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSuccess} severity="success" sx={{ width: '100%' }}>
          {successMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};
