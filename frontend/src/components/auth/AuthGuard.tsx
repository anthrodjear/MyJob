/**
 * AuthGuard — client-side route protection for /dashboard routes.
 *
 * Checks localStorage for JWT on mount. Redirects to /login if absent.
 * Uses React state + useEffect (not middleware) because tokens are in localStorage.
 *
 * Does NOT:
 * - Validate JWT expiry (backend rejects expired tokens on every API call)
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
import { getAuthToken } from "@/lib/auth";

interface AuthGuardProps {
  children: ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter();
  const [token, setToken] = useState<string | null | undefined>(undefined);
  const [authChecked, setAuthChecked] = useState(false);

  useEffect(() => {
    setToken(getAuthToken());
    setAuthChecked(true);
  }, []);

  // Redirect to login only after auth has been checked.
  useEffect(() => {
    if (authChecked && token === null) {
      router.replace("/login");
    }
  }, [authChecked, token, router]);

  if (!authChecked || token === null) {
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

  return <>{children}</>;
}
