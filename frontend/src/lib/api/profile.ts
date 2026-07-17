/**
 * API client for the Profile domain.
 *
 * Handles ETag-based optimistic concurrency control:
 * - GET returns the profile + ETag header
 * - PUT/PATCH require If-Match header with the current ETag
 * - On VERSION_CONFLICT (409), the caller should re-fetch and retry
 *
 * Does NOT cache — use TanStack Query for caching.
 *
 * @see backend/internal/profile/handler.go
 */

import { authFetch, ApiError } from "@/lib/api/client";
import type { Profile, PatchProfileRequest, UpdateProfileRequest } from "@/lib/types/profile";

/**
 * Profile response with ETag for optimistic concurrency.
 * The ETag is extracted from the response header, not the JSON body.
 */
export interface ProfileResponseWithETag {
  /** The profile data. */
  profile: Profile;
  /** ETag value for If-Match header on subsequent PUT/PATCH. */
  etag: string;
}

/**
 * Fetch the singleton profile.
 *
 * Extracts the ETag header from the response for use in subsequent
 * PUT/PATCH requests (optimistic concurrency control).
 *
 * @returns Profile + ETag for concurrency control
 * @throws ApiError on non-2xx responses
 *
 * @example
 *   const { profile, etag } = await fetchProfile();
 *   // Use etag for subsequent updates
 *   await updateProfile(etag, { preferences: { remote_only: true } });
 */
export async function fetchProfile(): Promise<ProfileResponseWithETag> {
  const res = await authFetch("/profile", { method: "GET" });
  const profile = (await res.json()) as Profile;
  const etag = res.headers.get("etag") ?? "";
  return { profile, etag };
}

/**
 * Update the entire profile (PUT).
 *
 * Requires the ETag from a previous GET to prevent concurrent overwrites.
 * On VERSION_CONFLICT (409), the caller should re-fetch and retry.
 *
 * @param etag - ETag from the most recent GET request
 * @param data - Full profile data to replace
 * @returns Updated profile + new ETag
 * @throws ApiError with status 409 on VERSION_CONFLICT
 *
 * @example
 *   const { profile, etag } = await fetchProfile();
 *   const updated = await updateProfile(etag, {
 *     preferences: { ...profile.data.preferences, remote_only: true },
 *     skills: profile.data.skills,
 *   });
 */
export async function updateProfile(
  etag: string,
  data: UpdateProfileRequest,
): Promise<ProfileResponseWithETag> {
  const res = await authFetch("/profile", {
    method: "PUT",
    headers: { "If-Match": etag },
    body: JSON.stringify(data),
  });
  const profile = (await res.json()) as Profile;
  const newEtag = res.headers.get("etag") ?? "";
  return { profile, etag: newEtag };
}

/**
 * Partially update the profile (PATCH).
 *
 * Only provided fields are merged into the existing profile.
 * Nil pointer fields are ignored (don't change).
 * Requires the ETag from a previous GET.
 *
 * @param etag - ETag from the most recent GET request
 * @param data - Partial profile data to merge
 * @returns Updated profile + new ETag
 * @throws ApiError with status 409 on VERSION_CONFLICT
 *
 * @example
 *   const { profile, etag } = await fetchProfile();
 *   // Only update remote_only, everything else stays the same
 *   const updated = await patchProfile(etag, {
 *     preferences: { remote_only: false },
 *   });
 */
export async function patchProfile(
  etag: string,
  data: PatchProfileRequest,
): Promise<ProfileResponseWithETag> {
  const res = await authFetch("/profile", {
    method: "PATCH",
    headers: { "If-Match": etag },
    body: JSON.stringify(data),
  });
  const profile = (await res.json()) as Profile;
  const newEtag = res.headers.get("etag") ?? "";
  return { profile, etag: newEtag };
}
