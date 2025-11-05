import React from 'react';
import { useLocation } from 'react-router-dom';
import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import { Button } from '@atoms/Button';
import { NavItem } from '@molecules/NavItem';
import { cn } from '@/utils';

export interface NavItemType {
  path: string;
  label: string;
  icon: React.ReactNode;
  priority: 'high' | 'medium' | 'low';
}

export interface NavigationProps {
  navigationItems: NavItemType[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function Navigation({ navigationItems, open, onOpenChange }: NavigationProps) {
  const location = useLocation();

  const handleNavClick = () => {
    onOpenChange(false);
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        {/* Overlay */}
        <Dialog.Overlay className="fixed inset-0 z-[1200] bg-black/50 data-[state=open]:animate-fade-in" />

        {/* Drawer content */}
        <Dialog.Content
          className={cn(
            'fixed top-0 left-0 z-[1200] h-full w-[85vw] max-w-[400px]',
            'bg-[var(--drawer-bg)] border-r border-[var(--drawer-border)]',
            'shadow-2xl',
            'data-[state=open]:animate-slide-in',
            'focus:outline-none'
          )}
        >
          {/* Drawer header */}
          <div className="flex items-center justify-between min-h-[56px] md:min-h-[64px] px-4 border-b border-[var(--drawer-border)]">
            <Dialog.Title className="text-lg font-bold bg-gradient-to-r from-primary-600 to-secondary-600 bg-clip-text text-transparent">
              Hyperion
            </Dialog.Title>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon" aria-label="Close menu">
                <X className="h-5 w-5" />
              </Button>
            </Dialog.Close>
          </div>

          {/* Navigation items */}
          <nav className="flex flex-col p-2 gap-1">
            {navigationItems.map((item) => (
              <NavItem
                key={item.path}
                to={item.path}
                icon={item.icon}
                label={item.label}
                active={location.pathname === item.path}
                onClick={handleNavClick}
                className={cn(
                  'w-full justify-start h-14 px-6 rounded-lg',
                  location.pathname === item.path
                    ? 'bg-[var(--drawer-item-bg-active)] text-[var(--nav-item-text-active)] font-semibold'
                    : 'hover:bg-[var(--drawer-item-bg-hover)]'
                )}
              />
            ))}
          </nav>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
