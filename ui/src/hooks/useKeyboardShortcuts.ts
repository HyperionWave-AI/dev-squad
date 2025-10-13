import { useEffect } from 'react';

export interface KeyboardShortcut {
  key: string;
  metaKey?: boolean;
  ctrlKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  handler: () => void;
}

/**
 * Hook for registering global keyboard shortcuts
 * @param shortcuts - Array of keyboard shortcuts to register
 *
 * @example
 * useKeyboardShortcuts([
 *   { key: 'k', metaKey: true, handler: () => focusSearch() },
 *   { key: 'Escape', handler: () => clearSearch() }
 * ]);
 */
export const useKeyboardShortcuts = (shortcuts: KeyboardShortcut[]): void => {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Find matching shortcut
      const matchingShortcut = shortcuts.find((shortcut) => {
        // Check if key matches (case-insensitive)
        const keyMatches = event.key.toLowerCase() === shortcut.key.toLowerCase();

        // Check modifier keys
        // For metaKey: allow both metaKey and ctrlKey (cross-platform Cmd/Ctrl support)
        // For ctrlKey: strict ctrlKey only (when explicitly specified)
        const metaKeyMatches = shortcut.metaKey
          ? (event.metaKey || event.ctrlKey)
          : !event.metaKey && !event.ctrlKey;

        const ctrlKeyMatches = shortcut.ctrlKey !== undefined
          ? shortcut.ctrlKey === event.ctrlKey
          : true;

        const shiftKeyMatches = shortcut.shiftKey !== undefined
          ? shortcut.shiftKey === event.shiftKey
          : true;

        const altKeyMatches = shortcut.altKey !== undefined
          ? shortcut.altKey === event.altKey
          : true;

        // If both metaKey and ctrlKey are undefined, allow matching without modifier check
        const needsMetaCheck = shortcut.metaKey !== undefined;
        const needsCtrlCheck = shortcut.ctrlKey !== undefined;

        return keyMatches &&
          (needsMetaCheck ? metaKeyMatches : true) &&
          (needsCtrlCheck ? ctrlKeyMatches : true) &&
          shiftKeyMatches &&
          altKeyMatches;
      });

      // If found, prevent default and call handler
      if (matchingShortcut) {
        event.preventDefault();
        matchingShortcut.handler();
      }
    };

    // Add event listener
    window.addEventListener('keydown', handleKeyDown);

    // Cleanup on unmount
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [shortcuts]);
};
