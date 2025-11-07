import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, ArrowLeft } from 'lucide-react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * ErrorBoundary - React error boundary component to catch and handle errors gracefully
 *
 * Prevents single component failures from crashing the entire application.
 * Shows user-friendly error messages with recovery options.
 *
 * @example
 * ```tsx
 * <ErrorBoundary>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 */
class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  /**
   * Update state when an error is caught
   */
  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return {
      hasError: true,
      error,
    };
  }

  /**
   * Log error details and call optional error handler
   */
  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log error to console in development
    if (import.meta.env.DEV) {
      console.error('ErrorBoundary caught an error:', error);
      console.error('Component stack:', errorInfo.componentStack);
    }

    // Update state with error info
    this.setState({
      errorInfo,
    });

    // Call optional error handler prop
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // TODO: Send error to logging service in production
    // Example: logErrorToService(error, errorInfo);
  }

  /**
   * Reload the current page
   */
  handleReload = (): void => {
    window.location.reload();
  };

  /**
   * Navigate back in browser history
   */
  handleGoBack = (): void => {
    window.history.back();
  };

  /**
   * Reset error state to try rendering children again
   */
  handleReset = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback UI with glassmorphic design
      return (
        <div className="min-h-screen flex items-center justify-center p-6 bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800">
          <div className="w-full max-w-2xl">
            {/* Error Card - Glassmorphic Design */}
            <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl shadow-xl border border-gray-200 dark:border-gray-700 p-8">
              {/* Error Icon */}
              <div className="flex items-center justify-center mb-6">
                <div className="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center">
                  <AlertTriangle className="w-8 h-8 text-red-600 dark:text-red-400" />
                </div>
              </div>

              {/* Error Title */}
              <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-4">
                Something went wrong
              </h1>

              {/* Error Message */}
              <div className="mb-6">
                <p className="text-center text-gray-600 dark:text-gray-300 mb-4">
                  An unexpected error occurred. Don't worry, your data is safe.
                </p>

                {this.state.error && (
                  <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-sm font-mono text-red-800 dark:text-red-300 break-all">
                      {this.state.error.toString()}
                    </p>
                  </div>
                )}
              </div>

              {/* Stack Trace (Development Only) */}
              {import.meta.env.DEV && this.state.errorInfo && (
                <details className="mb-6">
                  <summary className="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white mb-2">
                    Show technical details
                  </summary>
                  <div className="bg-gray-100 dark:bg-gray-900 rounded-lg p-4 overflow-auto max-h-64">
                    <pre className="text-xs font-mono text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                      {this.state.errorInfo.componentStack}
                    </pre>
                  </div>
                </details>
              )}

              {/* Action Buttons */}
              <div className="flex flex-col sm:flex-row gap-3 justify-center">
                <button
                  onClick={this.handleReload}
                  className="flex items-center justify-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors duration-200 shadow-md hover:shadow-lg"
                >
                  <RefreshCw className="w-4 h-4" />
                  Reload Page
                </button>

                <button
                  onClick={this.handleGoBack}
                  className="flex items-center justify-center gap-2 px-6 py-3 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded-lg transition-colors duration-200 shadow-md hover:shadow-lg"
                >
                  <ArrowLeft className="w-4 h-4" />
                  Go Back
                </button>

                {import.meta.env.DEV && (
                  <button
                    onClick={this.handleReset}
                    className="flex items-center justify-center gap-2 px-6 py-3 bg-yellow-500 hover:bg-yellow-600 text-white font-medium rounded-lg transition-colors duration-200 shadow-md hover:shadow-lg"
                  >
                    Try Again
                  </button>
                )}
              </div>

              {/* Help Text */}
              <p className="text-center text-sm text-gray-500 dark:text-gray-400 mt-6">
                If this problem persists, please contact support or try clearing your browser cache.
              </p>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
