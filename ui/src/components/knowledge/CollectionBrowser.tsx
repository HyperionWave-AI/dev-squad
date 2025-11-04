import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Tabs,
  Tab,
  Skeleton,
  Alert,
  Stack,
  Button,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { useKnowledge } from './KnowledgeLayout';
import { knowledgeApi } from '../../services/knowledgeApi';
import { CreateCollectionModal } from './CreateCollectionModal';

const categoryIcons: Record<string, string> = {
  Tech: '🔧',
  Task: '📋',
  UI: '🎨',
  Ops: '⚙️',
  Other: '📚',
};

export const CollectionBrowser: React.FC = () => {
  const { selectedCollection, setSelectedCollection, collections, setCollections } = useKnowledge();
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [modalOpen, setModalOpen] = useState<boolean>(false);

  useEffect(() => {
    const fetchCollections = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await knowledgeApi.listCollections();
        setCollections(response.collections);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load collections');
        setCollections([]);
      } finally {
        setLoading(false);
      }
    };

    fetchCollections();
  }, [setCollections]);

  const categories = ['All', ...Array.from(new Set(collections.map((c) => c.category)))];
  const filteredCollections =
    selectedCategory === 'All'
      ? collections
      : collections.filter((c) => c.category === selectedCategory);

  const handleCollectionClick = (collectionName: string) => {
    setSelectedCollection(collectionName);
  };

  const handleCategoryChange = (_event: React.SyntheticEvent, newValue: string) => {
    setSelectedCategory(newValue);
  };

  const handleCreateSuccess = () => {
    // Refresh collections after successful creation
    const fetchCollections = async () => {
      try {
        const response = await knowledgeApi.listCollections();
        setCollections(response.collections);
      } catch (err) {
        console.error('Failed to refresh collections:', err);
      }
    };
    fetchCollections();
  };

  if (loading) {
    return (
      <Box>
        <Typography variant="h5" sx={{ mb: 2, fontWeight: 600 }}>
          Knowledge Collections
        </Typography>
        <Stack spacing={2}>
          {[...Array(6)].map((_, index) => (
            <Skeleton key={index} variant="rectangular" height={100} sx={{ borderRadius: 2 }} />
          ))}
        </Stack>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Box>
      {/* Header with Create Button */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h5" sx={{ fontWeight: 600 }}>
          Knowledge Collections
        </Typography>
        <Button
          variant="contained"
          color="primary"
          startIcon={<AddIcon />}
          onClick={() => setModalOpen(true)}
        >
          Create Collection
        </Button>
      </Box>

      {/* Category Tabs */}
      <Tabs
        value={selectedCategory}
        onChange={handleCategoryChange}
        variant="scrollable"
        scrollButtons="auto"
        sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
      >
        {categories.map((category) => (
          <Tab key={category} label={category} value={category} />
        ))}
      </Tabs>

      {/* Collection Grid */}
      <Stack spacing={2}>
        {filteredCollections.length === 0 ? (
          <Alert severity="info">No collections found in this category</Alert>
        ) : (
          filteredCollections.map((collection) => (
            <Card
              key={collection.name}
                sx={{
                  border: 2,
                  borderColor:
                    selectedCollection === collection.name ? 'primary.main' : 'divider',
                  transition: 'all 0.2s',
                  '&:hover': {
                    borderColor: 'primary.light',
                  },
                }}
              >
                <CardActionArea onClick={() => handleCollectionClick(collection.name)}>
                  <CardContent>
                    {/* Collection Header */}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography sx={{ fontSize: '1.5rem' }} component="span">
                          {categoryIcons[collection.category] || categoryIcons.Other}
                        </Typography>
                        <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
                          {collection.name}
                        </Typography>
                      </Box>
                      <Chip
                        label={collection.count}
                        color="primary"
                        size="small"
                        sx={{ fontWeight: 600 }}
                      />
                    </Box>

                    {/* Collection Description */}
                    {collection.description && (
                      <Tooltip title={collection.description} arrow placement="top">
                        <Typography
                          variant="body2"
                          color="text.secondary"
                          sx={{
                            mb: 1,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {collection.description}
                        </Typography>
                      </Tooltip>
                    )}

                    {/* Collection Category and Tags */}
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center' }}>
                      <Chip
                        label={collection.category}
                        size="small"
                        variant="outlined"
                        sx={{ fontSize: '0.75rem' }}
                      />
                      {collection.tags && collection.tags.map((tag) => (
                        <Chip
                          key={tag}
                          label={tag}
                          size="small"
                          variant="filled"
                          color="default"
                          sx={{ fontSize: '0.7rem' }}
                        />
                      ))}
                    </Box>
                  </CardContent>
                </CardActionArea>
            </Card>
          ))
        )}
      </Stack>

      {/* Summary */}
      {filteredCollections.length > 0 && (
        <Box
          sx={{
            mt: 3,
            p: 2,
            bgcolor: 'background.default',
            borderRadius: 1,
            border: 1,
            borderColor: 'divider',
          }}
        >
          <Typography variant="body2" color="text.secondary">
            <strong>
              {filteredCollections.length} collection
              {filteredCollections.length !== 1 ? 's' : ''}
            </strong>
            {' · '}
            <span>{filteredCollections.reduce((sum, c) => sum + c.count, 0)} total entries</span>
          </Typography>
        </Box>
      )}

      {/* Create Collection Modal */}
      <CreateCollectionModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSuccess={handleCreateSuccess}
      />
    </Box>
  );
};
