/**
 * ErrorBoundary — catches render errors in child components.
 *
 * Implements `static getDerivedStateFromError` for fallback UI
 * and `componentDidCatch` for error logging.
 *
 * @example
 *   <ErrorBoundary>
 *     <AppShell />
 *   </ErrorBoundary>
 *
 *   // With custom fallback:
 *   <ErrorBoundary fallback={<CustomErrorUI />}>...</ErrorBoundary>
 */

"use client";

import * as React from "react";
import { type ReactNode } from "react";
import { Button } from "@/components/shared/Button";

interface ErrorBoundaryProps {
  /** Child components to monitor for errors. */
  children: ReactNode;
  /** Optional custom fallback UI. Defaults to built-in error display. */
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  /** Whether an error has been caught. */
  hasError: boolean;
  /** The caught error, if any. */
  error: Error | null;
}

/**
 * ErrorBoundary — class-based error boundary component.
 *
 * Catches errors during rendering, in lifecycle methods, and in
 * constructors of the whole tree below. Does NOT catch:
 * - Event handlers (use try/catch in handlers)
 * - Async code (use try/catch in async functions)
 * - Server-side rendering
 * - Errors in the error boundary itself
 */
export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  /**
   * Called after an error is thrown by a descendant component.
   * Updates state to trigger fallback render.
   */
  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  /**
   * Called after an error is caught.
   * Used for side effects (logging, analytics).
   */
  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    // Log error with component stack for debugging
    console.error("[ErrorBoundary] Caught error:", error);
    console.error("[ErrorBoundary] Component stack:", info.componentStack);
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      // Custom fallback takes precedence
      if (this.props.fallback != null) {
        return this.props.fallback;
      }

      // Built-in fallback UI
      return (
        <div className="flex flex-col items-center justify-center py-12">
          <h2 className="text-xl font-semibold text-text-primary">
            Something went wrong
          </h2>
          <p className="mt-2 max-w-md text-center text-sm text-text-secondary">
            {this.state.error?.message ?? "An unexpected error occurred."}
          </p>
          <Button
            onClick={() => this.setState({ hasError: false, error: null })}
            className="mt-4"
          >
            Try again
          </Button>
        </div>
      );
    }

    return this.props.children;
  }
}
