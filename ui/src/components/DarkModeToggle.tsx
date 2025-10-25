import React, { useState, useEffect } from 'react';

interface DarkModeToggleProps {
  className?: string;
  onToggle?: (isDark: boolean) => void;
}

export const DarkModeToggle: React.FC<DarkModeToggleProps> = ({ 
  className = '', 
  onToggle 
}) => {
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
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
      document.documentElement.classList.remove('dark');
    }
    
    // Persist to localStorage
    localStorage.setItem('darkMode', JSON.stringify(isDarkMode));
    
    // Call callback if provided
    onToggle?.(isDarkMode);
  }, [isDarkMode, onToggle]);

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      // Only update if no manual preference is stored
      const saved = localStorage.getItem('darkMode');
      if (saved === null) {
        setIsDarkMode(e.matches);
      }
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const handleToggle = () => {
    setIsDarkMode(!isDarkMode);
  };

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <span 
        className="text-sm font-medium"
        style={{ color: 'var(--text-primary)' }}
      >
        {isDarkMode ? '🌙 Dark' : '☀️ Light'}
      </span>
      
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={isDarkMode}
          onChange={handleToggle}
          className="sr-only"
          aria-label="Toggle dark mode"
          aria-describedby="dark-mode-description"
        />
        
        <div 
          className={`
            relative w-11 h-6 rounded-full transition-colors duration-200 ease-in-out
            ${isDarkMode ? 'bg-blue-600' : 'bg-gray-300'}
            focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2
          `}
          style={{
            backgroundColor: isDarkMode ? 'var(--accent)' : '#d1d5db'
          }}
        >
          <div
            className={`
              absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow-md
              transform transition-transform duration-200 ease-in-out
              ${isDarkMode ? 'translate-x-5' : 'translate-x-0'}
            `}
          />
        </div>
      </label>
      
      <span 
        id="dark-mode-description" 
        className="text-xs"
        style={{ color: 'var(--text-secondary)' }}
      >
        Switch between light and dark themes
      </span>
    </div>
  );
};

// Hook for using dark mode state in other components
export const useDarkMode = () => {
  const [isDarkMode, setIsDarkMode] = useState<boolean>(() => {
    const saved = localStorage.getItem('darkMode');
    if (saved !== null) {
      return JSON.parse(saved);
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  const toggleDarkMode = () => {
    const newMode = !isDarkMode;
    setIsDarkMode(newMode);
    
    if (newMode) {
      document.documentElement.setAttribute('data-theme', 'dark');
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
      document.documentElement.classList.remove('dark');
    }
    
    localStorage.setItem('darkMode', JSON.stringify(newMode));
  };

  return { isDarkMode, toggleDarkMode };
};