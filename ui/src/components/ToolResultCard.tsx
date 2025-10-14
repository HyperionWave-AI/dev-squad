/**
 * ToolResultCard Component
 *
 * Displays tool execution results with type-specific rendering:
 * - Bash: Terminal output with ANSI colors
 * - ReadFile: Code viewer with syntax highlighting
 * - WriteFile: Success message
 * - ListDirectory: File table
 * - ApplyPatch: Diff viewer
 */

import { useState } from 'react';
import {
  Card,
  CardContent,
  Alert,
  Chip,
  Typography,
  Box,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Collapse,
  IconButton,
} from '@mui/material';
import { CheckCircle, Error as ErrorIcon, ExpandMore } from '@mui/icons-material';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { AnsiUp } from 'ansi_up';

interface ToolResultCardProps {
  tool: string;
  result: any;
  error: string | null;
  durationMs: number;
}

export function ToolResultCard({
  tool,
  result,
  error,
  durationMs,
}: ToolResultCardProps) {
  const [expanded, setExpanded] = useState(false);

  const handleExpandClick = () => {
    setExpanded(!expanded);
  };

  const getDurationColor = () => {
    if (durationMs < 1000) return 'success';
    if (durationMs < 5000) return 'warning';
    return 'error';
  };

  const formatDuration = () => {
    if (durationMs < 1000) return `${durationMs}ms`;
    return `${(durationMs / 1000).toFixed(2)}s`;
  };

  const detectLanguageFromPath = (path: string): string => {
    const ext = path.split('.').pop()?.toLowerCase();
    const langMap: Record<string, string> = {
      js: 'javascript',
      ts: 'typescript',
      tsx: 'tsx',
      jsx: 'jsx',
      py: 'python',
      go: 'go',
      java: 'java',
      cpp: 'cpp',
      c: 'c',
      rs: 'rust',
      rb: 'ruby',
      php: 'php',
      sh: 'bash',
      yaml: 'yaml',
      yml: 'yaml',
      json: 'json',
      xml: 'xml',
      html: 'html',
      css: 'css',
      md: 'markdown',
      sql: 'sql',
    };
    return langMap[ext || ''] || 'text';
  };

  const renderBashOutput = (output: any) => {
    // Defensive parsing: handle JSON string or object
    let outputStr = output;
    if (typeof output === 'string') {
      try {
        const parsed = JSON.parse(output);
        outputStr = parsed.stdout || parsed.output || parsed;
      } catch {
        // Already a string, use as-is
        outputStr = output;
      }
    } else if (output && typeof output === 'object') {
      outputStr = output.stdout || output.output || JSON.stringify(output);
    }

    const ansi_up = new AnsiUp();
    const html = ansi_up.ansi_to_html(String(outputStr));

    return (
      <Box
        sx={{
          backgroundColor: 'grey.900',
          color: 'white',
          p: 2,
          borderRadius: 1,
          fontFamily: 'monospace',
          fontSize: '0.875rem',
          overflowX: 'auto',
          maxHeight: 400,
          overflowY: 'auto',
        }}
        dangerouslySetInnerHTML={{ __html: html }}
      />
    );
  };

  const renderReadFile = (data: any) => {
    // Defensive parsing: handle JSON string or object
    let parsedData = data;
    if (typeof data === 'string') {
      try {
        parsedData = JSON.parse(data);
      } catch {
        // Not JSON, treat as plain content
        parsedData = { content: data };
      }
    }

    const content = parsedData.content || parsedData;
    const filePath = parsedData.filePath || parsedData.path || 'file';
    const language = detectLanguageFromPath(filePath);

    return (
      <Box>
        <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
          File: {filePath}
        </Typography>
        <Box sx={{ maxHeight: 400, overflowY: 'auto' }}>
          <SyntaxHighlighter
            language={language}
            style={vscDarkPlus}
            customStyle={{
              fontSize: '0.85rem',
              borderRadius: '4px',
              margin: 0,
            }}
            showLineNumbers
          >
            {typeof content === 'string' ? content : JSON.stringify(content, null, 2)}
          </SyntaxHighlighter>
        </Box>
      </Box>
    );
  };

  const renderWriteFile = (data: any) => {
    // Defensive parsing: handle JSON string or object
    let parsedData = data;
    if (typeof data === 'string') {
      try {
        parsedData = JSON.parse(data);
      } catch {
        // Not JSON, use default
        parsedData = {};
      }
    }

    const filePath = parsedData.filePath || parsedData.path || 'file';
    const bytesWritten = parsedData.bytesWritten || parsedData.size || 0;

    return (
      <Alert severity="success" icon={<CheckCircle />}>
        <Typography variant="body2">
          File written successfully: <strong>{filePath}</strong>
        </Typography>
        {bytesWritten > 0 && (
          <Typography variant="caption" color="text.secondary">
            {bytesWritten} bytes written
          </Typography>
        )}
      </Alert>
    );
  };

  const renderListDirectory = (data: any) => {
    // Defensive parsing: handle JSON string or object
    let parsedData = data;
    if (typeof data === 'string') {
      try {
        parsedData = JSON.parse(data);
      } catch {
        // Not JSON, treat as empty
        parsedData = { files: [] };
      }
    }

    const files = Array.isArray(parsedData) ? parsedData : parsedData.files || [];

    if (files.length === 0) {
      return (
        <Typography variant="body2" color="text.secondary">
          Empty directory
        </Typography>
      );
    }

    return (
      <Box sx={{ maxHeight: 400, overflowY: 'auto' }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell align="right">Size</TableCell>
              <TableCell align="right">Modified</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {files.map((file: any, idx: number) => {
              // Handle both string arrays and object arrays
              const fileName = typeof file === 'string' ? file : file.name;
              const fileSize = typeof file === 'object' ? file.size : undefined;
              const fileModified = typeof file === 'object' ? file.modified : undefined;
              const isDirectory = typeof file === 'object' ? file.isDirectory : false;

              return (
                <TableRow key={idx} hover>
                  <TableCell>
                    <Typography
                      variant="body2"
                      sx={{
                        fontFamily: 'monospace',
                        fontWeight: isDirectory ? 'bold' : 'normal',
                      }}
                    >
                      {isDirectory ? '📁 ' : '📄 '}
                      {fileName}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="caption" color="text.secondary">
                      {fileSize ? `${(fileSize / 1024).toFixed(1)} KB` : '-'}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="caption" color="text.secondary">
                      {fileModified ? new Date(fileModified).toLocaleDateString() : '-'}
                    </Typography>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </Box>
    );
  };

  const renderPatchDiff = (data: any) => {
    // Defensive parsing: handle JSON string or object
    let parsedData = data;
    if (typeof data === 'string') {
      try {
        parsedData = JSON.parse(data);
      } catch {
        // Not JSON, treat as plain diff text
        parsedData = data;
      }
    }

    const diffText = typeof parsedData === 'object'
      ? (parsedData.diff || parsedData.patch || JSON.stringify(parsedData, null, 2))
      : parsedData;

    return (
      <Box sx={{ maxHeight: 400, overflowY: 'auto' }}>
        <SyntaxHighlighter
          language="diff"
          style={vscDarkPlus}
          customStyle={{
            fontSize: '0.85rem',
            borderRadius: '4px',
            margin: 0,
          }}
          showLineNumbers
        >
          {typeof diffText === 'string' ? diffText : JSON.stringify(diffText, null, 2)}
        </SyntaxHighlighter>
      </Box>
    );
  };

  const renderGenericResult = (data: any) => {
    const content = typeof data === 'string' ? data : JSON.stringify(data, null, 2);

    return (
      <Box sx={{ maxHeight: 400, overflowY: 'auto' }}>
        <SyntaxHighlighter
          language="json"
          style={vscDarkPlus}
          customStyle={{
            fontSize: '0.85rem',
            borderRadius: '4px',
            margin: 0,
          }}
        >
          {content}
        </SyntaxHighlighter>
      </Box>
    );
  };

  const renderResult = () => {
    const toolLower = tool.toLowerCase();

    if (toolLower.includes('bash') || toolLower.includes('exec') || toolLower.includes('command')) {
      return renderBashOutput(result.stdout || result.output || result);
    }

    if (toolLower.includes('read') && toolLower.includes('file')) {
      return renderReadFile(result);
    }

    if (toolLower.includes('write') && toolLower.includes('file')) {
      return renderWriteFile(result);
    }

    if (toolLower.includes('list') || toolLower.includes('directory') || toolLower.includes('ls')) {
      return renderListDirectory(result);
    }

    if (toolLower.includes('patch') || toolLower.includes('diff')) {
      return renderPatchDiff(result);
    }

    return renderGenericResult(result);
  };

  if (error) {
    return (
      <Card
        elevation={2}
        sx={{
          mb: 1,
          backgroundColor: 'error.50',
          borderLeft: 4,
          borderLeftColor: 'error.main',
        }}
      >
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
            <Alert severity="error" icon={<ErrorIcon />} sx={{ flex: 1 }}>
              <Typography variant="body2" fontWeight="bold">
                Tool execution failed
              </Typography>
            </Alert>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Chip
                label={formatDuration()}
                size="small"
                color={getDurationColor()}
              />
              <IconButton
                onClick={handleExpandClick}
                aria-expanded={expanded}
                aria-label="show error details"
                size="small"
                sx={{
                  transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 0.3s',
                }}
              >
                <ExpandMore />
              </IconButton>
            </Box>
          </Box>
          <Collapse in={expanded} timeout="auto" unmountOnExit>
            <Box
              sx={{
                backgroundColor: 'grey.900',
                color: 'error.light',
                p: 2,
                borderRadius: 1,
                fontFamily: 'monospace',
                fontSize: '0.875rem',
                overflowX: 'auto',
                maxHeight: 200,
                overflowY: 'auto',
              }}
            >
              {error}
            </Box>
          </Collapse>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card
      elevation={2}
      sx={{
        mb: 1,
        backgroundColor: 'success.50',
        borderLeft: 4,
        borderLeftColor: 'success.main',
      }}
    >
      <CardContent>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
          <Typography variant="caption" color="text.secondary">
            Result
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              label={formatDuration()}
              size="small"
              color={getDurationColor()}
            />
            <IconButton
              onClick={handleExpandClick}
              aria-expanded={expanded}
              aria-label="show result"
              size="small"
              sx={{
                transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
                transition: 'transform 0.3s',
              }}
            >
              <ExpandMore />
            </IconButton>
          </Box>
        </Box>
        <Collapse in={expanded} timeout="auto" unmountOnExit>
          {renderResult()}
        </Collapse>
      </CardContent>
    </Card>
  );
}
