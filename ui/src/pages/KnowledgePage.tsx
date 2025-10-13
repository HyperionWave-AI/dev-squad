import React from 'react';
import { Container, Box, Typography, Stack } from '@mui/material';
import {
  KnowledgeProvider,
  KnowledgeSearch,
  CollectionBrowser,
  SearchResults,
} from '../components/knowledge';

export const KnowledgePage: React.FC = () => {
  return (
    <KnowledgeProvider>
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
              Knowledge Base
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Search, browse, and explore knowledge entries across collections
            </Typography>
          </Box>

          {/* Main Layout: Search + Results (left) + Collections (right) */}
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', md: '2fr 1fr' },
              gap: 3,
            }}
          >
            {/* Search and Results Section (2/3 width on medium+ screens) */}
            <Stack spacing={3}>
              {/* Search Form */}
              <KnowledgeSearch />

              {/* Search Results */}
              <SearchResults />
            </Stack>

            {/* Collections Browser (1/3 width on medium+ screens) */}
            <Box>
              <CollectionBrowser />
            </Box>
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
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>
                  Click collection
                </Typography>
                <Typography variant="caption">Auto-select for search</Typography>
              </Box>
            </Box>
          </Box>
        </Container>
      </Box>
    </KnowledgeProvider>
  );
};
