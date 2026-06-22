/**
 * StatusBadge — application status display with semantic colors.
 *
 * Server Component. Static label (no role="status").
 * Colors map to pipeline stages: draft (default), queued (info),
 * applied (success), assessment/phone_screen/technical (warning),
 * final (info), offer (success), rejected (danger).
 *
 * @example
 *   <StatusBadge status="applied" />
 */

import { cn } from "@/lib/utils";
import type { ApplicationStatus } from "@/lib/types/applications";

/** Status → badge color mapping. */
const STATUS_STYLES: Record<ApplicationStatus, string> = {
  draft: "bg-bg-tertiary text-text-secondary",
  queued: "bg-info-light text-info-dark",
  applied: "bg-success-light text-success-dark",
  assessment: "bg-warning-light text-warning-dark",
  phone_screen: "bg-warning-light text-warning-dark",
  technical: "bg-warning-light text-warning-dark",
  final: "bg-info-light text-info-dark",
  offer: "bg-success-light text-success-dark",
  rejected: "bg-danger-light text-danger-dark",
};

/** Status → human-readable label. */
const STATUS_LABELS: Record<ApplicationStatus, string> = {
  draft: "Draft",
  queued: "Queued",
  applied: "Applied",
  assessment: "Assessment",
  phone_screen: "Phone Screen",
  technical: "Technical",
  final: "Final Round",
  offer: "Offer",
  rejected: "Rejected",
};

interface StatusBadgeProps {
  /** Application status value. */
  status: ApplicationStatus;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * StatusBadge — application status display.
 *
 * Accessibility:
 * - Uses `aria-label` for screen reader announcement
 * - No `role="status"` (static label, not live update)
 */
export function StatusBadge({ status, className }: StatusBadgeProps) {
  const colorClass = STATUS_STYLES[status] ?? STATUS_STYLES.draft;
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
