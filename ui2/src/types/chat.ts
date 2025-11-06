/**
 * Chat-related type definitions
 */

export interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  timestamp: string;
  metadata?: {
    tool?: string;
    [key: string]: any;
  };
}

export interface Session {
  id: string;
  name: string;
  messages: Message[];
  createdAt: string;
  updatedAt: string;
  parentChatId?: string; // Alternative field name for compatibility with chatService
  parentSessionId?: string;
  isSubchat?: boolean;
  subchats?: Session[];
  metadata?: {
    [key: string]: any;
  };
}

export interface ChatState {
  sessions: Session[];
  currentSession: Session | null;
  isLoading: boolean;
  error: string | null;
}

export interface CreateSessionOptions {
  name?: string;
  parentId?: string;
}

export interface SendMessageOptions {
  sessionId: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  metadata?: {
    tool?: string;
    [key: string]: any;
  };
}