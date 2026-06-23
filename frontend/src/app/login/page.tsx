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
 */

"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useLogin } from "@/hooks/useAuth";
import { Button } from "@/components/shared/Button";

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
  const [error, setError] = useState<string | null>(null);

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

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-primary">
      <div className="w-full max-w-sm space-y-8 px-4">
        {/* Brand */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-2 text-sm text-text-secondary">
            AI Job Search Agent
          </p>
        </div>

        {/* Login form */}
        <form onSubmit={handleSubmit} className="space-y-6" noValidate>
          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-text-primary"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              autoFocus
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loginMutation.isPending}
              className="mt-1 block w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
              placeholder="Enter your password"
              aria-describedby={error ? "login-error" : undefined}
            />
          </div>

          {/* Error message */}
          {error != null && (
            <div
              id="login-error"
              role="alert"
              className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
            >
              {error}
            </div>
          )}

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

        {/* Help text */}
        <p className="text-center text-xs text-text-tertiary">
          This is a local-first application. Your data stays on your machine.
        </p>
      </div>
    </main>
  );
}
