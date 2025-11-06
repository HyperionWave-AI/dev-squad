import React, { useState } from 'react';
import type { Session } from '../../types/chat';
import { formatSessionDate, isValidDate } from '../../utils/dateUtils';

interface SessionListProps {
  sessions: Session[];
  currentSessionId?: string;
  onSessionSelect: (sessionId: string) => void;
  onSessionDelete?: (sessionId: string) => void;
  onSessionRename?: (sessionId: string, newName: string) => void;
}

// Tree node structure for hierarchical rendering
interface SessionTreeNode {
  session: Session;
  children: SessionTreeNode[];
  depth: number;
}

// Helper function to detect if a session is a subchat
const isSubchat = (session: Session): boolean => {
  return session.name?.startsWith('Subchat:') || !!session.metadata?.parentChatId;
};

// Organize sessions into hierarchical structure with proper sorting
const organizeSessionsHierarchy = (sessions: Session[]): SessionTreeNode[] => {
  const sessionMap = new Map<string, SessionTreeNode>();
  const rootNodes: SessionTreeNode[] = [];

  console.log('🔍 organizeSessionsHierarchy called with sessions:', sessions.map(s => ({
    id: s.id,
    name: s.name,
    parentChatId: s.metadata?.parentChatId,
    updatedAt: s.updatedAt,
    isSubchat: isSubchat(s)
  })));

  // First pass: create all nodes
  sessions.forEach(session => {
    sessionMap.set(session.id, { session, children: [], depth: 0 });
  });

  // Second pass: build parent-child relationships
  sessions.forEach(session => {
    const node = sessionMap.get(session.id)!;
    const parentChatId = session.metadata?.parentChatId;

    if (parentChatId && sessionMap.has(parentChatId)) {
      // This is a subchat with a valid parent
      console.log(`✅ Found parent-child link: ${session.name} → parent: ${parentChatId}`);
      const parentNode = sessionMap.get(parentChatId)!;
      parentNode.children.push(node);
    } else if (isSubchat(session) && !parentChatId) {
      // Legacy subchat detection by name - try to find parent by parsing name
      console.log(`⚠️ Subchat without parentChatId: ${session.name}`);
      const nameToCheck = session.name || '';
      const parentIdMatch = nameToCheck.match(/Subchat:\s*(.+?)\s*-/);
      if (parentIdMatch) {
        const parentName = parentIdMatch[1].trim();
        const parentSession = sessions.find(s => 
          s.name === parentName && s.id !== session.id
        );
        if (parentSession) {
          console.log(`✅ Found parent by name match: ${session.name} → ${parentName}`);
          const parentNode = sessionMap.get(parentSession.id)!;
          parentNode.children.push(node);
        } else {
          console.log(`❌ No parent found for subchat: ${session.name} (looking for: ${parentName})`);
          rootNodes.push(node);
        }
      } else {
        console.log(`❌ Could not parse parent from name: ${session.name}`);
        rootNodes.push(node);
      }
    } else {
      // This is a root chat
      console.log(`📁 Root chat: ${session.name}`);
      rootNodes.push(node);
    }
  });

  // Set depths recursively
  const setDepth = (node: SessionTreeNode, depth: number) => {
    node.depth = depth;
    node.children.forEach(child => setDepth(child, depth + 1));
  };

  // Sort root nodes by date (most recent first) with proper date validation
  rootNodes.sort((a, b) => {
    const dateA = isValidDate(a.session.updatedAt) ? new Date(a.session.updatedAt).getTime() : 0;
    const dateB = isValidDate(b.session.updatedAt) ? new Date(b.session.updatedAt).getTime() : 0;
    return dateB - dateA;
  });

  // Sort children by date (most recent first) with proper date validation
  sessionMap.forEach(node => {
    node.children.sort((a, b) => {
      const dateA = isValidDate(a.session.updatedAt) ? new Date(a.session.updatedAt).getTime() : 0;
      const dateB = isValidDate(b.session.updatedAt) ? new Date(b.session.updatedAt).getTime() : 0;
      return dateB - dateA;
    });
  });

  // Set depths for all nodes
  rootNodes.forEach(node => setDepth(node, 0));

  console.log('📊 Final hierarchical structure (root nodes):', rootNodes.map(node => ({
    id: node.session.id,
    name: node.session.name,
    depth: node.depth,
    childrenCount: node.children.length,
    children: node.children.map(child => ({
      id: child.session.id,
      name: child.session.name,
      parentId: child.session.metadata?.parentChatId,
      depth: child.depth
    }))
  })));

  return rootNodes;
};

export function SessionList({ 
  sessions, 
  currentSessionId, 
  onSessionSelect, 
  onSessionDelete,
  onSessionRename 
}: SessionListProps) {
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');

  // Organize sessions into hierarchical structure
  const organizedSessions = organizeSessionsHierarchy(sessions);

  const handleRename = (sessionId: string, currentName: string) => {
    setEditingSessionId(sessionId);
    setEditingName(currentName);
  };

  const handleSaveRename = (sessionId: string) => {
    if (editingName.trim() && onSessionRename) {
      onSessionRename(sessionId, editingName.trim());
    }
    setEditingSessionId(null);
    setEditingName('');
  };

  const handleCancelRename = () => {
    setEditingSessionId(null);
    setEditingName('');
  };

  const renderSessionItem = (node: SessionTreeNode) => {
    const { session, depth } = node;
    const isActive = session.id === currentSessionId;
    const isEditing = editingSessionId === session.id;
    const hasSubchats = node.children.length > 0;
    const isSubchatItem = isSubchat(session);

    return (
      <div key={session.id} className="session-item-container">
        <div 
          className={`session-item ${isActive ? 'active' : ''} ${isSubchatItem ? 'subchat' : 'parent-chat'}`}
          style={{ 
            paddingLeft: `${depth * 24 + 12}px`,
            position: 'relative'
          }}
        >
          {/* Indentation indicator for subchats */}
          {isSubchatItem && depth > 0 && (
            <div 
              className="subchat-indicator"
              style={{
                position: 'absolute',
                left: `${(depth - 1) * 24 + 24}px`,
                top: '50%',
                transform: 'translateY(-50%)',
                width: '12px',
                height: '1px',
                backgroundColor: 'var(--border-color, #e0e0e0)',
              }}
            />
          )}
          
          <div 
            className="session-content"
            onClick={() => !isEditing && onSessionSelect(session.id)}
          >
            <div className="session-info">
              {isEditing ? (
                <input
                  type="text"
                  value={editingName}
                  onChange={(e) => setEditingName(e.target.value)}
                  onBlur={() => handleSaveRename(session.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleSaveRename(session.id);
                    } else if (e.key === 'Escape') {
                      handleCancelRename();
                    }
                  }}
                  className="session-name-input"
                  autoFocus
                />
              ) : (
                <>
                  <div className={`session-name ${isSubchatItem ? 'subchat-name' : ''}`}>
                    {session.name || 'Untitled Chat'}
                  </div>
                  <div className="session-date">
                    {formatSessionDate(session.updatedAt)}
                  </div>
                </>
              )}
            </div>
            
            {!isEditing && (
              <div className="session-actions">
                {onSessionRename && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleRename(session.id, session.name || 'Untitled Chat');
                    }}
                    className="session-action-btn"
                    title="Rename session"
                  >
                    ✏️
                  </button>
                )}
                {onSessionDelete && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm('Are you sure you want to delete this session?')) {
                        onSessionDelete(session.id);
                      }
                    }}
                    className="session-action-btn delete"
                    title="Delete session"
                  >
                    🗑️
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
        
        {/* Render subchats recursively */}
        {hasSubchats && node.children.map(childNode => 
          renderSessionItem(childNode)
        )}
      </div>
    );
  };

  if (sessions.length === 0) {
    return (
      <div className="session-list-empty">
        <p>No chat sessions yet</p>
        <p className="text-sm text-gray-500">Start a new conversation to see it here</p>
      </div>
    );
  }

  return (
    <div className="session-list">
      {organizedSessions.map(node => renderSessionItem(node))}
    </div>
  );
}