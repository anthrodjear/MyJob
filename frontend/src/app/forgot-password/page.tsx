"use client";

/**
 * ForgotPasswordPage — request a password reset token.
 *
 * Client Component (needs form state, hooks, browser APIs).
 *
 * Flow:
 * 1. User enters their email (the one used during setup)
 * 2. POST /auth/password/reset → { reset_token, message }
 * 3. Display the reset token for user to copy
 * 4. User clicks "Continue to Reset" → navigates to /reset-password with token
 *
 * Accessibility:
 * - <main> landmark
 * - <h1> page title
 * - <form> with proper labels
 * - Error announced via role="alert"
 * - Auto-focus on email input
 */

import { useState, useCallback, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { requestPasswordReset } from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { AlertCircle, Check } from "lucide-react";

/** Validate email format. */
function validateEmail(email: string): string | null {
  const trimmed = email.trim();
  if (trimmed.length === 0) return "Email is required.";
  // RFC 5322 simplified email validation
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(trimmed)) return "Please enter a valid email address.";
  return null;
}

export default function ForgotPasswordPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [token, setToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSubmit = useCallback(
    async (e: FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      setError(null);
      setToken(null);
      setCopied(false);

      const validationError = validateEmail(email);
      if (validationError != null) {
        setError(validationError);
        return;
      }

      setIsSubmitting(true);
      try {
        const data = await requestPasswordReset(email.trim());

        // Token received — display it (local-first: token returned in body)
        if (data.reset_token) {
          setToken(data.reset_token);
        } else {
          // Generic success path — no token returned (email mismatch, etc.)
          setError(null);
        }
      } catch {
        setError("Cannot reach the server. Is the backend running?");
      } finally {
        setIsSubmitting(false);
      }
    },
    [email]
  );

  const copyToken = useCallback(() => {
    if (!token) return;
    navigator.clipboard.writeText(token);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [token]);

  const continueToReset = useCallback(() => {
    if (!token) return;
    router.push(`/reset-password?token=${encodeURIComponent(token)}`);
  }, [router, token]);

  // Show success state if token received
  if (token) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
        <div className="w-full max-w-md space-y-6 px-4">
          {/* Brand */}
          <div className="text-center">
            <h1 className="text-3xl font-bold text-primary">MyJob</h1>
            <p className="mt-1 text-sm text-text-secondary">AI Job Search Agent</p>
          </div>

          {/* Success card */}
          <div className="rounded-lg border border-success bg-success-light p-6 shadow-sm">
            <div className="flex items-center gap-3 text-success-dark mb-4">
              <Check className="h-6 w-6 flex-shrink-0" aria-hidden="true" />
              <h2 className="text-xl font-semibold">Reset Token Generated</h2>
            </div>

            <p className="text-sm text-text-secondary mb-4">
              Copy the token below and use it on the next page to set your new password.
            </p>

            {/* Token display */}
            <div className="relative mb-4">
              <div className="flex gap-2">
                <input
                  type="text"
                  readOnly
                  value={token}
                  className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-text-primary font-mono select-all"
                  aria-label="Password reset token"
                />
                <button
                  type="button"
                  onClick={copyToken}
                  className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-2 text-sm font-medium text-text-primary hover:bg-bg-tertiary transition-colors"
                  aria-label={copied ? "Copied to clipboard" : "Copy token to clipboard"}
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 text-success" aria-hidden="true" />
                      <span>Copied</span>
                    </>
                  ) : (
                    <>
                      <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                      </svg>
                      <span>Copy</span>
                    </>
                  )}
                </button>
              </div>
              {copied && (
                <p className="mt-2 text-xs text-success" role="status" aria-live="polite">
                  Token copied to clipboard
                </p>
              )}
            </div>

            <p className="text-xs text-text-tertiary mb-4">
              Token expires in 1 hour. Keep it secure — anyone with this token can reset your password.
            </p>

            <Button
              variant="primary"
              size="lg"
              onClick={continueToReset}
              className="w-full"
            >
              Continue to Reset Password
            </Button>
          </div>

          {/* Help text */}
          <p className="text-center text-xs text-text-tertiary">
            This is a local-first application. Your data stays on your machine.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-secondary">
      <div className="w-full max-w-md space-y-6 px-4">
        {/* Brand */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-1 text-sm text-text-secondary">AI Job Search Agent</p>
        </div>

        {/* Forgot password card */}
        <div className="rounded-lg border border-border bg-bg-secondary p-6 shadow-sm">
          <h2 className="text-xl font-semibold text-text-primary">Forgot Password?</h2>
          <p className="mt-1 text-sm text-text-secondary">
            Enter your email to receive a password reset token.
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
            {/* Email input */}
            <Input
              id="email"
              label="Email"
              type="email"
              autoComplete="email"
              required
              autoFocus
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isSubmitting}
              placeholder="your@email.com"
              helperText="The email you used during initial setup"
              error={error != null ? error : undefined}
            />

            {/* Submit button */}
            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={isSubmitting}
              loadingText="Generating token…"
              className="w-full"
            >
              Generate Reset Token
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