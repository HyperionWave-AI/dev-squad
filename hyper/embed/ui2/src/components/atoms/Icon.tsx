import React from 'react';
import { cn } from '@/utils';

export interface IconProps extends React.SVGAttributes<SVGElement> {
  children: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeClasses = {
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
  xl: 'w-8 h-8',
};

export function Icon({ children, size = 'md', className, ...props }: IconProps) {
  return (
    <span className={cn('inline-flex items-center justify-center', sizeClasses[size], className)} {...props}>
      {children}
    </span>
  );
}
