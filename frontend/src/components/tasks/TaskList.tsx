/**
 * TaskList — renders a list of tasks with empty state.
 *
 * Uses TaskCard for each item. Shows EmptyState when no tasks exist.
 *
 * @example
 *   <TaskList tasks={tasks} />
 */

import { ClipboardList } from "lucide-react";
import type { TaskResponse } from "@/lib/types/tasks";
import { TaskCard } from "./TaskCard";
import { EmptyState } from "@/components/shared/EmptyState";

interface TaskListProps {
  tasks: TaskResponse[];
}

export function TaskList({ tasks }: TaskListProps) {
  if (tasks.length === 0) {
    return (
      <EmptyState
        icon={<ClipboardList className="h-12 w-12" />}
        title="No tasks yet"
        description="Tasks appear here when the system processes jobs, generates resumes, or fills applications."
      />
    );
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {tasks.map((task) => (
        <TaskCard key={task.id} task={task} />
      ))}
    </div>
  );
}
