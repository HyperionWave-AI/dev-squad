import React, { lazy, Suspense } from 'react';
import {
  Card,
  CardContent,
  Typography,
  LinearProgress,
  Chip,
  Box,
  IconButton,
  Tooltip,
} from '@mui/material';
import { ContentCopy, Description } from '@mui/icons-material';
import type { SearchResult } from '../../types/codeIndex';

// Lazy load syntax highlighter to reduce bundle size
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then((mod) => ({
    default: mod.Prism,
  }))
);

// Import theme - using vscDarkPlus which is available in prism styles
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface CodeResultsProps {
  results: SearchResult[];
  loading?: boolean;
}

export const CodeResults: React.FC<CodeResultsProps> = ({ results, loading }) => {
  const handleCopyPath = (filePath: string) => {
    navigator.clipboard.writeText(filePath);
  };

  const getLanguageForFile = (fileName: string): string => {
    const ext = fileName.split('.').pop()?.toLowerCase();
    const languageMap: Record<string, string> = {
      go: 'go',
      ts: 'typescript',
      tsx: 'tsx',
      js: 'javascript',
      jsx: 'jsx',
      py: 'python',
      java: 'java',
      c: 'c',
      cpp: 'cpp',
      h: 'c',
      hpp: 'cpp',
      rs: 'rust',
      rb: 'ruby',
      php: 'php',
      css: 'css',
      html: 'html',
      json: 'json',
      yaml: 'yaml',
      yml: 'yaml',
      md: 'markdown',
      sh: 'bash',
    };
    return languageMap[ext || ''] || 'text';
  };

  if (loading) {
    return (
      <Card>
        <CardContent>
          <LinearProgress />
          <Typography variant="body2" color="text.secondary" sx={{ mt: 2, textAlign: 'center' }}>
            Searching...
          </Typography>
        </CardContent>
      </Card>
    );
  }

  if (results.length === 0) {
    return (
      <Card>
        <CardContent>
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Description sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
            <Typography variant="h6" color="text.secondary">
              No results found
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Try adjusting your search query or filters
            </Typography>
          </Box>
        </CardContent>
      </Card>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Typography variant="h6" sx={{ mb: 1 }}>
        Search Results ({results.length})
      </Typography>

      {results.map((result, index) => {
        const language = result.language || getLanguageForFile(result.fileName);
        const scorePercentage = Math.round(result.score * 100);

        return (
          <Card key={index}>
            <CardContent>
              {/* File Header */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, flexWrap: 'wrap' }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, flexGrow: 1 }}>
                  {result.fileName}
                </Typography>
                <Chip
                  label={language.toUpperCase()}
                  size="small"
                  color="primary"
                  variant="outlined"
                />
                <Chip
                  label={`${result.lines} lines`}
                  size="small"
                  variant="outlined"
                />
              </Box>

              {/* Score Bar */}
              <Box sx={{ mb: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                  <Typography variant="caption" color="text.secondary">
                    Relevance Score:
                  </Typography>
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>
                    {scorePercentage}%
                  </Typography>
                </Box>
                <LinearProgress
                  variant="determinate"
                  value={scorePercentage}
                  sx={{ height: 6, borderRadius: 1 }}
                />
              </Box>

              {/* File Path */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{
                    fontFamily: 'monospace',
                    flexGrow: 1,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {result.filePath}
                </Typography>
                <Tooltip title="Copy path">
                  <IconButton
                    size="small"
                    onClick={() => handleCopyPath(result.filePath)}
                  >
                    <ContentCopy fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Box>

              {/* Code Excerpt */}
              {result.excerpt && (
                <Box sx={{ mt: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                    Code Excerpt:
                  </Typography>
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
                      wrapLines
                    >
                      {result.excerpt}
                    </SyntaxHighlighter>
                  </Suspense>
                </Box>
              )}
            </CardContent>
          </Card>
        );
      })}
    </Box>
  );
};
