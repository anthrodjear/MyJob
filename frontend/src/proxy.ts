/**
 * Proxy (formerly middleware.ts in Next.js 16).
 *
 * Runs before any page render. Validates the JWT session cookie
 * and redirects unauthenticated users to /login.
 *
 * This is the OPTIMISTIC auth layer — fast redirects only.
 * Real enforcement happens in lib/dal.ts (Data Access Layer)
 * and the Go backend's AuthMiddleware.
 *
 * Renamed from middleware.ts per Next.js 16 convention.
 *
 * @see https://nextjs.org/docs/app/api-reference/file-conventions/proxy
 */

import { NextRequest, NextResponse } from "next/server";
import { decrypt, type SessionPayload } from "@/lib/session";

/** Routes that require authentication. */
const PROTECTED_ROUTES = ["/dashboard"];

/** Routes that are always public (never redirect away). */
const PUBLIC_ROUTES = ["/login", "/setup", "/forgot-password", "/reset-password"];

/**
 * Paths excluded from proxy execution entirely.
 * These are checked both here (safety net) and in config.matcher (primary gate).
 */
const EXCLUDED_PATHS = ["/api", "/_next", "/favicon.ico", "/swagger"];

function isExcludedPath(path: string): boolean {
  return EXCLUDED_PATHS.some((excluded) => path.startsWith(excluded));
}

function isProtectedRoute(path: string): boolean {
  return PROTECTED_ROUTES.some((route) => path.startsWith(route));
}

function isPublicRoute(path: string): boolean {
  return PUBLIC_ROUTES.some((route) => path.startsWith(route));
}

/**
 * Proxy function — validates session cookie and redirects.
 *
 * Does NOT forward auth headers to backend (Server Components do that
 * via lib/dal.ts). This layer only handles client-facing redirects.
 */
export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Skip excluded paths (API routes, static files, swagger)
  if (isExcludedPath(pathname)) {
    return NextResponse.next();
  }

  // Read session cookie and decrypt
  const sessionCookie = request.cookies.get("session")?.value;
  const session: SessionPayload | null = await decrypt(sessionCookie);

  const isAuthenticated =
    session?.accessToken != null &&
    // Check if the access token inside the session is still valid
    (session.expiresAt ?? 0) > Math.floor(Date.now() / 1000);

  // Redirect unauthenticated users away from protected routes
  if (isProtectedRoute(pathname) && !isAuthenticated) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Redirect authenticated users away from public auth pages
  if (isPublicRoute(pathname) && isAuthenticated) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next (all Next.js internal routes)
     * - favicon.ico (favicon file)
     * - swagger (swagger UI)
     */
    "/((?!api|_next|favicon.ico|swagger).*)",
  ],
};
