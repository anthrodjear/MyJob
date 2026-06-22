/**
 * Jobs Page — job listings with search, filter, and sort.
 *
 * Client Component that manages filter state and fetches jobs
 * using the useJobs hook. Renders JobFilters for controls and
 * JobList for the results grid.
 *
 * URL search params are read on mount to initialize filters.
 *
 * @example
 *   /dashboard/jobs?search=react&status=matched
 */

"use client";

import { useState, useCallback, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { JobFilters } from "@/components/jobs/JobFilters";
import { JobList } from "@/components/jobs/JobList";
import { useJobs, useApplyToJob, useScoreJob, useSaveJob, useUpdateJobStatus } from "@/hooks/useJobs";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { cn } from "@/lib/utils";
import type { JobListParams } from "@/lib/types/jobs";

/** Default filter values. */
const DEFAULT_FILTERS: JobListParams = {
  page: 1,
  limit: DEFAULT_PAGE_SIZE,
};

/**
 * JobsPage — job listings with search, filter, and sort.
 *
 * Accessibility:
 * - Page has descriptive heading
 * - Filters are labeled and grouped
 * - Results announced to screen readers
 * - Loading states use aria-busy
 */
export default function JobsPage() {
  const router = useRouter();
  const searchParams = useSearchParams();

  /** Initialize filters from URL search params. */
  const [filters, setFilters] = useState<JobListParams>(() => {
    const initial: JobListParams = { ...DEFAULT_FILTERS };
    const search = searchParams.get("search");
    const source = searchParams.get("source");
    const status = searchParams.get("status");
    const minScore = searchParams.get("min_score");
    const page = searchParams.get("page");
    const sortBy = searchParams.get("sort_by");
    const sortDir = searchParams.get("sort_dir");

    if (search) initial.search = search;
    if (source) initial.source = source;
    if (status) initial.status = status;
    if (minScore) {
      const parsed = Number(minScore);
      if (!Number.isNaN(parsed)) initial.min_score = parsed;
    }
    if (page) {
      const parsed = Number(page);
      if (!Number.isNaN(parsed) && parsed > 0) initial.page = parsed;
    }
    if (sortBy) initial.sort_by = sortBy;
    if (sortDir === "asc" || sortDir === "desc") initial.sort_dir = sortDir;

    return initial;
  });

  /** Sync URL when filters change. */
  useEffect(() => {
    const params = new URLSearchParams();
    if (filters.search) params.set("search", filters.search);
    if (filters.source) params.set("source", filters.source);
    if (filters.status) params.set("status", filters.status);
    if (filters.min_score) params.set("min_score", String(filters.min_score));
    if (filters.page && filters.page > 1) params.set("page", String(filters.page));
    if (filters.sort_by) params.set("sort_by", filters.sort_by);
    if (filters.sort_dir) params.set("sort_dir", filters.sort_dir);

    const qs = params.toString();
    const url = qs ? `/dashboard/jobs?${qs}` : "/dashboard/jobs";
    router.replace(url, { scroll: false });
  }, [filters, router]);

  /** Fetch jobs with current filters. */
  const { data, isLoading, error } = useJobs(filters);

  /** Mutations for job actions. */
  const applyMutation = useApplyToJob();
  const scoreMutation = useScoreJob();
  const saveMutation = useSaveJob();
  const statusMutation = useUpdateJobStatus();

  /** Handle filter changes from JobFilters — always reset to page 1. */
  const handleFilterChange = useCallback((newFilters: JobListParams) => {
    setFilters({ ...newFilters, page: 1 });
  }, []);

  /** Handle page changes from JobList pagination. */
  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }));
  }, []);

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Jobs</h1>
          <p
            className="mt-1 text-text-secondary"
            aria-live="polite"
            aria-atomic="true"
          >
            {isLoading ? "Loading jobs..." : `${data?.total ?? 0} jobs found`}
          </p>
        </div>
      </div>

      {/* Error state */}
      {error && !isLoading && (
        <div role="alert" className="rounded-md border border-danger-light bg-danger-light/10 p-4 text-danger-dark">
          <p className="font-medium">Failed to load jobs</p>
          <p className="text-sm">{error.message}</p>
        </div>
      )}

      {/* Filters */}
      <JobFilters filters={filters} onFilterChange={handleFilterChange} />

      {/* Results */}
      <JobList
        jobs={data?.items ?? []}
        isLoading={isLoading}
        error={error?.message}
        onApply={(jobId) => applyMutation.mutate({ jobId })}
        onScore={(jobId) => scoreMutation.mutate({ jobId })}
        onSave={(jobId, saved) => saveMutation.mutate({ jobId, save: saved })}
        onArchive={(jobId) => statusMutation.mutate({ jobId, status: "archived" })}
      />

      {/* Pagination */}
      {data && data.total > DEFAULT_PAGE_SIZE && (
        <nav aria-label="Job list pagination" className="flex justify-center gap-2">
          <button
            type="button"
            onClick={() => handlePageChange((filters.page ?? 1) - 1)}
            disabled={(filters.page ?? 1) <= 1}
            className={cn(
              "rounded-md border border-border bg-bg-secondary px-3 py-1.5",
              "text-sm font-medium text-text-primary transition-colors",
              "hover:bg-bg-tertiary focus-visible:outline-none focus-visible:ring-2",
              "focus-visible:ring-primary disabled:opacity-50",
            )}
            aria-label="Previous page"
          >
            Previous
          </button>
          <span className="flex items-center px-3 text-sm text-text-secondary">
            Page {filters.page ?? 1} of {Math.ceil(data.total / DEFAULT_PAGE_SIZE)}
          </span>
          <button
            type="button"
            onClick={() => handlePageChange((filters.page ?? 1) + 1)}
            disabled={(filters.page ?? 1) >= Math.ceil(data.total / DEFAULT_PAGE_SIZE)}
            className={cn(
              "rounded-md border border-border bg-bg-secondary px-3 py-1.5",
              "text-sm font-medium text-text-primary transition-colors",
              "hover:bg-bg-tertiary focus-visible:outline-none focus-visible:ring-2",
              "focus-visible:ring-primary disabled:opacity-50",
            )}
            aria-label="Next page"
          >
            Next
          </button>
        </nav>
      )}
    </div>
  );
}
