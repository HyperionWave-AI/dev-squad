/**
 * Date utility functions for consistent date handling across the application
 */

/**
 * Checks if a date value is valid
 */
export function isValidDate(date: any): boolean {
  if (!date) return false;
  
  const dateObj = date instanceof Date ? date : new Date(date);
  return dateObj instanceof Date && !isNaN(dateObj.getTime());
}

/**
 * Formats a date for session display with fallback handling
 */
export function formatSessionDate(date: any): string {
  if (!isValidDate(date)) {
    return 'Unknown';
  }
  
  const dateObj = date instanceof Date ? date : new Date(date);
  const now = new Date();
  const diffMs = now.getTime() - dateObj.getTime();
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
  // Return relative time for recent dates
  if (diffMinutes < 1) {
    return 'Just now';
  } else if (diffMinutes < 60) {
    return `${diffMinutes} minute${diffMinutes === 1 ? '' : 's'} ago`;
  } else if (diffHours < 24) {
    return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
  } else if (diffDays < 7) {
    return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
  } else {
    // Return formatted date for older dates
    return dateObj.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: dateObj.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
    });
  }
}

/**
 * Formats relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(date: any): string {
  if (!isValidDate(date)) {
    return 'Unknown time';
  }
  
  return formatSessionDate(date);
}

/**
 * Formats absolute date (e.g., "Nov 6, 2024")
 */
export function formatAbsoluteDate(date: any): string {
  if (!isValidDate(date)) {
    return 'Unknown date';
  }
  
  const dateObj = date instanceof Date ? date : new Date(date);
  return dateObj.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  });
}

/**
 * Safe date creation with validation
 */
export function createSafeDate(dateInput: any): Date {
  if (!dateInput) {
    return new Date();
  }
  
  if (dateInput instanceof Date) {
    return isValidDate(dateInput) ? dateInput : new Date();
  }
  
  const parsed = new Date(dateInput);
  return isValidDate(parsed) ? parsed : new Date();
}