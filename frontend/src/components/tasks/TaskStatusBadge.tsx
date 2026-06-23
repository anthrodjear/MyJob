/**
 * TaskStatusBadge — displays task status with color coding.
 *
 * Shows status text with appropriate background color.
 * Uses `role="status"` for dynamic status updates.
 *
 * @example
 *   <TaskStatusBadge status="running" />
 */

import { cn } from "@/lib/utils";
import type { TaskStatus } from "@/lib/types/tasks";

/** Status → display config mapping. Aligned with backend/internal/tasks/model.go. */
const STATUS_CONFIG: Record<
  TaskStatus,
  { label: string; color: string }
> = {
  pending: { label: "Pending", color: "bg-bg-tertiary text-text-secondary" },
  running: { label: "Running", color: "bg-info-light text-info-dark" },
  completed: { label: "Completed", color: "bg-success-light text-success-dark" },
  failed: { label: "Failed", color: "bg-danger-light text-danger-dark" },
  cancelled: { label: "Cancelled", color: "bg-bg-tertiary text-text-tertiary" },
};

interface TaskStatusBadgeProps {
  status: TaskStatus;
  className?: string;
}

export function TaskStatusBadge({ status, className }: TaskStatusBadgeProps) {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.pending;
  return (
    <span
      role="status"
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        config.color,
        className,
      )}
    >
      {config.label}
    </span>
  );
}
