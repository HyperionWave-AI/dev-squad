import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Pagination,
  Alert,
  IconButton,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import type { KnowledgeEntry } from '../../types/knowledge';
import { useKnowledge } from './KnowledgeLayout';

const RESULTS_PER_PAGE = 10;

// Code block regex: ```language\ncode```
const CODE_BLOCK_REGEX = /```(\w+)?\n([\s\S]*?)```/g;

interface SearchResultsProps {
  // Can accept results as prop or use from context
  results?: KnowledgeEntry[];
}

// Get score color based on value
const getScoreColor = (score: number): 'success' | 'warning' | 'error' => {
  if (score > 0.8) return 'success';
  if (score > 0.5) return 'warning';
  return 'error';
};

// Parse text and render with syntax highlighting for code blocks
const renderTextWithHighlighting = (text: string): React.ReactNode => {
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  // Reset regex lastIndex
  CODE_BLOCK_REGEX.lastIndex = 0;

  while ((match = CODE_BLOCK_REGEX.exec(text)) !== null) {
    // Add text before code block
    if (match.index > lastIndex) {
      parts.push(
        <Typography
          key={`text-${lastIndex}`}
          variant="body2"
          component="div"
          sx={{ whiteSpace: 'pre-wrap', mb: 1 }}
        >
          {text.slice(lastIndex, match.index)}
        </Typography>
      );
    }

    // Add code block with syntax highlighting
    const language = match[1] || 'text';
    const code = match[2];
    parts.push(
      <Box key={`code-${match.index}`} sx={{ mb: 2 }}>
        <SyntaxHighlighter
          language={language}
          style={vscDarkPlus}
          customStyle={{
            borderRadius: '8px',
            fontSize: '0.875rem',
          }}
        >
          {code}
        </SyntaxHighlighter>
      </Box>
    );

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < text.length) {
    parts.push(
      <Typography
        key={`text-${lastIndex}`}
        variant="body2"
        component="div"
        sx={{ whiteSpace: 'pre-wrap' }}
      >
        {text.slice(lastIndex)}
      </Typography>
    );
  }

  return parts.length > 0 ? parts : (
    <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
      {text}
    </Typography>
  );
};

export const SearchResults: React.FC<SearchResultsProps> = ({ results: propResults }) => {
  const { results: contextResults } = useKnowledge();
  const [page, setPage] = useState<number>(1);
  const [expanded, setExpanded] = useState<string | false>(false);
  const [expandedContent, setExpandedContent] = useState<Set<string>>(new Set());

  // Use prop results or context results
  const results = propResults ?? contextResults;

  // Paginate results
  const paginatedResults = useMemo(() => {
    const startIndex = (page - 1) * RESULTS_PER_PAGE;
    const endIndex = startIndex + RESULTS_PER_PAGE;
    return results.slice(startIndex, endIndex);
  }, [results, page]);

  const totalPages = Math.ceil(results.length / RESULTS_PER_PAGE);

  const handleAccordionChange = (panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
    setExpanded(isExpanded ? panel : false);
  };

  const handlePageChange = (_event: React.ChangeEvent<unknown>, value: number) => {
    setPage(value);
    setExpanded(false); // Collapse all when changing page
    setExpandedContent(new Set()); // Clear expanded content state
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const toggleContentExpansion = (entryId: string) => {
    setExpandedContent((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(entryId)) {
        newSet.delete(entryId);
      } else {
        newSet.add(entryId);
      }
      return newSet;
    });
  };

  if (results.length === 0) {
    return null;
  }

  return (
    <Box sx={{ mt: 3 }}>
      {/* Results Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Search Results ({results.length})
        </Typography>
        {totalPages > 1 && (
          <Typography variant="body2" color="text.secondary">
            Page {page} of {totalPages}
          </Typography>
        )}
      </Box>

      {/* Results List */}
      <Box sx={{ mb: 3 }}>
        {paginatedResults.map((entry, index) => {
          const panelId = `result-${entry.id}`;
          const textPreview = entry.text.slice(0, 200) + (entry.text.length > 200 ? '...' : '');

          return (
            <Accordion
              key={entry.id}
              expanded={expanded === panelId}
              onChange={handleAccordionChange(panelId)}
              sx={{ mb: 1 }}
            >
              <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                aria-controls={`${panelId}-content`}
                id={`${panelId}-header`}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, width: '100%', pr: 2 }}>
                  {/* Result number */}
                  <Chip
                    label={`#${(page - 1) * RESULTS_PER_PAGE + index + 1}`}
                    size="small"
                    color="primary"
                    sx={{ minWidth: '48px' }}
                  />

                  {/* Text preview */}
                  <Typography
                    variant="body2"
                    sx={{
                      flex: 1,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {textPreview}
                  </Typography>

                  {/* Score badge */}
                  {entry.score !== undefined && (
                    <Chip
                      label={entry.score.toFixed(3)}
                      size="small"
                      color={getScoreColor(entry.score)}
                      sx={{ fontWeight: 600 }}
                    />
                  )}
                </Box>
              </AccordionSummary>

              <AccordionDetails>
                {/* Full text with syntax highlighting and expand/collapse */}
                <Box sx={{ mb: 2 }}>
                  {(() => {
                    const isExpanded = expandedContent.has(entry.id);
                    const lines = entry.text.split('\n');
                    const needsExpansion = lines.length > 10;
                    const displayText = !isExpanded && needsExpansion
                      ? lines.slice(0, 10).join('\n')
                      : entry.text;

                    return (
                      <>
                        <Box
                          sx={{
                            position: 'relative',
                            ...(needsExpansion && !isExpanded && {
                              '&::after': {
                                content: '""',
                                position: 'absolute',
                                bottom: 0,
                                left: 0,
                                right: 0,
                                height: '40px',
                                background: 'linear-gradient(to bottom, transparent, rgba(18, 18, 18, 0.9))',
                                pointerEvents: 'none',
                              },
                            }),
                          }}
                        >
                          {renderTextWithHighlighting(displayText)}
                        </Box>

                        {needsExpansion && (
                          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
                            <IconButton
                              onClick={() => toggleContentExpansion(entry.id)}
                              size="small"
                              sx={{
                                border: 1,
                                borderColor: 'divider',
                                borderRadius: 1,
                                px: 2,
                                gap: 1,
                              }}
                            >
                              {isExpanded ? (
                                <>
                                  <ExpandLessIcon fontSize="small" />
                                  <Typography variant="caption">Show Less</Typography>
                                </>
                              ) : (
                                <>
                                  <ExpandMoreIcon fontSize="small" />
                                  <Typography variant="caption">
                                    Show More ({lines.length - 10} more lines)
                                  </Typography>
                                </>
                              )}
                            </IconButton>
                          </Box>
                        )}
                      </>
                    );
                  })()}
                </Box>

                {/* Metadata */}
                {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                  <Box sx={{ mt: 2, pt: 2, borderTop: 1, borderColor: 'divider' }}>
                    <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
                      Metadata:
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                      {Object.entries(entry.metadata).map(([key, value]) => (
                        <Chip
                          key={key}
                          label={`${key}: ${String(value)}`}
                          size="small"
                          variant="outlined"
                        />
                      ))}
                    </Box>
                  </Box>
                )}

                {/* Created date */}
                {entry.createdAt && (
                  <Typography variant="caption" color="text.secondary" sx={{ mt: 2, display: 'block' }}>
                    Created: {new Date(entry.createdAt).toLocaleString()}
                  </Typography>
                )}
              </AccordionDetails>
            </Accordion>
          );
        })}
      </Box>

      {/* Pagination */}
      {totalPages > 1 && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 3 }}>
          <Pagination
            count={totalPages}
            page={page}
            onChange={handlePageChange}
            color="primary"
            size="large"
            showFirstButton
            showLastButton
          />
        </Box>
      )}

      {/* No Results Message */}
      {results.length === 0 && (
        <Alert severity="info">
          No results found. Try adjusting your search query or selecting a different collection.
        </Alert>
      )}
    </Box>
  );
};
