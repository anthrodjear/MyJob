/**
 * Error boundary for /dashboard/resumes route segment.
 *
 * Catches rendering errors in resumes list page.
 * Provides retry action and safe error message.
 */

"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/shared/Button";

/**
 * Map raw error to safe user-friendly message.
 * Never expose error.message to users.
 */
function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
  }
  return "Failed to load resumes. Please try again.";
}

interface ResumeErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function ResumeError({ error, reset }: ResumeErrorProps) {
  useEffect(() => {
    console.error("[Resumes Page Error]", error);
  }, [error]);

  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center gap-4 text-center">
      <AlertTriangle className="h-12 w-12 text-danger" aria-hidden="true" />
      <div>
        <h2 className="text-lg font-semibold text-foreground">
          Something went wrong
        </h2>
        <p className="mt-1 text-sm text-text-secondary" role="alert">
          {getUserMessage(error)}
        </p>
      </div>
      <Button variant="secondary" onClick={reset}>
        Try Again
      </Button>
    </div>
  );
}
