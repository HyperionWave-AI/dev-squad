/**
 * ToolCallCard Component
 *
 * Displays tool invocation with collapsible arguments section.
 * Shows tool name, icon, timestamp, and syntax-highlighted JSON args.
 */

import { useState } from 'react';
import {
  Card,
  CardHeader,
  CardContent,
  Collapse,
  IconButton,
  Typography,
  Chip,
  Box,
} from '@mui/material';
import { ExpandMore } from '@mui/icons-material';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface ToolCallCardProps {
  tool: string;
  args: Record<string, any>;
  id: string;
  timestamp: Date;
  isPending?: boolean;
}

export function ToolCallCard({
  tool,
  args,
  id,
  timestamp,
  isPending = false,
}: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);

  const getToolIcon = () => {
    if (tool.toLowerCase().includes('bash') || tool.toLowerCase().includes('exec')) {
      return '🔧';
    }
    if (tool.toLowerCase().includes('file') || tool.toLowerCase().includes('read') || tool.toLowerCase().includes('write')) {
      return '📄';
    }
    if (tool.toLowerCase().includes('list') || tool.toLowerCase().includes('directory')) {
      return '📁';
    }
    if (tool.toLowerCase().includes('patch') || tool.toLowerCase().includes('diff')) {
      return '🔀';
    }
    return '🛠️';
  };

  const formatTimestamp = (date: Date) => {
    return date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      second: '2-digit',
      hour12: true,
    });
  };

  const handleExpandClick = () => {
    setExpanded(!expanded);
  };

  return (
    <Card
      elevation={2}
      sx={{
        mb: 1,
        backgroundColor: 'primary.50',
        borderLeft: 4,
        borderLeftColor: 'primary.main',
      }}
    >
      <CardHeader
        avatar={
          <Box sx={{ fontSize: '1.5rem', display: 'flex', alignItems: 'center' }}>
            {getToolIcon()}
          </Box>
        }
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="subtitle2" fontWeight="bold">
              {tool}
            </Typography>
            {isPending && (
              <Chip
                label="Executing..."
                size="small"
                color="warning"
                sx={{ height: 20, fontSize: '0.7rem' }}
              />
            )}
          </Box>
        }
        subheader={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="caption" color="text.secondary">
              {formatTimestamp(timestamp)} • ID: {id.slice(0, 8)}
            </Typography>
          </Box>
        }
        action={
          <IconButton
            onClick={handleExpandClick}
            aria-expanded={expanded}
            aria-label="show arguments"
            size="small"
            sx={{
              transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
              transition: 'transform 0.3s',
            }}
          >
            <ExpandMore />
          </IconButton>
        }
        sx={{ pb: 0.5 }}
      />

      <Collapse in={expanded} timeout="auto" unmountOnExit>
        <CardContent sx={{ pt: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
            <Typography variant="caption" color="text.secondary">
              Arguments
            </Typography>
          </Box>
          <Box
            sx={{
              maxHeight: 300,
              overflowY: 'auto',
              '& pre': { margin: 0 },
            }}
          >
            <SyntaxHighlighter
              language="json"
              style={vscDarkPlus}
              customStyle={{
                fontSize: '0.85rem',
                borderRadius: '4px',
                margin: 0,
                padding: '12px',
              }}
            >
              {JSON.stringify(args, null, 2)}
            </SyntaxHighlighter>
          </Box>
        </CardContent>
      </Collapse>
    </Card>
  );
}
