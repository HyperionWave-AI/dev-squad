import React, { useState } from 'react';
import {
  Card,
  CardContent,
  TextField,
  Button,
  Box,
  Chip,
  Slider,
  Typography,
  Stack,
  InputAdornment,
} from '@mui/material';
import { Search, Clear } from '@mui/icons-material';

interface CodeSearchProps {
  onSearch: (query: string, options: SearchOptions) => void;
  loading?: boolean;
}

export interface SearchOptions {
  fileTypes: string[];
  minScore: number;
  limit: number;
  folderPath?: string;  // Optional: filter results to specific folder
  retrieve?: 'chunk' | 'full';  // Optional: content retrieval mode
}

const FILE_TYPE_OPTIONS = [
  { label: 'Go', value: '.go' },
  { label: 'TypeScript', value: '.ts' },
  { label: 'TSX', value: '.tsx' },
  { label: 'JavaScript', value: '.js' },
  { label: 'Python', value: '.py' },
  { label: 'Java', value: '.java' },
];

export const CodeSearch: React.FC<CodeSearchProps> = ({ onSearch, loading }) => {
  const [query, setQuery] = useState('');
  const [selectedFileTypes, setSelectedFileTypes] = useState<string[]>([]);
  const [minScore, setMinScore] = useState(0.2);
  const [limit, setLimit] = useState(10);

  const handleSearch = () => {
    if (!query.trim()) return;

    onSearch(query, {
      fileTypes: selectedFileTypes,
      minScore,
      limit,
    });
  };

  const handleClear = () => {
    setQuery('');
    setSelectedFileTypes([]);
    setMinScore(0.2);
    setLimit(10);
  };

  const handleFileTypeToggle = (fileType: string) => {
    setSelectedFileTypes((prev) =>
      prev.includes(fileType)
        ? prev.filter((ft) => ft !== fileType)
        : [...prev, fileType]
    );
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSearch();
    }
  };

  return (
    <Card sx={{ width: '100%', maxWidth: '100%', overflow: 'hidden', boxSizing: 'border-box' }}>
      <CardContent sx={{ width: '100%', maxWidth: '100%', boxSizing: 'border-box' }}>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Search />
          Semantic Code Search
        </Typography>

        <Stack spacing={3} sx={{ width: '100%', maxWidth: '100%', boxSizing: 'border-box' }}>
          {/* Search Query */}
          <TextField
            fullWidth
            label="Search Query"
            placeholder="e.g., JWT authentication middleware, database connection pool, React component..."
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
            disabled={loading}
            sx={{ maxWidth: '100%', boxSizing: 'border-box' }}
          />

          {/* File Type Filters */}
          <Box>
            <Typography variant="subtitle2" gutterBottom>
              File Types
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              {FILE_TYPE_OPTIONS.map((option) => (
                <Chip
                  key={option.value}
                  label={option.label}
                  onClick={() => handleFileTypeToggle(option.value)}
                  color={selectedFileTypes.includes(option.value) ? 'primary' : 'default'}
                  variant={selectedFileTypes.includes(option.value) ? 'filled' : 'outlined'}
                  disabled={loading}
                />
              ))}
            </Box>
          </Box>

          {/* Min Score Slider */}
          <Box sx={{ width: '100%', maxWidth: '100%' }}>
            <Typography variant="subtitle2" gutterBottom>
              Minimum Relevance Score: {(minScore * 100).toFixed(0)}%
            </Typography>
            <Slider
              value={minScore}
              onChange={(_, value) => setMinScore(value as number)}
              min={0.0}
              max={0.8}
              step={0.05}
              marks={[
                { value: 0.0, label: '0%' },
                { value: 0.2, label: '20%' },
                { value: 0.4, label: '40%' },
                { value: 0.6, label: '60%' },
                { value: 0.8, label: '80%' },
              ]}
              valueLabelDisplay="auto"
              valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
              disabled={loading}
              sx={{ width: '100%' }}
            />
          </Box>

          {/* Result Limit Slider */}
          <Box sx={{ width: '100%', maxWidth: '100%' }}>
            <Typography variant="subtitle2" gutterBottom>
              Maximum Results: {limit}
            </Typography>
            <Slider
              value={limit}
              onChange={(_, value) => setLimit(value as number)}
              min={5}
              max={50}
              step={5}
              marks={[
                { value: 5, label: '5' },
                { value: 25, label: '25' },
                { value: 50, label: '50' },
              ]}
              valueLabelDisplay="auto"
              disabled={loading}
              sx={{ width: '100%' }}
            />
          </Box>

          {/* Action Buttons */}
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="contained"
              startIcon={<Search />}
              onClick={handleSearch}
              disabled={!query.trim() || loading}
              fullWidth
            >
              {loading ? 'Searching...' : 'Search Code'}
            </Button>
            <Button
              variant="outlined"
              startIcon={<Clear />}
              onClick={handleClear}
              disabled={loading}
            >
              Clear
            </Button>
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
};
