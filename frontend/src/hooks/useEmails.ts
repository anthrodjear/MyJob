/**
 * TanStack Query hooks for Emails domain.
 *
 * Provides list/detail queries and mutations for email operations
 * including optimistic updates for read status.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchEmails,
  fetchEmail,
  updateEmail,
  classifyEmail,
} from "@/lib/api/emails";
import type { EmailFilterInput, UpdateEmailInput } from "@/lib/schemas/emails";
import type { Email, EmailListResponse } from "@/lib/types/emails";

/** Empty email list response for graceful degradation. */
const emptyEmails: EmailListResponse = { emails: [], total: 0, limit: 0, offset: 0 };

/**
 * Fetch paginated email list with optional filters.
 *
 * @param params - Filter and pagination parameters
 */
export function useEmails(params?: EmailFilterInput) {
  return useQuery({
    queryKey: ["emails", params],
    queryFn: ({ signal }) => fetchEmails(params, signal),
    placeholderData: emptyEmails,
  });
}

/**
 * Fetch a single email by ID.
 *
 * @param id - Email UUID (enabled only when non-empty)
 */
export function useEmail(id: string) {
  return useQuery({
    queryKey: ["emails", id],
    queryFn: ({ signal }) => fetchEmail(id, signal),
    enabled: !!id,
  });
}

/**
 * Update email read status or reply draft with optimistic cache rollback.
 */
export function useUpdateEmail() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateEmailInput }) =>
      updateEmail(id, data),
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: ["emails", id] });

      const previous = queryClient.getQueryData<Email>(["emails", id]);

      if (previous) {
        queryClient.setQueryData<Email>(["emails", id], {
          ...previous,
          ...data,
        });
      }

      return { previous };
    },
    onError: (_err, { id }, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["emails", id], context.previous);
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ["emails"] });
    },
  });
}

/**
 * Trigger email re-classification with cache invalidation.
 */
export function useClassifyEmail() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => classifyEmail(id),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ["emails"] });
    },
  });
}
