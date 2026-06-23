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
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    const token = getAuthToken();
    if (token == null) {
      router.replace("/login");
      return;
    }
    setChecked(true);
  }, [router]);

  // Show loading while checking auth
  if (!checked) {
    return (
      <div
        className="flex min-h-screen items-center justify-center bg-bg-primary"
        role="status"
        aria-label="Checking authentication"
      >
        <div className="text-sm text-text-secondary">Loading…</div>
      </div>
    );
  }

  return <>{children}</>;
}
