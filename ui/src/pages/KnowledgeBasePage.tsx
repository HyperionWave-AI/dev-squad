import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Alert,
  Snackbar,
  CircularProgress,
  Typography,
  TextField,
  InputAdornment,
  IconButton,
} from '@mui/material';
import { Search as SearchIcon, Clear as ClearIcon } from '@mui/icons-material';
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

  // Universal search state
  const [universalSearchQuery, setUniversalSearchQuery] = useState<string>('');
  const [isUniversalSearchMode, setIsUniversalSearchMode] = useState<boolean>(false);
  const [searchCollections, setSearchCollections] = useState<string[]>([]);

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

  // Universal search handler
  const handleUniversalSearch = async () => {
    if (!universalSearchQuery.trim()) {
      // Clear search mode
      setIsUniversalSearchMode(false);
      setEntries([]);
      setSearchCollections([]);
      setSelectedEntry(null);
      setSelectedCollection(null);
      return;
    }

    setLoading(true);
    setIsUniversalSearchMode(true);
    setSelectedCollection(null); // Clear collection selection in universal search mode

    try {
      const { entries: searchResults, collectionsWithData } = await knowledgeService.universalSearch(
        universalSearchQuery,
        100
      );
      setEntries(searchResults);
      setSearchCollections(collectionsWithData);
      setSelectedEntry(null);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Universal search failed');
      setEntries([]);
      setSearchCollections([]);
    } finally {
      setLoading(false);
    }
  };

  // Handle Enter key in search box
  const handleSearchKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleUniversalSearch();
    }
  };

  // Clear universal search
  const handleClearUniversalSearch = () => {
    setUniversalSearchQuery('');
    setIsUniversalSearchMode(false);
    setEntries([]);
    setSearchCollections([]);
    setSelectedEntry(null);
  };

  return (
    <Box sx={{ height: 'calc(100vh - 64px)', overflow: 'hidden', p: 2 }}>
      {/* Universal Search Bar */}
      <Box sx={{ mb: 2 }}>
        <Paper elevation={2} sx={{ p: 2 }}>
          <TextField
            fullWidth
            placeholder="Search across all collections... (press Enter)"
            value={universalSearchQuery}
            onChange={(e) => setUniversalSearchQuery(e.target.value)}
            onKeyPress={handleSearchKeyPress}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
              endAdornment: universalSearchQuery && (
                <InputAdornment position="end">
                  <IconButton size="small" onClick={handleClearUniversalSearch}>
                    <ClearIcon />
                  </IconButton>
                </InputAdornment>
              ),
            }}
          />
          {isUniversalSearchMode && (
            <Box sx={{ mt: 1, display: 'flex', gap: 1, alignItems: 'center' }}>
              <Typography variant="body2" color="text.secondary">
                Found {entries.length} results across {searchCollections.length} collections
                {entries.length >= 100 && ' (limited to top 100)'}
              </Typography>
            </Box>
          )}
        </Paper>
      </Box>

      {/* Main layout with 3 columns */}
      <Box sx={{ display: 'flex', gap: 2, height: 'calc(100% - 100px)' }}>
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
            {!selectedCollection && !isUniversalSearchMode ? (
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
                  Select a collection or use universal search
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
