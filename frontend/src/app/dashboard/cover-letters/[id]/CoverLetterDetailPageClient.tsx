/**
 * CoverLetterDetailPageClient — cover letter detail with content.
 */

"use client";

import { useCoverLetter } from "@/hooks/useResumes";
import { CoverLetterDetail } from "@/components/cover-letters/CoverLetterDetail";
import { CardSkeleton } from "@/components/shared/LoadingSkeleton";

interface CoverLetterDetailPageClientProps {
  id: string;
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("not_found")) {
      return "Cover letter not found.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
  }
  return "Failed to load cover letter. Please try again.";
}

export function CoverLetterDetailPageClient({ id }: CoverLetterDetailPageClientProps) {
  const { data: coverLetter, isLoading, error } = useCoverLetter(id);

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
      </div>
    );
  }

  if (coverLetter == null) {
    return (
      <div role="alert" className="text-sm text-text-secondary">
        Cover letter not found.
      </div>
    );
  }

  return <CoverLetterDetail coverLetter={coverLetter} />;
}
