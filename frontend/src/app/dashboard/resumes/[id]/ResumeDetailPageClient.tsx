/**
 * ResumeDetailPageClient — resume detail with content sections.
 *
 * Client Component (uses hooks for data fetching and mutations).
 */

"use client";

import { useResume } from "@/hooks/useResumes";
import { ResumeDetail } from "@/components/resumes/ResumeDetail";
import { CardSkeleton } from "@/components/shared/LoadingSkeleton";

interface ResumeDetailPageClientProps {
  id: string;
}

/**
 * Map raw error to safe user-friendly message.
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("not_found")) {
      return "Resume not found.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
  }
  return "Failed to load resume. Please try again.";
}

export function ResumeDetailPageClient({ id }: ResumeDetailPageClientProps) {
  const { data: resume, isLoading, error } = useResume(id);

  if (error != null) {
    return (
      <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
        {getErrorMessage(error)}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <CardSkeleton />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  if (resume == null) {
    return (
      <div role="alert" className="text-sm text-text-secondary">
        Resume not found.
      </div>
    );
  }

  return <ResumeDetail resume={resume} />;
}
