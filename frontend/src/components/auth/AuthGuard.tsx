/**
 * AuthGuard — client-side route protection for /dashboard routes.
 *
 * Checks localStorage for JWT on mount. Redirects to /login if absent or expired.
 * Uses React state + useEffect (not middleware) because tokens are in localStorage.
 *
 * Does NOT:
 * - Validate JWT signature (backend validates on every API call)
 * - Refresh tokens (backend issues long-lived JWTs)
 * - Handle server-side auth (this is client-only protection)
 *
 * Accessibility:
 * - Shows loading spinner while checking auth
 * - Announces loading state to screen readers
 *
 * @example
 *   // In dashboard layout:
 *   <AuthGuard>{children}</AuthGuard>
 */

"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { getAuthToken, isTokenExpired } from "@/lib/auth";

interface AuthGuardProps {
  children: ReactNode;
}

// Synchronous token check for initial render — runs before first paint
function getInitialAuthState(): { token: string | null; shouldRedirect: boolean } {
  if (typeof window === "undefined") {
    return { token: null, shouldRedirect: false };
  }
  const token = getAuthToken();
  const expired = isTokenExpired();
  return { token, shouldRedirect: token === null || expired };
}

export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(() => getInitialAuthState().token);
  const [checked, setChecked] = useState(false);

  // Check token on mount and periodically
  useEffect(() => {
    const checkAuth = () => {
      const currentToken = getAuthToken();
      const expired = isTokenExpired();

      if (currentToken === null || expired) {
        setToken(currentToken);
        router.replace(`/login?redirect=${encodeURIComponent(window.location.pathname)}`);
      } else {
        setToken(currentToken);
      }
      setChecked(true);
    };

    // Initial check (may be redundant with initial state, but handles SSR edge cases)
    checkAuth();

    // Re-check every 30 seconds in case token expires while page is open
    const interval = setInterval(checkAuth, 30_000);
    return () => clearInterval(interval);
  }, [router]);

  // Show loading state until initial check completes
  if (!checked) {
    return (
      <div
        className="flex min-h-screen items-center justify-center bg-bg-secondary"
        role="status"
        aria-label="Checking authentication"
      >
        <div className="text-sm text-text-secondary">Loading…</div>
      </div>
    );
  }

  // If token is null after check, redirect already happened — render nothing
  if (token === null) {
    return null;
  }

  return <>{children}</>;
}
