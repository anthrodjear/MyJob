/**
 * TanStack Query hooks for applications data.
 *
 * Provides useQuery hooks for fetching applications with caching, and
 * useMutation hooks for application actions (create, status transition, notes).
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchApplications,
  fetchApplicationStats,
  fetchApplication,
  createApplication,
  updateApplicationStatus,
  updateApplicationNotes,
  fetchApplicationTimeline,
} from "@/lib/api/applications";
import type {
  Application,
  ApplicationListParams,
  ApplicationStatus,
  ApplicationListResponse,
} from "@/lib/types/applications";

/** Stable stringify for query keys — sorts keys for consistent references. */
function stableStringify(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

/** Query keys for applications — consistent cache invalidation. */
export const applicationsKeys = {
  all: ["applications"] as const,
  lists: () => [...applicationsKeys.all, "list"] as const,
  list: (params: ApplicationListParams) =>
    [...applicationsKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  stats: () => [...applicationsKeys.all, "stats"] as const,
  details: () => [...applicationsKeys.all, "detail"] as const,
  detail: (id: string) => [...applicationsKeys.details(), id] as const,
  timeline: (id: string) => [...applicationsKeys.detail(id), "timeline"] as const,
};

/** Empty application list response for graceful degradation. */
const emptyApplications: ApplicationListResponse = { applications: [], total: 0, limit: 0, offset: 0 };

/**
 * Hook to fetch paginated application list with filters.
 *
 * @param params - Query parameters (status, min_score, page, limit)
 * @returns TanStack Query result with applications data
 */
export function useApplications(params: ApplicationListParams = {}) {
  return useQuery({
    queryKey: applicationsKeys.list(params),
    queryFn: () => fetchApplications(params),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    placeholderData: emptyApplications,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30_000),
    structuralSharing: true,
  });
}

/**
 * Hook to fetch application stats (by_status, by_tier).
 *
 * @returns TanStack Query result with stats data
 */
export function useApplicationStats() {
  return useQuery({
    queryKey: applicationsKeys.stats(),
    queryFn: () => fetchApplicationStats(),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    structuralSharing: true,
  });
}

/**
 * Hook to fetch a single application by ID.
 *
 * @param id - Application UUID
 * @returns TanStack Query result with application data
 */
export function useApplication(id: string) {
  return useQuery({
    queryKey: applicationsKeys.detail(id),
    queryFn: () => fetchApplication(id),
    enabled: !!id,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    structuralSharing: true,
  });
}

/**
 * Hook to fetch audit trail timeline for an application.
 *
 * @param id - Application UUID
 * @returns TanStack Query result with timeline events
 */
export function useApplicationTimeline(id: string) {
  return useQuery({
    queryKey: applicationsKeys.timeline(id),
    queryFn: () => fetchApplicationTimeline(id),
    enabled: !!id,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 2,
    structuralSharing: true,
  });
}

/**
 * Hook to create a new application.
 * Invalidates application list and stats caches on success.
 *
 * @returns TanStack Mutation result
 */
export function useCreateApplication() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: {
      job_id: string;
      resume_id?: string;
      cover_letter_id?: string;
      portal_type?: string;
      portal_url?: string;
    }) => createApplication(payload),
    onSuccess: () => {
      // Invalidate lists to show new application
      queryClient.invalidateQueries({ queryKey: applicationsKeys.lists() });
      queryClient.invalidateQueries({ queryKey: applicationsKeys.stats() });
    },
  });
}

/**
 * Hook to transition application status.
 * Optimistically updates the status in cache, rolls back on error.
 * Backend returns { message }, not the full Application — so we invalidate
 * to ensure the list and detail views refetch the authoritative state.
 *
 * @returns TanStack Mutation result
 */
export function useUpdateApplicationStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      status,
      notes,
    }: {
      id: string;
      status: ApplicationStatus;
      notes?: string;
    }) => updateApplicationStatus(id, status, notes),
    onMutate: async ({ id, status }) => {
      await queryClient.cancelQueries({ queryKey: applicationsKeys.detail(id) });

      const previousApp = queryClient.getQueryData<Application>(
        applicationsKeys.detail(id),
      );

      // Optimistically set status — onSettled will refetch authoritative data
      queryClient.setQueryData<Application>(applicationsKeys.detail(id), (old) =>
        old ? { ...old, status } : old,
      );

      return { previousApp };
    },
    onError: (_err, variables, context) => {
      // Roll back optimistic update on failure
      if (context?.previousApp) {
        queryClient.setQueryData(
          applicationsKeys.detail(variables.id),
          context.previousApp,
        );
      }
    },
    onSettled: (_data, _error, variables) => {
      // Always refetch to get authoritative state from server
      queryClient.invalidateQueries({
        queryKey: applicationsKeys.detail(variables.id),
      });
      queryClient.invalidateQueries({ queryKey: applicationsKeys.lists() });
      queryClient.invalidateQueries({ queryKey: applicationsKeys.stats() });
      queryClient.invalidateQueries({
        queryKey: applicationsKeys.timeline(variables.id),
      });
    },
  });
}

/**
 * Hook to update application notes.
 * Optimistically updates notes in cache, rolls back on error.
 * Backend returns { message }, not the full Application.
 *
 * @returns TanStack Mutation result
 */
export function useUpdateApplicationNotes() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) =>
      updateApplicationNotes(id, notes),
    onMutate: async ({ id, notes }) => {
      await queryClient.cancelQueries({ queryKey: applicationsKeys.detail(id) });

      const previousApp = queryClient.getQueryData<Application>(
        applicationsKeys.detail(id),
      );

      // Optimistically set notes — onSettled will refetch authoritative data
      queryClient.setQueryData<Application>(applicationsKeys.detail(id), (old) =>
        old ? { ...old, notes } : old,
      );

      return { previousApp };
    },
    onError: (_err, variables, context) => {
      // Roll back optimistic update on failure
      if (context?.previousApp) {
        queryClient.setQueryData(
          applicationsKeys.detail(variables.id),
          context.previousApp,
        );
      }
    },
    onSettled: (_data, _error, variables) => {
      // Always refetch to get authoritative state from server
      queryClient.invalidateQueries({
        queryKey: applicationsKeys.detail(variables.id),
      });
    },
  });
}
