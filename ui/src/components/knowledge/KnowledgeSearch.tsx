import React, { useState, useEffect, useRef } from 'react';
import {
  Box,
  TextField,
  Autocomplete,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Slider,
  Typography,
  Paper,
  Alert,
  CircularProgress,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ClearIcon from '@mui/icons-material/Clear';
import { useKnowledge } from './KnowledgeLayout';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import { knowledgeApi } from '../../services/knowledgeApi';

const RECENT_SEARCHES_KEY = 'knowledgeRecentSearches';
const MAX_RECENT_SEARCHES = 10;

// Load recent searches from localStorage
const loadRecentSearches = (): string[] => {
  try {
    const stored = localStorage.getItem(RECENT_SEARCHES_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
};

// Save recent search to localStorage
const saveRecentSearch = (query: string): void => {
  try {
    const recent = loadRecentSearches();
    const updated = [query, ...recent.filter((q) => q !== query)].slice(0, MAX_RECENT_SEARCHES);
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(updated));
  } catch (error) {
    console.error('Failed to save recent search:', error);
  }
};

export const KnowledgeSearch: React.FC = () => {
  const { selectedCollection, setSelectedCollection, collections, setResults, filters, setFilters } = useKnowledge();
  const [query, setQuery] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [recentSearches, setRecentSearches] = useState<string[]>(loadRecentSearches());
  const queryInputRef = useRef<HTMLInputElement>(null);

  // Auto-fill collection when selected from CollectionBrowser
  useEffect(() => {
    // selectedCollection is already set by context, no need to do anything else
  }, [selectedCollection]);

  // Keyboard shortcut: Cmd+K to focus search
  useKeyboardShortcuts([
    {
      key: 'k',
      metaKey: true,
      handler: () => {
        queryInputRef.current?.focus();
      },
    },
    {
      key: 'Escape',
      handler: () => {
        handleClear();
      },
    },
  ]);

  const handleSearch = async (e?: React.FormEvent) => {
    e?.preventDefault();

    if (!selectedCollection || !query.trim()) {
      setError('Please select a collection and enter a search query');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await knowledgeApi.searchKnowledge({
        collection: selectedCollection,
        query: query.trim(),
        limit: filters.limit,
      });

      setResults(response.results);

      // Save to recent searches
      saveRecentSearch(query.trim());
      setRecentSearches(loadRecentSearches());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleClear = () => {
    setQuery('');
    setResults([]);
    setError(null);
  };

  const handleLimitChange = (_event: Event, newValue: number | number[]) => {
    setFilters({ ...filters, limit: newValue as number });
  };

  return (
    <Paper elevation={2} sx={{ p: 3 }}>
      <Box component="form" onSubmit={handleSearch}>
        <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
          Search Knowledge
        </Typography>

        {/* Collection Select */}
        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel id="collection-select-label">Collection</InputLabel>
          <Select
            labelId="collection-select-label"
            id="collection-select"
            value={selectedCollection}
            label="Collection"
            onChange={(e) => setSelectedCollection(e.target.value)}
          >
            <MenuItem value="">
              <em>Select a collection...</em>
            </MenuItem>
            {collections.map((col) => (
              <MenuItem key={col.name} value={col.name}>
                {col.name} ({col.count} entries)
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Search Query with Autocomplete */}
        <Autocomplete
          freeSolo
          options={recentSearches}
          value={query}
          onInputChange={(_event, newValue) => setQuery(newValue)}
          renderInput={(params) => (
            <TextField
              {...params}
              inputRef={queryInputRef}
              label="Search Query"
              placeholder="Enter search terms... (Cmd+K to focus)"
              required
              fullWidth
              sx={{ mb: 2 }}
            />
          )}
        />

        {/* Result Limit Slider */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Result Limit: {filters.limit}
          </Typography>
          <Slider
            value={filters.limit}
            onChange={handleLimitChange}
            min={5}
            max={20}
            step={5}
            marks
            valueLabelDisplay="auto"
          />
        </Box>

        {/* Action Buttons */}
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            type="submit"
            variant="contained"
            color="primary"
            disabled={loading}
            startIcon={loading ? <CircularProgress size={20} /> : <SearchIcon />}
            fullWidth
          >
            {loading ? 'Searching...' : 'Search'}
          </Button>
          <Button
            type="button"
            variant="outlined"
            color="secondary"
            onClick={handleClear}
            startIcon={<ClearIcon />}
            sx={{ minWidth: '120px' }}
          >
            Clear
          </Button>
        </Box>

        {/* Error Display */}
        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}

        {/* Keyboard Shortcuts Hint */}
        <Box sx={{ mt: 2, p: 1.5, bgcolor: 'background.default', borderRadius: 1 }}>
          <Typography variant="caption" color="text.secondary">
            <strong>Keyboard shortcuts:</strong> Cmd+K to focus search, Esc to clear
          </Typography>
        </Box>
      </Box>
    </Paper>
  );
};
