/**
 * Emails error boundary — catches and displays errors from the emails route.
 *
 * Implements getUserMessage() to map technical errors to safe, user-friendly strings.
 * Includes role="alert" for immediate screen reader announcement.
 * Provides both "Try again" and "Go to dashboard" navigation options.
 */

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/shared/Button";

interface EmailsErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

/** Maps technical error messages to safe, user-friendly strings. */
function getUserMessage(error: Error): string {
  const msg = error.message.toLowerCase();
  if (msg.includes("network") || msg.includes("fetch")) {
    return "Network error — check your connection and try again.";
  }
  if (msg.includes("401") || msg.includes("unauthorized")) {
    return "Session expired — please refresh the page.";
  }
  if (msg.includes("404") || msg.includes("not found")) {
    return "Email not found — it may have been removed.";
  }
  return "Something went wrong loading emails.";
}

export default function EmailsError({ error, reset }: EmailsErrorProps) {
  const router = useRouter();

  useEffect(() => {
    console.error("Emails page error:", error);
  }, [error]);

  return (
    <div
      role="alert"
      className="flex flex-col items-center justify-center gap-4 py-16 text-center"
    >
      <AlertTriangle className="h-12 w-12 text-destructive" aria-hidden="true" />
      <h2 className="text-xl font-semibold">Failed to load emails</h2>
      <p className="max-w-md text-muted-foreground">
        {getUserMessage(error)}
      </p>
      <div className="flex gap-3">
        <Button onClick={reset} variant="secondary">
          Try again
        </Button>
        <Button
          onClick={() => router.push("/dashboard", { scroll: false })}
          variant="secondary"
        >
          Go to dashboard
        </Button>
      </div>
    </div>
  );
}
