# Knowledge UI Components - Quick Implementation Guide

## For ui-dev Agent

The specifications from Frontend Experience Specialist didn't persist to disk. Use existing patterns instead:

### Existing Patterns to Follow:
- **Reference Component**: `coordinator/ui/src/components/KnowledgeBrowser.tsx` (Tailwind CSS patterns)
- **API Client**: `coordinator/ui/src/services/mcpClient.ts` (MCP client patterns)
- **Types**: `coordinator/ui/src/types/coordinator.ts` and `knowledge.ts`

### Quick Implementation (Use contextHint from TODOs):

1. **KnowledgeSearch.tsx** - Search interface with Tailwind
2. **KnowledgeCreate.tsx** - Form component
3. **CollectionBrowser.tsx** - Grid view of collections

### API Endpoints (Backend Ready):
- GET `/api/knowledge/search?collectionName=X&query=Y&limit=N`
- POST `/api/knowledge` - body: `{collectionName, information, metadata}`
- GET `/api/knowledge/collections`

### Use Tailwind CSS (not Material-UI)
The existing codebase uses Tailwind. Follow KnowledgeBrowser.tsx patterns.

## Settings Component with Dark Mode Toggle

### DarkModeToggle.tsx - Toggle component with state management
```tsx
import React, { useState, useEffect } from 'react';

interface DarkModeToggleProps {
  className?: string;
}

export const DarkModeToggle: React.FC<DarkModeToggleProps> = ({ className = '' }) => {
  const [isDarkMode, setIsDarkMode] = useState<boolean>(() => {
    // Check localStorage first, then system preference
    const saved = localStorage.getItem('darkMode');
    if (saved !== null) {
      return JSON.parse(saved);
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  useEffect(() => {
    // Apply theme to document
    if (isDarkMode) {
      document.documentElement.setAttribute('data-theme', 'dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
    
    // Persist to localStorage
    localStorage.setItem('darkMode', JSON.stringify(isDarkMode));
  }, [isDarkMode]);

  const handleToggle = () => {
    setIsDarkMode(!isDarkMode);
  };

  return (
    <div className={`settings-item ${className}`}>
      <div>
        <div className="settings-label">Dark Mode</div>
        <div className="settings-description">
          Switch between light and dark themes
        </div>
      </div>
      <label className="dark-mode-toggle">
        <input
          type="checkbox"
          checked={isDarkMode}
          onChange={handleToggle}
          aria-label="Toggle dark mode"
        />
        <span className="toggle-slider"></span>
      </label>
    </div>
  );
};
```

### Settings.tsx - Main settings page component
```tsx
import React from 'react';
import { DarkModeToggle } from './DarkModeToggle';

export const Settings: React.FC = () => {
  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6 text-gray-900 dark:text-white">
        Settings
      </h1>
      
      <div className="settings-container rounded-lg">
        <DarkModeToggle />
        
        {/* Additional settings can be added here */}
        <div className="settings-item">
          <div>
            <div className="settings-label">Notifications</div>
            <div className="settings-description">
              Manage your notification preferences
            </div>
          </div>
          <button className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors">
            Configure
          </button>
        </div>
      </div>
    </div>
  );
};
```

### Usage:
Import and use the Settings component in your main app routing or as a page component.