/**
 * Dashboard API client — stats, activity, tasks endpoints.
 *
 * Provides typed fetchers for dashboard data used in Server Components.
 * All functions use dalFetch (Data Access Layer) which reads the session
 * cookie server-side and forwards the auth token to the Go backend.
 *
 * Why dalFetch instead of apiFetch:
 *   - Server Components can't use localStorage (browser API)
 *   - dalFetch reads the encrypted session cookie via next/headers cookies()
 *   - dalFetch forwards the access token as Authorization header
 *
 * @see lib/dal.ts — Data Access Layer (auth + fetch)
 * @see lib/api/client.ts — Client-side API client (localStorage-based)
 */

import { dalFetch } from "@/lib/dal";
import type { ApplicationStatsResponse } from "@/lib/types/applications";
import type { ActivityListResponse } from "@/lib/types/activity";
import type { TaskListResponse } from "@/lib/types/tasks";

/**
 * Fetch application statistics for KPI cards and pipeline funnel.
 *
 * @returns Application stats including totals by status and tier
 * @throws AuthError if session cookie is missing/invalid
 * @throws DalError if backend returns non-2xx
 */
export async function fetchDashboardStats(): Promise<ApplicationStatsResponse> {
  const result = await dalFetch<ApplicationStatsResponse>("/applications/stats");
  if (result === undefined) {
    throw new Error("Unexpected empty response from applications/stats");
  }
  return result;
}

/**
 * Fetch recent activity logs for the activity feed.
 *
 * @param limit - Number of activities to fetch (default: 10)
 * @param offset - Pagination offset (default: 0)
 * @returns Paginated activity list
 * @throws AuthError if session cookie is missing/invalid
 * @throws DalError if backend returns non-2xx
 */
export async function fetchRecentActivity(
  limit = 10,
  offset = 0,
): Promise<ActivityListResponse> {
  const params = new URLSearchParams();
  params.set("limit", String(limit));
  params.set("offset", String(offset));

  const result = await dalFetch<ActivityListResponse>(`/activity-logs?${params}`);
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
 * @throws AuthError if session cookie is missing/invalid
 * @throws DalError if backend returns non-2xx
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

  const result = await dalFetch<TaskListResponse>(`/tasks?${params}`);
  if (result === undefined) {
    throw new Error("Unexpected empty response from tasks");
  }
  return result;
}
