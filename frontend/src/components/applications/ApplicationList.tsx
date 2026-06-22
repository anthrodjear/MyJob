/**
 * ApplicationList — paginated application list with loading, empty, and error states.
 *
 * Renders ApplicationCard components. Shows skeleton loader during initial load.
 * Used on the main applications page.
 *
 * @example
 *   <ApplicationList
 *     applications={apps}
 *     isLoading={false}
 *     onSelect={handleSelect}
 *     onStatusChange={handleStatus}
 *   />
 */

"use client";

import { Inbox } from "lucide-react";
import { cn } from "@/lib/utils";
import { ApplicationCard } from "./ApplicationCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { Button } from "@/components/shared/Button";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import type { Application } from "@/lib/types/applications";

/**
 * Loading skeleton — shows placeholder cards matching ApplicationCard layout.
 */
function ApplicationListSkeleton() {
  return (
    <div className="space-y-3" aria-busy="true" aria-label="Loading applications">
      <span className="sr-only" aria-live="polite">Loading applications…</span>
      {Array.from({ length: 5 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-border bg-bg-secondary p-4"
        >
          <div className="mb-3 flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <Skeleton className="mb-2 h-5 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
            </div>
            <div className="flex flex-col items-end gap-1">
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-5 w-14 rounded-full" />
            </div>
          </div>
          <div className="mb-3 flex gap-4">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-3 w-24" />
          </div>
          <Skeleton className="mb-3 h-3 w-full" />
          <div className="flex gap-2">
            <Skeleton className="h-8 w-20" />
          </div>
        </div>
      ))}
    </div>
  );
}

interface ApplicationListProps {
  /** Array of applications to display. */
  applications: Application[];
  /** Whether data is currently loading (shows skeleton). */
  isLoading?: boolean;
  /** Callback when an application card is selected. */
  onSelect?: (id: string) => void;
  /** Callback when a status transition is triggered. */
  onStatusChange?: (id: string, status: Application["status"]) => void;
  /** Whether more pages are available. */
  hasMore?: boolean;
  /** Callback to load the next page. */
  onLoadMore?: () => void;
  /** Whether loading more pages. */
  loadingMore?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApplicationList — paginated application listing.
 *
 * Accessibility:
 * - Uses `<ul>/<li>` for list semantics
 * - Loading state uses `aria-busy` to announce loading
 * - EmptyState has icon + description for context
 * - Load more button has aria-label
 */
export function ApplicationList({
  applications,
  isLoading = false,
  onSelect,
  onStatusChange,
  hasMore,
  onLoadMore,
  loadingMore,
  className,
}: ApplicationListProps) {
  // Loading state — skeleton loader
  if (isLoading) {
    return <ApplicationListSkeleton />;
  }

  // Empty state
  if (applications.length === 0) {
    return (
      <EmptyState
        icon={<Inbox className="h-12 w-12" />}
        title="No applications found"
        description="Applications will appear here once jobs are submitted."
      />
    );
  }

  return (
    <div className={cn("space-y-4", className)}>
      <ul className="space-y-3">
        {applications.map((app) => (
          <li key={app.id}>
            <ApplicationCard
              application={app}
              onStatusChange={onStatusChange}
              onClick={onSelect}
            />
          </li>
        ))}
      </ul>

      {hasMore && onLoadMore && (
        <div className="flex justify-center pt-2">
          <Button
            variant="secondary"
            onClick={onLoadMore}
            disabled={loadingMore}
            aria-label="Load more applications"
          >
            {loadingMore ? "Loading..." : "Load More"}
          </Button>
        </div>
      )}
    </div>
  );
}
