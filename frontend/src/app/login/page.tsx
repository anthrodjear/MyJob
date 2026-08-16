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

import { Suspense, useState, useEffect, useCallback, type FormEvent, type ReactNode } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useLogin } from "@/hooks/useAuth";
import { getSetupStatus } from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import {
  Eye,
  EyeOff,
  Lock,
  AlertCircle,
  CheckCircle,
  Shield,
  ShieldAlert,
  Zap,
  Target,
  FileText,
  Briefcase,
} from "lucide-react";

/** Password strength levels. */
type PasswordStrength = "weak" | "fair" | "good" | "strong";

/** Ordered strength levels for iteration. */
const STRENGTH_LEVELS: PasswordStrength[] = ["weak", "fair", "good", "strong"];

/** Color and label for each strength level. */
const STRENGTH_CONFIG: Record<PasswordStrength, { color: string; label: string; icon: ReactNode }> = {
  weak: { color: "text-danger", label: "Weak", icon: <ShieldAlert className="h-3 w-3" /> },
  fair: { color: "text-warning", label: "Fair", icon: <Shield className="h-3 w-3" /> },
  good: { color: "text-primary", label: "Good", icon: <Shield className="h-3 w-3" /> },
  strong: { color: "text-success", label: "Strong", icon: <CheckCircle className="h-3 w-3" /> },
};

/** Key features to display on the brand side. */
const FEATURES = [
  { icon: <Zap className="h-5 w-5" />, text: "Auto-Apply to matching jobs" },
  { icon: <Target className="h-5 w-5" />, text: "Smart job matching & alerts" },
  { icon: <FileText className="h-5 w-5" />, text: "AI resume tailoring per role" },
  { icon: <Briefcase className="h-5 w-5" />, text: "Track applications in one place" },
];

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
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <p className="text-sm text-text-secondary">Loading…</p>
        </div>
      }
    >
      <LoginInner />
    </Suspense>
  );
}

/**
 * Sanitize a post-login redirect target.
 *
 * Only allow same-origin relative paths. Reject anything that:
 *   - does not start with "/" (could be an absolute URL or a scheme)
 *   - starts with "//" or "/\\" (protocol-relative — would be treated as
 *     an absolute URL by the browser and used for open-redirect attacks)
 *   - contains CR/LF (header-splitting guard)
 *   - is otherwise malformed
 *
 * Falls back to "/dashboard" when the value is not safe.
 */
function sanitizeRedirect(value: string | null | undefined): string {
  const fallback = "/dashboard";
  if (!value) return fallback;
  // Only accept simple relative paths starting with a single "/".
  // Reject protocol-relative ("//evil.com"), scheme-bearing values, and
  // anything containing control characters.
  if (
    !value.startsWith("/") ||
    value.startsWith("//") ||
    value.startsWith("/\\") ||
    /[\r\n]/.test(value)
  ) {
    return fallback;
  }
  return value;
}

function LoginInner() {
  const router = useRouter();
  const loginMutation = useLogin();
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checkingSetup, setCheckingSetup] = useState(true);
  const [touched, setTouched] = useState(false);

  // Get redirect URL from query params — sanitize to prevent open redirect.
  const searchParams = useSearchParams();
  const redirectUrl = sanitizeRedirect(searchParams.get("redirect"));

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
          // Use full page reload (not router.push) to ensure the session cookie
          // Set-Cookie header is fully processed before the proxy runs on the
          // next page. Soft navigation can race with cookie persistence.
          window.location.href = redirectUrl;
        },
        onError: (err) => {
          setError(getUserMessage(err));
        },
      });
    },
    [password, loginMutation, redirectUrl],
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
    <main className="flex min-h-screen bg-bg-secondary">
      {/* ── Brand / Hero Section ─────────────────────────────────────── */}
      <section
        className="relative hidden w-1/2 items-center justify-center overflow-hidden lg:flex"
        aria-hidden="true"
      >
        {/* Gradient background */}
        <div
          className="absolute inset-0"
          style={{
            background: "linear-gradient(135deg, #2563eb 0%, #7c3aed 100%)",
          }}
        />

        {/* Decorative floating shapes */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -left-20 -top-20 h-72 w-72 rounded-full bg-white/10" />
          <div className="absolute -bottom-32 -right-32 h-96 w-96 rounded-full bg-white/5" />
          <div className="absolute left-1/4 top-1/3 h-40 w-40 rounded-full bg-white/5" />
        </div>

        {/* Content */}
        <div className="relative z-10 max-w-md px-8">
          <h1 className="text-5xl font-extrabold tracking-tight text-white">
            MyJob
          </h1>
          <p className="mt-3 text-lg font-medium text-white/90">
            Your AI-Powered Job Search Assistant
          </p>

          <ul className="mt-10 space-y-4">
            {FEATURES.map((f) => (
              <li key={f.text} className="flex items-center gap-3 text-white/90">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white/15 backdrop-blur-sm">
                  {f.icon}
                </span>
                <span className="text-sm font-medium">{f.text}</span>
              </li>
            ))}
          </ul>

          <p className="mt-12 text-xs text-white/60">
            Local-first &middot; Your data stays on your machine
          </p>
        </div>
      </section>

      {/* ── Mobile brand header (visible < lg) ────────────────────────── */}
      <div
        className="flex w-full items-center justify-center px-6 pt-10 lg:hidden"
        style={{
          background: "linear-gradient(135deg, #2563eb 0%, #7c3aed 100%)",
        }}
      >
        <div className="text-center">
          <h1 className="text-3xl font-extrabold text-white">MyJob</h1>
          <p className="mt-1 text-sm text-white/80">
            AI-Powered Job Search Assistant
          </p>
        </div>
      </div>

      {/* ── Form Section ─────────────────────────────────────────────── */}
      <section className="flex w-full items-center justify-center bg-bg-secondary px-4 py-12 lg:w-1/2">
        <div className="animate-fade-in w-full max-w-sm space-y-6">
          {/* Mobile-only brand icon */}
          <div className="text-center lg:hidden">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary-light">
              <Lock className="h-6 w-6 text-primary" aria-hidden="true" />
            </div>
          </div>

          {/* Desktop heading */}
          <div className="hidden text-center lg:block">
            <h1 className="sr-only">Sign in to MyJob</h1>
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary-light">
              <Lock className="h-6 w-6 text-primary" aria-hidden="true" />
            </div>
            <h2 className="text-2xl font-bold text-text-primary">Welcome back</h2>
            <p className="mt-1 text-sm text-text-secondary">
              Sign in to your account
            </p>
          </div>

          {/* Login card */}
          <div className="rounded-xl border border-border bg-bg-secondary p-6 shadow-sm">
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
                    className="flex h-8 w-8 items-center justify-center rounded-md text-text-tertiary transition-colors hover:bg-bg-tertiary hover:text-text-secondary focus:outline-none focus:ring-2 focus:ring-primary/50"
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
                      <div
                        className="flex gap-1"
                        role="progressbar"
                        aria-valuenow={STRENGTH_LEVELS.indexOf(strength) + 1}
                        aria-valuemin={1}
                        aria-valuemax={4}
                      >
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

            {/* Forgot password & Sign up links */}
            <div className="flex justify-between pt-2 text-sm">
              <a
                href="/forgot-password"
                className="text-primary transition-colors hover:text-primary/80"
              >
                Forgot password?
              </a>
              <a
                href="/setup"
                className="text-primary transition-colors hover:text-primary/80"
              >
                Don&apos;t have an account? Sign up
              </a>
            </div>
          </div>

          {/* Help text */}
          <p className="text-center text-xs text-text-tertiary">
            This is a local-first application. Your data stays on your machine.
          </p>
        </div>
      </section>

    </main>
  );
}
