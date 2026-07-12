/**
 * TaskList — renders a list of tasks with loading, empty, and list states.
 *
 * Uses TaskCard for each item. Shows skeleton loader during initial load.
 * Used on the main tasks page and dashboard.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <TaskList tasks={tasks} isLoading={false} />
 */

"use client";

import { ClipboardList } from "lucide-react";
import type { TaskResponse } from "@/lib/types/tasks";
import { TaskCard } from "./TaskCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { TaskCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";

interface TaskListProps {
  /** Array of tasks to display. */
  tasks: TaskResponse[];
  /** Whether data is currently loading (shows skeleton). */
  isLoading?: boolean;
}

/** Skeleton placeholder matching the list layout. */
function TaskListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading tasks">
      <span className="sr-only" aria-live="polite">Loading tasks…</span>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <TaskCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function TaskList({ tasks, isLoading = false }: TaskListProps) {
  // Empty state
  if (tasks.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<ClipboardList className="h-12 w-12" />}
        title="No tasks yet"
        description="Tasks appear here when the system processes jobs, generates resumes, or fills applications."
      />
    );
  }

  // Render list with SkeletonWrapper
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<TaskListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Tasks loaded"
    >
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} />
        ))}
      </div>
    </SkeletonWrapper>
  );
}
