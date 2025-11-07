import { useState } from 'react';
import { Button } from '@/components/atoms/Button';
import { AlertTriangle } from 'lucide-react';

/**
 * ErrorBoundaryTest - Test component to verify ErrorBoundary functionality
 *
 * This component allows you to trigger intentional errors to test that:
 * 1. Error boundaries catch React component errors
 * 2. Fallback UI is displayed correctly
 * 3. Reload and Go Back buttons work
 * 4. Error doesn't crash the entire application
 *
 * HOW TO USE:
 * 1. Import this component into a page wrapped with ErrorBoundary
 * 2. Click "Trigger Render Error" to test
 * 3. Verify fallback UI appears with error message
 * 4. Test reload and go back buttons
 * 5. Remove this component after testing
 */
export function ErrorBoundaryTest() {
  const [shouldError, setShouldError] = useState(false);

  if (shouldError) {
    // Intentionally throw error to test ErrorBoundary
    throw new Error('Test error from ErrorBoundaryTest component - ErrorBoundary is working!');
  }

  return (
    <div className="fixed bottom-4 right-4 z-50">
      <div className="bg-yellow-50 dark:bg-yellow-900/20 border-2 border-yellow-400 dark:border-yellow-600 rounded-lg p-4 shadow-xl max-w-sm">
        <div className="flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h3 className="font-semibold text-yellow-900 dark:text-yellow-100 mb-1">
              Error Boundary Test
            </h3>
            <p className="text-sm text-yellow-800 dark:text-yellow-200 mb-3">
              Click the button below to trigger an intentional error and verify the ErrorBoundary catches it.
            </p>
            <Button
              variant="danger"
              size="sm"
              onClick={() => setShouldError(true)}
              className="w-full"
            >
              🧪 Trigger Render Error
            </Button>
            <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-2">
              <strong>Expected:</strong> Error boundary fallback UI should appear
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ErrorBoundaryTest;
