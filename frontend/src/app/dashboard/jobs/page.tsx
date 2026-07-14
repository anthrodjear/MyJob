/**
 * Jobs Page — job listings with search, filter, and sort.
 *
 * Client Component that manages filter state and fetches jobs
 * using the useJobs hook. Renders JobFilters for controls and
 * JobList for the results grid.
 *
 * URL search params are read on mount to initialize filters.
 * Pagination uses offset-based computation (page → offset).
 *
 * @example
 *   /dashboard/jobs?status=matched&offset=20
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
  offset: 0,
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

  /** Derive page number from offset for display. */
  const getPageFromOffset = (offset: number) => Math.floor(offset / DEFAULT_PAGE_SIZE) + 1;
  const getOffsetFromPage = (page: number) => (page - 1) * DEFAULT_PAGE_SIZE;

  /** Initialize filters from URL search params. */
  const [filters, setFilters] = useState<JobListParams>(() => {
    const initial: JobListParams = { ...DEFAULT_FILTERS };
    const status = searchParams.get("status");
    const sourceId = searchParams.get("source_id");
    const minScore = searchParams.get("min_score");
    const offset = searchParams.get("offset");
    const limit = searchParams.get("limit");

    if (status) initial.status = status;
    if (sourceId) initial.source_id = sourceId;
    if (minScore) {
      const parsed = Number(minScore);
      if (!Number.isNaN(parsed)) initial.min_score = parsed;
    }
    if (offset) {
      const parsed = Number(offset);
      if (!Number.isNaN(parsed) && parsed >= 0) initial.offset = parsed;
    }
    if (limit) {
      const parsed = Number(limit);
      if (!Number.isNaN(parsed) && parsed > 0) initial.limit = parsed;
    }

    return initial;
  });

  /** Current page for display (derived from offset). */
  const currentPage = getPageFromOffset(filters.offset ?? 0);

  /** Sync URL when filters change. */
  useEffect(() => {
    const params = new URLSearchParams();
    if (filters.status) params.set("status", filters.status);
    if (filters.source_id) params.set("source_id", filters.source_id);
    if (filters.min_score) params.set("min_score", String(filters.min_score));
    if (filters.offset && filters.offset > 0) params.set("offset", String(filters.offset));
    if (filters.limit && filters.limit !== DEFAULT_PAGE_SIZE) params.set("limit", String(filters.limit));

    const qs = params.toString();
    const url = qs ? `/dashboard/jobs?${qs}` : "/dashboard/jobs";
    router.replace(url, { scroll: false });
  }, [filters, router]);

  /** Fetch jobs with current filters. */
  const { data, isLoading, isPlaceholderData } = useJobs(filters);

  /** Mutations for job actions. */
  const applyMutation = useApplyToJob();
  const scoreMutation = useScoreJob();
  const saveMutation = useSaveJob();
  const statusMutation = useUpdateJobStatus();

  /** Handle filter changes from JobFilters — always reset to offset 0. */
  const handleFilterChange = useCallback((newFilters: JobListParams) => {
    setFilters({ ...newFilters, offset: 0 });
  }, []);

  /** Handle page changes — convert page number to offset. */
  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, offset: getOffsetFromPage(page) }));
  }, []);

  const totalPages = data ? Math.ceil(data.total / DEFAULT_PAGE_SIZE) : 0;

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
            {isLoading && !isPlaceholderData ? "Loading jobs..." : `${data?.total ?? 0} jobs found`}
          </p>
        </div>
      </div>

      {/* Filters */}
      <JobFilters filters={filters} onFilterChange={handleFilterChange} />

      {/* Results */}
      <JobList
        jobs={data?.jobs ?? []}
        isLoading={isLoading && !isPlaceholderData}
        onApply={(jobId) => applyMutation.mutate({ jobId })}
        onScore={(jobId) => scoreMutation.mutate({ jobId })}
        onSave={(jobId, saved) => saveMutation.mutate({ jobId, save: saved })}
        onArchive={(jobId) => statusMutation.mutate({ jobId, status: "archived" })}
      />

      {/* Pagination */}
      {data && totalPages > 1 && (
        <nav aria-label="Job list pagination" className="flex justify-center gap-2">
          <button
            type="button"
            onClick={() => handlePageChange(currentPage - 1)}
            disabled={currentPage <= 1}
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
            Page {currentPage} of {totalPages}
          </span>
          <button
            type="button"
            onClick={() => handlePageChange(currentPage + 1)}
            disabled={currentPage >= totalPages}
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
