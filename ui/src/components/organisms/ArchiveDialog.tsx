import React, { useState, useEffect } from 'react';
import { type Message, type ArchiveRequest } from '../../types/chat';
import './ArchiveDialog.css';

interface ArchiveDialogProps {
  isOpen: boolean;
  messages: Message[];
  onClose: () => void;
  onArchive: (request: ArchiveRequest, sessionId: string) => Promise<void>;
  sessionId: string;
  isLoading?: boolean;
  error?: string;
}

/**
 * ArchiveDialog Component
 * Allows users to select messages to archive and preview token savings
 */
export const ArchiveDialog: React.FC<ArchiveDialogProps> = ({
  isOpen,
  messages,
  onClose,
  onArchive,
  sessionId,
  isLoading = false,
  error,
}) => {
  const [selectedMessageIds, setSelectedMessageIds] = useState<Set<string>>(new Set());
  const [estimatedTokensSaved, setEstimatedTokensSaved] = useState(0);
  const [localError, setLocalError] = useState<string | null>(null);

  // Estimate tokens saved (1 token ≈ 4 characters)
  useEffect(() => {
    let totalTokens = 0;
    selectedMessageIds.forEach((id) => {
      const message = messages.find((m) => m.id === id);
      if (message) {
        totalTokens += Math.ceil(message.content.length / 4);
      }
    });
    setEstimatedTokensSaved(totalTokens);
  }, [selectedMessageIds, messages]);

  const handleSelectMessage = (messageId: string) => {
    const newSelected = new Set(selectedMessageIds);
    if (newSelected.has(messageId)) {
      newSelected.delete(messageId);
    } else {
      newSelected.add(messageId);
    }
    setSelectedMessageIds(newSelected);
  };

  const handleSelectAll = () => {
    if (selectedMessageIds.size === messages.length) {
      setSelectedMessageIds(new Set());
    } else {
      setSelectedMessageIds(new Set(messages.map((m) => m.id)));
    }
  };

  const handleArchive = async () => {
    if (selectedMessageIds.size === 0) {
      setLocalError('Please select at least one message to archive');
      return;
    }

    try {
      setLocalError(null);
      await onArchive(
        {
          sessionId,
          messageIds: Array.from(selectedMessageIds),
          reason: 'User-initiated archive to free up context',
        },
        sessionId
      );
      setSelectedMessageIds(new Set());
      onClose();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Failed to archive messages');
    }
  };

  if (!isOpen) {
    return null;
  }

  return (
    <div className="archive-dialog-overlay" onClick={onClose}>
      <div className="archive-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="archive-dialog-header">
          <h2>Archive Messages</h2>
          <button className="archive-dialog-close" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="archive-dialog-content">
          {error || localError ? (
            <div className="archive-error">
              <span className="error-icon">⚠️</span>
              <span>{error || localError}</span>
            </div>
          ) : null}

          <div className="archive-preview">
            <div className="preview-header">
              <label className="select-all-checkbox">
                <input
                  type="checkbox"
                  checked={selectedMessageIds.size === messages.length && messages.length > 0}
                  onChange={handleSelectAll}
                  disabled={messages.length === 0}
                />
                <span>Select All ({selectedMessageIds.size}/{messages.length})</span>
              </label>
              <div className="preview-stats">
                <span className="stat">
                  <strong>Estimated Tokens Saved:</strong> {estimatedTokensSaved.toLocaleString()}
                </span>
              </div>
            </div>

            <div className="archive-message-list">
              {messages.length === 0 ? (
                <div className="empty-state">No messages to archive</div>
              ) : (
                messages.map((message) => (
                  <div
                    key={message.id}
                    className={`archive-message-item ${
                      selectedMessageIds.has(message.id) ? 'selected' : ''
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedMessageIds.has(message.id)}
                      onChange={() => handleSelectMessage(message.id)}
                    />
                    <div className="message-content">
                      <div className="message-role">
                        <span className={`role-badge role-${message.role}`}>{message.role}</span>
                        <span className="message-time">
                          {new Date(message.timestamp).toLocaleString()}
                        </span>
                      </div>
                      <div className="message-text">
                        {message.content.substring(0, 100)}
                        {message.content.length > 100 ? '...' : ''}
                      </div>
                      <div className="message-tokens">
                        ~{Math.ceil(message.content.length / 4)} tokens
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        <div className="archive-dialog-footer">
          <button
            className="archive-button-cancel"
            onClick={onClose}
            disabled={isLoading}
          >
            Cancel
          </button>
          <button
            className="archive-button-archive"
            onClick={handleArchive}
            disabled={isLoading || selectedMessageIds.size === 0}
          >
            {isLoading ? 'Archiving...' : `Archive ${selectedMessageIds.size} Message(s)`}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ArchiveDialog;
