/**
 * DEPRECATED — localStorage auth helpers removed.
 *
 * This module previously stored JWT access/refresh tokens in localStorage.
 * The application has migrated to HTTP-only session cookies managed by:
 * - /api/auth/[...proxy]/route.ts — sets/encrypts the session cookie
 * - /api/auth/session/route.ts — reads the session cookie for client checks
 * - lib/session.ts — JWT encrypt/decrypt for the cookie payload
 * - lib/dal.ts — Server Component auth (reads cookie, forwards Bearer header)
 *
 * If you need to log out a user, call the logout Route Handler:
 *   await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
 *
 * To force logout from localStorage (e.g., on password reset):
 *   await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
 */

export {};
