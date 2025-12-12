import React, { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { cn } from '@/utils';

interface CopyButtonProps {
  text: string;
  className?: string;
}

export const CopyButton: React.FC<CopyButtonProps> = ({ text, className }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy text: ', err);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className={cn(
        'text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300',
        'transition-colors duration-200 ease-in-out',
        'opacity-0 group-hover:opacity-100', // Only visible on hover of parent
        'focus:outline-none focus:ring-2 focus:ring-primary-300',
        className
      )}
      aria-label="Copy message"
    >
      {copied ? (
        <Check className="w-4 h-4 text-green-500" />
      ) : (
        <Copy className="w-4 h-4" />
      )}
    </button>
  );
};