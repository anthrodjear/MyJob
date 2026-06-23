/**
 * Tasks API — aligned with backend/internal/tasks/handler.go.
 *
 * Backend endpoints:
 * - GET /tasks?status=&type=&limit=&offset= → TaskListResponse
 * - GET /tasks/:id → TaskResponse
 * - POST /tasks → TaskResponse
 *
 * @example
 *   import { fetchTasks } from "@/lib/api/tasks";
 *   const { tasks, total } = await fetchTasks({ status: "running" });
 */

import { apiGet, apiPost } from "./client";
import type {
  TaskResponse,
  TaskListResponse,
  TaskStatus,
  TaskType,
} from "@/lib/types/tasks";

/** List params for tasks. */
export interface TaskListParams {
  status?: TaskStatus;
  type?: TaskType;
  limit?: number;
  offset?: number;
}

/**
 * Fetch tasks with optional filters and pagination.
 *
 * @param params - Status, type, limit, offset filters
 * @returns Paginated task list
 * @throws ApiError on server error
 */
export async function fetchTasks(
  params: TaskListParams = {},
): Promise<TaskListResponse> {
  const searchParams = new URLSearchParams();
  if (params.status != null) searchParams.set("status", params.status);
  if (params.type != null) searchParams.set("type", params.type);
  if (params.limit != null) searchParams.set("limit", String(params.limit));
  if (params.offset != null) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  const path = qs.length > 0 ? `tasks?${qs}` : "tasks";
  const resp = await apiGet<TaskListResponse>(path);
  return resp ?? { tasks: [], total: 0 };
}

/**
 * Fetch a single task by ID.
 *
 * @param id - Task UUID
 * @returns Task detail
 * @throws ApiError on 404 or server error
 */
export async function fetchTask(id: string): Promise<TaskResponse> {
  const resp = await apiGet<TaskResponse>(`tasks/${id}`);
  if (resp == null) {
    throw new Error("Task not found");
  }
  return resp;
}

/**
 * Create a new task (for manual task dispatch).
 *
 * @param type - Task type
 * @param params - Task-specific parameters
 * @param priority - Task priority (default: 0)
 * @returns Created task
 * @throws ApiError on invalid type or server error
 */
export async function createTask(
  type: TaskType,
  params?: Record<string, unknown>,
  priority: number = 0,
): Promise<TaskResponse> {
  const resp = await apiPost<TaskResponse>("tasks", {
    type,
    params: params ?? {},
    priority,
  });
  if (resp == null) {
    throw new Error("Failed to create task");
  }
  return resp;
}
