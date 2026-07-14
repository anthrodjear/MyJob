import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Middleware for route protection.
 *
 * Checks for auth_status cookie (set by client on login/logout).
 * Redirects unauthenticated users to /login for protected routes.
 * The AuthGuard component handles the actual JWT validation client-side.
 *
 * For full server-side validation, tokens would need to be in HTTP-only cookies.
 */

const PUBLIC_PATHS = ["/login", "/setup", "/api", "/_next", "/favicon.ico", "/swagger"];

function isPublicPath(path: string): boolean {
  return PUBLIC_PATHS.some((publicPath) => path.startsWith(publicPath));
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths
  if (isPublicPath(pathname)) {
    return NextResponse.next();
  }

  // For dashboard routes, check auth_status cookie
  const authStatus = request.cookies.get("auth_status")?.value;

  if (pathname.startsWith("/dashboard") && authStatus !== "authenticated") {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - swagger (swagger UI)
     */
    "/((?!api|_next/static|_next/image|favicon.ico|swagger).*)",
  ],
};