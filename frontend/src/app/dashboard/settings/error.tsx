/**
 * Settings error boundary — catches errors in the settings route segment.
 *
 * Displays a user-friendly error message with retry and navigation options.
 */

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/shared/Button";

interface SettingsErrorProps {
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
    return "Profile not found. It may need to be created first.";
  }
  if (msg.includes("network") || msg.includes("fetch")) {
    return "Unable to connect to the server. Please check your connection.";
  }
  if (msg.includes("version") || msg.includes("conflict")) {
    return "Your profile was modified elsewhere. Please refresh and try again.";
  }
  return "Something went wrong while loading settings.";
}

export default function SettingsError({ error, reset }: SettingsErrorProps) {
  const router = useRouter();

  useEffect(() => {
    console.error("Settings error:", error);
  }, [error]);

  return (
    <div
      className="flex flex-col items-center justify-center py-16 text-center"
      aria-live="assertive"
      role="alert"
    >
      <h2 className="mb-2 text-lg font-semibold text-text-primary">
        Settings Error
      </h2>
      <p className="mb-6 max-w-md text-sm text-text-secondary">
        {getUserMessage(error)}
      </p>
      <div className="flex gap-3">
        <Button variant="primary" onClick={reset}>
          Try Again
        </Button>
        <Button variant="secondary" onClick={() => router.push("/dashboard", { scroll: false })}>
          Back to Dashboard
        </Button>
      </div>
    </div>
  );
}
