/**
 * TanStack Query hooks for authentication.
 *
 * Provides useMutation hooks for login and password change.
 * Server Components should use the API client directly;
 * Client Components use these hooks.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { login, changePassword } from "@/lib/api/auth";
import { clearAuthToken } from "@/lib/auth";

/** Query keys for auth — consistent cache invalidation. */
export const authKeys = {
  all: ["auth"] as const,
  profile: () => [...authKeys.all, "profile"] as const,
};

/**
 * Login mutation — authenticates and stores the JWT.
 *
 * On success: stores token in localStorage, optionally refetch profile.
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
    mutationFn: (password: string) => login(password),
    onSuccess: () => {
      // Invalidate profile query so it refetches with new token
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
 * Logout — clears token and invalidates all queries.
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
    clearAuthToken();
    queryClient.clear();
    // Redirect to login
    window.location.href = "/login";
  };
}
