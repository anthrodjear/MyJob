/**
 * Token management for JWT authentication.
 *
 * Stores the access token in localStorage (client-side only).
 * Single-user local app — localStorage is acceptable (no multi-user concerns).
 *
 * Does NOT:
 * - Validate tokens (backend validates on every request)
 * - Handle refresh tokens (backend issues long-lived JWTs)
 * - Manage login/logout UI (use hooks/useAuth.ts for that)
 *
 * @example
 *   import { setAuthToken, getAuthToken, clearAuthToken } from "@/lib/auth";
 *
 *   // After login
 *   setAuthToken("eyJhbGciOi...");
 *
 *   // Before API call (apiFetch reads this automatically)
 *   const token = getAuthToken();
 *
 *   // On logout
 *   clearAuthToken();
 */

const TOKEN_KEY = "token" as const;

/**
 * Store the auth token in localStorage.
 * Called after successful login.
 *
 * @param token - JWT access token from POST /auth/login
 */
export function setAuthToken(token: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(TOKEN_KEY, token);
}

/**
 * Retrieve the current auth token from localStorage.
 * Returns null if not logged in or on the server.
 *
 * @returns JWT string or null
 */
export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

/**
 * Remove the auth token from localStorage.
 * Called on logout or when token expires.
 */
export function clearAuthToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(TOKEN_KEY);
}

/**
 * Check if the user is authenticated (has a stored token).
 * Does NOT validate the token — use this for optimistic UI only.
 *
 * @returns true if a token exists in localStorage
 */
export function isAuthenticated(): boolean {
  return getAuthToken() !== null;
}
