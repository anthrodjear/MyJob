"use client";

import { Suspense, useState, useCallback, useEffect, type FormEvent, type ReactNode } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { resetPassword } from "@/lib/api/auth";
import { clearAuthStatusCookie, clearAuthTokens } from "@/lib/auth";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { AlertCircle, Eye, EyeOff, CheckCircle, Shield, ShieldAlert } from "lucide-react";

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

/**
 * Calculate password strength based on common criteria.
 * Returns a strength level from "weak" to "strong".
 */
function calculatePasswordStrength(password: string): PasswordStrength {
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

/** Map API error codes to user-friendly messages. */
function getUserMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("invalid_token") || msg.includes("invalid token")) {
      return "This reset link is invalid, expired, or has already been used.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Cannot reach the server. Is the backend running?";
    }
    if (msg.includes("timeout")) {
      return "Server took too long to respond. Please try again.";
    }
    if (msg.includes("401") || msg.includes("unauthorized")) {
      return "This reset link is invalid or expired.";
    }
  }
  return "Something went wrong. Please try again.";
}

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
          <p className="text-sm text-text-secondary">Loading…</p>
        </main>
      }
    >
      <ResetPasswordInner />
    </Suspense>
  );
}

function ResetPasswordInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [touched, setTouched] = useState(false);
  const [success, setSuccess] = useState(false);

  // Password strength (show after user has touched the field)
  const strength = calculatePasswordStrength(newPassword);
  const showStrength = touched;

  // Redirect to login if no token
  useEffect(() => {
    if (!token) {
      router.replace("/login");
    }
  }, [token, router]);

  const validate = useCallback((): string | null => {
    const pw = newPassword.trim();
    const confirm = confirmPassword.trim();

    if (pw.length === 0) return "Password is required.";
    if (pw.length < 8) return "Password must be at least 8 characters.";
    if (pw !== confirm) return "Passwords do not match.";

    return null;
  }, [newPassword, confirmPassword]);

  const handleSubmit = useCallback(
    async (e: FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      setError(null);

      if (!token) {
        setError("Missing or invalid reset token. Please request a new link.");
        return;
      }

      const validationError = validate();
      if (validationError != null) {
        setError(validationError);
        return;
      }

      setIsSubmitting(true);
      try {
        await resetPassword(token, newPassword.trim());
        // Clear any stale auth session so the user re-authenticates with the new password
        clearAuthTokens();
        clearAuthStatusCookie();
        setSuccess(true);
      } catch (err) {
        setError(getUserMessage(err));
      } finally {
        setIsSubmitting(false);
      }
    },
    [token, newPassword, validate]
  );

  // Show success state
  if (success) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
        <div className="w-full max-w-md space-y-6 px-4">
          {/* Brand */}
          <div className="text-center">
            <h1 className="text-3xl font-bold text-primary">MyJob</h1>
            <p className="mt-1 text-sm text-text-secondary">AI Job Search Agent</p>
          </div>

          {/* Success card */}
          <div className="rounded-lg border border-success bg-success/10 p-6 shadow-sm">
            <div className="flex items-center gap-3 text-success mb-4">
              <CheckCircle className="h-6 w-6 flex-shrink-0" aria-hidden="true" />
              <h2 className="text-xl font-semibold">Password Reset Successful</h2>
            </div>

            <p className="text-sm text-text-secondary mb-6">
              Your password has been updated. You can now sign in with your new password.
            </p>

            <Button
              variant="primary"
              size="lg"
              onClick={() => router.push("/login")}
              className="w-full"
            >
              Go to Login
            </Button>
          </div>

          <p className="text-center text-xs text-text-tertiary">
            This is a local-first application. Your data stays on your machine.
          </p>
        </div>
      </main>
    );
  }

  const strengthConfig = STRENGTH_CONFIG[strength];

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
      <div className="w-full max-w-md space-y-6 px-4">
        {/* Brand */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-1 text-sm text-text-secondary">AI Job Search Agent</p>
        </div>

        {/* Reset password card */}
        <div className="rounded-lg border border-border bg-bg-secondary p-6 shadow-sm">
          <h2 className="text-xl font-semibold text-text-primary">Set New Password</h2>
          <p className="mt-1 text-sm text-text-secondary">
            Enter a strong password below. Your reset token is valid for 1 hour.
          </p>

          {/* Error message */}
          {error != null && (
            <div
              role="alert"
              className="mb-4 flex items-start gap-2 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
            >
              <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            {/* New Password input with visibility toggle */}
            <Input
              id="newPassword"
              label="New Password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              required
              autoFocus
              value={newPassword}
              onChange={(e) => {
                setNewPassword(e.target.value);
                setTouched(true);
                setError(null);
              }}
              onBlur={() => setTouched(true)}
              disabled={isSubmitting}
              placeholder="Enter new password"
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="cursor-pointer text-text-tertiary hover:text-text-secondary focus:outline-none"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
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

            {/* Confirm Password input */}
            <Input
              id="confirmPassword"
              label="Confirm Password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              required
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value);
                setError(null);
              }}
              disabled={isSubmitting}
              placeholder="Confirm new password"
              error={confirmPassword && newPassword !== confirmPassword ? "Passwords do not match" : undefined}
            />

            {/* Submit button */}
            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={isSubmitting}
              loadingText="Resetting password…"
              className="w-full"
            >
              Reset Password
            </Button>
          </form>
        </div>

        {/* Back to login */}
        <p className="text-center text-sm text-text-secondary">
          <a href="/login" className="text-primary hover:text-primary/80 transition-colors">
            ← Back to Sign In
          </a>
        </p>

        {/* Help text */}
        <p className="text-center text-xs text-text-tertiary">
          This is a local-first application. Your data stays on your machine.
        </p>
      </div>
    </main>
  );
}