import React from 'react';
import { Link } from 'react-router-dom';
import { cn } from '@/utils';

export interface NavItemProps {
  to: string;
  icon?: React.ReactNode;
  label: string;
  active?: boolean;
  onClick?: () => void;
  className?: string;
}

export function NavItem({ to, icon, label, active, onClick, className }: NavItemProps) {
  return (
    <Link
      to={to}
      onClick={onClick}
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
        'hover:bg-primary-500 hover:text-white',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2',
        active
          ? 'bg-primary-500 text-white'
          : 'bg-transparent text-gray-700 dark:text-gray-200',
        className
      )}
    >
      {icon && <span className="flex items-center justify-center">{icon}</span>}
      <span>{label}</span>
    </Link>
  );
}
