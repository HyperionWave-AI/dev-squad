import type { ReactNode } from 'react';
import { cn } from '@/utils/cn';

interface PageHeaderProps {
  /**
   * The main title of the page
   */
  title: string;

  /**
   * Optional description text shown below the title
   */
  description?: string;

  /**
   * Icon to display - can be any React node (e.g., Lucide icon component)
   */
  icon: ReactNode;

  /**
   * Starting color for the gradient background (CSS color value)
   */
  gradientFrom: string;

  /**
   * Ending color for the gradient background (CSS color value)
   */
  gradientTo: string;

  /**
   * Optional additional CSS classes
   */
  className?: string;
}

/**
 * PageHeader - A unified glassmorphic header component for all pages
 *
 * Features:
 * - Glassmorphic background with blur effect
 * - Animated gradient icon with pulse effect
 * - Responsive design
 * - Dark mode support
 * - Customizable gradient colors per page
 *
 * @example
 * ```tsx
 * <PageHeader
 *   title="Task Board"
 *   description="Manage and track your tasks"
 *   icon={<LayoutDashboard />}
 *   gradientFrom="#a855f7"
 *   gradientTo="#6366f1"
 * />
 * ```
 */
export function PageHeader({
  title,
  description,
  icon,
  gradientFrom,
  gradientTo,
  className,
}: PageHeaderProps) {
  return (
    <div
      className={cn(
        // Glassmorphic container
        'backdrop-blur-md',
        'bg-white/70 dark:bg-gray-800/70',
        'border border-white/30 dark:border-gray-700/30',
        'rounded-lg',
        'p-6',
        'shadow-lg',
        // Smooth transitions
        'transition-all duration-300',
        className
      )}
    >
      <div className="flex items-center gap-3">
        {/* Animated Icon Container */}
        <div className="relative">
          {/* Animated glow effect */}
          <div
            className="absolute inset-0 rounded-xl blur-lg opacity-30 animate-pulse"
            style={{
              background: `linear-gradient(to bottom right, ${gradientFrom}, ${gradientTo})`,
            }}
          />

          {/* Icon with gradient background */}
          <div
            className="relative p-3 rounded-xl shadow-xl"
            style={{
              background: `linear-gradient(to bottom right, ${gradientFrom}, ${gradientTo})`,
            }}
          >
            <div className="h-8 w-8 text-white flex items-center justify-center">
              {icon}
            </div>
          </div>
        </div>

        {/* Title and Description */}
        <div className="flex-1">
          <h1
            className="text-3xl font-bold bg-clip-text text-transparent"
            style={{
              backgroundImage: `linear-gradient(to right, ${gradientFrom}, ${gradientTo}, ${gradientFrom})`,
            }}
          >
            {title}
          </h1>
          {description && (
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              {description}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

export default PageHeader;
