/**
 * Jobs Error Boundary — catches runtime errors in the jobs route segment.
 *
 * Renders a user-friendly error message with recovery actions:
 * - "Try again" re-throws to trigger Next.js error boundary reset
 * - "Go to dashboard" navigates to /dashboard
 *
 * Follows the getUserMessage pattern from /dashboard/error.tsx:
 * maps technical errors to safe, user-friendly strings.
 *
 * @see /app/dashboard/error.tsx — same pattern
 */

"use client";

import { useRouter } from "next/navigation";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/shared/Button";

/**
 * Map technical error messages to user-friendly strings.
 * Never expose raw error details to the user.
 */
function getUserMessage(error: Error): string {
  const msg = error.message.toLowerCase();

  if (msg.includes("network") || msg.includes("fetch")) {
    return "Network error — check your connection and try again.";
  }
  if (msg.includes("401") || msg.includes("unauthorized")) {
    return "Session expired — please refresh the page.";
  }
  if (msg.includes("404") || msg.includes("not found")) {
    return "Job not found — it may have been removed.";
  }
  if (msg.includes("500") || msg.includes("internal")) {
    return "Server error — please try again later.";
  }
  if (msg.includes("timeout")) {
    return "Request timed out — please try again.";
  }

  return "Something went wrong loading jobs.";
}

interface JobsErrorProps {
  error: Error;
  reset: () => void;
}

/**
 * JobsError — error boundary fallback for the jobs route.
 *
 * Accessibility:
 * - role="alert" for immediate screen reader announcement
 * - Error message uses getUserMessage for safe display
 * - Both actions are keyboard accessible
 */
export default function JobsError({ error, reset }: JobsErrorProps) {
  const router = useRouter();

  return (
    <div
      role="alert"
      className="flex flex-col items-center justify-center gap-4 py-16 text-center"
    >
      <AlertTriangle
        className="h-12 w-12 text-danger"
        aria-hidden="true"
      />
      <h2 className="text-xl font-semibold text-text-primary">
        Failed to load jobs
      </h2>
      <p className="max-w-md text-text-secondary">
        {getUserMessage(error)}
      </p>
      <div className="flex gap-3">
        <Button
          onClick={() => reset()}
          variant="primary"
        >
          Try again
        </Button>
        <Button
          onClick={() => router.push("/dashboard")}
          variant="secondary"
        >
          Go to dashboard
        </Button>
      </div>
    </div>
  );
}
