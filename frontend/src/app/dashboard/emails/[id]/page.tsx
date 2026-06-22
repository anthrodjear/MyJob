/**
 * Email detail page — Server Component wrapper for [id] route.
 *
 * Shows loading skeleton, error state with role="alert", and detail view.
 * Uses scroll: false on navigation to preserve scroll position.
 */

"use client";

import { useParams, useRouter } from "next/navigation";
import { useEmail } from "@/hooks/useEmails";
import { EmailDetail } from "@/components/emails/EmailDetail";

export default function EmailDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { data: email, isLoading, error } = useEmail(params.id);

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl space-y-4" aria-busy="true">
        <div className="h-8 w-24 animate-pulse rounded bg-muted" />
        <div className="h-12 w-2/3 animate-pulse rounded bg-muted" />
        <div className="h-4 w-1/3 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-lg border bg-card" />
      </div>
    );
  }

  if (error || !email) {
    return (
      <div className="mx-auto max-w-4xl text-center py-12" role="alert">
        <p className="text-muted-foreground">
          {error instanceof Error ? error.message : "Email not found"}
        </p>
        <button
          type="button"
          onClick={() => router.push("/dashboard/emails", { scroll: false })}
          className="mt-4 text-sm text-primary underline-offset-4 hover:underline"
        >
          Back to emails
        </button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl">
      <EmailDetail
        email={email}
        onBack={() => router.push("/dashboard/emails", { scroll: false })}
      />
    </div>
  );
}
