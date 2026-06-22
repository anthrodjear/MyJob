/**
 * JobList — responsive grid of job cards.
 *
 * Handles loading, empty, and error states.
 * Renders jobs in a responsive grid that adapts from 1 to 3 columns.
 * Uses JobCard for individual job display.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <JobList jobs={jobs} isLoading={false} onApply={handleApply} />
 */

import { Briefcase, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { JobCard } from "./JobCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import type { Job } from "@/lib/types/jobs";

interface JobListProps {
  /** Array of jobs to display. */
  jobs: Job[];
  /** Whether data is currently loading. */
  isLoading?: boolean;
  /** Error message to display. */
  error?: string;
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
  /** Callback when the "Try Again" error action is clicked. */
  onRetry?: () => void;
  /** Additional CSS classes for the grid container. */
  className?: string;
}

/**
 * Loading skeleton — shows 6 placeholder cards in a grid.
 */
function JobListSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-border bg-bg-secondary p-4"
        >
          <Skeleton className="mb-3 h-5 w-3/4" />
          <Skeleton className="mb-2 h-4 w-1/2" />
          <Skeleton className="mb-3 h-3 w-full" />
          <Skeleton className="mb-4 h-2 w-full" />
          <div className="flex gap-2">
            <Skeleton className="h-8 w-16" />
            <Skeleton className="h-8 w-16" />
          </div>
        </div>
      ))}
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
 * - Error state uses `role="alert"` for immediate announcement
 */
export function JobList({
  jobs,
  isLoading = false,
  error,
  onApply,
  onScore,
  onSave,
  onArchive,
  onSearch,
  onRetry,
  className,
}: JobListProps) {
  // Loading state
  if (isLoading) {
    return (
      <div aria-busy="true" aria-label="Loading jobs">
        <span className="sr-only" aria-live="polite">Loading jobs…</span>
        <JobListSkeleton />
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div
        role="alert"
        className="rounded-lg border border-danger-light bg-danger-light/10 p-4 text-center"
      >
        <AlertTriangle className="mx-auto mb-2 h-6 w-6 text-danger-dark" />
        <p className="text-sm text-danger-dark">{error}</p>
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="mt-3 rounded-md bg-danger px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-danger-hover focus:outline-none focus:ring-2 focus:ring-danger focus:ring-offset-2"
          >
            Try Again
          </button>
        )}
      </div>
    );
  }

  // Empty state
  if (jobs.length === 0) {
    return (
      <EmptyState
        icon={<Briefcase className="h-12 w-12" />}
        title="No jobs found"
        description="Start a search to discover new opportunities."
        action={onSearch ? { label: "Start Search", onClick: onSearch } : undefined}
      />
    );
  }

  // Job grid
  return (
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
  );
}
