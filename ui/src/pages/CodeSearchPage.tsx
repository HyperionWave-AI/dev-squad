import React, { useState } from 'react';
import { Container, Box, Typography, Stack } from '@mui/material';
import {
  CodeIndexConfig,
  CodeSearch,
  CodeResults,
  IndexStatus,
} from '../components/code';
import type { SearchResult } from '../types/codeIndex';
import type { SearchOptions } from '../components/code/CodeSearch';
import { restCodeClient } from '../services/restCodeClient';

export const CodeSearchPage: React.FC = () => {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  const handleSearch = async (query: string, options: SearchOptions) => {
    try {
      setLoading(true);
      const searchResults = await restCodeClient.search(query, {
        limit: options.limit,
      });
      setResults(searchResults);
    } catch (err) {
      console.error('Search failed:', err);
      alert(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        bgcolor: 'background.default',
        py: 4,
      }}
    >
      <Container maxWidth="xl">
        {/* Page Header */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h3" sx={{ fontWeight: 700, mb: 1 }}>
            Code Search
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Semantic search across your indexed code repositories
          </Typography>
        </Box>

        {/* Main Layout: Search + Results (left 2/3) + Config + Status (right 1/3) */}
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', md: '2fr 1fr' },
            gap: 3,
          }}
        >
          {/* Search and Results Section */}
          <Stack spacing={3}>
            {/* Search Form */}
            <CodeSearch onSearch={handleSearch} loading={loading} />

            {/* Search Results */}
            <CodeResults results={results} loading={loading} />
          </Stack>

          {/* Configuration and Status Sidebar */}
          <Stack spacing={3}>
            {/* Folder Configuration */}
            <CodeIndexConfig />

            {/* Index Status */}
            <IndexStatus />
          </Stack>
        </Box>

        {/* Keyboard Shortcuts Hint */}
        <Box
          sx={{
            mt: 6,
            p: 3,
            bgcolor: 'primary.main',
            color: 'primary.contrastText',
            borderRadius: 2,
            textAlign: 'center',
          }}
        >
          <Typography variant="h6" sx={{ fontWeight: 600, mb: 1 }}>
            Keyboard Shortcuts
          </Typography>
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'center',
              gap: 4,
              flexWrap: 'wrap',
            }}
          >
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Cmd + K
              </Typography>
              <Typography variant="caption">Focus search</Typography>
            </Box>
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Esc
              </Typography>
              <Typography variant="caption">Clear search</Typography>
            </Box>
          </Box>
        </Box>
      </Container>
    </Box>
  );
};
