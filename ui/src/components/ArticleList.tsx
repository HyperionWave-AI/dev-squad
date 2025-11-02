import React from 'react';
import {
  Box,
  Card,
  CardContent,
  CardActionArea,
  Typography,
  Chip,
} from '@mui/material';
import type { KnowledgeEntry } from '../services/knowledgeService';

interface ArticleListProps {
  entries: KnowledgeEntry[];
  selectedEntryId: string | null;
  onSelectEntry: (entry: KnowledgeEntry) => void;
}

export const ArticleList: React.FC<ArticleListProps> = ({
  entries,
  selectedEntryId,
  onSelectEntry,
}) => {
  const getTitle = (text: string): string => {
    const firstLine = text.split('\n')[0];
    // Remove markdown heading markers
    return firstLine.replace(/^#+\s*/, '').trim();
  };

  const getPreview = (text: string): string => {
    // Remove first line (title) and get preview
    const lines = text.split('\n').slice(1);
    const preview = lines.join(' ').trim();

    // Truncate to 150 characters
    if (preview.length > 150) {
      return preview.substring(0, 150) + '...';
    }
    return preview || 'No preview available';
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return '';
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return '';
    }
  };

  if (entries.length === 0) {
    return (
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
          No entries in this collection
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2, height: '100%', overflow: 'auto' }}>
      <Typography variant="h6" gutterBottom>
        {entries.length} {entries.length === 1 ? 'Entry' : 'Entries'}
      </Typography>

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {entries.map((entry) => (
          <Card
            key={entry.id}
            variant={selectedEntryId === entry.id ? 'elevation' : 'outlined'}
            elevation={selectedEntryId === entry.id ? 4 : 0}
            sx={{
              borderColor: selectedEntryId === entry.id ? 'primary.main' : undefined,
              borderWidth: selectedEntryId === entry.id ? 2 : 1,
            }}
          >
            <CardActionArea onClick={() => onSelectEntry(entry)}>
              <CardContent>
                <Typography variant="h6" gutterBottom sx={{ fontSize: '1rem', fontWeight: 600 }}>
                  {getTitle(entry.text)}
                </Typography>

                <Typography variant="body2" color="text.secondary" paragraph>
                  {getPreview(entry.text)}
                </Typography>

                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                  {entry.createdAt && (
                    <Typography variant="caption" color="text.secondary">
                      {formatDate(entry.createdAt)}
                    </Typography>
                  )}

                  {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                    <>
                      {Object.entries(entry.metadata).slice(0, 3).map(([key, value]) => (
                        <Chip
                          key={key}
                          label={`${key}: ${value}`}
                          size="small"
                          variant="outlined"
                        />
                      ))}
                      {Object.keys(entry.metadata).length > 3 && (
                        <Chip
                          label={`+${Object.keys(entry.metadata).length - 3} more`}
                          size="small"
                          variant="outlined"
                        />
                      )}
                    </>
                  )}
                </Box>
              </CardContent>
            </CardActionArea>
          </Card>
        ))}
      </Box>
    </Box>
  );
};
