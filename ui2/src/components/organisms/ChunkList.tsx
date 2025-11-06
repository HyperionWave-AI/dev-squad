import React, { lazy, Suspense } from 'react';
import * as Collapsible from '@radix-ui/react-collapsible';
import { ChevronDown, Code2, Box, FileType } from 'lucide-react';
import { Badge } from '../atoms/Badge';
import type { FileChunkDetails, ASTNode } from '../../services/codeIndexService';

// Lazy load syntax highlighter
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(module => ({
    default: module.Prism
  }))
);

import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface ChunkListProps {
  chunks: FileChunkDetails[];
  language: string;
}

const getNodeTypeIcon = (type: string) => {
  switch (type.toLowerCase()) {
    case 'function':
    case 'method':
      return <Code2 className="h-3 w-3" />;
    case 'class':
    case 'struct':
      return <Box className="h-3 w-3" />;
    case 'interface':
    case 'type':
      return <FileType className="h-3 w-3" />;
    default:
      return <Code2 className="h-3 w-3" />;
  }
};

const getNodeTypeBadgeColor = (type: string): 'default' | 'secondary' | 'success' | 'warning' => {
  switch (type.toLowerCase()) {
    case 'function':
    case 'method':
      return 'success';
    case 'class':
    case 'struct':
      return 'warning';
    case 'interface':
    case 'type':
      return 'secondary';
    default:
      return 'default';
  }
};

const ASTNodeDisplay: React.FC<{ node: ASTNode }> = ({ node }) => (
  <div className="flex items-start gap-2 p-2 bg-gray-50 dark:bg-gray-700/50 rounded border border-gray-200 dark:border-gray-600">
    <div className="flex items-center gap-1 mt-0.5">
      {getNodeTypeIcon(node.type)}
    </div>
    <div className="flex-1 min-w-0">
      <div className="flex items-center gap-2 mb-1">
        <Badge variant={getNodeTypeBadgeColor(node.type)} className="text-xs">
          {node.type}
        </Badge>
        <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
          {node.name}
        </span>
      </div>
      {node.signature && (
        <code className="text-xs text-gray-600 dark:text-gray-400 block truncate">
          {node.signature}
        </code>
      )}
      <span className="text-xs text-gray-500 dark:text-gray-400">
        Lines {node.startLine}-{node.endLine}
      </span>
    </div>
  </div>
);

export const ChunkList: React.FC<ChunkListProps> = ({ chunks, language }) => {
  const [openChunks, setOpenChunks] = React.useState<Set<string>>(new Set());

  const toggleChunk = (chunkId: string) => {
    setOpenChunks(prev => {
      const next = new Set(prev);
      if (next.has(chunkId)) {
        next.delete(chunkId);
      } else {
        next.add(chunkId);
      }
      return next;
    });
  };

  if (chunks.length === 0) {
    return (
      <div className="p-8 text-center text-gray-500 dark:text-gray-400">
        <p>No chunks available for this file.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {chunks.map((chunk, index) => {
        const isOpen = openChunks.has(chunk.chunkId);

        return (
          <Collapsible.Root
            key={chunk.chunkId}
            open={isOpen}
            onOpenChange={() => toggleChunk(chunk.chunkId)}
            className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-800"
          >
            <Collapsible.Trigger className="w-full p-4 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
              <div className="flex items-center gap-3">
                <ChevronDown
                  className={`h-4 w-4 text-gray-500 transition-transform ${
                    isOpen ? 'transform rotate-180' : ''
                  }`}
                />
                <div className="text-left">
                  <div className="flex items-center gap-2 mb-1">
                    <Badge variant="secondary" className="text-xs">
                      Chunk {index + 1}
                    </Badge>
                    <Badge variant={chunk.chunkType === 'ast' ? 'success' : 'default'} className="text-xs">
                      {chunk.chunkType}
                    </Badge>
                    {chunk.astNodes && chunk.astNodes.length > 0 && (
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {chunk.astNodes.length} node{chunk.astNodes.length > 1 ? 's' : ''}
                      </span>
                    )}
                  </div>
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    Lines {chunk.startLine}-{chunk.endLine} ({chunk.endLine - chunk.startLine + 1} lines)
                  </span>
                </div>
              </div>
            </Collapsible.Trigger>

            <Collapsible.Content className="border-t border-gray-200 dark:border-gray-700">
              <div className="p-4 space-y-4">
                {/* AST Nodes */}
                {chunk.astNodes && chunk.astNodes.length > 0 && (
                  <div className="space-y-2">
                    <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                      AST Nodes
                    </h4>
                    {chunk.astNodes.map((node, nodeIndex) => (
                      <ASTNodeDisplay key={`${chunk.chunkId}-node-${nodeIndex}`} node={node} />
                    ))}
                  </div>
                )}

                {/* Code Content */}
                <div>
                  <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    Code
                  </h4>
                  <div className="rounded overflow-hidden border border-gray-200 dark:border-gray-700">
                    <Suspense
                      fallback={
                        <div className="p-4 text-center text-gray-500 dark:text-gray-400 text-sm">
                          Loading syntax highlighting...
                        </div>
                      }
                    >
                      <SyntaxHighlighter
                        language={language}
                        style={vscDarkPlus}
                        customStyle={{
                          margin: 0,
                          fontSize: '0.75rem',
                          maxHeight: '400px',
                        }}
                        showLineNumbers
                        startingLineNumber={chunk.startLine}
                        wrapLines
                      >
                        {chunk.content}
                      </SyntaxHighlighter>
                    </Suspense>
                  </div>
                </div>
              </div>
            </Collapsible.Content>
          </Collapsible.Root>
        );
      })}
    </div>
  );
};
