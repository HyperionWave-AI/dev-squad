/**
 * ChatSessionList Component
 *
 * Displays list of chat sessions in left sidebar with Git-graph-style visualization.
 * Features: session selection, new chat creation, delete with confirmation, delete all functionality.
 * Git-graph visualization: parent chats and subchats shown with visual branching lines,
 * similar to VS Code's Git commit graph.
 */

import { useState, useMemo, useRef, useEffect } from 'react';
import {
  Box,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  IconButton,
  Button,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
  TextField,
  useTheme,
} from '@mui/material';
import {
  Add,
  Delete,
  Chat,
  DeleteSweep,
  SmartToy,
  ExpandMore
} from '@mui/icons-material';
import type { ChatSession } from '../services/chatService';

// Helper function to detect if a session is a subchat
const isSubchat = (session: ChatSession): boolean => {
  return session.title.startsWith('Subchat:') || !!session.parentChatId;
};

// Tree node structure for hierarchical rendering
interface ChatTreeNode {
  session: ChatSession;
  children: ChatTreeNode[];
  yPosition?: number; // Y position for graph rendering
  depth: number; // Depth in tree (0 for root)
}

// Graph node for SVG rendering
interface GraphNode {
  id: string;
  x: number;
  y: number;
  depth: number;
  isSubchat: boolean;
  isActive: boolean;
  hasChildren: boolean;
  parentId?: string;
}

// Build hierarchical tree structure from flat sessions array
const buildChatTree = (sessions: ChatSession[]): ChatTreeNode[] => {
  const sessionMap = new Map<string, ChatTreeNode>();
  const rootNodes: ChatTreeNode[] = [];

  console.log('🔍 buildChatTree called with sessions:', sessions.map(s => ({
    id: s.id,
    title: s.title,
    parentChatId: s.parentChatId,
    createdAt: s.createdAt,
    isSubchat: isSubchat(s)
  })));

  // First pass: create all nodes
  sessions.forEach(session => {
    sessionMap.set(session.id, { session, children: [], depth: 0 });
  });

  // Second pass: build parent-child relationships and set depths
  const setDepth = (node: ChatTreeNode, depth: number) => {
    node.depth = depth;
    node.children.forEach(child => setDepth(child, depth + 1));
  };

  sessions.forEach(session => {
    const node = sessionMap.get(session.id)!;

    if (session.parentChatId && sessionMap.has(session.parentChatId)) {
      // This is a subchat with a valid parent
      console.log(`✅ Found parent-child link: ${session.title} → parent: ${session.parentChatId}`);
      const parentNode = sessionMap.get(session.parentChatId)!;
      parentNode.children.push(node);
    } else if (isSubchat(session) && !session.parentChatId) {
      // Legacy subchat detection by title - try to find parent by parsing title
      console.log(`⚠️ Subchat without parentChatId: ${session.title}`);
      const parentIdMatch = session.title.match(/Subchat:\s*(.+?)\s*-/);
      if (parentIdMatch) {
        const parentTitle = parentIdMatch[1].trim();
        const parentSession = sessions.find(s => s.title === parentTitle);
        if (parentSession) {
          console.log(`✅ Found parent by title match: ${session.title} → ${parentTitle}`);
          const parentNode = sessionMap.get(parentSession.id)!;
          parentNode.children.push(node);
        } else {
          console.log(`❌ No parent found for subchat: ${session.title} (looking for: ${parentTitle})`);
          rootNodes.push(node);
        }
      } else {
        console.log(`❌ Could not parse parent from title: ${session.title}`);
        rootNodes.push(node);
      }
    } else {
      // This is a root chat
      console.log(`📁 Root chat: ${session.title}`);
      rootNodes.push(node);
    }
  });

  // Sort root nodes by createdAt (oldest first, chronological order)
  rootNodes.sort((a, b) =>
    new Date(a.session.createdAt).getTime() - new Date(b.session.createdAt).getTime()
  );

  // Sort children by createdAt (oldest first, chronological order)
  sessionMap.forEach(node => {
    node.children.sort((a, b) =>
      new Date(a.session.createdAt).getTime() - new Date(b.session.createdAt).getTime()
    );
  });

  // Set depths for all nodes
  rootNodes.forEach(node => setDepth(node, 0));

  console.log('📊 Final tree structure (root nodes):', rootNodes.map(node => ({
    id: node.session.id,
    title: node.session.title,
    depth: node.depth,
    childrenCount: node.children.length,
    children: node.children.map(child => ({
      id: child.session.id,
      title: child.session.title,
      parentId: child.session.parentChatId
    }))
  })));

  return rootNodes;
};

// Constants for Git graph visualization
const GRAPH_CONFIG = {
  nodeRadius: 5,
  lineWidth: 2,
  branchLineWidth: 1.5,
  graphLeftMargin: 16,
  nodeXPosition: 24,
  itemHeight: 56, // Increased for better breathing room
  branchOffsetX: 12, // Horizontal offset for branch lines
};

// Calculate positions for all visible nodes in the tree
const calculateNodePositions = (
  tree: ChatTreeNode[],
  expandedNodes: Set<string>,
  activeSessionId: string | null
): GraphNode[] => {
  const positions: GraphNode[] = [];
  let currentY = GRAPH_CONFIG.itemHeight / 2;

  const traverse = (node: ChatTreeNode, depth: number) => {
    const isExpanded = expandedNodes.has(node.session.id);
    const hasChildren = node.children.length > 0;
    const xPos = GRAPH_CONFIG.nodeXPosition + (depth * GRAPH_CONFIG.branchOffsetX);

    positions.push({
      id: node.session.id,
      x: xPos,
      y: currentY,
      depth,
      isSubchat: isSubchat(node.session),
      isActive: node.session.id === activeSessionId,
      hasChildren,
      parentId: node.session.parentChatId,
    });

    node.yPosition = currentY;
    currentY += GRAPH_CONFIG.itemHeight;

    // Only traverse children if node is expanded
    if (isExpanded && hasChildren) {
      node.children.forEach(child => traverse(child, depth + 1));
    }
  };

  tree.forEach(node => traverse(node, 0));
  return positions;
};

// Generate SVG path data for connections between nodes
const generateGraphPaths = (
  nodes: GraphNode[]
): { path: string; color: string; width: number; isActive: boolean }[] => {
  const paths: { path: string; color: string; width: number; isActive: boolean }[] = [];
  const nodeMap = new Map(nodes.map(n => [n.id, n]));

  nodes.forEach(node => {
    if (node.parentId) {
      const parentNode = nodeMap.get(node.parentId);
      if (parentNode) {
        // Draw curved line from parent to child
        const startX = parentNode.x;
        const startY = parentNode.y;
        const endX = node.x;
        const endY = node.y;

        // Create smooth curve using quadratic bezier
        const controlX = startX + (endX - startX) / 2;
        const controlY = startY;

        const pathData = `M ${startX} ${startY} Q ${controlX} ${controlY}, ${endX} ${endY}`;

        paths.push({
          path: pathData,
          color: node.isActive ? '#ff9800' : '#4caf50',
          width: GRAPH_CONFIG.branchLineWidth,
          isActive: node.isActive || parentNode.isActive,
        });
      }
    }
  });

  return paths;
};

// Git Graph Visualization Component
interface GitGraphProps {
  nodes: GraphNode[];
  paths: { path: string; color: string; width: number; isActive: boolean }[];
  height: number;
  theme: any;
}

const GitGraph: React.FC<GitGraphProps> = ({ nodes, paths, height, theme }) => {
  return (
    <svg
      style={{
        position: 'absolute',
        left: 0,
        top: 0,
        width: 60,
        height: `${height}px`,
        pointerEvents: 'none',
        zIndex: 1,
      }}
    >
      {/* Draw connection paths */}
      {paths.map((pathData, idx) => (
        <path
          key={`path-${idx}`}
          d={pathData.path}
          stroke={pathData.color}
          strokeWidth={pathData.width}
          fill="none"
          opacity={pathData.isActive ? 1 : 0.7}
          strokeLinecap="round"
        />
      ))}

      {/* Draw vertical timeline for root nodes */}
      {nodes
        .filter(n => n.depth === 0)
        .map((node, idx, arr) => {
          if (idx < arr.length - 1) {
            const nextNode = arr[idx + 1];
            return (
              <line
                key={`timeline-${node.id}`}
                x1={node.x}
                y1={node.y}
                x2={nextNode.x}
                y2={nextNode.y}
                stroke={theme.palette.divider}
                strokeWidth={GRAPH_CONFIG.lineWidth}
                opacity={0.3}
              />
            );
          }
          return null;
        })}

      {/* Draw nodes (circles) */}
      {nodes.map(node => (
        <circle
          key={`node-${node.id}`}
          cx={node.x}
          cy={node.y}
          r={node.isActive ? GRAPH_CONFIG.nodeRadius + 1 : GRAPH_CONFIG.nodeRadius}
          fill={
            node.isActive
              ? theme.palette.primary.main
              : node.isSubchat
              ? theme.palette.secondary.main
              : theme.palette.success.main
          }
          stroke={theme.palette.background.paper}
          strokeWidth={2}
        />
      ))}
    </svg>
  );
};

interface ChatSessionListProps {
  sessions: ChatSession[];
  activeSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  onNewChat: () => void;
  onDeleteSession: (sessionId: string) => void;
  onDeleteAllSessions: () => void;
  onRenameSession?: (sessionId: string, newTitle: string) => void;
}

export const ChatSessionList: React.FC<ChatSessionListProps> = ({
  sessions,
  activeSessionId,
  onSessionSelect,
  onNewChat,
  onDeleteSession,
  onDeleteAllSessions,
  onRenameSession,
}) => {
  const theme = useTheme();
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleteAllConfirmOpen, setDeleteAllConfirmOpen] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [renameDialogOpen, setRenameDialogOpen] = useState(false);
  const [sessionToRename, setSessionToRename] = useState<string | null>(null);
  const [newTitle, setNewTitle] = useState('');
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const listRef = useRef<HTMLDivElement>(null);

  // Build tree structure from sessions
  const chatTree = useMemo(() => buildChatTree(sessions), [sessions]);

  // Calculate node positions for Git graph
  const graphNodes = useMemo(
    () => calculateNodePositions(chatTree, expandedNodes, activeSessionId),
    [chatTree, expandedNodes, activeSessionId]
  );

  // Generate connection paths
  const graphPaths = useMemo(() => generateGraphPaths(graphNodes), [graphNodes]);

  // Calculate total height for SVG
  const totalHeight = useMemo(() => {
    const visibleNodeCount = graphNodes.length;
    return Math.max(visibleNodeCount * GRAPH_CONFIG.itemHeight, 200);
  }, [graphNodes.length]);

  // Auto-expand nodes that have active children
  useEffect(() => {
    if (activeSessionId) {
      const activeSession = sessions.find(s => s.id === activeSessionId);
      if (activeSession?.parentChatId) {
        setExpandedNodes(prev => new Set([...prev, activeSession.parentChatId!]));
      }
    }
  }, [activeSessionId, sessions]);

  const handleDeleteClick = (sessionId: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setSessionToDelete(sessionId);
    setDeleteConfirmOpen(true);
  };

  const handleDeleteConfirm = () => {
    if (sessionToDelete) {
      onDeleteSession(sessionToDelete);
      setSessionToDelete(null);
    }
    setDeleteConfirmOpen(false);
  };

  const handleDeleteAllClick = () => {
    setDeleteAllConfirmOpen(true);
  };

  const handleDeleteAllConfirm = () => {
    onDeleteAllSessions();
    setDeleteAllConfirmOpen(false);
  };

  const handleRenameClick = (sessionId: string, currentTitle: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setSessionToRename(sessionId);
    setNewTitle(currentTitle);
    setRenameDialogOpen(true);
  };

  const handleRenameConfirm = () => {
    if (sessionToRename && onRenameSession && newTitle.trim()) {
      onRenameSession(sessionToRename, newTitle.trim());
      setSessionToRename(null);
      setNewTitle('');
    }
    setRenameDialogOpen(false);
  };

  const handleToggleExpand = (sessionId: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setExpandedNodes(prev => {
      const newSet = new Set(prev);
      if (newSet.has(sessionId)) {
        newSet.delete(sessionId);
      } else {
        newSet.add(sessionId);
      }
      return newSet;
    });
  };

  // Flatten tree for rendering while preserving hierarchy
  const flattenTreeForRendering = (nodes: ChatTreeNode[]): ChatTreeNode[] => {
    const result: ChatTreeNode[] = [];

    const traverse = (node: ChatTreeNode) => {
      result.push(node);
      if (expandedNodes.has(node.session.id) && node.children.length > 0) {
        node.children.forEach(traverse);
      }
    };

    nodes.forEach(traverse);
    return result;
  };

  const flattenedSessions = flattenTreeForRendering(chatTree);

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header with New Chat and Delete All buttons */}
      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={onNewChat}
            fullWidth
            sx={{ textTransform: 'none' }}
          >
            New Chat
          </Button>
          <IconButton
            onClick={handleDeleteAllClick}
            disabled={sessions.length === 0}
            color="error"
            sx={{
              minWidth: 48,
              '& .MuiSvgIcon-root': {
                fontSize: '2rem', // Increased from default 1.5rem to 2rem (33% larger)
              }
            }}
            title="Delete All Chats"
          >
            <DeleteSweep />
          </IconButton>
        </Box>
        <Typography variant="body2" color="text.secondary">
          {sessions.length} {sessions.length === 1 ? 'chat' : 'chats'}
        </Typography>
      </Box>

      {/* Sessions List with Git Graph */}
      <Box sx={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        {/* Git Graph SVG Overlay */}
        <GitGraph
          nodes={graphNodes}
          paths={graphPaths}
          height={totalHeight}
          theme={theme}
        />

        {/* Sessions List */}
        <Box
          ref={listRef}
          sx={{
            height: '100%',
            overflowY: 'auto',
            pl: 4, // Reduced from 7 to 4 for tighter spacing with graph
          }}
        >
          {sessions.length === 0 ? (
            <Box sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="body2" color="text.secondary">
                No chat sessions yet. Create your first chat to get started!
              </Typography>
            </Box>
          ) : (
            <List sx={{ py: 0 }}>
              {flattenedSessions.map((treeNode) => {
                const { session } = treeNode;
                const isActive = session.id === activeSessionId;
                const hasChildren = treeNode.children.length > 0;
                const isExpanded = expandedNodes.has(session.id);
                const isSubchatNode = isSubchat(session);

                return (
                  <ListItem
                    key={session.id}
                    disablePadding
                    sx={{
                      pl: treeNode.depth * 2, // Indent based on depth
                      borderLeft: isActive ? '4px solid' : 0,
                      borderColor: 'primary.main',
                      backgroundColor: isActive ? 'rgba(25, 118, 210, 0.08)' : 'transparent',
                      borderRadius: isActive ? '0 8px 8px 0' : 0,
                      mb: 1, // Increased spacing between items
                      transition: 'all 0.2s ease-in-out',
                      '&:hover': {
                        backgroundColor: isActive ? 'rgba(25, 118, 210, 0.12)' : 'action.hover',
                        '& .action-buttons': {
                          opacity: 1,
                        },
                      },
                    }}
                  >
                    <ListItemButton
                      onClick={() => onSessionSelect(session.id)}
                      sx={{
                        py: 1.5, // Increased vertical padding
                        px: 2, // Increased horizontal padding
                        minHeight: GRAPH_CONFIG.itemHeight,
                        alignItems: 'center', // Center align for better visual balance
                      }}
                    >
                      {/* Expand/Collapse button for parent chats */}
                      {hasChildren && (
                        <IconButton
                          size="small"
                          onClick={(e) => handleToggleExpand(session.id, e)}
                          sx={{
                            mr: 1.5,
                            transform: isExpanded ? 'rotate(0deg)' : 'rotate(-90deg)',
                            transition: 'transform 0.2s',
                          }}
                        >
                          <ExpandMore fontSize="small" />
                        </IconButton>
                      )}

                      {/* Chat icon */}
                      <Box sx={{ mr: 2, flexShrink: 0 }}>
                        {isSubchatNode ? (
                          <SmartToy
                            fontSize="small"
                            sx={{ color: theme.palette.secondary.main }}
                          />
                        ) : (
                          <Chat
                            fontSize="small"
                            sx={{ color: theme.palette.primary.main }}
                          />
                        )}
                      </Box>

                      {/* Session title and metadata */}
                      <ListItemText
                        primary={
                          <Typography
                            variant="body2"
                            sx={{
                              fontWeight: isActive ? 600 : 400,
                              color: isActive ? 'primary.main' : 'text.primary',
                              whiteSpace: 'nowrap',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              lineHeight: 1.3,
                            }}
                          >
                            {session.title}
                          </Typography>
                        }
                        secondary={
                          <Typography
                            variant="caption"
                            sx={{
                              color: 'text.secondary',
                              display: 'block',
                              whiteSpace: 'nowrap',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              mt: 0.75, // More space above date
                              fontSize: '0.75rem',
                            }}
                          >
                            {new Date(session.createdAt).toLocaleDateString()}
                            {hasChildren && (
                              <span style={{ marginLeft: 12 }}>
                                • {treeNode.children.length} subchat{treeNode.children.length !== 1 ? 's' : ''}
                              </span>
                            )}
                          </Typography>
                        }
                      />

                      {/* Action buttons - show on hover */}
                      <Box
                        className="action-buttons"
                        sx={{
                          display: 'flex',
                          gap: 1, // Increased gap between buttons
                          ml: 'auto', // Push to the right
                          opacity: isActive ? 1 : 0, // Show when active, hide otherwise
                          transition: 'opacity 0.2s ease-in-out',
                        }}
                      >
                        {onRenameSession && (
                          <IconButton
                            size="small"
                            onClick={(e) => handleRenameClick(session.id, session.title, e)}
                            sx={{
                              padding: '6px',
                              '&:hover': {
                                backgroundColor: 'action.hover',
                                color: 'primary.main',
                              },
                            }}
                          >
                            <Typography variant="caption" sx={{ fontSize: '1.1rem' }}>✏️</Typography>
                          </IconButton>
                        )}
                        <IconButton
                          size="small"
                          onClick={(e) => handleDeleteClick(session.id, e)}
                          sx={{
                            padding: '6px',
                            '&:hover': {
                              backgroundColor: 'error.lighter',
                              color: 'error.main',
                            },
                          }}
                        >
                          <Delete fontSize="small" />
                        </IconButton>
                      </Box>
                    </ListItemButton>
                  </ListItem>
                );
              })}
            </List>
          )}
        </Box>
      </Box>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)}>
        <DialogTitle>Delete Chat Session</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete this chat session? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirmOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete All Confirmation Dialog */}
      <Dialog open={deleteAllConfirmOpen} onClose={() => setDeleteAllConfirmOpen(false)}>
        <DialogTitle>Delete All Chat Sessions</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete all chat sessions? This will permanently remove all your conversations and cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteAllConfirmOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteAllConfirm} color="error" variant="contained">
            Delete All
          </Button>
        </DialogActions>
      </Dialog>

      {/* Rename Dialog */}
      <Dialog open={renameDialogOpen} onClose={() => setRenameDialogOpen(false)}>
        <DialogTitle>Rename Chat Session</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="New Title"
            fullWidth
            variant="outlined"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onKeyPress={(e) => {
              if (e.key === 'Enter') {
                handleRenameConfirm();
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenameDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleRenameConfirm} variant="contained">
            Rename
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ChatSessionList;
