/**
 * Approvals error boundary — catches errors in the approvals route segment.
 *
 * Displays a user-friendly error message with retry and navigation options.
 */

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/shared/Button";

interface ApprovalsErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

/** Maps technical errors to user-friendly messages. */
function getUserMessage(error: Error): string {
  const msg = error.message.toLowerCase();
  if (msg.includes("unauthorized") || msg.includes("401")) {
    return "Your session has expired. Please sign in again.";
  }
  if (msg.includes("not found") || msg.includes("404")) {
    return "Approval request not found. It may have been processed already.";
  }
  if (msg.includes("network") || msg.includes("fetch")) {
    return "Unable to connect to the server. Please check your connection.";
  }
  return "Something went wrong while loading approvals.";
}

export default function ApprovalsError({ error, reset }: ApprovalsErrorProps) {
  const router = useRouter();

  useEffect(() => {
    console.error("Approvals error:", error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center py-16 text-center" aria-live="assertive">
      <h2 className="mb-2 text-lg font-semibold text-text-primary">
        Approval Error
      </h2>
      <p className="mb-6 max-w-md text-sm text-text-secondary">
        {getUserMessage(error)}
      </p>
      <div className="flex gap-3">
        <Button variant="primary" onClick={reset}>
          Try Again
        </Button>
        <Button variant="secondary" onClick={() => router.push("/dashboard")}>
          Back to Dashboard
        </Button>
      </div>
    </div>
  );
}
