import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
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
  House,
  ChevronDown,
  Search,
  X,
  Sun,
  Moon,
  RefreshCw,
  Sparkles,
  KanbanSquare,
} from 'lucide-react';
import { Button } from '@atoms/Button';
import { Badge } from '@atoms/Badge';
import { cn } from '@/utils';
import * as Switch from '@radix-ui/react-switch';

export interface NavItemType {
  path: string;
  label: string;
  icon: React.ReactNode;
  priority: 'high' | 'medium' | 'low';
}

export interface SidebarProps {
  navigationItems: NavItemType[];
  onRefresh: () => void;
  theme: 'light' | 'dark';
  onThemeToggle: () => void;
  isMobileOpen?: boolean;
  onMobileToggle?: () => void;
}

interface NavigationSection {
  id: string;
  title: string;
  icon: React.ComponentType<any>;
  items?: Array<{
    path: string;
    label: string;
    icon: React.ComponentType<any>;
  }>;
  href?: string;
  isDirectLink?: boolean;
}

export function Sidebar({
  navigationItems,
  onRefresh,
  theme,
  onThemeToggle,
  isMobileOpen = false,
  onMobileToggle,
}: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(['core', 'management'])
  );
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Handle global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        setIsSearchOpen(true);
      }
      if (event.key === 'Escape' && isSearchOpen) {
        setIsSearchOpen(false);
        setSearchQuery('');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isSearchOpen]);

  // Build navigation sections
  const navigationSections: NavigationSection[] = [
    {
      id: 'dashboard',
      title: 'Progress blog',
      icon: Sparkles,
      href: '/blog',
      isDirectLink: true,
    },
    {
      id: 'core',
      title: 'Core Tools',
      icon: LayoutDashboard,
      items: [
        { path: '/chat', label: 'Chat', icon: MessageSquare },
        { path: '/tasks', label: 'Tasks', icon: KanbanSquare },
        { path: '/code', label: 'Code Search', icon: Code },
      ],
    },
    {
      id: 'management',
      title: 'Management',
      icon: Brain,
      items: [
        { path: '/knowledge', label: 'Knowledge Base', icon: BookOpen },
        { path: '/reflection', label: 'Reflection', icon: Brain },
        { path: '/subagents', label: 'Subagents', icon: Bot },
        { path: '/mcp-servers', label: 'MCP Servers', icon: Plug },
      ],
    },
    {
      id: 'system',
      title: 'System',
      icon: Settings,
      items: [
        { path: '/tools', label: 'Tools', icon: Wrench },
        { path: '/settings', label: 'Settings', icon: Settings },
      ],
    },
  ];

  const toggleSection = (sectionId: string) => {
    const newExpanded = new Set(expandedSections);
    if (newExpanded.has(sectionId)) {
      newExpanded.delete(sectionId);
    } else {
      newExpanded.add(sectionId);
    }
    setExpandedSections(newExpanded);
  };

  const handleNavigate = (path: string) => {
    navigate(path);
    if (onMobileToggle) {
      onMobileToggle();
    }
  };

  // Filter navigation items based on search query
  const filteredSections = searchQuery
    ? navigationSections
        .map((section) => ({
          ...section,
          items: section.items?.filter((item) =>
            item.label.toLowerCase().includes(searchQuery.toLowerCase())
          ),
        }))
        .filter((section) => section.isDirectLink || (section.items && section.items.length > 0))
    : navigationSections;

  return (
    <>
      {/* Mobile backdrop */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={onMobileToggle}
        />
      )}

      {/* Sidebar */}
      <div
        className={cn(
          'w-80 h-screen overflow-y-auto relative z-50',
          'bg-[#0e1e3e]',
          'border-r border-[#0a1628]',
          'lg:relative lg:translate-x-0',
          isMobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
          'transition-transform duration-200 ease-in-out',
          'fixed lg:relative top-0 left-0'
        )}
      >
        {/* Mobile close button */}
        {onMobileToggle && (
          <button
            onClick={onMobileToggle}
            className="absolute top-4 right-4 lg:hidden p-2 hover:bg-[#1e3352] rounded-lg transition-colors z-10"
          >
            <X className="h-5 w-5 text-[#8fa9c7]" />
          </button>
        )}

        {/* Sidebar Header */}
        <div className="border-b border-[#0a1628]">
          {/* Workspace Header */}
          <div className="p-4 pb-2">
            <div className="flex items-center gap-3 p-2">
              <div className="relative">
                <div className="w-9 h-9 bg-gradient-to-br from-blue-600 to-purple-600 rounded-lg flex items-center justify-center shadow-lg">
                  <Bot className="h-5 w-5 text-white" />
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <h1 className="text-lg font-bold text-white">
                  Hyperion
                </h1>
                <p className="text-xs text-[#8fa9c7]">
                  AI Orchestration Platform
                </p>
              </div>
            </div>
          </div>

          {/* Status Bar */}
          <div className="px-4 pb-3">
            <div className="flex items-center justify-between text-xs">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse"></div>
                <span className="text-[#8fa9c7]">All Systems Operational</span>
              </div>
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4">
                v2.0
              </Badge>
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div className="px-4 py-3 border-b border-[#0a1628]">
          <button
            onClick={() => setIsSearchOpen(true)}
            className="w-full flex items-center gap-3 px-3 py-2 bg-[#162942] hover:bg-[#1e3352] rounded-lg transition-colors text-left"
          >
            <Search className="h-4 w-4 text-[#8fa9c7]" />
            <span className="flex-1 text-sm text-[#8fa9c7]">
              Search navigation...
            </span>
            <kbd className="px-1.5 py-0.5 bg-[#0a1628] rounded text-xs text-[#8fa9c7] font-mono">
              ⌘K
            </kbd>
          </button>
        </div>

        {/* Search Overlay */}
        {isSearchOpen && (
          <div className="fixed inset-0 z-[100] flex items-start justify-center pt-20 px-4">
            <div
              className="fixed inset-0 bg-black/50"
              onClick={() => {
                setIsSearchOpen(false);
                setSearchQuery('');
              }}
            />
            <div className="relative w-full max-w-2xl bg-[#162942] rounded-2xl shadow-2xl border border-[#0a1628]">
              <div className="p-4">
                <div className="flex items-center gap-3 mb-4">
                  <Search className="h-5 w-5 text-[#8fa9c7]" />
                  <input
                    type="text"
                    placeholder="Search navigation..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="flex-1 bg-transparent border-none outline-none text-white placeholder-[#8fa9c7]"
                    autoFocus
                  />
                  <button
                    onClick={() => {
                      setIsSearchOpen(false);
                      setSearchQuery('');
                    }}
                    className="p-1 hover:bg-[#1e3352] rounded"
                  >
                    <X className="h-4 w-4 text-[#8fa9c7]" />
                  </button>
                </div>
                {searchQuery && (
                  <div className="space-y-1 max-h-96 overflow-y-auto">
                    {filteredSections.map((section) =>
                      section.items?.map((item) => (
                        <button
                          key={item.path}
                          onClick={() => {
                            handleNavigate(item.path);
                            setIsSearchOpen(false);
                            setSearchQuery('');
                          }}
                          className="w-full flex items-center gap-3 px-3 py-2 hover:bg-[#1e3352] rounded-lg transition-colors text-left"
                        >
                          <item.icon className="h-4 w-4 text-[#8fa9c7]" />
                          <span className="text-sm text-white">
                            {item.label}
                          </span>
                        </button>
                      ))
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Navigation Sections */}
        <div className="p-4 space-y-6">
          {navigationSections.map((section) => (
            <div key={section.id}>
              {/* Section Header */}
              <div
                onClick={() => {
                  if (section.isDirectLink && section.href) {
                    handleNavigate(section.href);
                  } else {
                    toggleSection(section.id);
                  }
                }}
                className={cn(
                  'flex items-center justify-between w-full text-left mb-2 px-3 py-2 rounded-lg transition-colors cursor-pointer group',
                  section.isDirectLink && location.pathname === section.href
                    ? 'bg-[#1264a3]'
                    : 'hover:bg-[#1e3352]'
                )}
              >
                <div className="flex items-center gap-2">
                  <section.icon
                    className={cn(
                      'h-4 w-4',
                      section.isDirectLink && location.pathname === section.href
                        ? 'text-white'
                        : 'text-[#8fa9c7] group-hover:text-white'
                    )}
                  />
                  <h3
                    className={cn(
                      'text-sm font-semibold',
                      section.isDirectLink && location.pathname === section.href
                        ? 'text-white'
                        : 'text-[#8fa9c7] group-hover:text-white'
                    )}
                  >
                    {section.title}
                  </h3>
                </div>
                {!section.isDirectLink && (
                  <motion.div
                    animate={{ rotate: expandedSections.has(section.id) ? 0 : -90 }}
                    transition={{ duration: 0.15 }}
                  >
                    <ChevronDown className="h-3 w-3 text-[#8fa9c7] group-hover:text-white" />
                  </motion.div>
                )}
              </div>

              {/* Section Items */}
              <AnimatePresence>
                {!section.isDirectLink && expandedSections.has(section.id) && section.items && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ duration: 0.2 }}
                    className="overflow-hidden space-y-0.5"
                  >
                    {section.items.map((item) => {
                      const isActive = location.pathname === item.path;
                      const ItemIcon = item.icon;

                      return (
                        <motion.div
                          key={item.path}
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          transition={{ duration: 0.2 }}
                          whileHover={{ x: 4 }}
                          className={cn(
                            'rounded-lg cursor-pointer transition-all duration-150',
                            isActive
                              ? 'bg-[#1264a3]'
                              : 'hover:bg-[#1e3352]'
                          )}
                          onClick={() => handleNavigate(item.path)}
                        >
                          <div className="flex items-center gap-3 px-4 py-2.5">
                            <ItemIcon
                              className={cn(
                                'h-4 w-4',
                                isActive
                                  ? 'text-white'
                                  : 'text-[#8fa9c7]'
                              )}
                            />
                            <span
                              className={cn(
                                'text-sm',
                                isActive
                                  ? 'text-white font-medium'
                                  : 'text-[#8fa9c7]'
                              )}
                            >
                              {item.label}
                            </span>
                          </div>
                        </motion.div>
                      );
                    })}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          ))}
        </div>

        {/* Footer - Controls */}
        <div className="mt-auto p-4 border-t border-[#0a1628]">
          <div className="space-y-3">
            {/* Theme Toggle */}
            <div className="flex items-center justify-between px-3 py-2 bg-[#162942] rounded-lg border border-[#0a1628]">
              <div className="flex items-center gap-2">
                <Sun className="h-4 w-4 text-[#8fa9c7]" />
                <span className="text-sm text-[#8fa9c7]">Theme</span>
              </div>
              <div className="flex items-center gap-2">
                <Switch.Root
                  checked={theme === 'dark'}
                  onCheckedChange={onThemeToggle}
                  className={cn(
                    'w-11 h-6 rounded-full relative transition-colors',
                    theme === 'dark' ? 'bg-blue-500' : 'bg-gray-300'
                  )}
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full shadow-lg transition-transform duration-100 translate-x-0.5 will-change-transform data-[state=checked]:translate-x-[22px]" />
                </Switch.Root>
                <Moon className="h-4 w-4 text-[#8fa9c7]" />
              </div>
            </div>

            {/* Refresh Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={onRefresh}
              className="w-full justify-center gap-2 bg-[#162942] hover:bg-[#1e3352] border-[#0a1628] text-[#8fa9c7] hover:text-white"
            >
              <RefreshCw className="h-4 w-4" />
              <span>Refresh</span>
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
