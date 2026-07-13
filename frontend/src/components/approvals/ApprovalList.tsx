/**
 * ApprovalList — paginated approval list with loading and empty states.
 *
 * Renders ApprovalCard components. Shows skeleton loader during initial load.
 * Uses SkeletonWrapper to enforce minimum/maximum display times and prevent pop-ins.
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
import { ApprovalCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";
import type { Approval } from "@/lib/types/approvals";

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

/** Skeleton placeholder matching the list layout. */
function ApprovalListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading approvals">
      <span className="sr-only" aria-live="polite">Loading approvals…</span>
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <ApprovalCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
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
  // Empty state
  if (approvals.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<ClipboardCheck className="h-12 w-12" />}
        title="No approval requests"
        description="Approval requests will appear here when jobs score in the review tier."
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<ApprovalListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Approvals loaded"
    >
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
    </SkeletonWrapper>
  );
}
