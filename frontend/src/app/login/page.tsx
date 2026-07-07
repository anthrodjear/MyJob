/**
 * Login page — password-only authentication for single-user local app.
 *
 * Client Component (needs form state, hooks, browser APIs).
 *
 * Flow:
 * 1. User enters password
 * 2. POST /auth/login → { access_token, expires_at }
 * 3. Token stored in localStorage
 * 4. Redirect to /dashboard
 *
 * Accessibility:
 * - `<main>` landmark
 * - `<h1>` page title
 * - `<form>` with proper labels
 * - Error announced via `role="alert"`
 * - Auto-focus on password input
 * - Password visibility toggle with aria-label
 */

"use client";

import { useState, useEffect, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useLogin } from "@/hooks/useAuth";
import { getSetupStatus } from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { Eye, EyeOff, Lock, AlertCircle } from "lucide-react";

/**
 * Map API error codes to user-friendly messages.
 * Never expose raw error.message to users.
 */
function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("invalid_credentials") || msg.includes("invalid credentials")) {
      return "Incorrect password. Please try again.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
    if (msg.includes("timeout")) {
      return "Server took too long to respond. Please try again.";
    }
  }
  return "Something went wrong. Please try again.";
}

export default function LoginPage() {
  const router = useRouter();
  const loginMutation = useLogin();
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checkingSetup, setCheckingSetup] = useState(true);

  // Check setup status on mount — redirect to /setup if required
  useEffect(() => {
    let cancelled = false;

    async function checkSetup() {
      try {
        const status = await getSetupStatus();
        if (!cancelled && status.setup_required) {
          router.replace("/setup");
          return;
        }
      } catch {
        // If we can't reach the server, let the user try to log in
        // The backend will return 403 if setup is required
      } finally {
        if (!cancelled) {
          setCheckingSetup(false);
        }
      }
    }

    void checkSetup();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (password.trim().length === 0) {
      setError("Password is required.");
      return;
    }

    loginMutation.mutate(password, {
      onSuccess: () => {
        router.push("/dashboard");
      },
      onError: (err) => {
        setError(getUserMessage(err));
      },
    });
  };

  // Show loading spinner while checking setup status
  if (checkingSetup) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
        <div className="text-center">
          <svg
            className="mx-auto h-8 w-8 animate-spin text-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            />
          </svg>
          <p className="mt-3 text-sm text-text-secondary">Loading…</p>
        </div>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
      <div className="w-full max-w-sm space-y-6 px-4">
        {/* Brand */}
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary-light">
            <Lock className="h-6 w-6 text-primary" aria-hidden="true" />
          </div>
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-1 text-sm text-text-secondary">
            AI Job Search Agent
          </p>
        </div>

        {/* Login card */}
        <div className="rounded-lg border border-border bg-bg-secondary p-6 shadow-sm">
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            {/* Error message */}
            {error != null && (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
              >
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
                <span>{error}</span>
              </div>
            )}

            {/* Password input with visibility toggle */}
            <Input
              id="password"
              label="Password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              required
              autoFocus
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loginMutation.isPending}
              placeholder="Enter your password"
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="cursor-pointer text-text-tertiary hover:text-text-secondary focus:outline-none"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              }
            />

            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={loginMutation.isPending}
              loadingText="Signing in…"
              className="w-full"
            >
              Sign In
            </Button>
          </form>
        </div>

        {/* Help text */}
        <p className="text-center text-xs text-text-tertiary">
          This is a local-first application. Your data stays on your machine.
        </p>
      </div>
    </main>
  );
}
