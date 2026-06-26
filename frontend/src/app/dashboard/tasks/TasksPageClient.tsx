/**
 * TasksPageClient — tasks list with polling and status filter.
 *
 * Client Component (uses hooks for data fetching with real-time polling).
 *
 * Features:
 * - 5-second polling for active tasks
 * - Status filter (all, pending, running, completed, failed)
 * - URL-synced pagination
 *
 * @example
 *   <TasksPageClient />
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useTasks } from "@/hooks/useTasks";
import { TaskList } from "@/components/tasks/TaskList";
import { CardSkeleton } from "@/components/shared/LoadingSkeleton";
import { Pagination } from "@/components/shared/Pagination";
import { Button } from "@/components/shared/Button";
import type { TaskStatus } from "@/lib/types/tasks";

const PAGE_SIZE = 20;

/** Status filter options. */
const STATUS_FILTERS: Array<{ value: TaskStatus | "all"; label: string }> = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
];

export function TasksPageClient() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const statusFilter = (searchParams.get("status") as TaskStatus | "all") ?? "all";
  const offset = parseInt(searchParams.get("offset") ?? "0", 10);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const { data, isLoading, error } = useTasks({
    status: statusFilter === "all" ? undefined : statusFilter,
    limit: PAGE_SIZE,
    offset,
  });

  const tasks = data?.tasks ?? [];
  const total = data?.total ?? 0;

  const updateParams = (key: string, value: string) => {
    const params = new URLSearchParams(searchParams.toString());
    if (value === "all" || value === "0") {
      params.delete(key);
    } else {
      params.set(key, value);
    }
    // Reset offset when filter changes
    if (key === "status") params.delete("offset");
    router.push(`/dashboard/tasks?${params.toString()}`, { scroll: false });
  };

  if (error != null) {
    return (
      <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
        Failed to load tasks. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Tasks</h1>
        <p className="text-sm text-text-secondary">
          Monitor background jobs, scoring, and form submissions.
        </p>
      </div>

      {/* Status filter */}
      <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Filter tasks by status">
        {STATUS_FILTERS.map((filter) => (
          <Button
            key={filter.value}
            variant={statusFilter === filter.value ? "primary" : "secondary"}
            size="sm"
            onClick={() => updateParams("status", filter.value)}
            role="radio"
            aria-checked={statusFilter === filter.value}
          >
            {filter.label}
          </Button>
        ))}
      </div>

      {/* Task list */}
      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : (
        <>
          <TaskList tasks={tasks} />
          <Pagination
            page={currentPage}
            total={total}
            limit={PAGE_SIZE}
            onPageChange={(page) => {
              const newOffset = (page - 1) * PAGE_SIZE;
              updateParams("offset", String(newOffset));
            }}
          />
        </>
      )}
    </div>
  );
}
