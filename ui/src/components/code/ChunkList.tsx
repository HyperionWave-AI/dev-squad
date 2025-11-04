import React, { useState, lazy, Suspense } from 'react';
import {
  List,
  ListItem,
  ListItemButton,
  Box,
  Typography,
  Chip,
  Collapse,
  CircularProgress,
} from '@mui/material';
import {
  Code as CodeIcon,
  Class as ClassIcon,
  ExpandMore,
  ChevronRight,
} from '@mui/icons-material';
import type { FileChunkDetails } from '../../types/codeIndex';

// Lazy load syntax highlighter
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then((mod) => ({
    default: mod.Prism,
  }))
);

import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface ChunkListProps {
  chunks: FileChunkDetails[];
  fileId: string;
  language: string;
}

export const ChunkList: React.FC<ChunkListProps> = ({ chunks, language }) => {
  const [expandedChunks, setExpandedChunks] = useState<Set<number>>(new Set());
  const [chunkContents, setChunkContents] = useState<Map<number, string>>(new Map());
  const [loadingChunks, setLoadingChunks] = useState<Set<number>>(new Set());

  const toggleChunk = async (chunkNum: number) => {
    const isExpanding = !expandedChunks.has(chunkNum);

    // Update expanded state
    const newExpanded = new Set(expandedChunks);
    if (isExpanding) {
      newExpanded.add(chunkNum);

      // Load content if not already loaded
      if (!chunkContents.has(chunkNum)) {
        setLoadingChunks(prev => new Set(prev).add(chunkNum));
        try {
          // Note: We need to fetch the full chunk content
          // For now, we'll show a placeholder
          // TODO: Add endpoint to get individual chunk content
          const content = `// Chunk ${chunkNum} content would be loaded here\n// This requires a backend endpoint to fetch chunk content`;
          setChunkContents(prev => new Map(prev).set(chunkNum, content));
        } catch (error) {
          console.error('Failed to load chunk content:', error);
        } finally {
          setLoadingChunks(prev => {
            const newSet = new Set(prev);
            newSet.delete(chunkNum);
            return newSet;
          });
        }
      }
    } else {
      newExpanded.delete(chunkNum);
    }
    setExpandedChunks(newExpanded);
  };

  const getNodeIcon = (nodeType?: string) => {
    if (!nodeType) return undefined;
    const type = nodeType.toLowerCase();
    if (type.includes('function') || type.includes('method')) {
      return <CodeIcon fontSize="small" />;
    }
    if (type.includes('class') || type.includes('interface') || type.includes('struct')) {
      return <ClassIcon fontSize="small" />;
    }
    return <CodeIcon fontSize="small" />;
  };

  if (chunks.length === 0) {
    return (
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          No chunks found
        </Typography>
      </Box>
    );
  }

  return (
    <List sx={{ width: '100%', p: 0 }}>
      {chunks.map((chunk) => {
        const isExpanded = expandedChunks.has(chunk.chunkNum);
        const isLoading = loadingChunks.has(chunk.chunkNum);
        const content = chunkContents.get(chunk.chunkNum);

        return (
          <React.Fragment key={chunk.chunkNum}>
            <ListItem disablePadding>
              <ListItemButton
                onClick={() => toggleChunk(chunk.chunkNum)}
                sx={{
                  flexDirection: 'column',
                  alignItems: 'flex-start',
                  py: 1.5,
                  borderBottom: '1px solid',
                  borderColor: 'divider',
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%', mb: 1 }}>
                  {isExpanded ? <ExpandMore fontSize="small" /> : <ChevronRight fontSize="small" />}
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    Chunk #{chunk.chunkNum}
                  </Typography>
                  <Chip
                    label={`Lines ${chunk.startLine}-${chunk.endLine}`}
                    size="small"
                    variant="outlined"
                    sx={{ ml: 'auto' }}
                  />
                </Box>

                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', ml: 3 }}>
                  {/* Chunk Type Badge */}
                  <Chip
                    label={chunk.chunkType === 'ast' ? 'AST' : 'Line-based'}
                    size="small"
                    color={chunk.chunkType === 'ast' ? 'success' : 'default'}
                    sx={{ fontSize: '0.75rem' }}
                  />

                  {/* Node Type Badge */}
                  {chunk.nodeType && (
                    <Chip
                      icon={getNodeIcon(chunk.nodeType)}
                      label={chunk.nodeType}
                      size="small"
                      color="info"
                      sx={{ fontSize: '0.75rem' }}
                    />
                  )}

                  {/* Node Name */}
                  {chunk.nodeName && (
                    <Typography
                      variant="caption"
                      sx={{
                        fontFamily: 'monospace',
                        bgcolor: 'action.hover',
                        px: 1,
                        py: 0.5,
                        borderRadius: 1,
                      }}
                    >
                      {chunk.nodeName}
                    </Typography>
                  )}

                  {/* Signature */}
                  {chunk.signature && (
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      sx={{
                        fontFamily: 'monospace',
                        fontStyle: 'italic',
                        maxWidth: '400px',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {chunk.signature}
                    </Typography>
                  )}
                </Box>
              </ListItemButton>
            </ListItem>

            {/* Expanded Content */}
            <Collapse in={isExpanded} timeout="auto" unmountOnExit>
              <Box sx={{ p: 2, bgcolor: 'action.hover' }}>
                {isLoading ? (
                  <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                    <CircularProgress size={24} />
                  </Box>
                ) : content ? (
                  <Suspense
                    fallback={
                      <Box sx={{ p: 2, bgcolor: '#1e1e1e', borderRadius: 1 }}>
                        <Typography variant="caption" color="rgba(255,255,255,0.7)">
                          Loading syntax highlighter...
                        </Typography>
                      </Box>
                    }
                  >
                    <SyntaxHighlighter
                      language={language}
                      style={vscDarkPlus}
                      customStyle={{
                        borderRadius: 8,
                        fontSize: '0.875rem',
                        margin: 0,
                      }}
                      showLineNumbers
                      startingLineNumber={chunk.startLine}
                      wrapLines
                    >
                      {content}
                    </SyntaxHighlighter>
                  </Suspense>
                ) : (
                  <Typography variant="caption" color="text.secondary">
                    Content not available
                  </Typography>
                )}
              </Box>
            </Collapse>
          </React.Fragment>
        );
      })}
    </List>
  );
};
