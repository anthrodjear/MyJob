/**
 * TanStack Query hooks for authentication.
 *
 * Provides useMutation hooks for login, refresh, and password change.
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { login, refreshAccessToken, changePassword } from "@/lib/api/auth";
import { clearAuthTokens, getAuthToken, getTokenExpiry } from "@/lib/auth";

/** Query keys for auth — consistent cache invalidation. */
export const authKeys = {
  all: ["auth"] as const,
  profile: () => [...authKeys.all, "profile"] as const,
};

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
 * Logout — clears tokens and invalidates all queries.
 *
 * Not a mutation (no backend call needed — token is stateless).
 * Increments session version on backend to invalidate all tokens,
 * but for a local app, clearing localStorage is sufficient.
 *
 * @example
 *   const logout = useLogout();
 *   logout();
 */
export function useLogout() {
  const queryClient = useQueryClient();

  return () => {
    clearAuthTokens();
    queryClient.clear();
    // Redirect to login
    window.location.href = "/login";
  };
}

/**
 * Hook to proactively refresh the access token before it expires.
 * Sets up an interval to check and refresh tokens periodically.
 *
 * @param intervalMs - How often to check/refresh (default: 5 minutes)
 * @returns Object with the last refresh result
 *
 * @example
 *   // In a layout or provider component
 *   useTokenRefresher({ intervalMs: 5 * 60 * 1000 }); // Every 5 minutes
 */
export function useTokenRefresher({
  intervalMs = 5 * 60 * 1000,
}: { intervalMs?: number } = {}) {
  const { mutate: refresh } = useRefreshToken();
  const refreshInProgress = useRef(false);

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
        if (nowSeconds >= expiry - 120 && !refreshInProgress.current) {
          refreshInProgress.current = true;
          refresh(undefined, {
            onSettled: () => {
              refreshInProgress.current = false;
            },
          });
        }
      }
    }, intervalMs);

    return () => clearInterval(id);
  }, [intervalMs, refresh]);

  return {
    refresh,
    isRefreshing: refreshInProgress.current,
  };
}
