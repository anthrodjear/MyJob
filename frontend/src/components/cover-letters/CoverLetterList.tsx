/**
 * CoverLetterList — renders a list of cover letters with loading and empty states.
 *
 * Uses CoverLetterCard for each item. Shows skeleton loader during initial load.
 * Shows EmptyState when no cover letters exist.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <CoverLetterList coverLetters={coverLetters} isLoading={false} />
 */

"use client";

import { Mail } from "lucide-react";
import type { CoverLetter } from "@/lib/types/resumes";
import { CoverLetterCard } from "./CoverLetterCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { CoverLetterCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";

interface CoverLetterListProps {
  /** Array of cover letters to display. */
  coverLetters: CoverLetter[];
  /** Whether data is currently loading (initial load, not placeholder). */
  isLoading?: boolean;
}

/** Skeleton placeholder matching the list layout. */
function CoverLetterListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading cover letters">
      <span className="sr-only" aria-live="polite">Loading cover letters…</span>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <CoverLetterCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function CoverLetterList({ coverLetters, isLoading = false }: CoverLetterListProps) {
  // Empty state
  if (coverLetters.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<Mail className="h-12 w-12" />}
        title="No cover letters yet"
        description="Cover letters are generated when you apply to jobs. Submit an application to get started."
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<CoverLetterListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Cover letters loaded"
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {coverLetters.map((cl) => (
          <CoverLetterCard key={cl.id} coverLetter={cl} />
        ))}
      </div>
    </SkeletonWrapper>
  );
}
