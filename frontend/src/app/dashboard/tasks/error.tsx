/**
 * Error boundary for /dashboard/tasks route segment.
 */

"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/shared/Button";

function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
  }
  return "Failed to load tasks. Please try again.";
}

interface TasksErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function TasksError({ error, reset }: TasksErrorProps) {
  useEffect(() => {
    console.error("[Tasks Page Error]", error);
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
