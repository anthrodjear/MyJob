/**
 * Dashboard Page — main dashboard view (Server Component).
 *
 * Fetches stats, activity, and tasks in parallel with graceful degradation,
 * then composes all dashboard widgets with the fetched data.
 *
 * Auth flow:
 *   1. proxy.ts validates session cookie, redirects to /login if missing
 *   2. dashboard.ts uses dalFetch which reads session cookie server-side
 *   3. AuthError caught here redirects to /login (safety net)
 */

import { redirect } from "next/navigation";
import { DashboardStats } from "@/components/dashboard/DashboardStats";
import { PipelineSummary } from "@/components/dashboard/PipelineSummary";
import { ActivityFeed } from "@/components/dashboard/ActivityFeed";
import { QuickActions } from "@/components/dashboard/QuickActions";
import { UpcomingTasks } from "@/components/dashboard/UpcomingTasks";
import {
  fetchDashboardStats,
  fetchRecentActivity,
  fetchPendingTasks,
} from "@/lib/api/dashboard";
import { AuthError } from "@/lib/dal";
import type { ApplicationStatsResponse } from "@/lib/types/applications";
import type { ActivityListResponse } from "@/lib/types/activity";
import type { TaskListResponse } from "@/lib/types/tasks";

/** Empty fallback for stats when fetch fails. */
const emptyStats: ApplicationStatsResponse = {
  total: 0,
  by_status: {},
  by_tier: {},
};

/** Empty fallback for activity when fetch fails. */
const emptyActivity: ActivityListResponse = {
  activities: [],
  total: 0,
  limit: 10,
  offset: 0,
};

/** Empty fallback for tasks when fetch fails. */
const emptyTasks: TaskListResponse = {
  tasks: [],
  total: 0,
};

/**
 * Safe fetch wrapper that returns fallback on error instead of throwing.
 * Allows partial dashboard rendering when some endpoints fail.
 */
async function safeFetch<T>(fn: () => Promise<T>, fallback: T): Promise<T> {
  try {
    return await fn();
  } catch (error) {
    // AuthError is a flow-control error — must propagate for redirect handling
    if (error instanceof AuthError) throw error;
    console.error("[DashboardPage] Fetch failed, using fallback:", error);
    return fallback;
  }
}

export default async function DashboardPage() {
  // Fetch all dashboard data in parallel with graceful degradation
  let stats: ApplicationStatsResponse;
  let activity: ActivityListResponse;
  let tasks: TaskListResponse;

  try {
    [stats, activity, tasks] = await Promise.all([
      safeFetch(() => fetchDashboardStats(), emptyStats),
      safeFetch(() => fetchRecentActivity(10), emptyActivity),
      safeFetch(() => fetchPendingTasks(5), emptyTasks),
    ]);
  } catch (error) {
    // AuthError = session cookie missing/invalid → redirect to login
    if (error instanceof AuthError) {
      redirect("/login");
    }
    // Other errors fall through to empty fallbacks
    console.error("[DashboardPage] Unexpected error:", error);
    stats = emptyStats;
    activity = emptyActivity;
    tasks = emptyTasks;
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Dashboard</h1>
          <p className="text-text-secondary mt-1">
            Overview of your job search pipeline
          </p>
        </div>
      </div>

      {/* KPI Stats */}
      <DashboardStats stats={stats} />

      {/* Pipeline funnel */}
      <PipelineSummary stats={stats} />

      {/* Main content grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left column: Activity + Tasks */}
        <div className="lg:col-span-2 space-y-6">
          <ActivityFeed activities={activity.activities} />
          <UpcomingTasks tasks={tasks.tasks} />
        </div>

        {/* Right column: Quick Actions */}
        <div className="lg:col-span-1">
          <QuickActions />
        </div>
      </div>
    </div>
  );
}