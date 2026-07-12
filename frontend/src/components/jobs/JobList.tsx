/**
 * JobList — responsive grid of job cards.
 *
 * Handles loading and empty states.
 * Renders jobs in a responsive grid that adapts from 1 to 3 columns.
 * Uses JobCard for individual job display.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <JobList jobs={jobs} isLoading={false} onApply={handleApply} />
 */

"use client";

import { Briefcase } from "lucide-react";
import { cn } from "@/lib/utils";
import { JobCard } from "./JobCard";
import { JobCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";
import { EmptyState } from "@/components/shared/EmptyState";
import type { Job } from "@/lib/types/jobs";

interface JobListProps {
  /** Array of jobs to display. */
  jobs: Job[];
  /** Whether data is currently loading (initial load, not placeholder). */
  isLoading?: boolean;
  /** Callback when Apply button is clicked on a job. */
  onApply?: (jobId: string) => void;
  /** Callback when Score button is clicked on a job. */
  onScore?: (jobId: string) => void;
  /** Callback when Save/Unsave button is clicked on a job. */
  onSave?: (jobId: string, saved: boolean) => void;
  /** Callback when Archive button is clicked on a job. */
  onArchive?: (jobId: string) => void;
  /** Callback when the "Start Search" empty state action is clicked. */
  onSearch?: () => void;
  /** Additional CSS classes for the grid container. */
  className?: string;
}

/** Skeleton placeholder matching the list layout. */
function JobListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading jobs">
      <span className="sr-only" aria-live="polite">Loading jobs…</span>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <JobCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

/**
 * JobList — responsive grid of job cards.
 *
 * Accessibility:
 * - Grid uses `role="list"` with `aria-label` for screen readers
 * - Each JobCard uses `role="group"` for individual card identification
 * - Loading state uses `aria-busy` to announce loading
 */
export function JobList({
  jobs,
  isLoading = false,
  onApply,
  onScore,
  onSave,
  onArchive,
  onSearch,
  className,
}: JobListProps) {
  // Empty state
  if (jobs.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<Briefcase className="h-12 w-12" />}
        title="No jobs found"
        description="Start a search to discover new opportunities."
        action={onSearch ? { label: "Start Search", onClick: onSearch } : undefined}
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<JobListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Jobs loaded"
    >
      <div
        role="list"
        aria-label="Job listings"
        className={cn(
          "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3",
          className,
        )}
      >
        {jobs.map((job) => (
          <div key={job.id} role="listitem">
            <JobCard
              job={job}
              onApply={onApply}
              onScore={onScore}
              onSave={onSave}
              onArchive={onArchive}
            />
          </div>
        ))}
      </div>
    </SkeletonWrapper>
  );
}
