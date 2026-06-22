/**
 * Interview detail page — Server Component wrapper for [id] route.
 *
 * Shows loading skeleton, error state with role="alert", and detail view.
 * Uses scroll: false on navigation to preserve scroll position.
 */

"use client";

import { useParams, useRouter } from "next/navigation";
import { useInterview } from "@/hooks/useInterviews";
import { InterviewDetail } from "@/components/interviews/InterviewDetail";

/** Maps technical error messages to safe, user-friendly strings. */
function getUserMessage(error: unknown): string {
  if (!(error instanceof Error)) return "Interview not found.";
  const msg = error.message.toLowerCase();
  if (msg.includes("network") || msg.includes("fetch")) {
    return "Network error — check your connection and try again.";
  }
  if (msg.includes("404") || msg.includes("not found")) {
    return "Interview not found — it may have been removed.";
  }
  return "Something went wrong loading this interview.";
}

export default function InterviewDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { data: interview, isLoading, error } = useInterview(params.id);

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl space-y-4" aria-busy="true">
        <div className="h-8 w-24 animate-pulse rounded bg-muted" />
        <div className="h-12 w-1/2 animate-pulse rounded bg-muted" />
        <div className="grid gap-4 sm:grid-cols-3">
          {Array.from({ length: 3 }, (_, i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg border bg-card" />
          ))}
        </div>
        <div className="h-64 animate-pulse rounded-lg border bg-card" />
      </div>
    );
  }

  if (error || !interview) {
    return (
      <div className="mx-auto max-w-4xl text-center py-12" role="alert">
        <p className="text-muted-foreground">
          {getUserMessage(error)}
        </p>
        <button
          type="button"
          onClick={() => router.push("/dashboard/interviews", { scroll: false })}
          className="mt-4 text-sm text-primary underline-offset-4 hover:underline"
        >
          Back to interviews
        </button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl">
      <InterviewDetail
        interview={interview}
        onBack={() => router.push("/dashboard/interviews", { scroll: false })}
      />
    </div>
  );
}
