/**
 * StatusBadge — visual indicator for interview session status.
 *
 * Renders a color-coded pill with accessible label.
 * Server Component — no "use client" needed.
 *
 * @example
 *   <StatusBadge status="active" />
 */

import { cn } from "@/lib/utils";
import type { InterviewStatus } from "@/lib/types/interviews";

const STATUS_STYLES: Record<InterviewStatus, string> = {
  pending: "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300",
  starting: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  completed: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
  failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
  cancelled: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
};

const STATUS_LABELS: Record<InterviewStatus, string> = {
  pending: "Pending",
  starting: "Starting",
  active: "Active",
  completed: "Completed",
  failed: "Failed",
  cancelled: "Cancelled",
};

interface StatusBadgeProps {
  /** Current interview status to display. */
  status: InterviewStatus;
  /** Additional CSS classes. */
  className?: string;
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        STATUS_STYLES[status],
        className
      )}
      role="status"
      aria-label={`Status: ${STATUS_LABELS[status]}`}
    >
      {STATUS_LABELS[status]}
    </span>
  );
}
