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

### DarkModeToggle.tsx - Toggle component with state management and consistent styling
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
      <div className="settings-item-content">
        <div className="settings-label">Dark Mode</div>
        <div className="settings-description">
          Switch between light and dark themes for better viewing experience
        </div>
      </div>
      <label className="dark-mode-toggle">
        <input
          type="checkbox"
          checked={isDarkMode}
          onChange={handleToggle}
          aria-label="Toggle dark mode"
          aria-describedby="dark-mode-description"
        />
        <span className="toggle-slider"></span>
      </label>
    </div>
  );
};
```

### Settings.tsx - Main settings page component with consistent design
```tsx
import React from 'react';
import { DarkModeToggle } from './DarkModeToggle';

export const Settings: React.FC = () => {
  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
          Settings
        </h1>
        <p className="text-lg" style={{ color: 'var(--text-secondary)' }}>
          Customize your experience and preferences
        </p>
      </div>
      
      <div className="space-y-6">
        {/* Appearance Section */}
        <section>
          <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
            Appearance
          </h2>
          <div className="settings-container">
            <DarkModeToggle />
          </div>
        </section>

        {/* Additional Settings Sections */}
        <section>
          <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
            Notifications
          </h2>
          <div className="settings-container">
            <div className="settings-item">
              <div className="settings-item-content">
                <div className="settings-label">Push Notifications</div>
                <div className="settings-description">
                  Receive notifications for important updates and messages
                </div>
              </div>
              <button 
                className="px-4 py-2 rounded-md transition-colors"
                style={{ 
                  backgroundColor: 'var(--accent)',
                  color: 'white'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = 'var(--accent-hover)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = 'var(--accent)';
                }}
              >
                Configure
              </button>
            </div>
            
            <div className="settings-item">
              <div className="settings-item-content">
                <div className="settings-label">Email Notifications</div>
                <div className="settings-description">
                  Get email updates about your account and activity
                </div>
              </div>
              <label className="dark-mode-toggle">
                <input
                  type="checkbox"
                  defaultChecked={true}
                  aria-label="Toggle email notifications"
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>
        </section>

        <section>
          <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
            Privacy & Security
          </h2>
          <div className="settings-container">
            <div className="settings-item">
              <div className="settings-item-content">
                <div className="settings-label">Data Collection</div>
                <div className="settings-description">
                  Allow anonymous usage data collection to improve the service
                </div>
              </div>
              <label className="dark-mode-toggle">
                <input
                  type="checkbox"
                  defaultChecked={false}
                  aria-label="Toggle data collection"
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};
```

## Styling Guidelines for Settings Page Design Consistency

### CSS Variables Usage
All components should use CSS custom properties for theming:
- `var(--bg-primary)` - Main background color
- `var(--bg-secondary)` - Secondary background (cards, panels)
- `var(--bg-tertiary)` - Tertiary background (hover states)
- `var(--text-primary)` - Primary text color
- `var(--text-secondary)` - Secondary text color (descriptions)
- `var(--border-color)` - Border colors
- `var(--accent)` - Accent color for buttons and active states
- `var(--accent-hover)` - Hover state for accent elements

### Component Structure
```css
.settings-container {
  /* Container for settings sections */
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px var(--shadow);
}

.settings-item {
  /* Individual setting row */
  background-color: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  transition: background-color 0.2s ease;
}

.settings-item:hover {
  background-color: var(--bg-tertiary);
}

.settings-item-content {
  /* Content area with label and description */
  flex: 1;
  margin-right: 1rem;
}

.settings-label {
  /* Setting title */
  color: var(--text-primary);
  font-weight: 500;
  font-size: 1rem;
  margin-bottom: 0.25rem;
}

.settings-description {
  /* Setting description */
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.4;
}
```

### Dark Mode Toggle Styling
The toggle component uses enhanced styling with:
- Smooth transitions and animations
- Focus states for accessibility
- Visual feedback with icons (sun/moon)
- Hover and active states
- High contrast mode support
- Reduced motion support for accessibility

### Responsive Design
- Mobile-first approach with breakpoints
- Stacked layout on smaller screens
- Touch-friendly interaction areas
- Proper spacing and typography scaling

### Accessibility Features
- ARIA labels and descriptions
- Keyboard navigation support
- Focus indicators
- Screen reader compatibility
- High contrast mode support
- Reduced motion preferences

### Visual Feedback Implementation
The toggle provides clear visual feedback for current dark mode state through:
1. **Toggle Position**: Slider moves left (light) or right (dark)
2. **Color Changes**: Background color changes based on state
3. **Icon Indicators**: Sun (☀️) for light mode, Moon (🌙) for dark mode
4. **Smooth Transitions**: All state changes are animated
5. **Focus States**: Clear focus indicators for keyboard navigation

### Testing Considerations
- Test with screen readers
- Verify keyboard navigation
- Check color contrast ratios
- Test on mobile devices
- Validate theme persistence across page reloads
- Test system preference detection