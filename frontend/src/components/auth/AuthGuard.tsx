/**
 * AuthGuard — client-side route protection for /dashboard routes.
 *
 * Checks session cookie via /api/auth/session on mount. Redirects to /login
 * if not authenticated.
 *
 * Does NOT:
 * - Validate JWT signature (backend validates on every API call)
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

interface AuthGuardProps {
  children: ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter();
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function checkSession() {
      try {
        const res = await fetch("/api/auth/session", { credentials: "include" });
        if (cancelled) return;
        const data = (await res.json()) as { authenticated: boolean };
        if (data.authenticated) {
          setAuthenticated(true);
        } else {
          setAuthenticated(false);
          router.replace(`/login?redirect=${encodeURIComponent(window.location.pathname)}`);
        }
      } catch {
        if (!cancelled) {
          setAuthenticated(false);
          router.replace(`/login?redirect=${encodeURIComponent(window.location.pathname)}`);
        }
      }
    }

    void checkSession();

    // Re-check every 30 seconds in case session expires while page is open
    const interval = setInterval(checkSession, 30_000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [router]);

  // Show loading state until initial check completes
  if (authenticated === null) {
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

  // If not authenticated, redirect already happened — render nothing
  if (!authenticated) {
    return null;
  }

  return <>{children}</>;
}
