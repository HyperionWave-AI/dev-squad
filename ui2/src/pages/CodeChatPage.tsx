import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { SessionList } from '@/components/organisms/SessionList';
import { ChatMessage } from '@/components/organisms/ChatMessage';
import type { Session } from '../types/chat';
import chatService from '../services/chatService';
import { createSafeDate } from '../utils/dateUtils';

export default function CodeChatPage() {
  const { sessionId } = useParams<{ sessionId?: string }>();
  const navigate = useNavigate();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentSession, setCurrentSession] = useState<Session | null>(null);
  const [message, setMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Load sessions on component mount
  useEffect(() => {
    chatService.loadFromStorage();
    const loadedSessions = chatService.getAllSessions();
    setSessions(loadedSessions);

    // Set current session based on URL parameter or create new one
    if (sessionId) {
      const session = chatService.getSession(sessionId);
      if (session) {
        setCurrentSession(session);
        chatService.setCurrentSession(sessionId);
      } else {
        // Session not found, redirect to new session
        navigate('/chat');
      }
    } else if (loadedSessions.length === 0) {
      // No sessions exist, create a new one
      handleNewSession();
    } else {
      // Load the most recent session
      const mostRecent = loadedSessions.sort((a, b) => 
        new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
      )[0];
      setCurrentSession(mostRecent);
      chatService.setCurrentSession(mostRecent.id);
      navigate(`/chat/${mostRecent.id}`, { replace: true });
    }
  }, [sessionId, navigate]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [currentSession?.messages]);

  const handleNewSession = () => {
    const newSession = chatService.createSession();
    setSessions(chatService.getAllSessions());
    setCurrentSession(newSession);
    navigate(`/chat/${newSession.id}`);
    chatService.saveToStorage();
  };

  const handleSessionSelect = (selectedSessionId: string) => {
    const session = chatService.getSession(selectedSessionId);
    if (session) {
      setCurrentSession(session);
      chatService.setCurrentSession(selectedSessionId);
      navigate(`/chat/${selectedSessionId}`);
    }
  };

  const handleSessionDelete = (sessionIdToDelete: string) => {
    const success = chatService.deleteSession(sessionIdToDelete);
    if (success) {
      const updatedSessions = chatService.getAllSessions();
      setSessions(updatedSessions);
      
      // If we deleted the current session, navigate to another or create new
      if (currentSession?.id === sessionIdToDelete) {
        if (updatedSessions.length > 0) {
          const nextSession = updatedSessions[0];
          setCurrentSession(nextSession);
          chatService.setCurrentSession(nextSession.id);
          navigate(`/chat/${nextSession.id}`);
        } else {
          handleNewSession();
        }
      }
      
      chatService.saveToStorage();
    }
  };

  const handleSessionRename = (sessionIdToRename: string, newName: string) => {
    const updatedSession = chatService.updateSession(sessionIdToRename, { name: newName });
    if (updatedSession) {
      setSessions(chatService.getAllSessions());
      if (currentSession?.id === sessionIdToRename) {
        setCurrentSession(updatedSession);
      }
      chatService.saveToStorage();
    }
  };

  const handleSendMessage = async () => {
    if (!message.trim() || !currentSession || isLoading) return;

    setIsLoading(true);
    
    try {
      // Add user message
      const userMessage = chatService.addMessage(currentSession.id, {
        role: 'user',
        content: message
      });

      if (userMessage) {
        // Update local state
        const updatedSession = chatService.getSession(currentSession.id);
        if (updatedSession) {
          setCurrentSession(updatedSession);
          setSessions(chatService.getAllSessions());
        }
        
        setMessage('');
        
        // Here you would typically call your AI service
        // For now, we'll add a simple echo response
        setTimeout(() => {
          const assistantMessage = chatService.addMessage(currentSession.id, {
            role: 'assistant',
            content: `Echo: ${message}`
          });

          if (assistantMessage) {
            const finalSession = chatService.getSession(currentSession.id);
            if (finalSession) {
              setCurrentSession(finalSession);
              setSessions(chatService.getAllSessions());
            }
          }
          
          chatService.saveToStorage();
          setIsLoading(false);
        }, 1000);
      }
    } catch (error) {
      console.error('Error sending message:', error);
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar with session list */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <Button 
            onClick={handleNewSession}
            className="w-full"
          >
            New Chat
          </Button>
        </div>
        
        <div className="flex-1 overflow-y-auto">
          <SessionList
            sessions={sessions}
            currentSessionId={currentSession?.id}
            onSessionSelect={handleSessionSelect}
            onSessionDelete={handleSessionDelete}
            onSessionRename={handleSessionRename}
          />
        </div>
      </div>

      {/* Main chat area */}
      <div className="flex-1 flex flex-col">
        {currentSession ? (
          <>
            {/* Chat header */}
            <div className="bg-white border-b border-gray-200 p-4">
              <h1 className="text-xl font-semibold text-gray-900">
                {currentSession.name}
              </h1>
              <p className="text-sm text-gray-500">
                Created {createSafeDate(currentSession.createdAt).toLocaleDateString()}
              </p>
            </div>

            {/* Messages area */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {currentSession.messages.length === 0 ? (
                <div className="text-center text-gray-500 mt-8">
                  <p>No messages yet. Start the conversation!</p>
                </div>
              ) : (
                currentSession.messages.map((msg) => (
                  <ChatMessage key={msg.id} message={msg} />
                ))
              )}
              {isLoading && (
                <div className="text-center text-gray-500">
                  <p>Thinking...</p>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>

            {/* Input area */}
            <div className="bg-white border-t border-gray-200 p-4">
              <div className="flex space-x-2">
                <Textarea
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  onKeyDown={handleKeyPress}
                  placeholder="Type your message..."
                  className="flex-1 min-h-[60px] max-h-[200px]"
                  disabled={isLoading}
                />
                <Button 
                  onClick={handleSendMessage}
                  disabled={!message.trim() || isLoading}
                  className="self-end"
                >
                  Send
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center text-gray-500">
              <p>Select a chat session or create a new one to get started</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}