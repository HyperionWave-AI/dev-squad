import React, { useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Menu, RefreshCw, Sun, Moon } from 'lucide-react';
import { Button } from '@atoms/Button';
import { Badge } from '@atoms/Badge';
import { NavItem } from '@molecules/NavItem';
import { cn } from '@/utils';
import * as Switch from '@radix-ui/react-switch';

export interface NavItemType {
  path: string;
  label: string;
  icon: React.ReactNode;
  priority: 'high' | 'medium' | 'low';
}

export interface HeaderProps {
  navigationItems: NavItemType[];
  onRefresh: () => void;
  onMobileMenuToggle: () => void;
  theme: 'light' | 'dark';
  onThemeToggle: () => void;
  className?: string;
}

export function Header({
  navigationItems,
  onRefresh,
  onMobileMenuToggle,
  theme,
  onThemeToggle,
  className,
}: HeaderProps) {
  const location = useLocation();
  const [isScrolled, setIsScrolled] = useState(false);

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 10);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const currentPage = navigationItems.find((item) => item.path === location.pathname);

  return (
    <header
      className={cn(
        'sticky top-0 z-[1100] w-full backdrop-blur-sm transition-all duration-200',
        isScrolled
          ? 'bg-[var(--header-bg-scrolled)] shadow-[var(--header-shadow-scrolled)]'
          : 'bg-[var(--header-bg)] shadow-[var(--header-shadow)]',
        'border-b border-[var(--header-border)]',
        className
      )}
    >
      <div className="flex items-center justify-between w-full min-h-[56px] md:min-h-[64px] px-4 md:px-6 lg:px-8 gap-2 md:gap-4">
        {/* Left section: Mobile menu + Logo + Page indicator */}
        <div className="flex items-center gap-2 md:gap-3 flex-1 min-w-0">
          {/* Mobile menu button */}
          <Button
            variant="ghost"
            size="icon"
            onClick={onMobileMenuToggle}
            className="md:hidden flex-shrink-0"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </Button>

          {/* Logo */}
          <div className="flex items-center gap-2 min-w-0">
            <h1 className="text-lg md:text-xl lg:text-2xl font-bold bg-gradient-to-r from-primary-600 to-secondary-600 bg-clip-text text-transparent whitespace-nowrap overflow-hidden text-ellipsis">
              Hyperion
            </h1>

            {/* Current page indicator (mobile) */}
            {currentPage && (
              <Badge variant="outline" className="md:hidden flex-shrink-0">
                {currentPage.label}
              </Badge>
            )}
          </div>
        </div>

        {/* Center section: Desktop navigation */}
        <nav className="hidden md:flex items-center gap-1 lg:gap-2">
          {navigationItems
            .filter((item) => item.priority === 'high' || item.priority === 'medium')
            .map((item) => (
              <NavItem
                key={item.path}
                to={item.path}
                icon={item.icon}
                label={item.label}
                active={location.pathname === item.path}
              />
            ))}
        </nav>

        {/* Right section: Actions */}
        <div className="flex items-center gap-1 md:gap-2 flex-shrink-0">
          {/* Refresh button */}
          <Button
            variant="ghost"
            size="icon"
            onClick={onRefresh}
            aria-label="Refresh"
            className="hover:rotate-90 transition-transform duration-200"
          >
            <RefreshCw className="h-5 w-5" />
          </Button>

          {/* Theme toggle */}
          <div className="flex items-center gap-2 px-2">
            <Sun className="h-4 w-4 text-gray-500" />
            <Switch.Root
              checked={theme === 'dark'}
              onCheckedChange={onThemeToggle}
              className={cn(
                'w-11 h-6 rounded-full relative transition-colors',
                theme === 'dark' ? 'bg-primary-500' : 'bg-gray-300'
              )}
            >
              <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform duration-100 translate-x-0.5 will-change-transform data-[state=checked]:translate-x-[22px]" />
            </Switch.Root>
            <Moon className="h-4 w-4 text-gray-500" />
          </div>
        </div>
      </div>
    </header>
  );
}
