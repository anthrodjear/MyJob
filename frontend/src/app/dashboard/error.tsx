/**
 * Dashboard Error Boundary — catches render errors in the dashboard route.
 *
 * Provides a fallback UI with retry button when dashboard components fail.
 * Sanitizes error messages for user display.
 * Client Component — uses useState for retry state.
 */

"use client";

import { Button } from "@/components/shared/Button";

interface DashboardErrorProps {
  error: Error;
  reset: () => void;
}

/**
 * Maps technical errors to user-friendly messages.
 */
function getUserMessage(err: Error): string {
  const message = err.message.toLowerCase();

  if (message.includes("fetch") || message.includes("network") || err.name === "NetworkError") {
    return "Unable to connect to the server. Check your connection and try again.";
  }
  if (message.includes("401") || message.includes("403") || message.includes("unauthorized")) {
    return "Your session expired. Please refresh the page to sign in again.";
  }
  if (message.includes("404")) {
    return "The requested resource was not found.";
  }
  if (message.includes("500") || message.includes("502") || message.includes("503")) {
    return "The server is temporarily unavailable. Please try again in a moment.";
  }
  if (message.includes("timeout") || message.includes("timed out")) {
    return "The request timed out. Please try again.";
  }

  // Generic fallback — don't expose raw error details
  return "An unexpected error occurred while loading the dashboard. Please try again.";
}

export default function DashboardError({ error, reset }: DashboardErrorProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <h2 className="text-xl font-semibold text-text-primary">
        Dashboard failed to load
      </h2>
      <p className="mt-2 text-text-secondary text-center max-w-md">
        {getUserMessage(error)}
      </p>
      <Button onClick={reset} className="mt-4">
        Try again
      </Button>
    </div>
  );
}