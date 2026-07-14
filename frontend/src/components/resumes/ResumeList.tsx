/**
 * ResumeList — renders a list of resumes with loading and empty states.
 *
 * Uses ResumeCard for each item. Shows skeleton loader during initial load.
 * Shows EmptyState when no resumes exist.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <ResumeList resumes={resumes} isLoading={false} />
 */

"use client";

import { FileText } from "lucide-react";
import type { Resume } from "@/lib/types/resumes";
import { ResumeCard } from "./ResumeCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { ResumeCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";

interface ResumeListProps {
  /** Array of resumes to display. */
  resumes: Resume[];
  /** Whether data is currently loading (initial load, not placeholder). */
  isLoading?: boolean;
}

/** Skeleton placeholder matching the list layout. */
function ResumeListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading resumes">
      <span className="sr-only" aria-live="polite">Loading resumes…</span>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <ResumeCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function ResumeList({ resumes, isLoading = false }: ResumeListProps) {
  // Empty state
  if (resumes.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<FileText className="h-12 w-12" />}
        title="No resumes yet"
        description="Create your first resume to get started with AI-powered job applications."
        action={{
          label: "Create Resume",
          onClick: () => { window.location.href = "/dashboard/resumes/new"; },
        }}
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<ResumeListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Resumes loaded"
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {resumes.map((resume) => (
          <ResumeCard key={resume.id} resume={resume} />
        ))}
      </div>
    </SkeletonWrapper>
  );
}
