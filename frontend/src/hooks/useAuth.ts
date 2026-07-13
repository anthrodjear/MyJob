/**
 * TanStack Query hooks for authentication.
 *
 * Provides useMutation hooks for login, refresh, password change, and logout.
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { login, refreshAccessToken, changePassword, logout as logoutApi } from "@/lib/api/auth";
import { clearAuthTokens, getAuthToken, getTokenExpiry } from "@/lib/auth";

/** Query keys for auth — consistent cache invalidation. */
export const authKeys = {
  all: ["auth"] as const,
  profile: () => [...authKeys.all, "profile"] as const,
};

/**
 * Internal helper: force logout when refresh token is invalid.
 * Clears local state and redirects to login.
 */
function forceLogout(): void {
  clearAuthTokens();
  // Full page reload to clear all in-memory state
  window.location.href = "/login";
}

/**
 * Login mutation — authenticates and stores the JWT + refresh token.
 *
 * Note: login() stores tokens in localStorage internally (via lib/auth.ts).
 * The side effect is in the API function, not the hook.
 *
 * On success: stores tokens in localStorage, optionally refetch profile.
 * On error: throws ApiError (caller handles display).
 *
 * @example
 *   const loginMutation = useLogin();
 *   loginMutation.mutate("my-password", {
 *     onSuccess: () => router.push("/dashboard"),
 *   });
 */
export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation({
    // Note: login() stores tokens in localStorage internally
    mutationFn: (password: string) => login(password),
    onSuccess: () => {
      // Invalidate profile query so it refetches with new token
      void queryClient.invalidateQueries({ queryKey: authKeys.profile() });
    },
  });
}

/**
 * Refresh access token mutation.
 *
 * Uses the stored refresh token to get new tokens.
 * On failure: clears tokens and redirects to login (refresh token invalid/expired).
 *
 * @example
 *   const refreshMutation = useRefreshToken();
 *   refreshMutation.mutate(undefined, {
 *     onSuccess: () => console.log("Tokens refreshed"),
 *   });
 */
export function useRefreshToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => refreshAccessToken(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: authKeys.profile() });
    },
    onError: () => {
      // Refresh token is invalid/expired — force re-authentication
      queryClient.clear();
      forceLogout();
    },
  });
}

/**
 * Change password mutation.
 *
 * Requires current password for verification.
 * Does NOT log the user out — token remains valid after password change.
 *
 * @example
 *   const changePasswordMutation = useChangePassword();
 *   changePasswordMutation.mutate(
 *     { currentPassword: "old", newPassword: "new-strong-pass" },
 *     { onSuccess: () => showToast("Password changed") },
 *   );
 */
export function useChangePassword() {
  return useMutation({
    mutationFn: ({
      currentPassword,
      newPassword,
    }: {
      currentPassword: string;
      newPassword: string;
    }) => changePassword(currentPassword, newPassword),
  });
}

/**
 * Logout mutation — revokes refresh tokens and clears local state.
 *
 * Calls backend to revoke all refresh tokens, then clears local state.
 * If the backend call fails, local tokens are still cleared (best-effort).
 * Returns mutation object with isPending for loading state.
 *
 * @example
 *   const logoutMutation = useLogout();
 *   logoutMutation.mutate();
 *   // or check logoutMutation.isPending for button loading state
 */
export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      // Best-effort server-side revocation
      try {
        await logoutApi();
      } catch {
        // Backend may be unreachable — still clear local state
      }
    },
    onSettled: () => {
      clearAuthTokens();
      queryClient.clear();
      // Full page reload to clear all in-memory state
      window.location.href = "/login";
    },
  });
}

/**
 * Hook to proactively refresh the access token before it expires.
 * Sets up an interval to check and refresh tokens periodically.
 *
 * @param intervalMs - How often to check/refresh (default: 5 minutes)
 * @returns Object with refresh function and reactive isRefreshing state
 *
 * @example
 *   // In a layout or provider component
 *   const { isRefreshing } = useTokenRefresher({ intervalMs: 5 * 60 * 1000 });
 */
export function useTokenRefresher({
  intervalMs = 5 * 60 * 1000,
}: { intervalMs?: number } = {}) {
  const refreshMutation = useRefreshToken();
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    // Skip if interval is not positive
    if (intervalMs <= 0) return;

    const id = setInterval(() => {
      // Only refresh if token exists and is expiring soon
      const token = getAuthToken();
      const expiry = getTokenExpiry();
      if (token != null && expiry != null) {
        const nowSeconds = Math.floor(Date.now() / 1000);
        // Refresh if expiring within 2 minutes and no refresh in progress
        if (nowSeconds >= expiry - 120 && !isRefreshing) {
          setIsRefreshing(true);
          refreshMutation.mutate(undefined, {
            onSettled: () => setIsRefreshing(false),
          });
        }
      }
    }, intervalMs);

    return () => clearInterval(id);
  }, [intervalMs, refreshMutation, isRefreshing]);

  return {
    refresh: refreshMutation.mutate,
    isRefreshing,
  };
}
