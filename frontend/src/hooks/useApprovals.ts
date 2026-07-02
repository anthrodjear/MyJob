/**
 * TanStack Query hooks for approvals data.
 *
 * Provides useQuery hooks for fetching approvals with caching, and
 * useMutation hooks for approve/reject actions.
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchApprovals,
  fetchApproval,
  approveApproval,
  rejectApproval,
} from "@/lib/api/approvals";
import type { ApprovalListParams } from "@/lib/types/approvals";

/** Stable stringify for query keys — sorts keys for consistent references. */
function stableStringify(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

/** Query keys for approvals — consistent cache invalidation. */
export const approvalsKeys = {
  all: ["approvals"] as const,
  lists: () => [...approvalsKeys.all, "list"] as const,
  list: (params: ApprovalListParams) =>
    [...approvalsKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  details: () => [...approvalsKeys.all, "detail"] as const,
  detail: (id: string) => [...approvalsKeys.details(), id] as const,
};

/**
 * Hook to fetch paginated approval list with filters.
 *
 * @param params - Query parameters (status, application_id, limit, offset)
 * @returns TanStack Query result with approvals data
 */
export function useApprovals(params: ApprovalListParams = {}) {
  return useQuery({
    queryKey: approvalsKeys.list(params),
    queryFn: () => fetchApprovals(params),
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    placeholderData: (previous) => previous,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30_000),
  });
}

/**
 * Hook to fetch a single approval by ID.
 *
 * @param id - Approval UUID
 * @returns TanStack Query result with approval data
 */
export function useApproval(id: string) {
  return useQuery({
    queryKey: approvalsKeys.detail(id),
    queryFn: () => fetchApproval(id),
    enabled: !!id,
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: 2,
  });
}

/**
 * Hook to approve an approval request.
 * Invalidates approval list and detail caches on success.
 *
 * @returns TanStack Mutation result
 */
export function useApproveApproval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: { id: string }) => approveApproval(id),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: approvalsKeys.detail(variables.id) });
      queryClient.invalidateQueries({ queryKey: approvalsKeys.lists() });
    },
  });
}

/**
 * Hook to reject an approval request with a reason.
 * Invalidates approval list and detail caches on success.
 *
 * @returns TanStack Mutation result
 */
export function useRejectApproval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      rejectApproval(id, reason),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: approvalsKeys.detail(variables.id) });
      queryClient.invalidateQueries({ queryKey: approvalsKeys.lists() });
    },
  });
}
