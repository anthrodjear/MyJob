/**
 * TanStack Query hooks for authentication.
 *
 * Provides useMutation hooks for login, refresh, password change, and logout.
 * Server Components should use dalFetch (lib/dal.ts);
 * Client Components use these hooks.
 *
 * Auth flow:
 *   - Login: calls POST /api/auth/login (Route Handler) → sets HTTP-only session cookie
 *   - Refresh: calls POST /api/auth/refresh (Route Handler) → updates session cookie
 *   - Logout: calls POST /api/auth/logout (Route Handler) → clears session cookie
 *   - Password change: calls Go backend directly via apiPost (no cookie change needed)
 *
 * The session cookie is HTTP-only — JS cannot read it directly.
 * proxy.ts reads it for client-side redirects.
 * lib/dal.ts reads it for Server Component auth.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { changePassword } from "@/lib/api/auth";

/** Query keys for auth — consistent cache invalidation. */
export const authKeys = {
  all: ["auth"] as const,
  profile: () => [...authKeys.all, "profile"] as const,
};

/**
 * Login mutation — authenticates via Route Handler and sets session cookie.
 *
 * Calls POST /api/auth/login (Next.js Route Handler) which:
 *   1. Proxies to Go backend POST /api/v1/auth/login
 *   2. Encrypts tokens into HTTP-only session cookie
 *   3. Returns success to the client
 *
 * On success: session cookie is set (HTTP-only, not accessible by JS).
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
    mutationFn: async (password: string) => {
      const resp = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });

      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}));
        throw new Error(body?.error ?? "Login failed");
      }

      return resp.json();
    },
    onSuccess: () => {
      // Invalidate profile query so it refetches with new session
      void queryClient.invalidateQueries({ queryKey: authKeys.profile() });
    },
  });
}

/**
 * Refresh access token via Route Handler.
 *
 * Calls POST /api/auth/refresh which reads the session cookie,
 * uses the refresh token to get new tokens from Go backend,
 * and updates the session cookie.
 *
 * On failure: session cookie is cleared, user must re-login.
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
    mutationFn: async () => {
      const resp = await fetch("/api/auth/refresh", {
        method: "POST",
      });

      if (!resp.ok) {
        throw new Error("Refresh failed");
      }

      return resp.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: authKeys.profile() });
    },
    onError: () => {
      // Refresh token is invalid/expired — Route Handler cleared the cookie
      queryClient.clear();
      window.location.href = "/login";
    },
  });
}

/**
 * Change password mutation.
 *
 * Requires current password for verification.
 * Does NOT log the user out — session cookie remains valid after password change.
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
 * Logout mutation — clears session cookie and redirects.
 *
 * Calls POST /api/auth/logout (clears HTTP-only cookie),
 * then redirects to /login.
 *
 * @example
 *   const logoutMutation = useLogout();
 *   logoutMutation.mutate();
 */
export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      // Best-effort cookie clearing via Route Handler
      try {
        await fetch("/api/auth/logout", { method: "POST" });
      } catch {
        // Route Handler may be unreachable — still redirect
      }
    },
    onSettled: () => {
      queryClient.clear();
      // Full page reload to clear all in-memory state
      window.location.href = "/login";
    },
  });
}
