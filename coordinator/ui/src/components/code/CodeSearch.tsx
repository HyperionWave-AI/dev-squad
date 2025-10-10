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
  const [minScore, setMinScore] = useState(0.5);
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
    setMinScore(0.5);
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
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Search />
          Semantic Code Search
        </Typography>

        <Stack spacing={3}>
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
          <Box>
            <Typography variant="subtitle2" gutterBottom>
              Minimum Relevance Score: {(minScore * 100).toFixed(0)}%
            </Typography>
            <Slider
              value={minScore}
              onChange={(_, value) => setMinScore(value as number)}
              min={0.5}
              max={1.0}
              step={0.05}
              marks={[
                { value: 0.5, label: '50%' },
                { value: 0.75, label: '75%' },
                { value: 1.0, label: '100%' },
              ]}
              valueLabelDisplay="auto"
              valueLabelFormat={(value) => `${(value * 100).toFixed(0)}%`}
              disabled={loading}
            />
          </Box>

          {/* Result Limit Slider */}
          <Box>
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
