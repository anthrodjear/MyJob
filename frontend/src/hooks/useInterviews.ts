/**
 * TanStack Query hooks for Interviews domain.
 *
 * Provides list/detail queries and mutations for interview operations
 * including start/stop lifecycle management.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchInterviews,
  fetchInterview,
  createInterview,
  startInterview,
  stopInterview,
} from "@/lib/api/interviews";
import type { InterviewFilterInput, CreateInterviewInput } from "@/lib/schemas/interviews";

/**
 * Fetch paginated interview list with optional filters.
 *
 * @param params - Filter and pagination parameters
 */
export function useInterviews(params?: InterviewFilterInput) {
  return useQuery({
    queryKey: ["interviews", params],
    queryFn: ({ signal }) => fetchInterviews(params, signal),
  });
}

/**
 * Fetch a single interview session by ID.
 *
 * @param id - Interview session UUID (enabled only when non-empty)
 */
export function useInterview(id: string) {
  return useQuery({
    queryKey: ["interviews", id],
    queryFn: ({ signal }) => fetchInterview(id, signal),
    enabled: !!id,
  });
}

/**
 * Create a new interview session with cache invalidation.
 */
export function useCreateInterview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateInterviewInput) => createInterview(data),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
    },
  });
}

/**
 * Start an interview session with optimistic status update.
 */
export function useStartInterview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data?: { provider?: string; model?: string };
    }) => startInterview(id, data),
    onMutate: async ({ id }) => {
      await queryClient.cancelQueries({ queryKey: ["interviews", id] });
      const previous = queryClient.getQueryData(["interviews", id]);
      queryClient.setQueryData(["interviews", id], (old: Record<string, unknown> | undefined) =>
        old ? { ...old, status: "starting" } : old
      );
      return { previous };
    },
    onError: (_err, { id }, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["interviews", id], context.previous);
      }
    },
    onSettled: (_data, _error, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
      void queryClient.invalidateQueries({ queryKey: ["interviews", id] });
    },
  });
}

/**
 * Stop an interview session with optimistic status update.
 */
export function useStopInterview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data?: { reason?: string };
    }) => stopInterview(id, data),
    onMutate: async ({ id }) => {
      await queryClient.cancelQueries({ queryKey: ["interviews", id] });
      const previous = queryClient.getQueryData(["interviews", id]);
      queryClient.setQueryData(["interviews", id], (old: Record<string, unknown> | undefined) =>
        old ? { ...old, status: "cancelled" } : old
      );
      return { previous };
    },
    onError: (_err, { id }, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["interviews", id], context.previous);
      }
    },
    onSettled: (_data, _error, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
      void queryClient.invalidateQueries({ queryKey: ["interviews", id] });
    },
  });
}
