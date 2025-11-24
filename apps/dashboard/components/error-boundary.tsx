"use client";

import { Component, type ReactNode } from "react";

type ErrorBoundaryProps = {
  children: ReactNode;
  fallback?: (error: Error, reset: () => void) => ReactNode;
};

type ErrorBoundaryState = {
  hasError: boolean;
  error: Error | null;
};

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("Tree layout error:", error, errorInfo);
  }

  reset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError && this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.reset);
      }

      return (
        <div className="flex items-center justify-center w-full h-full bg-red-50 dark:bg-red-950/20">
          <div className="max-w-md p-6 bg-white dark:bg-gray-900 border border-red-200 dark:border-red-800 rounded-lg shadow-lg">
            <h2 className="text-lg font-semibold text-red-600 dark:text-red-400 mb-2">
              Something went wrong
            </h2>
            <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">
              {this.state.error.message}
            </p>
            <button
              type="button"
              onClick={this.reset}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded transition-colors"
            >
              Try again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
