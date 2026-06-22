/**
 * TanStack Query hooks for jobs data.
 *
 * Provides useQuery hooks for fetching jobs with caching, polling, and
 * optimistic updates, plus useMutation hooks for job actions.
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchJob,
  fetchJobs,
  applyToJob,
  scoreJob,
  saveJob,
  updateJobStatus,
  deleteJob,
  fetchSimilarJobs,
  fetchJobApplications,
} from "@/lib/api/jobs";
import type { Job, JobListParams, JobStatus } from "@/lib/types/jobs";

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
  similar: (id: string) => [...jobsKeys.detail(id), "similar"] as const,
  applications: (id: string) => [...jobsKeys.detail(id), "applications"] as const,
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
    gcTime: 5 * 60_000, // 5 minutes
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

/**
 * Hook to fetch similar jobs for a given job.
 *
 * @param jobId - Job UUID
 * @returns TanStack Query result with similar jobs
 */
export function useSimilarJobs(jobId: string) {
  return useQuery({
    queryKey: jobsKeys.similar(jobId),
    queryFn: () => fetchSimilarJobs(jobId),
    enabled: !!jobId,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    structuralSharing: true,
  });
}

/**
 * Hook to fetch job application history.
 *
 * @param jobId - Job UUID
 * @returns TanStack Query result with application history
 */
export function useJobApplications(jobId: string) {
  return useQuery({
    queryKey: jobsKeys.applications(jobId),
    queryFn: () => fetchJobApplications(jobId),
    enabled: !!jobId,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    structuralSharing: true,
  });
}

/**
 * Hook to apply to a job.
 * Invalidates job list and job detail caches on success.
 *
 * @returns TanStack Mutation result
 */
export function useApplyToJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ jobId }: { jobId: string }) => applyToJob(jobId),
    onSuccess: (data, variables) => {
      // Invalidate job lists to reflect application
      queryClient.invalidateQueries({ queryKey: jobsKeys.lists() });
      // Invalidate job detail to show application status
      queryClient.invalidateQueries({ queryKey: jobsKeys.detail(variables.jobId) });
      // Invalidate similar jobs if shown
      queryClient.invalidateQueries({ queryKey: jobsKeys.similar(variables.jobId) });
    },
  });
}

/**
 * Hook to score a job against user profile.
 * Invalidates job list and job detail caches on success.
 *
 * @returns TanStack Mutation result
 */
export function useScoreJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ jobId }: { jobId: string }) => scoreJob(jobId),
    onSuccess: (data, variables) => {
      // Invalidate job lists to show new scores
      queryClient.invalidateQueries({ queryKey: jobsKeys.lists() });
      // Invalidate job detail
      queryClient.invalidateQueries({ queryKey: jobsKeys.detail(variables.jobId) });
    },
  });
}

/**
 * Hook to save/unsave a job.
 * Optimistically updates the job in cache.
 *
 * @returns TanStack Mutation result
 */
export function useSaveJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ jobId, save }: { jobId: string; save: boolean }) => saveJob(jobId, save),
    onMutate: async ({ jobId, save }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: jobsKeys.detail(jobId) });

      // Snapshot previous value
      const previousJob = queryClient.getQueryData<Job>(jobsKeys.detail(jobId));

      queryClient.setQueryData<Job>(jobsKeys.detail(jobId), (old) =>
        old ? { ...old, match_details: { ...old.match_details, saved: save } } : old,
      );

      return { previousJob };
    },
    onError: (err, variables, context) => {
      // Rollback on error
      if (context?.previousJob) {
        queryClient.setQueryData(jobsKeys.detail(variables.jobId), context.previousJob);
      }
    },
    onSettled: (data, error, variables) => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: jobsKeys.detail(variables.jobId) });
    },
  });
}

/**
 * Hook to update job status.
 * Optimistically updates the job in cache.
 *
 * @returns TanStack Mutation result
 */
export function useUpdateJobStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ jobId, status }: { jobId: string; status: JobStatus }) => updateJobStatus(jobId, status),
    onMutate: async ({ jobId, status }) => {
      await queryClient.cancelQueries({ queryKey: jobsKeys.detail(jobId) });

      const previousJob = queryClient.getQueryData<Job>(jobsKeys.detail(jobId));

      queryClient.setQueryData<Job>(jobsKeys.detail(jobId), (old) =>
        old ? { ...old, status } : old,
      );

      return { previousJob };
    },
    onError: (err, variables, context) => {
      if (context?.previousJob) {
        queryClient.setQueryData(jobsKeys.detail(variables.jobId), context.previousJob);
      }
    },
    onSettled: (data, error, variables) => {
      queryClient.invalidateQueries({ queryKey: jobsKeys.detail(variables.jobId) });
      queryClient.invalidateQueries({ queryKey: jobsKeys.lists() });
    },
  });
}

/**
 * Hook to delete a job.
 *
 * @returns TanStack Mutation result
 */
export function useDeleteJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ jobId }: { jobId: string }) => deleteJob(jobId),
    onSuccess: (data, variables) => {
      // Remove from lists
      queryClient.invalidateQueries({ queryKey: jobsKeys.lists() });
      // Remove detail
      queryClient.removeQueries({ queryKey: jobsKeys.detail(variables.jobId) });
    },
  });
}