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

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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
  if (typeof window !== "undefined") {
    return localStorage.getItem("token");
  }
  return null;
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
 * - Auth token injection via configurable provider
 * - Error response parsing with text fallback for non-JSON errors
 * - 204 No Content and non-JSON responses (returns undefined)
 * - 30s timeout via AbortController (skipped if caller provides signal)
 *
 * @param path - API path (e.g., "jobs", "applications/123")
 * @param options - Standard RequestInit options
 * @returns Parsed JSON response, or undefined for 204/non-JSON responses
 * @throws ApiError on non-2xx responses
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T | undefined> {
  // new URL() prevents double slashes from misconfigured env vars
  const url = new URL(`${API_PREFIX}/${path}`, BACKEND_URL);

  // Build headers — only set Content-Type if not already provided
  const headers = new Headers(options?.headers);
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  // Inject auth token via configurable provider
  const token = getAuthToken();
  if (token != null) {
    headers.set("Authorization", `Bearer ${token}`);
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
      // Try JSON first (standard backend error envelope),
      // fall back to raw text for non-JSON errors (nginx HTML, plain text)
      let body: {
        error?: { code?: string; message?: string };
        raw?: string;
      } | null = null;
      let rawBody = "";

      try {
        body = await res.json();
      } catch {
        rawBody = await res.text().catch(() => "");
        body = { raw: rawBody };
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
  options?: RequestInit,
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
