/**
 * TanStack Query hooks for tasks data.
 *
 * Provides useQuery hooks for fetching tasks with polling (5s interval),
 * plus useQuery for single task detail.
 *
 * Tasks are polled frequently because they represent background work
 * (job discovery, scoring, form filling) that users want to monitor in real-time.
 */

import { useQuery } from "@tanstack/react-query";
import { fetchTasks, fetchTask } from "@/lib/api/tasks";
import { POLL_INTERVAL } from "@/lib/constants";
import type { TaskListParams } from "@/lib/api/tasks";
import type { TaskListResponse } from "@/lib/types/tasks";

/** Stable stringify for query keys — sorts keys for consistent references. Assumes flat objects. */
function stableStringify(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

/** Query keys for tasks — consistent cache invalidation. */
export const tasksKeys = {
  all: ["tasks"] as const,
  lists: () => [...tasksKeys.all, "list"] as const,
  list: (params: TaskListParams) =>
    [...tasksKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  details: () => [...tasksKeys.all, "detail"] as const,
  detail: (id: string) => [...tasksKeys.details(), id] as const,
};

/** Empty task list response for graceful degradation. */
const emptyTasks: TaskListResponse = { tasks: [], total: 0 };

/**
 * Fetch tasks with polling for real-time updates.
 *
 * Polls every 5 seconds when there are active (pending/running) tasks.
 * Stops polling when all tasks are terminal (completed/failed/cancelled).
 *
 * Uses placeholderData for graceful degradation — shows empty state instead of error.
 *
 * @param params - Status, type, limit, offset filters
 */
export function useTasks(params: TaskListParams = {}) {
  return useQuery({
    queryKey: tasksKeys.list(params),
    queryFn: () => fetchTasks(params),
    placeholderData: emptyTasks,
    refetchInterval: (query) => {
      // Stop polling if query errored
      if (query.state.error != null) return false;
      const tasks = query.state.data?.tasks ?? [];
      // Poll if any task is still active
      const hasActive = tasks.some(
        (t) => t.status === "pending" || t.status === "running",
      );
      return hasActive ? POLL_INTERVAL.tasks : false;
    },
  });
}

/**
 * Fetch a single task with polling.
 *
 * Polls every 5 seconds while the task is pending or running.
 *
 * @param id - Task UUID
 */
export function useTask(id: string) {
  return useQuery({
    queryKey: tasksKeys.detail(id),
    queryFn: () => fetchTask(id),
    enabled: id.length > 0,
    refetchInterval: (query) => {
      if (query.state.error != null) return false;
      const task = query.state.data;
      if (task == null) return false;
      return task.status === "pending" || task.status === "running"
        ? POLL_INTERVAL.tasks
        : false;
    },
  });
}
