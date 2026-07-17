/**
 * Base API client for all backend HTTP calls.
 *
 * Provides:
 * - JSON serialization/deserialization
 * - Error parsing with structured ApiError
 * - Configurable auth token injection (via setAuthToken)
 * - Request helpers: apiGet, apiPost, apiPatch, apiDelete
 *
 * Does NOT:
 * - Cache responses (use TanStack Query for caching)
 * - Handle retries (use TanStack Query retry config)
 * - Manage authentication state (login/logout handled elsewhere)
 *
 * @example
 *   import { apiGet } from "@/lib/api/client";
 *   import type { Job } from "@/lib/types/jobs";
 *   const jobs = await apiGet<{ items: Job[]; total: number }>("jobs");
 */

import { API_PREFIX } from "@/lib/constants";
import { getAuthToken as getStoredAuthToken, getRefreshToken, setAuthTokens, clearAuthTokens } from "@/lib/auth";

/**
 * Get the backend URL based on execution context.
 * - Server-side (Server Components, SSR): use INTERNAL_API_URL (Docker network)
 * - Client-side (browser): use NEXT_PUBLIC_API_URL (host machine)
 */
function getBackendUrl(): string {
  // Server-side: window is undefined, use internal Docker network URL
  if (typeof window === "undefined") {
    return process.env.INTERNAL_API_URL || "http://api:8080";
  }
  // Client-side: use public URL (localhost from browser perspective)
  return process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
}

const BACKEND_URL = getBackendUrl();

/**
 * Custom error class for API failures.
 * Preserves status code, error code, human-readable message,
 * and optional raw body for production debugging.
 */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public rawBody?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/**
 * Thrown when token refresh fails and the user must re-login.
 * Distinct from ApiError(401) to differentiate "request expired" from "refresh also failed".
 */
export class RefreshFailedError extends ApiError {
  constructor(message = "Session expired. Please log in again.") {
    super(401, "REFRESH_FAILED", message);
    this.name = "RefreshFailedError";
  }
}

/**
 * Auth token provider — decoupled from localStorage.
 *
 * Default: reads from localStorage (client-side only).
 * Override via setAuthToken() for testing, HTTP-only cookies, or
 * other auth mechanisms without rewriting the client.
 */
let authTokenProvider: (() => string | null) | null = null;

/**
 * Configure how the API client retrieves the auth token.
 * Call this at app startup (e.g., in a provider or test setup).
 *
 * @param provider - Function that returns the current auth token, or null
 *
 * @example
 *   // Test setup — bypass localStorage
 *   setAuthToken(() => "test-token-123");
 *
 *   // HTTP-only cookies — no token needed, browser sends cookies automatically
 *   setAuthToken(() => null);
 *
 *   // Reset to default (localStorage)
 *   setAuthToken(null);
 */
export function setAuthToken(provider: (() => string | null) | null): void {
  authTokenProvider = provider;
}

/**
 * Get the current auth token using the configured provider.
 * Falls back to localStorage if no custom provider is set.
 * Returns null on the server (no localStorage available).
 */
function getAuthToken(): string | null {
  if (authTokenProvider != null) {
    return authTokenProvider();
  }
  // Default: localStorage (client-side only)
  return getStoredAuthToken();
}

/**
 * Shared refresh promise — all concurrent 401 waiters share the same Promise.
 * Prevents multiple simultaneous refresh attempts.
 */
let refreshPromise: Promise<RefreshResult> | null = null;

/**
 * Result of a token refresh attempt.
 */
interface RefreshResult {
  /** New access token, or null if refresh failed. */
  token: string | null;
  /** Whether the failure was permanent (token definitively invalid). */
  permanent: boolean;
}

/**
 * Refresh the access token using the stored refresh token.
 * Returns the new access token, or null if refresh fails.
 * Concurrent callers share the same refresh attempt.
 */
async function refreshToken(): Promise<RefreshResult> {
  // If a refresh is already in-flight, wait for it
  if (refreshPromise != null) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const storedRefreshToken = getRefreshToken();
      if (storedRefreshToken == null) {
        return { token: null, permanent: true };
      }

      const res = await fetch(
        new URL(`${API_PREFIX}/auth/refresh`, BACKEND_URL).toString(),
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: storedRefreshToken }),
          cache: "no-store",
        }
      );

      if (!res.ok) {
        // 401/403 = token definitively invalid (permanent)
        // 5xx = server error (transient, don't clear tokens)
        const permanent = res.status < 500;
        return { token: null, permanent };
      }

      const data = (await res.json()) as {
        access_token: string;
        refresh_token: string;
        expires_at: number;
      };

      // Store new tokens
      setAuthTokens({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        expiresAt: data.expires_at,
      });

      return { token: data.access_token, permanent: false };
    } catch {
      // Network error = transient
      return { token: null, permanent: false };
    }
  })();

  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

/**
 * Wrapper around apiFetch that automatically refreshes tokens on 401.
 * Use this for all API calls that require authentication.
 *
 * On 401: attempts token refresh, retries original request once.
 * On permanent refresh failure (401/403): clears tokens and throws RefreshFailedError.
 * On transient refresh failure (5xx): throws RefreshFailedError without clearing tokens.
 */
export async function apiFetchWithRefresh<T>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  const token = getAuthToken();

  try {
    return await apiFetch<T>(path, {
      ...options,
      headers: {
        ...options?.headers,
        ...(token != null ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
  } catch (error) {
    // Only retry on 401 Unauthorized
    if (error instanceof ApiError && error.status === 401) {
      const result = await refreshToken();
      if (result.token != null) {
        // Retry the original request with the new token
        return apiFetch<T>(path, {
          ...options,
          headers: {
            ...options?.headers,
            Authorization: `Bearer ${result.token}`,
          },
        });
      }
      // Refresh failed — only clear tokens on permanent failure (token invalid)
      if (result.permanent) {
        clearAuthTokens();
      }
      throw new RefreshFailedError();
    }
    throw error;
  }
}

/**
 * Safely parse response body — handles empty bodies and non-JSON content.
 * Returns undefined if body is empty or not JSON.
 */
async function safeJsonParse(res: Response): Promise<unknown> {
  const contentType = res.headers.get("content-type");
  if (contentType != null && contentType.includes("application/json")) {
    return res.json();
  }
  // Non-JSON or missing content-type — return undefined
  return undefined;
}

/**
 * Base fetch wrapper for all backend API calls.
 *
 * Handles:
 * - URL construction (new URL prevents double slashes)
 * - JSON Content-Type header (preserves existing if already set)
 * - Auth token injection via configurable provider (skippable for public endpoints)
 * - Error response parsing with text fallback for non-JSON errors
 * - 204 No Content and non-JSON responses (returns undefined)
 * - 30s timeout via AbortController (skipped if caller provides signal)
 *
 * @param path - API path (e.g., "jobs", "applications/123")
 * @param options - Standard RequestInit options + skipAuth flag
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 * @throws ApiError on non-2xx responses
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit & { skipAuth?: boolean },
): Promise<T | undefined> {
  // new URL() prevents double slashes from misconfigured env vars
  // Strip leading slash from path to avoid /api/v1//endpoint when path = "/endpoint"
  const url = new URL(`${API_PREFIX}/${path.replace(/^\//, "")}`, BACKEND_URL);

  // Build headers — only set Content-Type if not already provided
  const headers = new Headers(options?.headers);
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  // Inject auth token via configurable provider (skip if skipAuth is true)
  if (options?.skipAuth !== true) {
    const token = getAuthToken();
    if (token != null) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  // Only create timeout if caller didn't provide their own signal.
  // If options.signal is provided, the caller owns timeout/cancellation.
  const controller = new AbortController();
  const timeoutId =
    options?.signal == null
      ? setTimeout(() => controller.abort(), 30_000)
      : null;

  try {
    const res = await fetch(url, {
      ...options,
      headers,
      cache: "no-store",
      signal: options?.signal ?? controller.signal,
    });

    if (!res.ok) {
      // Read body as text first, then try JSON parse
      // This avoids double-consuming the body stream
      const rawBody = await res.text().catch(() => "");
      let body: {
        error?: { code?: string; message?: string };
        raw?: string;
      } | null = null;

      if (rawBody.length > 0) {
        try {
          body = JSON.parse(rawBody) as typeof body;
        } catch {
          body = { raw: rawBody };
        }
      }

      const code = body?.error?.code ?? "UNKNOWN_ERROR";
      const message =
        body?.error?.message ?? `Request failed with status ${res.status}`;
      throw new ApiError(res.status, code, message, rawBody || undefined);
    }

    // 204 No Content — no body to parse
    if (res.status === 204) {
      return undefined;
    }

    // Safe JSON parse — handles empty bodies and non-JSON responses
    const parsed = await safeJsonParse(res);
    if (parsed === undefined) {
      return undefined;
    }
    return parsed as T;
  } finally {
    if (timeoutId != null) {
      clearTimeout(timeoutId);
    }
  }
}

/**
 * GET request helper.
 *
 * @param path - API path (e.g., "jobs", "jobs?status=applied")
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiGet<T>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetch<T>(path, { ...options, method: "GET" });
}

/**
 * POST request helper.
 *
 * @param path - API path (e.g., "jobs/123/approve")
 * @param data - Request body (auto-serialized to JSON). Pass undefined to skip body.
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPost<T>(
  path: string,
  data?: unknown,
  options?: RequestInit & { skipAuth?: boolean },
): Promise<T | undefined> {
  return apiFetch<T>(path, {
    ...options,
    method: "POST",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * PUT request helper.
 *
 * @param path - API path (e.g., "applications/123/status")
 * @param data - Request body (auto-serialized to JSON). Pass undefined to skip body.
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPut<T>(
  path: string,
  data?: unknown,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetch<T>(path, {
    ...options,
    method: "PUT",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * PATCH request helper.
 *
 * @param path - API path (e.g., "profile", "jobs/123")
 * @param data - Request body (partial update, auto-serialized to JSON)
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPatch<T>(
  path: string,
  data: unknown,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetch<T>(path, {
    ...options,
    method: "PATCH",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * DELETE request helper.
 * Most DELETE endpoints return 204 No Content (undefined).
 *
 * @param path - API path (e.g., "jobs/123")
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiDelete<T = void>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetch<T>(path, { ...options, method: "DELETE" });
}

/**
 * GET request helper WITH automatic token refresh.
 * Use this for all authenticated GET requests.
 *
 * @param path - API path (e.g., "jobs", "jobs?status=applied")
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiGetWithRefresh<T>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetchWithRefresh<T>(path, { ...options, method: "GET" });
}

/**
 * POST request helper WITH automatic token refresh.
 * Use this for all authenticated POST requests.
 *
 * @param path - API path (e.g., "jobs/123/approve")
 * @param data - Request body (auto-serialized to JSON). Pass undefined to skip body.
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPostWithRefresh<T>(
  path: string,
  data?: unknown,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetchWithRefresh<T>(path, {
    ...options,
    method: "POST",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * PUT request helper WITH automatic token refresh.
 * Use this for all authenticated PUT requests.
 *
 * @param path - API path (e.g., "applications/123/status")
 * @param data - Request body (auto-serialized to JSON). Pass undefined to skip body.
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPutWithRefresh<T>(
  path: string,
  data?: unknown,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetchWithRefresh<T>(path, {
    ...options,
    method: "PUT",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * PATCH request helper WITH automatic token refresh.
 * Use this for all authenticated PATCH requests.
 *
 * @param path - API path (e.g., "profile", "jobs/123")
 * @param data - Request body (partial update, auto-serialized to JSON)
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiPatchWithRefresh<T>(
  path: string,
  data: unknown,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetchWithRefresh<T>(path, {
    ...options,
    method: "PATCH",
    body: data != null ? JSON.stringify(data) : undefined,
  });
}

/**
 * DELETE request helper WITH automatic token refresh.
 * Use this for all authenticated DELETE requests.
 *
 * @param path - API path (e.g., "jobs/123")
 * @param options - Optional RequestInit overrides (headers, signal, etc.)
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 */
export function apiDeleteWithRefresh<T = void>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  return apiFetchWithRefresh<T>(path, { ...options, method: "DELETE" });
}

/**
 * Raw fetch with auth injection and error parsing.
 * Returns the full Response so callers can access headers (e.g., ETag).
 *
 * Use this when you need response headers that apiFetch doesn't expose.
 * Auth token is injected automatically. Errors are parsed into ApiError.
 */
export async function authFetch(
  input: string | URL,
  init?: RequestInit,
): Promise<Response> {
  const url = typeof input === "string"
    ? new URL(`${API_PREFIX}/${input.replace(/^\//, "")}`, BACKEND_URL)
    : input;

  const headers = new Headers(init?.headers);
  if (!headers.has("Content-Type") && init?.body != null) {
    headers.set("Content-Type", "application/json");
  }

  const token = getAuthToken();
  if (token != null) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(url, { ...init, headers, cache: "no-store" });

  if (!res.ok) {
    const rawBody = await res.text().catch(() => "");
    let parsed: { error?: { code?: string; message?: string } } | null = null;
    if (rawBody.length > 0) {
      try { parsed = JSON.parse(rawBody); } catch { /* non-JSON */ }
    }
    const code = parsed?.error?.code ?? "UNKNOWN_ERROR";
    const message = parsed?.error?.message ?? `Request failed with status ${res.status}`;
    throw new ApiError(res.status, code, message, rawBody || undefined);
  }

  return res;
}
