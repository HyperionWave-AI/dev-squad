import React, { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Menu } from 'lucide-react';
import { Sidebar } from '@organisms/Sidebar';
import { Button } from '@atoms/Button';

export interface NavItemType {
  path: string;
  label: string;
  icon: React.ReactNode;
  priority: 'high' | 'medium' | 'low';
}

export interface PageLayoutProps {
  navigationItems: NavItemType[];
  onRefresh: () => void;
  theme: 'light' | 'dark';
  onThemeToggle: () => void;
}

export function PageLayout({
  navigationItems,
  onRefresh,
  theme,
  onThemeToggle,
}: PageLayoutProps) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleMobileMenuToggle = () => {
    setMobileMenuOpen((prev) => !prev);
  };

  return (
    <div className="flex h-screen w-full overflow-hidden">
      {/* Left Sidebar */}
      <Sidebar
        navigationItems={navigationItems}
        onRefresh={onRefresh}
        theme={theme}
        onThemeToggle={onThemeToggle}
        isMobileOpen={mobileMenuOpen}
        onMobileToggle={handleMobileMenuToggle}
      />

      {/* Main Content Area */}
      <main className="flex-1 overflow-auto">
        {/* Mobile menu button - only visible on mobile */}
        <div className="lg:hidden fixed top-4 left-4 z-30">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleMobileMenuToggle}
            className="backdrop-blur-md bg-white/70 dark:bg-gray-900/70 border border-white/30 dark:border-gray-700/30 shadow-lg"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        {/* Page content */}
        <Outlet />
      </main>
    </div>
  );
}
