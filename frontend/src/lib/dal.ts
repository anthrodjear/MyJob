/**
 * Data Access Layer (DAL) — Server Component auth + data fetching.
 *
 * This is the SERVER-SIDE auth bridge. It reads the encrypted session
 * cookie (set by /api/auth/[...proxy]/route.ts) and forwards the
 * backend access token to the Go API as Authorization header.
 *
 * Usage in Server Components:
 *   import { dalFetch, dalRequireAuth } from "@/lib/dal";
 *   const data = await dalFetch("/jobs");
 *
 * Why this exists:
 *   - Server Components can't use localStorage (browser API)
 *   - Server Components can't call apiFetch (reads from localStorage)
 *   - This reads the session cookie via next/headers cookies()
 *   - Then forwards the decrypted accessToken to the backend
 *
 * @see app/api/auth/[...proxy]/route.ts — sets the session cookie
 * @see proxy.ts — reads the cookie for client-side redirects
 * @see lib/session.ts — JWT encrypt/decrypt
 */

import "server-only";
import { cookies } from "next/headers";
import { decrypt, type SessionPayload } from "@/lib/session";

/** Go backend base URL. */
const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

/** Backend API prefix. */
const API_PREFIX = "/api/v1";

/** Custom error for authentication failures in Server Components. */
export class AuthError extends Error {
  constructor(message = "Authentication required") {
    super(message);
    this.name = "AuthError";
  }
}

/** Custom error for backend fetch failures. */
export class DalError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly errorBody?: string,
  ) {
    super(message);
    this.name = "DalError";
  }
}

/**
 * Read and decrypt the session cookie. Returns null if not present or invalid.
 *
 * Used by dalRequireAuth() and dalFetch() to get the access token.
 */
export async function getSession(): Promise<SessionPayload | null> {
  const cookieStore = await cookies();
  const sessionCookie = cookieStore.get("session")?.value;
  return decrypt(sessionCookie);
}

/**
 * Require a valid session. Throws AuthError if not authenticated.
 *
 * Use in Server Components that must be behind auth:
 *   const session = await dalRequireAuth();
 *   // session.accessToken is guaranteed non-null
 */
export async function dalRequireAuth(): Promise<SessionPayload> {
  const session = await getSession();
  if (!session?.accessToken) {
    throw new AuthError();
  }
  return session;
}

/**
 * Fetch from the Go backend with auth headers from the session cookie.
 *
 * This is the primary data fetching function for Server Components.
 * It reads the session cookie, decrypts it, and forwards the access
 * token as Authorization: Bearer header.
 *
 * @param path - Backend API path (without /api/v1 prefix), e.g. "/jobs"
 * @param options - Standard fetch options (method, body, etc.)
 * @returns Parsed JSON response (or undefined for 204 No Content)
 * @throws AuthError if no session, DalError if backend returns non-2xx
 *
 * @example
 *   const jobs = await dalFetch("/jobs");
 *   const stats = await dalFetch("/dashboard/stats", { method: "POST" });
 *
 * Note: 204 No Content returns undefined. Callers should handle this.
 */
export async function dalFetch<T = unknown>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const session = await dalRequireAuth();

  const { headers: customHeaders, ...rest } = options;

  const headers = new Headers(customHeaders);
  headers.set("Authorization", `Bearer ${session.accessToken}`);
  if (!headers.has("Content-Type") && typeof options.body === "string") {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${BACKEND_URL}${API_PREFIX}${path}`, {
    ...rest,
    headers,
  });

  if (!response.ok) {
    // 401 = access token expired or invalid → throw AuthError (not DalError)
    if (response.status === 401) {
      throw new AuthError("Session expired");
    }
    const errorBody = await response.text();
    throw new DalError(
      `Backend request failed: ${response.status}`,
      response.status,
      errorBody,
    );
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}
