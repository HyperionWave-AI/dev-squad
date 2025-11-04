import React from 'react';
import { Search } from 'lucide-react';
import { Input } from '@atoms/Input';
import { cn } from '@/utils';

export interface SearchBarProps extends React.InputHTMLAttributes<HTMLInputElement> {
  onSearch?: (value: string) => void;
}

export function SearchBar({ className, onSearch, ...props }: SearchBarProps) {
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && onSearch) {
      onSearch(e.currentTarget.value);
    }
  };

  return (
    <div className={cn('relative w-full', className)}>
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
      <Input
        className="pl-10"
        placeholder="Search..."
        onKeyDown={handleKeyDown}
        {...props}
      />
    </div>
  );
}
