import React from 'react';
import { Library } from 'lucide-react';
import { Badge } from '@/components/atoms/Badge';
import { Icon } from '@/components/atoms/Icon';
import { cn } from '@/utils';

export interface KnowledgeHeaderProps {
  totalCollections: number;
  totalEntries: number;
  className?: string;
}

export function KnowledgeHeader({
  totalCollections,
  totalEntries,
  className,
}: KnowledgeHeaderProps) {
  return (
    <div
      className={cn(
        // Glassmorphism effect
        'backdrop-blur-xl bg-white/70 dark:bg-gray-900/70',
        'border border-white/20 dark:border-gray-700/50',
        'shadow-xl',
        'rounded-2xl',
        // Layout
        'p-8',
        'sticky top-0 z-10',
        // Transitions
        'transition-all duration-300',
        'hover:shadow-2xl hover:bg-white/80 dark:hover:bg-gray-900/80',
        className
      )}
    >
      <div className="flex items-center justify-between">
        {/* Left Section: Icon + Title + Subtitle */}
        <div className="flex items-center gap-6">
          {/* Icon with gradient background */}
          <div className="relative group">
            <div className="absolute inset-0 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl blur-lg opacity-60 group-hover:opacity-80 transition-opacity duration-300"></div>
            <div className="relative bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl p-4 shadow-lg">
              <Icon size="xl" className="text-white">
                <Library className="w-8 h-8" />
              </Icon>
            </div>
          </div>

          {/* Title and Subtitle */}
          <div className="space-y-2">
            <h1 className="text-4xl font-bold bg-gradient-to-r from-gray-900 to-gray-700 dark:from-white dark:to-gray-300 bg-clip-text text-transparent">
              Knowledge Base
            </h1>
            <p className="text-sm text-gray-600 dark:text-gray-400 font-medium">
              Explore, search, and manage your knowledge collections
            </p>
          </div>
        </div>

        {/* Right Section: Stats Badges */}
        <div className="flex items-center gap-4">
          {/* Total Collections Badge */}
          <div className="group relative">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-500 to-blue-600 rounded-full blur-md opacity-50 group-hover:opacity-70 transition-opacity duration-300"></div>
            <div className="relative flex items-center gap-3 px-6 py-3 bg-gradient-to-r from-blue-500 to-blue-600 rounded-full shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105">
              <div className="flex flex-col items-center">
                <span className="text-xs font-semibold text-blue-100 uppercase tracking-wider">
                  Collections
                </span>
                <span className="text-2xl font-bold text-white">
                  {totalCollections}
                </span>
              </div>
            </div>
          </div>

          {/* Total Entries Badge */}
          <div className="group relative">
            <div className="absolute inset-0 bg-gradient-to-r from-purple-500 to-purple-600 rounded-full blur-md opacity-50 group-hover:opacity-70 transition-opacity duration-300"></div>
            <div className="relative flex items-center gap-3 px-6 py-3 bg-gradient-to-r from-purple-500 to-purple-600 rounded-full shadow-lg hover:shadow-xl transition-all duration-300 hover:scale-105">
              <div className="flex flex-col items-center">
                <span className="text-xs font-semibold text-purple-100 uppercase tracking-wider">
                  Entries
                </span>
                <span className="text-2xl font-bold text-white">
                  {totalEntries}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
