/**
 * Login page — password-only authentication for single-user local app.
 *
 * Client Component (needs form state, hooks, browser APIs).
 *
 * Flow:
 * 1. User enters password
 * 2. POST /auth/login → { access_token, refresh_token, expires_at }
 * 3. Tokens stored in localStorage (access + refresh)
 * 4. Redirect to /dashboard
 *
 * UX improvements:
 * - Password strength indicator
 * - Clear error messages
 * - Loading states with accessible feedback
 * - Auto-focus on password input
 * - Password visibility toggle with aria-label
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

import { useState, useEffect, useCallback, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useLogin } from "@/hooks/useAuth";
import { getSetupStatus } from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { Eye, EyeOff, Lock, AlertCircle, CheckCircle, Shield, ShieldAlert } from "lucide-react";

/** Password strength levels. */
type PasswordStrength = "weak" | "fair" | "good" | "strong";

/** Ordered strength levels for iteration. */
const STRENGTH_LEVELS: PasswordStrength[] = ["weak", "fair", "good", "strong"];

/** Color and label for each strength level. */
const STRENGTH_CONFIG: Record<PasswordStrength, { color: string; label: string; icon: React.ReactNode }> = {
  weak: { color: "text-danger", label: "Weak", icon: <ShieldAlert className="h-3 w-3" /> },
  fair: { color: "text-warning", label: "Fair", icon: <Shield className="h-3 w-3" /> },
  good: { color: "text-primary", label: "Good", icon: <Shield className="h-3 w-3" /> },
  strong: { color: "text-success", label: "Strong", icon: <CheckCircle className="h-3 w-3" /> },
};

/**
 * Calculate password strength based on common criteria.
 * Returns a strength level from "weak" to "strong".
 */
function calculatePasswordStrength(password: string): PasswordStrength {
  // Trim whitespace for consistent scoring
  const trimmed = password.trim();
  if (trimmed.length === 0) return "weak";

  let score = 0;

  // Length
  if (trimmed.length >= 8) score++;
  if (trimmed.length >= 12) score++;
  if (trimmed.length >= 16) score++;

  // Character variety
  if (/[a-z]/.test(trimmed)) score++;
  if (/[A-Z]/.test(trimmed)) score++;
  if (/[0-9]/.test(trimmed)) score++;
  if (/[^a-zA-Z0-9]/.test(trimmed)) score++;

  // Penalize common patterns
  if (/(.)\1{2,}/.test(trimmed)) score--; // Repeated characters
  if (/^(?:password|admin|123456|qwerty|letmein)/i.test(trimmed)) score -= 2;

  if (score <= 2) return "weak";
  if (score <= 4) return "fair";
  if (score <= 5) return "good";
  return "strong";
}

/**
 * Map API error codes to user-friendly messages.
 * Never expose raw error messages to users.
 */
function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("invalid_credentials") || msg.includes("invalid credentials")) {
      return "Incorrect password. Please try again.";
    }
    if (msg.includes("invalid_refresh_token") || msg.includes("refresh token")) {
      return "Your session has expired. Please sign in again.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
    if (msg.includes("timeout")) {
      return "Server took too long to respond. Please try again.";
    }
    if (msg.includes("401") || msg.includes("unauthorized")) {
      return "Session expired. Please sign in again.";
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
  const [touched, setTouched] = useState(false);

  // Get redirect URL from query params
  const searchParams = useSearchParams();
  const redirectUrl = searchParams.get("redirect") || "/dashboard";

  // Get redirect param from URL
  const searchParams = new URLSearchParams(window.location.search);
  const redirectPath = searchParams.get("redirect") || "/dashboard";

  // Password strength (show after user has touched the field)
  const strength = calculatePasswordStrength(password);
  const showStrength = touched;

  // Check setup status on mount — redirect to /setup if required
  useEffect(() => {
    let cancelled = false;
    let redirectCount = 0;
    const MAX_REDIRECTS = 3;

    async function checkSetup() {
      if (redirectCount >= MAX_REDIRECTS) {
        // Too many redirects - stop to prevent loop
        if (!cancelled) setCheckingSetup(false);
        return;
      }

      try {
        const status = await getSetupStatus();
        if (!cancelled && status.setup_required) {
          // Setup required — redirect to setup page
          redirectCount++;
          router.replace("/setup");
          return;
        }
        // If setup not required but onboarding not complete, also go to setup
        if (!cancelled && !status.setup_required && !status.onboarding_completed) {
          redirectCount++;
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

  const handleSubmit = useCallback(
    (e: FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      setError(null);
      setTouched(true);

      const trimmedPassword = password.trim();
      if (trimmedPassword.length === 0) {
        setError("Password is required.");
        return;
      }

      // Send trimmed password to backend (consistent with strength calculation)
loginMutation.mutate(trimmedPassword, {
        onSuccess: () => {
          router.push(redirectUrl);
        },
        onError: (err) => {
          setError(getUserMessage(err));
        },
      });
    },
    [password, loginMutation, router, redirectPath],
  );

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

  const strengthConfig = STRENGTH_CONFIG[strength];

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
      <div className="w-full max-w-sm space-y-6 px-4">
        {/* Brand */}
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary-light">
            <Lock className="h-6 w-6 text-primary" aria-hidden="true" />
          </div>
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-1 text-sm text-text-secondary">AI Job Search Agent</p>
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
              onBlur={() => setTouched(true)}
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
              helperText={
                showStrength && (
                  <div className="mt-2 space-y-1.5">
                    {/* Strength bar */}
                    <div className="flex gap-1" role="progressbar" aria-valuenow={STRENGTH_LEVELS.indexOf(strength) + 1} aria-valuemin={1} aria-valuemax={4}>
                      {STRENGTH_LEVELS.map((level, index) => (
                        <div
                          key={level}
                          className={`h-1.5 flex-1 rounded transition-colors ${
                            index <= STRENGTH_LEVELS.indexOf(strength)
                              ? STRENGTH_CONFIG[level].color.replace("text-", "bg-")
                              : "bg-border"
                          }`}
                        />
                      ))}
                    </div>
                    {/* Strength label — sole source of screen reader feedback */}
                    <p className={`text-xs font-medium ${strengthConfig.color}`} aria-live="polite">
                      {strengthConfig.icon}
                      Password strength: {strengthConfig.label}
                    </p>
                  </div>
                )
              }
            />

            {/* Sign in button */}
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