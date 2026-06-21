/**
 * Dashboard API client — stats, activity, tasks endpoints.
 *
 * Provides typed fetchers for dashboard data used in Server Components.
 * All functions use the base apiFetch for consistent error handling.
 */

import { apiGet } from "@/lib/api/client";
import type { ApplicationStatsResponse } from "@/lib/types/applications";
import type { ActivityResponse, ActivityListResponse } from "@/lib/types/activity";
import type { TaskResponse, TaskListResponse } from "@/lib/types/tasks";

/**
 * Fetch application statistics for KPI cards and pipeline funnel.
 *
 * @returns Application stats including totals by status and tier
 * @throws ApiError if response is undefined or non-2xx
 */
export async function fetchDashboardStats(): Promise<ApplicationStatsResponse> {
  const result = await apiGet<ApplicationStatsResponse>("applications/stats");
  if (result === undefined) {
    throw new Error("Unexpected empty response shape response shape from applications/stats");
  }
  return result;
}

/**
 * Fetch recent activity logs for the activity feed.
 *
 * @param limit - Number of activities to fetch (default: 10)
 * @param offset - Pagination offset (default: 0)
 * @returns Paginated activity list
 * @throws ApiError if response is undefined or non-2xx
 */
export async function fetchRecentActivity(
  limit = 10,
  offset = 0,
): Promise<ActivityListResponse> {
  const params = new URLSearchParams();
  params.set("limit", String(limit));
  params.set("offset", String(offset));

  const result = await apiGet<ActivityListResponse>(`activity-logs?${params}`);
  if (result === undefined) {
    throw new Error("Unexpected empty response from activity-logs");
  }
  return result;
}

/**
 * Fetch pending tasks for the upcoming tasks widget.
 *
 * @param limit - Number of tasks to fetch (default: 5)
 * @param offset - Pagination offset (default: 0)
 * @param status - Task status filter (default: "pending")
 * @returns List of tasks matching the filter
 * @throws ApiError if response is undefined or non-2xx
 */
export async function fetchPendingTasks(
  limit = 5,
  offset = 0,
  status = "pending",
): Promise<TaskListResponse> {
  const params = new URLSearchParams();
  params.set("status", status);
  params.set("limit", String(limit));
  params.set("offset", String(offset));

  const result = await apiGet<TaskListResponse>(`tasks?${params}`);
  if (result === undefined) {
    throw new Error("Unexpected empty response from tasks");
  }
  return result;
}