/**
 * TanStack Query hooks for system configuration data.
 *
 * Provides useQuery for fetching the effective config, and useMutation
 * hooks for setting and deleting overrides. Also exports executeOverrides
 * helper for batch-setting multiple keys with proper error handling.
 *
 * Does NOT:
 * - Handle authentication (use auth hooks)
 * - Manage form state (use react-hook-form or controlled components)
 *
 * Server Components should use the API client directly.
 * Client Components use these hooks.
 *
 * @see lib/api/config.ts
 * @see lib/types/config.ts
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchSystemConfig, setOverride, deleteOverride } from "@/lib/api/config";
import type { SystemConfigResponse } from "@/lib/types/config";

/**
 * Query keys for system config — consistent cache invalidation.
 */
export const systemConfigKeys = {
  all: ["system-config"] as const,
  current: () => [...systemConfigKeys.all, "current"] as const,
};

/**
 * Fetch the fully resolved system configuration.
 *
 * Returns the merged config tree with sources tracking which layer
 * (yaml/env/db) produced each value, plus optional version string.
 *
 * @returns TanStack Query result with SystemConfigResponse data
 *
 * @example
 *   const { data, isLoading } = useSystemConfig();
 *   if (data) {
 *     console.log(data.config.scoring.mode); // "hybrid"
 *     console.log(data.version); // "abc123"
 *   }
 */
export function useSystemConfig() {
  return useQuery({
    queryKey: systemConfigKeys.current(),
    queryFn: async (): Promise<SystemConfigResponse> => {
      return fetchSystemConfig();
    },
    staleTime: 2 * 60 * 1000, // 2 minutes — config changes infrequently
    select: (data) => ({
      config: data.config,
      version: data.version,
    }),
  });
}

/**
 * Mutation to set a configuration override.
 *
 * On success, invalidates the config query to show the updated value.
 *
 * @returns TanStack Query mutation result
 *
 * @example
 *   const setMutation = useSetOverride();
 *   setMutation.mutate({ key: "scoring.auto_threshold", value: 95 });
 */
export function useSetOverride() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      key,
      value,
    }: {
      key: string;
      value: unknown;
    }) => {
      return setOverride(key, value);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemConfigKeys.all });
    },
  });
}

/**
 * Mutation to delete a configuration override.
 *
 * On success, invalidates the config query to show the reverted value.
 *
 * @returns TanStack Query mutation result
 *
 * @example
 *   const deleteMutation = useDeleteOverride();
 *   deleteMutation.mutate("scoring.auto_threshold");
 */
export function useDeleteOverride() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (key: string) => {
      return deleteOverride(key);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemConfigKeys.all });
    },
  });
}

/** Result of a batch override execution. */
export interface OverrideResult {
  /** Number of overrides that succeeded. */
  succeeded: number;
  /** Number of overrides that failed. */
  failed: number;
  /** Total overrides attempted. */
  total: number;
}

/**
 * Execute multiple config overrides with proper error handling.
 *
 * Uses mutateAsync + Promise.allSettled so partial failures don't
 * silently swallow errors. Always calls onSaved after all mutations
 * complete, regardless of individual failures.
 *
 * @param overrides - Array of [key, value] pairs to set
 * @param mutateAsync - The async mutate function from useSetOverride
 * @param onSaved - Callback after all mutations complete
 * @returns OverrideResult with success/failure counts
 *
 * @example
 *   const { mutateAsync } = useSetOverride();
 *   const result = await executeOverrides(
 *     [["scoring.auto_threshold", 95], ["scoring.mode", "hybrid"]],
 *     mutateAsync,
 *     () => console.log("done"),
 *   );
 *   if (result.failed > 0) {
 *     console.warn(`${result.failed} overrides failed`);
 *   }
 */
export async function executeOverrides(
  overrides: Array<[string, unknown]>,
  mutateAsync: (params: { key: string; value: unknown }) => Promise<unknown>,
  onSaved?: () => void,
): Promise<OverrideResult> {
  if (overrides.length === 0) {
    onSaved?.();
    return { succeeded: 0, failed: 0, total: 0 };
  }

  const results = await Promise.allSettled(
    overrides.map(([key, value]) => mutateAsync({ key, value })),
  );

  const succeeded = results.filter((r) => r.status === "fulfilled").length;
  const failed = results.filter((r) => r.status === "rejected").length;

  if (failed > 0) {
    console.warn(
      `[useSystemConfig] ${failed}/${overrides.length} overrides failed:`,
      results
        .filter((r): r is PromiseRejectedResult => r.status === "rejected")
        .map((r) => r.reason),
    );
  }

  onSaved?.();
  return { succeeded, failed, total: overrides.length };
}
