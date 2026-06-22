/**
 * StatusBadge — approval status display with semantic colors.
 *
 * Server Component. Static label (no role="status").
 *
 * @example
 *   <StatusBadge status="pending" />
 */

import { cn } from "@/lib/utils";
import type { ApprovalStatus } from "@/lib/types/approvals";

/** Status → badge color mapping. */
const STATUS_STYLES: Record<ApprovalStatus, string> = {
  pending: "bg-warning-light text-warning-dark",
  approved: "bg-success-light text-success-dark",
  rejected: "bg-danger-light text-danger-dark",
};

/** Status → human-readable label. */
const STATUS_LABELS: Record<ApprovalStatus, string> = {
  pending: "Pending",
  approved: "Approved",
  rejected: "Rejected",
};

interface StatusBadgeProps {
  /** Approval status value. */
  status: ApprovalStatus;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * StatusBadge — approval status display.
 *
 * Accessibility:
 * - Uses `aria-label` for screen reader announcement
 */
export function StatusBadge({ status, className }: StatusBadgeProps) {
  const colorClass = STATUS_STYLES[status] ?? STATUS_STYLES.pending;
  const label = STATUS_LABELS[status] ?? status;

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        colorClass,
        className,
      )}
      aria-label={`Status: ${label}`}
    >
      {label}
    </span>
  );
}
