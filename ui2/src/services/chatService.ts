import { v4 as uuidv4 } from 'uuid';
import type { Session, Message } from '../types/chat';
import { createSafeDate, isValidDate } from '../utils/dateUtils';

class ChatService {
  private sessions: Session[] = [];
  private currentSession: Session | null = null;

  // Session management
  createSession(name?: string): Session {
    const now = createSafeDate(new Date()).toISOString();
    const session: Session = {
      id: uuidv4(),
      name: name || `Chat ${new Date().toLocaleDateString()}`,
      messages: [],
      createdAt: now,
      updatedAt: now,
      subchats: []
    };
    
    this.sessions.push(session);
    this.currentSession = session;
    return session;
  }

  getSession(sessionId: string): Session | null {
    return this.sessions.find(s => s.id === sessionId) || null;
  }

  getAllSessions(): Session[] {
    // Ensure all sessions have valid dates
    return this.sessions.map(session => ({
      ...session,
      createdAt: isValidDate(session.createdAt) ? session.createdAt : createSafeDate(new Date()).toISOString(),
      updatedAt: isValidDate(session.updatedAt) ? session.updatedAt : createSafeDate(new Date()).toISOString()
    }));
  }

  updateSession(sessionId: string, updates: Partial<Session>): Session | null {
    const sessionIndex = this.sessions.findIndex(s => s.id === sessionId);
    if (sessionIndex === -1) return null;

    const updatedSession = {
      ...this.sessions[sessionIndex],
      ...updates,
      updatedAt: createSafeDate(new Date()).toISOString()
    };
    
    this.sessions[sessionIndex] = updatedSession;
    
    if (this.currentSession?.id === sessionId) {
      this.currentSession = updatedSession;
    }
    
    return updatedSession;
  }

  deleteSession(sessionId: string): boolean {
    const sessionIndex = this.sessions.findIndex(s => s.id === sessionId);
    if (sessionIndex === -1) return false;

    this.sessions.splice(sessionIndex, 1);
    
    if (this.currentSession?.id === sessionId) {
      this.currentSession = null;
    }
    
    return true;
  }

  setCurrentSession(sessionId: string): Session | null {
    const session = this.getSession(sessionId);
    if (session) {
      this.currentSession = session;
    }
    return session;
  }

  getCurrentSession(): Session | null {
    return this.currentSession;
  }

  // Message management
  addMessage(sessionId: string, message: Omit<Message, 'id' | 'timestamp'>): Message | null {
    const session = this.getSession(sessionId);
    if (!session) return null;

    const newMessage: Message = {
      ...message,
      id: uuidv4(),
      timestamp: createSafeDate(new Date()).toISOString()
    };

    session.messages.push(newMessage);
    session.updatedAt = createSafeDate(new Date()).toISOString();
    
    return newMessage;
  }

  getMessages(sessionId: string): Message[] {
    const session = this.getSession(sessionId);
    return session?.messages || [];
  }

  // Subchat management
  createSubchat(parentSessionId: string, name?: string): Session | null {
    const parentSession = this.getSession(parentSessionId);
    if (!parentSession) return null;

    const now = createSafeDate(new Date()).toISOString();
    const subchat: Session = {
      id: uuidv4(),
      name: name || `Subchat ${(parentSession.subchats?.length || 0) + 1}`,
      messages: [],
      createdAt: now,
      updatedAt: now,
      subchats: []
    };

    if (!parentSession.subchats) {
      parentSession.subchats = [];
    }
    
    parentSession.subchats.push(subchat);
    parentSession.updatedAt = now;
    
    return subchat;
  }

  // Data persistence (localStorage)
  saveToStorage(): void {
    try {
      localStorage.setItem('chatSessions', JSON.stringify(this.sessions));
      if (this.currentSession) {
        localStorage.setItem('currentSessionId', this.currentSession.id);
      }
    } catch (error) {
      console.error('Failed to save sessions to storage:', error);
    }
  }

  loadFromStorage(): void {
    try {
      const sessionsData = localStorage.getItem('chatSessions');
      if (sessionsData) {
        const parsedSessions = JSON.parse(sessionsData);
        // Validate and fix dates during loading
        this.sessions = parsedSessions.map((session: any) => ({
          ...session,
          createdAt: isValidDate(session.createdAt) ? session.createdAt : createSafeDate(new Date()).toISOString(),
          updatedAt: isValidDate(session.updatedAt) ? session.updatedAt : createSafeDate(new Date()).toISOString(),
          messages: session.messages?.map((msg: any) => ({
            ...msg,
            timestamp: isValidDate(msg.timestamp) ? msg.timestamp : createSafeDate(new Date()).toISOString()
          })) || []
        }));
      }

      const currentSessionId = localStorage.getItem('currentSessionId');
      if (currentSessionId) {
        this.setCurrentSession(currentSessionId);
      }
    } catch (error) {
      console.error('Failed to load sessions from storage:', error);
      this.sessions = [];
      this.currentSession = null;
    }
  }

  // Clear all data
  clearAll(): void {
    this.sessions = [];
    this.currentSession = null;
    localStorage.removeItem('chatSessions');
    localStorage.removeItem('currentSessionId');
  }
}

// Export singleton instance
export const chatService = new ChatService();
export default chatService;