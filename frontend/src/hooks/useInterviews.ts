"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchInterviews,
  fetchInterview,
  createInterview,
  startInterview,
  stopInterview,
} from "../lib/api/interviews";
import type { InterviewFilterInput, CreateInterviewInput } from "../lib/schemas/interviews";

export function useInterviews(params?: InterviewFilterInput) {
  return useQuery({
    queryKey: ["interviews", params],
    queryFn: ({ signal }) => fetchInterviews(params, signal),
  });
}

export function useInterview(id: string) {
  return useQuery({
    queryKey: ["interviews", id],
    queryFn: ({ signal }) => fetchInterview(id, signal),
    enabled: !!id,
  });
}

export function useCreateInterview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateInterviewInput) => createInterview(data),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
    },
  });
}

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
    onSettled: (_data, _error, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
      void queryClient.invalidateQueries({ queryKey: ["interviews", id] });
    },
  });
}

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
    onSettled: (_data, _error, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["interviews"] });
      void queryClient.invalidateQueries({ queryKey: ["interviews", id] });
    },
  });
}
