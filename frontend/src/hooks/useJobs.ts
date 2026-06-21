/**
 * TanStack Query hooks for jobs data.
 *
 * Provides useQuery hooks for fetching jobs with caching, polling, and
 * optimistic updates. Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useQuery } from "@tanstack/react-query";
import { fetchJob, fetchJobs } from "@/lib/api/jobs";
import type { Job, JobListParams } from "@/lib/types/jobs";

/** Stable stringify for query keys — sorts keys for consistent references. */
function stableStringify(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

/** Query keys for jobs — consistent cache invalidation. */
export const jobsKeys = {
  all: ["jobs"] as const,
  lists: () => [...jobsKeys.all, "list"] as const,
  list: (params: JobListParams) => [...jobsKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  details: () => [...jobsKeys.all, "detail"] as const,
  detail: (id: string) => [...jobsKeys.details(), id] as const,
};

/**
 * Hook to fetch paginated job list with filters.
 *
 * @param params - Query parameters (search, source, status, min_score, page, limit)
 * @returns TanStack Query result with jobs data
 *
 * @example
 *   const { data, isLoading } = useJobs({ search: "react", page: 1 });
 */
export function useJobs(params: JobListParams = {}) {
  return useQuery({
    queryKey: jobsKeys.list(params),
    queryFn: () => fetchJobs(params),
    staleTime: 60_000, // 1 minute
    gcTime: 5 * 60_000000, // 5 minutes
    placeholderData: (previous) => previous,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30_000),
    structuralSharing: true,
  });
}

/**
 * Hook to fetch a single job by ID.
 *
 * @param id - Job UUID
 * @returns TanStack Query result with job data
 */
export function useJob(id: string) {
  return useQuery({
    queryKey: jobsKeys.detail(id),
    queryFn: () => fetchJob(id),
    enabled: !!id,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30_000),
    structuralSharing: true,
  });
}