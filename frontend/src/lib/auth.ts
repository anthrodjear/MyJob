/**
 * Token management for JWT authentication with refresh token support.
 *
 * Stores access token and refresh token in localStorage (client-side only).
 * Single-user local app — localStorage is acceptable (no multi-user concerns).
 *
 * Security note: Both tokens are in localStorage. If an XSS vulnerability exists,
 * an attacker can steal both. The refresh token allows persistent access until it
 * expires (7 days). For this local-only app, this is acceptable — but be aware of
 * the risk if the scope ever changes.
 *
 * Does NOT:
 * - Validate tokens (backend validates on every request)
 * - Manage login/logout UI (use hooks/useAuth.ts for that)
 *
 * Token lifecycle:
 * - Access token: short-lived (30min default), used for API calls
 * - Refresh token: long-lived (7 days default), used to get new access tokens
 * - Both stored in localStorage, both cleared on logout
 *
 * @example
 *   import { setAuthTokens, getAuthToken, clearAuthTokens } from "@/lib/auth";
 *
 *   // After login
 *   setAuthTokens({ accessToken: "eyJ...", refreshToken: "abc...", expiresAt: 1234567890 });
 *
 *   // Before API call (apiFetch reads this automatically)
 *   const token = getAuthToken();
 *
 *   // On logout
 *   clearAuthTokens();
 */

const TOKEN_KEY = "token" as const;
const REFRESH_TOKEN_KEY = "refresh_token" as const;
const TOKEN_EXPIRY_KEY = "token_expiry" as const;

/** Stored auth tokens. */
export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresAt: number; // Unix timestamp in seconds
}

/**
 * Store auth tokens in localStorage.
 * Called after successful login or token refresh.
 *
 * @param tokens - Access token, refresh token, and expiry from POST /auth/login or POST /auth/refresh
 */
export function setAuthTokens(tokens: AuthTokens): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(TOKEN_KEY, tokens.accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  localStorage.setItem(TOKEN_EXPIRY_KEY, String(tokens.expiresAt));
}

/**
 * Retrieve the current access token from localStorage.
 * Returns null if not logged in or on the server.
 *
 * @returns JWT string or null
 */
export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

/**
 * Retrieve the current refresh token from localStorage.
 *
 * @returns Refresh token string or null
 */
export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

/**
 * Retrieve the access token expiry time.
 *
 * @returns Unix timestamp in seconds, or null if not set or invalid
 */
export function getTokenExpiry(): number | null {
  if (typeof window === "undefined") return null;
  const val = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (val === null) return null;
  const num = Number(val);
  return Number.isFinite(num) && num > 0 ? num : null;
}

/**
 * Check if the access token is expired or will expire within the given buffer.
 *
 * @param bufferSeconds - Seconds before expiry to consider "expired" (default: 60)
 * @returns true if token is expired or missing
 */
export function isTokenExpired(bufferSeconds = 60): boolean {
  const expiry = getTokenExpiry();
  if (expiry === null) return true;
  const nowSeconds = Math.floor(Date.now() / 1000);
  return nowSeconds >= expiry - bufferSeconds;
}

/**
 * Remove all auth tokens from localStorage.
 * Called on logout or when tokens are invalid.
 */
export function clearAuthTokens(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(TOKEN_EXPIRY_KEY);
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
