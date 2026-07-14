/**
 * ApplicationList — paginated application list with loading, empty, and error states.
 *
 * Renders ApplicationCard components. Shows skeleton loader during initial load.
 * Used on the main applications page.
 * Uses SkeletonWrapper to enforce minimum/maximum display times and prevent pop-ins.
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
import { ApplicationCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";
import type { Application } from "@/lib/types/applications";

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

/** Skeleton placeholder matching the list layout. */
function ApplicationListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading applications">
      <span className="sr-only" aria-live="polite">Loading applications…</span>
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <ApplicationCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
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
  // Empty state
  if (applications.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<Inbox className="h-12 w-12" />}
        title="No applications found"
        description="Applications will appear here once jobs are submitted."
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<ApplicationListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Applications loaded"
    >
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
    </SkeletonWrapper>
  );
}
