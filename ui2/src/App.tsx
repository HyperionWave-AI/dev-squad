import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import {
  MessageSquare,
  LayoutDashboard,
  BookOpen,
  Brain,
  Code,
  Plug,
  Wrench,
  Bot,
  Settings,
  KanbanSquare,
} from 'lucide-react';
import { PageLayout } from '@templates/PageLayout';
import { initializeTheme, applyTheme, setStoredTheme, watchSystemTheme } from '@/utils/theme';
import { ConversationModeProvider } from '@/contexts/ConversationModeContext';
import { CodeChatPage } from '@/pages/CodeChatPage';
import { KnowledgeBasePage } from '@/pages/KnowledgeBasePage';
import KanbanBoard from '@/pages/KanbanBoard';
import { ReflectionPage } from '@/pages/ReflectionPage';
import CodeSearchPage from '@/pages/CodeSearchPage';
import { MCPServersPage } from '@/pages/MCPServersPage';
import { HTTPToolsPage } from '@/pages/HTTPToolsPage';
import { SubagentsPage } from '@/pages/SubagentsPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { BlogProgressPage } from '@/pages/BlogProgressPage';

function App() {
  const [refreshKey, setRefreshKey] = useState(0);
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  // Initialize theme on mount
  useEffect(() => {
    const initialTheme = initializeTheme();
    const effectiveTheme = initialTheme === 'system' 
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : initialTheme;
    setTheme(effectiveTheme);
  }, []);

  // Watch system theme changes
  useEffect(() => {
    const cleanup = watchSystemTheme(() => {
      const storedTheme = localStorage.getItem('hyperion-theme');
      if (storedTheme === 'system' || !storedTheme) {
        const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        setTheme(systemTheme);
        applyTheme('system');
      }
    });
    return cleanup;
  }, []);

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1);
  };

  const handleThemeToggle = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    setStoredTheme(newTheme);
    applyTheme(newTheme);
  };

  // Navigation items matching the original UI
  const navigationItems = [
    { path: '/chat', label: 'Chat', icon: <MessageSquare className="h-5 w-5" />, priority: 'high' as const },
    { path: '/blog', label: 'Dashboard', icon: <LayoutDashboard className="h-5 w-5" />, priority: 'high' as const },
    { path: '/tasks', label: 'Tasks', icon: <KanbanSquare className="h-5 w-5" />, priority: 'high' as const },
    { path: '/knowledge', label: 'Knowledge', icon: <BookOpen className="h-5 w-5" />, priority: 'medium' as const },
    { path: '/reflection', label: 'Reflection', icon: <Brain className="h-5 w-5" />, priority: 'medium' as const },
    { path: '/code', label: 'Code', icon: <Code className="h-5 w-5" />, priority: 'medium' as const },
    { path: '/mcp-servers', label: 'MCP Servers', icon: <Plug className="h-5 w-5" />, priority: 'medium' as const },
    { path: '/tools', label: 'Tools', icon: <Wrench className="h-5 w-5" />, priority: 'low' as const },
    { path: '/subagents', label: 'Subagents', icon: <Bot className="h-5 w-5" />, priority: 'low' as const },
    { path: '/settings', label: 'Settings', icon: <Settings className="h-5 w-5" />, priority: 'low' as const },
  ];

  return (
    <ConversationModeProvider>
      <BrowserRouter basename="/ui">
        <Routes>
          <Route
            element={
              <PageLayout
                navigationItems={navigationItems}
                onRefresh={handleRefresh}
                theme={theme}
                onThemeToggle={handleThemeToggle}
              />
            }
          >
            <Route path="/chat" element={<CodeChatPage key={refreshKey} />} />
            <Route path="/tasks" element={<KanbanBoard key={refreshKey} />} />
            <Route path="/blog" element={<BlogProgressPage key={refreshKey} />} />
            <Route path="/knowledge" element={<KnowledgeBasePage key={refreshKey} />} />
            <Route path="/reflection" element={<ReflectionPage key={refreshKey} />} />
            <Route path="/code" element={<CodeSearchPage key={refreshKey} />} />
            <Route path="/mcp-servers" element={<MCPServersPage key={refreshKey} />} />
            <Route path="/tools" element={<HTTPToolsPage key={refreshKey} />} />
            <Route path="/subagents" element={<SubagentsPage key={refreshKey} />} />
            <Route path="/settings" element={<SettingsPage key={refreshKey} />} />
            <Route path="/" element={<Navigate to="/blog" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConversationModeProvider>
  );
}

export default App;
