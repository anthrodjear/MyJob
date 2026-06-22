/**
 * ApprovalCard — individual approval request listing display.
 *
 * Shows job snapshot summary, status, and action buttons.
 * Provides approve/reject actions for pending requests.
 *
 * @example
 *   <ApprovalCard approval={approval} onApprove={handleApprove} onReject={handleReject} />
 */

"use client";

import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import { Button } from "@/components/shared/Button";
import type { Approval } from "@/lib/types/approvals";

interface ApprovalCardProps {
  /** Approval data to display. */
  approval: Approval;
  /** Callback when Approve is clicked. */
  onApprove?: (id: string) => void;
  /** Callback when Reject is clicked. */
  onReject?: (id: string) => void;
  /** Callback when the card is clicked (navigate to detail). */
  onClick?: (id: string) => void;
  /** Whether mutations are pending. */
  isPending?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApprovalCard — approval request listing display.
 *
 * Accessibility:
 * - `role="group"` with `aria-label` identifies the card
 * - Action buttons have descriptive `aria-label` attributes
 * - Score uses `font-mono tabular-nums` for alignment
 */
export function ApprovalCard({
  approval,
  onApprove,
  onReject,
  onClick,
  isPending,
  className,
}: ApprovalCardProps) {
  const { job_snapshot: snapshot } = approval;
  const scorePercent = Math.round(snapshot.score);

  return (
    <div
      className={cn(
        "group rounded-lg border border-border bg-bg-secondary p-4 transition-colors hover:border-border-hover",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2",
        onClick && "cursor-pointer",
        className,
      )}
      role={onClick ? "button" : "group"}
      aria-label={`Approval request for ${snapshot.title} at ${snapshot.company}`}
      onClick={onClick ? () => onClick(approval.id) : undefined}
      onKeyDown={onClick ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onClick(approval.id); } } : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Header: Title + Company + Score */}
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-text-primary">
            {snapshot.title}
          </h3>
          <p className="truncate text-sm text-text-secondary">
            {snapshot.company}
            {snapshot.location && ` · ${snapshot.location}`}
          </p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <span
            className={cn(
              "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
              "font-mono tabular-nums",
              scorePercent >= 80 && "bg-success-light text-success-dark",
              scorePercent >= 50 && scorePercent < 80 && "bg-warning-light text-warning-dark",
              scorePercent < 50 && "bg-danger-light text-danger-dark",
            )}
          >
            {scorePercent}%
          </span>
          <StatusBadge status={approval.status} />
        </div>
      </div>

      {/* Date */}
      <p className="mb-3 text-xs text-text-tertiary">
        Created {formatDate(approval.created_at)}
        {approval.reviewed_at && ` · Reviewed ${formatDate(approval.reviewed_at)}`}
      </p>

      {/* Rejection reason preview */}
      {approval.rejection_reason && (
        <p className="mb-3 line-clamp-2 rounded-md bg-danger-light/10 p-2 text-xs text-danger-dark">
          {approval.rejection_reason}
        </p>
      )}

      {/* Actions */}
      {approval.status === "pending" && onApprove && onReject && (
        <div className="flex gap-2">
          <Button
            variant="primary"
            size="sm"
            disabled={isPending}
            onClick={(e) => {
              e.stopPropagation();
              onApprove(approval.id);
            }}
            aria-label={`Approve ${snapshot.title}`}
          >
            Approve
          </Button>
          <Button
            variant="danger"
            size="sm"
            disabled={isPending}
            onClick={(e) => {
              e.stopPropagation();
              onReject(approval.id);
            }}
            aria-label={`Reject ${snapshot.title}`}
          >
            Reject
          </Button>
        </div>
      )}
    </div>
  );
}
