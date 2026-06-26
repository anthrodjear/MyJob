/**
 * Setup page — first-boot admin account creation.
 *
 * Client Component (needs form state, hooks, browser APIs).
 *
 * Flow:
 * 1. Page loads → already handled by login page redirect
 * 2. User fills form (username, email, password, confirm password)
 * 3. Client-side validation
 * 4. POST /auth/setup → backend creates user
 * 5. Redirect to /login
 *
 * Accessibility:
 * - `<main>` landmark
 * - `<h1>` page title
 * - `<form>` with proper labels
 * - Error announced via `role="alert"`
 * - Auto-focus on username input
 */

"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { completeSetup } from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";

/**
 * Validate the setup form client-side.
 * Returns an error message or null if valid.
 */
function validateForm(
  username: string,
  email: string,
  password: string,
  confirmPassword: string,
): string | null {
  if (username.trim().length < 3) {
    return "Username must be at least 3 characters.";
  }
  if (!email.includes("@") || !email.includes(".")) {
    return "Please enter a valid email address.";
  }
  if (password.length < 8) {
    return "Password must be at least 8 characters.";
  }
  if (password !== confirmPassword) {
    return "Passwords do not match.";
  }
  return null;
}

/**
 * Map API error codes to user-friendly messages.
 */
function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("setup_complete") || msg.includes("setup already completed")) {
      return "An admin account already exists. Please log in.";
    }
    if (msg.includes("invalid_request")) {
      return "Please check your input and try again.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
  }
  return "Something went wrong. Please try again.";
}

export default function SetupPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    const validationError = validateForm(username, email, password, confirmPassword);
    if (validationError != null) {
      setError(validationError);
      return;
    }

    setIsSubmitting(true);
    try {
      await completeSetup(username.trim(), email.trim(), password);
      router.push("/login");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-primary">
      <div className="w-full max-w-sm space-y-8 px-4">
        {/* Brand */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-2 text-sm text-text-secondary">
            First-Time Setup
          </p>
          <p className="mt-1 text-xs text-text-tertiary">
            Create your admin account to get started.
          </p>
        </div>

        {/* Setup form */}
        <form onSubmit={handleSubmit} className="space-y-5" noValidate>
          <div>
            <label
              htmlFor="username"
              className="block text-sm font-medium text-text-primary"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              autoComplete="username"
              required
              autoFocus
              minLength={3}
              maxLength={100}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isSubmitting}
              className="mt-1 block w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
              placeholder="Choose a username"
              aria-describedby={error ? "setup-error" : undefined}
            />
          </div>

          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-text-primary"
            >
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isSubmitting}
              className="mt-1 block w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
              placeholder="your@email.com"
              aria-describedby={error ? "setup-error" : undefined}
            />
          </div>

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
              autoComplete="new-password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isSubmitting}
              className="mt-1 block w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
              placeholder="Min. 8 characters"
              aria-describedby={error ? "setup-error" : undefined}
            />
          </div>

          <div>
            <label
              htmlFor="confirm-password"
              className="block text-sm font-medium text-text-primary"
            >
              Confirm Password
            </label>
            <input
              id="confirm-password"
              type="password"
              autoComplete="new-password"
              required
              minLength={8}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={isSubmitting}
              className="mt-1 block w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
              placeholder="Repeat your password"
              aria-describedby={error ? "setup-error" : undefined}
            />
          </div>

          {/* Error message */}
          {error != null && (
            <div
              id="setup-error"
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
            loading={isSubmitting}
            loadingText="Creating account…"
            className="w-full"
          >
            Create Admin Account
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
