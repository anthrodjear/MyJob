/**
 * TaskCard — displays a single task with status, type, and timing info.
 *
 * Shows task type badge, status badge, attempts, and created/completed timestamps.
 * Active tasks (pending/running) show a pulse animation.
 *
 * @example
 *   <TaskCard task={task} />
 */

import { cn } from "@/lib/utils";
import type { TaskResponse, TaskStatus } from "@/lib/types/tasks";
import { TaskStatusBadge } from "./TaskStatusBadge";
import { TaskTypeBadge } from "./TaskTypeBadge";
import { Clock, CheckCircle, XCircle, Loader2 } from "lucide-react";

interface TaskCardProps {
  task: TaskResponse;
  className?: string;
}

/**
 * Format duration between two timestamps.
 * Returns human-readable string like "12s", "2m 30s", "1h 5m".
 */
function formatDuration(start: string, end?: string | null): string {
  const startTime = new Date(start).getTime();
  const endTime = end != null ? new Date(end).getTime() : Date.now();
  const diffMs = endTime - startTime;
  if (diffMs < 0) return "—";

  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

/**
 * Status icon — decorative, shown next to status badge.
 */
function StatusIcon({ status }: { status: TaskStatus }) {
  const iconClass = "h-4 w-4";
  switch (status) {
    case "pending":
      return <Clock className={cn(iconClass, "text-text-tertiary")} aria-hidden="true" />;
    case "running":
      return <Loader2 className={cn(iconClass, "text-info animate-spin")} aria-hidden="true" />;
    case "completed":
      return <CheckCircle className={cn(iconClass, "text-success")} aria-hidden="true" />;
    case "failed":
      return <XCircle className={cn(iconClass, "text-danger")} aria-hidden="true" />;
    case "cancelled":
      return <Clock className={cn(iconClass, "text-text-tertiary")} aria-hidden="true" />;
  }
}

export function TaskCard({ task, className }: TaskCardProps) {
  const isActive = task.status === "pending" || task.status === "running";
  const duration =
    task.started_at != null
      ? formatDuration(task.started_at, task.completed_at)
      : null;

  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-bg-secondary p-4 transition-colors",
        isActive && "border-info/30",
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <StatusIcon status={task.status} />
          <TaskTypeBadge type={task.type} />
        </div>
        <TaskStatusBadge status={task.status} />
      </div>

      {/* Error message */}
      {task.error != null && task.error.length > 0 && (
        <p className="mt-2 text-xs text-danger-dark line-clamp-2" role="alert">
          {task.error}
        </p>
      )}

      {/* Metadata */}
      <div className="mt-3 flex items-center justify-between text-xs text-text-tertiary">
        <div className="flex items-center gap-3">
          <span>
            Attempt {task.attempts}/{task.max_attempts}
          </span>
          {task.priority > 0 && <span>Priority {task.priority}</span>}
        </div>
        <div className="flex items-center gap-3">
          {duration != null && <span>{duration}</span>}
          <span>{new Date(task.created_at).toLocaleTimeString()}</span>
        </div>
      </div>

      {/* Active task pulse */}
      {isActive && (
        <div className="mt-2 h-1 overflow-hidden rounded-full bg-bg-tertiary">
          <div className="h-full w-full animate-pulse rounded-full bg-info/50" />
        </div>
      )}
    </div>
  );
}
