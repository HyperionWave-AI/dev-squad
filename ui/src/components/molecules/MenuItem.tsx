import React from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { cn } from '@/utils';

export interface MenuItemProps {
  icon?: React.ReactNode;
  label: string;
  onClick?: () => void;
  disabled?: boolean;
  className?: string;
}

export function MenuItem({ icon, label, onClick, disabled, className }: MenuItemProps) {
  return (
    <DropdownMenu.Item
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'relative flex cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors',
        'focus:bg-gray-100 dark:focus:bg-gray-700',
        'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        className
      )}
    >
      {icon && <span className="flex items-center justify-center">{icon}</span>}
      <span>{label}</span>
    </DropdownMenu.Item>
  );
}
