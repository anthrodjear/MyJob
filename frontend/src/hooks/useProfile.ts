/**
 * TanStack Query hooks for profile data.
 *
 * Provides useQuery for fetching the profile with ETag tracking,
 * and useMutation hooks for full replace (PUT) and partial merge (PATCH).
 *
 * ETag-based optimistic concurrency is managed automatically:
 * - The query stores the current ETag alongside the profile in cache
 * - Mutations read the ETag from cache and attach it as If-Match header
 * - On VERSION_CONFLICT (409), the query is invalidated to re-fetch
 *
 * Does NOT:
 * - Handle authentication (use auth hooks)
 * - Manage form state (use react-hook-form or controlled components)
 *
 * Server Components should use the API client directly.
 * Client Components use these hooks.
 *
 * @see lib/api/profile.ts
 * @see lib/types/profile.ts
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchProfile, updateProfile, patchProfile } from "@/lib/api/profile";
import type { Profile, PatchProfileRequest, UpdateProfileRequest } from "@/lib/types/profile";
import { ApiError } from "@/lib/api/client";

// ---------------------------------------------------------------------------
// Query Keys
// ---------------------------------------------------------------------------

/**
 * Query keys for profile — consistent cache invalidation.
 *
 * @warning Prefer using the mutation hooks (useUpdateProfile, usePatchProfile)
 * which handle ETag concurrency automatically. Direct invalidation may
 * cause stale ETag errors if the cache is not properly updated.
 */
export const profileKeys = {
  all: ["profile"] as const,
  current: () => [...profileKeys.all, "current"] as const,
};

// ---------------------------------------------------------------------------
// Internal Types
// ---------------------------------------------------------------------------

/**
 * Profile + ETag stored together in the query cache.
 *
 * Co-locating the ETag with the profile data eliminates the SSR race condition
 * of module-level mutable state and makes tests deterministic.
 * The `select` option on useProfile ensures components see only Profile,
 * not this internal wrapper.
 */
interface ProfileWithETag {
  profile: Profile;
  etag: string;
}

// ---------------------------------------------------------------------------
// Query Hooks
// ---------------------------------------------------------------------------

/**
 * Fetch the singleton profile.
 *
 * Returns the profile with computed stats. The ETag is stored in the
 * query cache alongside the profile for use by mutation hooks.
 *
 * Components receive only the Profile (not the ETag) via `select`.
 *
 * @returns TanStack Query result with Profile data
 *
 * @example
 *   const { data: profile, isLoading } = useProfile();
 *   if (profile) {
 *     console.log(profile.data.preferences.remote_only);
 *   }
 */
export function useProfile() {
  return useQuery({
    queryKey: profileKeys.current(),
    queryFn: async (): Promise<ProfileWithETag> => {
      const result = await fetchProfile();
      return { profile: result.profile, etag: result.etag };
    },
    staleTime: 5 * 60 * 1000, // 5 minutes — profile changes infrequently
    select: (data) => data.profile,
  });
}

// ---------------------------------------------------------------------------
// Mutation Hooks
// ---------------------------------------------------------------------------

/**
 * Mutation to fully replace the profile (PUT).
 *
 * Reads the ETag from the query cache for optimistic concurrency.
 * On VERSION_CONFLICT (409), the query is invalidated so the user
 * sees the latest state.
 *
 * @returns TanStack Query mutation result
 *
 * @example
 *   const updateMutation = useUpdateProfile();
 *   updateMutation.mutate({
 *     preferences: { remote_only: true },
 *     skills: [{ name: "Go", proficiency: "advanced" }],
 *   });
 */
export function useUpdateProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: UpdateProfileRequest): Promise<ProfileWithETag> => {
      // Try cache first; if empty, fetch fresh (handles direct navigation to settings)
      let cached = queryClient.getQueryData<ProfileWithETag>(profileKeys.current());
      if (!cached?.etag) {
        cached = await queryClient.fetchQuery({
          queryKey: profileKeys.current(),
          queryFn: async (): Promise<ProfileWithETag> => {
            const result = await fetchProfile();
            return { profile: result.profile, etag: result.etag };
          },
        });
      }
      if (!cached?.etag) {
        throw new ApiError(0, "ETAG_MISSING", "Could not load profile for updating");
      }
      const result = await updateProfile(cached.etag, data);
      return { profile: result.profile, etag: result.etag };
    },
    onSuccess: (result) => {
      queryClient.setQueryData(profileKeys.current(), result);
    },
    onError: (error) => {
      // On VERSION_CONFLICT, invalidate to force re-fetch
      if (error instanceof ApiError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: profileKeys.all });
      }
    },
  });
}

/**
 * Mutation to partially update the profile (PATCH).
 *
 * Only provided fields are merged; nil fields are ignored.
 * Reads the ETag from the query cache for optimistic concurrency.
 *
 * @returns TanStack Query mutation result
 *
 * @example
 *   const patchMutation = usePatchProfile();
 *   // Only update remote_only, everything else stays the same
 *   patchMutation.mutate({
 *     preferences: { remote_only: false },
 *   });
 */
export function usePatchProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: PatchProfileRequest): Promise<ProfileWithETag> => {
      // Try cache first; if empty, fetch fresh (handles direct navigation to settings)
      let cached = queryClient.getQueryData<ProfileWithETag>(profileKeys.current());
      if (!cached?.etag) {
        cached = await queryClient.fetchQuery({
          queryKey: profileKeys.current(),
          queryFn: async (): Promise<ProfileWithETag> => {
            const result = await fetchProfile();
            return { profile: result.profile, etag: result.etag };
          },
        });
      }
      if (!cached?.etag) {
        throw new ApiError(0, "ETAG_MISSING", "Could not load profile for patching");
      }
      const result = await patchProfile(cached.etag, data);
      return { profile: result.profile, etag: result.etag };
    },
    onSuccess: (result) => {
      queryClient.setQueryData(profileKeys.current(), result);
    },
    onError: (error) => {
      // On VERSION_CONFLICT, invalidate to force re-fetch
      if (error instanceof ApiError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: profileKeys.all });
      }
    },
  });
}
