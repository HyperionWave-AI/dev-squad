import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Select,
  MenuItem,
  TextField,
  Button,
  CircularProgress,
  Alert,
  InputAdornment,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Divider,
} from '@mui/material';
import { Search } from '@mui/icons-material';
import type { KnowledgeEntry } from '../types/coordinator';
import { restClient } from '../services/restClient';

export const KnowledgeBrowser: React.FC = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<KnowledgeEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [collection, setCollection] = useState('All Collections');
  const [limit, setLimit] = useState(10);
  const [popularCollections, setPopularCollections] = useState<Array<{ collection: string; count: number }>>([]);
  const [collections, setCollections] = useState<string[]>(['All Collections']);

  useEffect(() => {
    const loadAllCollections = async () => {
      try {
        const allCollections = await restClient.getAllCollections();
        // Filter out task-specific collections (task:hyperion://...) and extract unique collection names
        const nonTaskCollections = allCollections
          .filter(c => !c.name.startsWith('task:hyperion://'))
          .map(c => c.name);
        const uniqueNames = ['All Collections', ...new Set(nonTaskCollections)];
        setCollections(uniqueNames);
      } catch (err) {
        console.error('Failed to load collections:', err);
      }
    };

    const loadPopularCollections = async () => {
      try {
        const popular = await restClient.getPopularCollections(5);
        setPopularCollections(popular.filter(c => !c.collection.startsWith('task:')));
      } catch (err) {
        console.error('Failed to load popular collections:', err);
      }
    };

    const loadInitialKnowledge = async () => {
      try {
        setLoading(true);
        const entries = await restClient.browseKnowledge(undefined, limit);
        setResults(entries);
      } catch (err) {
        console.error('Failed to load initial knowledge:', err);
      } finally {
        setLoading(false);
      }
    };

    loadAllCollections();
    loadPopularCollections();
    loadInitialKnowledge();
  }, [limit]);

  const handleBrowse = async () => {
    try {
      setLoading(true);
      setError(null);

      const col = collection === 'All Collections' ? undefined : collection;
      const entries = await restClient.browseKnowledge(col, limit);
      setResults(entries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Browse failed');
      console.error('Browse error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    // If no query, browse instead
    if (!query.trim()) {
      handleBrowse();
      return;
    }

    try {
      setLoading(true);
      setError(null);

      let allEntries: KnowledgeEntry[] = [];

      if (collection === 'All Collections' || !collection) {
        // Search all major collections and aggregate results
        // Note: task:hyperion://... collections are per-task, too many to query individually
        const collectionsToSearch = [
          'adr',
          'technical-knowledge',
          'npm-package-testing',
          'data-contracts',
          'team-coordination',
          'workflow-context'
        ];

        const searchPromises = collectionsToSearch.map(async (col) => {
          try {
            return await restClient.queryKnowledge(col, query, 5);
          } catch (err) {
            console.warn(`Failed to search collection ${col}:`, err);
            return [];
          }
        });

        const results = await Promise.all(searchPromises);
        allEntries = results.flat();

        // Sort by score (highest first)
        allEntries.sort((a, b) => (b.score || 0) - (a.score || 0));

        // Limit total results
        allEntries = allEntries.slice(0, limit);
      } else {
        // Search single collection
        allEntries = await restClient.queryKnowledge(collection, query, limit);
      }

      setResults(allEntries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch();
  };

  return (
    <Box sx={{ width: '100%' }}>
      {/* Header */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h5" sx={{ fontWeight: 600, mb: 0.5 }}>
          Knowledge Browser
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Search across knowledge collections
        </Typography>
      </Box>

      {/* Search Section */}
      <Paper
        elevation={0}
        sx={{
          p: 3,
          mb: 3,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 2,
          backgroundColor: 'white',
        }}
      >
        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mb: 2 }}>
          <Select
            value={collection}
            onChange={(e) => {
              setCollection(e.target.value);
              // Trigger browse with new collection
              if (!query.trim()) {
                setTimeout(() => handleBrowse(), 0);
              }
            }}
            displayEmpty
            size="small"
            sx={{ minWidth: 200 }}
          >
            {collections.map((col) => (
              <MenuItem key={col} value={col}>
                {col}
              </MenuItem>
            ))}
          </Select>

          <Select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            size="small"
            sx={{ minWidth: 100 }}
          >
            <MenuItem value={10}>10 results</MenuItem>
            <MenuItem value={50}>50 results</MenuItem>
            <MenuItem value={100}>100 results</MenuItem>
          </Select>

          <TextField
            fullWidth
            placeholder="Search knowledge base (leave empty to browse)..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyPress={handleKeyPress}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <Search color="action" />
                </InputAdornment>
              ),
            }}
          />

          <Button
            variant="contained"
            color="primary"
            onClick={handleSearch}
            disabled={loading}
            sx={{ px: 4 }}
          >
            {loading ? <CircularProgress size={20} sx={{ color: 'white' }} /> : query.trim() ? 'Search' : 'Browse'}
          </Button>
        </Box>
      </Paper>

      {/* Error */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Results */}
      {loading && (
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'center',
            py: 6,
          }}
        >
          <CircularProgress />
        </Box>
      )}

      {!loading && results.length > 0 && (
        <Box sx={{ mb: 4 }}>
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              mb: 2,
            }}
          >
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
              {results.length} {query.trim() ? 'result' : 'entry'}{results.length !== 1 ? 's' : ''} {query.trim() ? 'found' : ''}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {query.trim() ? 'in' : 'from'} {collection || 'All Collections'}
            </Typography>
          </Box>

          {results.map((entry, index) => (
            <Card
              key={entry.id}
              variant="outlined"
              sx={{
                mb: 2,
                borderRadius: 2,
                transition: 'all 0.2s ease',
                '&:hover': { boxShadow: 3, borderColor: 'primary.light' },
              }}
            >
              <CardHeader
                titleTypographyProps={{ variant: 'subtitle1', fontWeight: 500 }}
                title={entry.collection}
                subheader={`📅 ${new Date(entry.createdAt).toLocaleDateString()}`}
                sx={{
                  background: 'linear-gradient(to right, #eff6ff, #ffffff)',
                  pb: 1,
                }}
                action={
                  <Chip
                    label={`#${index + 1}`}
                    size="small"
                    color="primary"
                    sx={{ fontWeight: 600 }}
                  />
                }
              />
              <Divider />
              <CardContent>
                <Typography
                  variant="body2"
                  sx={{
                    color: 'text.primary',
                    mb: 2,
                    whiteSpace: 'pre-wrap',
                  }}
                >
                  {entry.text}
                </Typography>

                {/* Tags from metadata or direct tags field */}
                {((entry.metadata?.tags as string[]) || entry.tags || []).length > 0 && (
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
                    {((entry.metadata?.tags as string[]) || entry.tags || []).map((tag) => (
                      <Chip
                        key={tag}
                        label={tag}
                        variant="outlined"
                        size="small"
                        sx={{ borderRadius: 1 }}
                      />
                    ))}
                  </Box>
                )}

                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    borderTop: '1px solid',
                    borderColor: 'divider',
                    pt: 1.5,
                  }}
                >
                  {entry.score !== undefined && (
                    <Typography variant="caption" color="text.secondary">
                      🎯 Relevance:{' '}
                      <Typography component="span" fontWeight={500}>
                        {(entry.score * 100).toFixed(0)}%
                      </Typography>
                    </Typography>
                  )}

                  {entry.createdBy && (
                    <Typography variant="caption" color="text.secondary">
                      👤 Created by{' '}
                      <Typography component="span" fontWeight={500}>
                        {entry.createdBy}
                      </Typography>
                    </Typography>
                  )}

                  {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                    <details>
                      <summary className="cursor-pointer text-blue-600 text-xs">
                        📋 View metadata
                      </summary>
                      <Box
                        sx={{
                          mt: 1,
                          backgroundColor: 'grey.50',
                          borderRadius: 1,
                          p: 1,
                        }}
                      >
                        <pre style={{ fontSize: 11, overflowX: 'auto' }}>
                          {JSON.stringify(entry.metadata, null, 2)}
                        </pre>
                      </Box>
                    </details>
                  )}
                </Box>
              </CardContent>
            </Card>
          ))}
        </Box>
      )}

      {/* Empty States */}
      {!loading && results.length === 0 && query && !error && (
        <Paper
          elevation={0}
          sx={{
            py: 10,
            textAlign: 'center',
            border: '2px dashed',
            borderColor: 'divider',
            borderRadius: 3,
            backgroundColor: 'grey.50',
          }}
        >
          <Typography variant="h3" sx={{ mb: 1 }}>
            🔍
          </Typography>
          <Typography variant="h6" sx={{ mb: 0.5 }}>
            No results found
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Try different keywords or another collection
          </Typography>
        </Paper>
      )}

      {!loading && results.length === 0 && !query && !error && (
        <Paper
          elevation={0}
          sx={{
            p: 4,
            background: 'linear-gradient(to bottom right, #eff6ff, #f0f9ff)',
            borderRadius: 3,
            border: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
            💡 Quick Start
          </Typography>
          <Typography variant="body2" sx={{ color: 'text.secondary', mb: 2 }}>
            {popularCollections.length > 0
              ? 'Browse popular collections:'
              : 'Try these example searches:'}
          </Typography>

          <Box sx={{ display: 'grid', gap: 1.5, gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
            {popularCollections.length > 0 ? (
              popularCollections.map((popular) => (
                <Button
                  key={popular.collection}
                  variant="outlined"
                  color="primary"
                  fullWidth
                  onClick={() => {
                    setCollection(popular.collection);
                  }}
                  sx={{
                    justifyContent: 'space-between',
                    textTransform: 'none',
                    borderRadius: 2,
                  }}
                >
                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                    {popular.collection}
                  </Typography>
                  <Chip
                    label={`${popular.count} entries`}
                    size="small"
                    sx={{ height: 20, fontSize: 11 }}
                  />
                </Button>
              ))
            ) : (
              [
                { query: 'JWT authentication', collection: 'technical-knowledge' },
                { query: 'React component', collection: 'technical-knowledge' },
                { query: 'file upload', collection: 'technical-knowledge' },
                { query: 'MongoDB', collection: 'adr' },
              ].map((example) => (
                <Button
                  key={example.query}
                  variant="outlined"
                  color="primary"
                  fullWidth
                  onClick={() => {
                    setQuery(example.query);
                    setCollection(example.collection);
                  }}
                  sx={{
                    justifyContent: 'flex-start',
                    textTransform: 'none',
                    borderRadius: 2,
                  }}
                >
                  "{example.query}"{' '}
                  <Typography variant="caption" sx={{ ml: 1 }}>
                    in {example.collection}
                  </Typography>
                </Button>
              ))
            )}
          </Box>
        </Paper>
      )}
    </Box>
  );
};
