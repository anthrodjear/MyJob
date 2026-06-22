/**
 * ApprovalList — paginated approval list with loading and empty states.
 *
 * Renders ApprovalCard components. Shows skeleton loader during initial load.
 *
 * @example
 *   <ApprovalList approvals={approvals} isLoading={false} onApprove={handleApprove} />
 */

"use client";

import { ClipboardCheck } from "lucide-react";
import { cn } from "@/lib/utils";
import { ApprovalCard } from "./ApprovalCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { Button } from "@/components/shared/Button";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import type { Approval } from "@/lib/types/approvals";

/**
 * Loading skeleton — shows placeholder cards matching ApprovalCard layout.
 */
function ApprovalListSkeleton() {
  return (
    <div className="space-y-3" aria-busy="true" aria-label="Loading approvals">
      <span className="sr-only" aria-live="polite">Loading approvals…</span>
      {Array.from({ length: 4 }).map((_, i) => (
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
              <Skeleton className="h-5 w-12 rounded-full" />
              <Skeleton className="h-5 w-16 rounded-full" />
            </div>
          </div>
          <Skeleton className="mb-3 h-3 w-1/3" />
          <div className="flex gap-2">
            <Skeleton className="h-8 w-20" />
            <Skeleton className="h-8 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}

interface ApprovalListProps {
  /** Array of approvals to display. */
  approvals: Approval[];
  /** Whether data is currently loading (shows skeleton). */
  isLoading?: boolean;
  /** Callback when Approve is clicked on an approval. */
  onApprove?: (id: string) => void;
  /** Callback when Reject is clicked on an approval. */
  onReject?: (id: string) => void;
  /** Callback when an approval card is selected. */
  onSelect?: (id: string) => void;
  /** Whether mutations are pending. */
  isPending?: boolean;
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
 * ApprovalList — paginated approval listing.
 *
 * Accessibility:
 * - Uses `<ul>/<li>` for list semantics
 * - Loading state uses `aria-busy` to announce loading
 * - EmptyState has icon + description for context
 */
export function ApprovalList({
  approvals,
  isLoading = false,
  onApprove,
  onReject,
  onSelect,
  isPending,
  hasMore,
  onLoadMore,
  loadingMore,
  className,
}: ApprovalListProps) {
  // Loading state — skeleton loader
  if (isLoading) {
    return <ApprovalListSkeleton />;
  }

  // Empty state
  if (approvals.length === 0) {
    return (
      <EmptyState
        icon={<ClipboardCheck className="h-12 w-12" />}
        title="No approval requests"
        description="Approval requests will appear here when jobs score in the review tier."
      />
    );
  }

  return (
    <div className={cn("space-y-4", className)}>
      <ul className="space-y-3">
        {approvals.map((approval) => (
          <li key={approval.id}>
            <ApprovalCard
              approval={approval}
              onApprove={onApprove}
              onReject={onReject}
              onClick={onSelect}
              isPending={isPending}
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
            aria-label="Load more approvals"
          >
            {loadingMore ? "Loading..." : "Load More"}
          </Button>
        </div>
      )}
    </div>
  );
}
