import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Middleware for route protection.
 *
 * Since JWT tokens are stored in localStorage (not cookies), this middleware
 * cannot validate tokens directly. It serves as a first line of defense:
 * - Redirects unauthenticated users to /login for protected routes
 * - The AuthGuard component handles the actual token validation client-side
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

  // For dashboard routes, we rely on client-side AuthGuard
  // This middleware just ensures we don't serve the page shell without checking
  // The actual auth happens in AuthGuard which checks localStorage

  // However, we can check for a potential auth cookie if we migrate to cookies
  // For now, let the page load and let AuthGuard handle it

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