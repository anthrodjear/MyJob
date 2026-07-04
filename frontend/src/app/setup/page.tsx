"use client";

/**
 * Setup page — multi-step onboarding wizard.
 *
 * Client Component (needs form state, hooks, browser APIs).
 *
 * Flow:
 * 1. Account creation (username, email, password)
 * 2. LLM API keys (OpenAI, Anthropic) — optional
 * 3. Voice & Email (LiveKit, Microsoft 365) — skippable
 * 4. Preferences (scoring thresholds, job sources) — skippable
 * 5. Completion confirmation
 *
 * Accessibility:
 * - `<main>` landmark
 * - `<h1>` page title
 * - Step indicator with `aria-current="step"`
 * - Error announced via `role="alert"`
 * - Auto-focus on first input of each step
 */

import { useState, useCallback, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  completeSetup,
  login,
  saveOnboardingConfig,
  updateOnboardingStep,
  completeOnboarding,
} from "@/lib/api/auth";
import { Button } from "@/components/shared/Button";
import { SetupStepLLMKeys } from "@/components/setup/SetupStepLLMKeys";
import { SetupStepVoiceEmail } from "@/components/setup/SetupStepVoiceEmail";
import { SetupStepPreferences } from "@/components/setup/SetupStepPreferences";
import { SetupStepComplete } from "@/components/setup/SetupStepComplete";

/** Step identifiers for navigation and tracking. */
type StepId = "account" | "llm" | "voice" | "preferences" | "complete";

/** LLM keys from step 2. */
interface LLMKeys {
  openai: string;
  anthropic: string;
}

/** Voice/email config from step 3. */
interface VoiceEmailConfig {
  livekitUrl: string;
  livekitKey: string;
  livekitSecret: string;
  msTenantId: string;
  msClientId: string;
  msClientSecret: string;
}

/** Preferences from step 4. */
interface Preferences {
  autoThreshold: number;
  reviewThreshold: number;
  jobSources: string[];
}

/** Step metadata for rendering. */
const STEPS: { id: StepId; label: string; number: number }[] = [
  { id: "account", label: "Account", number: 1 },
  { id: "llm", label: "LLM Keys", number: 2 },
  { id: "voice", label: "Voice & Email", number: 3 },
  { id: "preferences", label: "Preferences", number: 4 },
  { id: "complete", label: "Complete", number: 5 },
];

/** Account form state. */
interface AccountForm {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

/**
 * Validate the account creation form.
 * Returns an error message or null if valid.
 */
function validateAccountForm(form: AccountForm): string | null {
  if (form.username.trim().length < 3) {
    return "Username must be at least 3 characters.";
  }
  // RFC 5322 simplified email validation
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(form.email)) {
    return "Please enter a valid email address.";
  }
  if (form.password.length < 8) {
    return "Password must be at least 8 characters.";
  }
  if (form.password !== form.confirmPassword) {
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

  // Current step state
  const [currentStep, setCurrentStep] = useState<StepId>("account");

  // Form data for each step
  const [account, setAccount] = useState<AccountForm>({
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
  });

  // Error state
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Step completion state
  const [completedSteps, setCompletedSteps] = useState<Set<StepId>>(
    new Set(),
  );

  // Get current step index
  const currentStepIndex = STEPS.findIndex((s) => s.id === currentStep);

  /**
   * Navigate to the next step.
   * Marks current step as completed.
   */
  const goToStep = useCallback((step: StepId) => {
    setError(null);
    setCurrentStep(step);
  }, []);

  /**
   * Handle account form submission (Step 1).
   * Creates user, logs in, and advances to LLM step.
   */
  const handleAccountSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    const validationError = validateAccountForm(account);
    if (validationError != null) {
      setError(validationError);
      return;
    }

    setIsSubmitting(true);
    try {
      await completeSetup(
        account.username.trim(),
        account.email.trim(),
        account.password,
      );
      // Auto-login after setup
      await login(account.password);
      setCompletedSteps((prev) => new Set(prev).add("account"));
      await updateOnboardingStep("llm");
      goToStep("llm");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  /**
   * Handle LLM keys step completion (Step 2).
   * Saves keys and advances to voice step.
   */
  const handleLLMComplete = async (keys: LLMKeys) => {
    setIsSubmitting(true);
    try {
      await saveOnboardingConfig({
        openai_key: keys.openai || undefined,
        anthropic_key: keys.anthropic || undefined,
      });
      setCompletedSteps((prev) => new Set(prev).add("llm"));
      await updateOnboardingStep("voice");
      goToStep("voice");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  /**
   * Handle voice/email step completion (Step 3).
   * Saves config and advances to preferences step.
   */
  const handleVoiceEmailComplete = async (config: VoiceEmailConfig) => {
    setIsSubmitting(true);
    try {
      await saveOnboardingConfig({
        livekit_url: config.livekitUrl || undefined,
        livekit_key: config.livekitKey || undefined,
        livekit_secret: config.livekitSecret || undefined,
        ms_tenant_id: config.msTenantId || undefined,
        ms_client_id: config.msClientId || undefined,
        ms_client_secret: config.msClientSecret || undefined,
      });
      setCompletedSteps((prev) => new Set(prev).add("voice"));
      await updateOnboardingStep("preferences");
      goToStep("preferences");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  /**
   * Handle preferences step completion (Step 4).
   * Saves preferences and advances to complete step.
   */
  const handlePreferencesComplete = async (prefs: Preferences) => {
    setIsSubmitting(true);
    try {
      await saveOnboardingConfig({
        auto_threshold: prefs.autoThreshold,
        review_threshold: prefs.reviewThreshold,
        job_sources: prefs.jobSources,
      });
      setCompletedSteps((prev) => new Set(prev).add("preferences"));
      await updateOnboardingStep("complete");
      goToStep("complete");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  /**
   * Handle final completion (Step 5).
   * Marks onboarding complete and redirects to dashboard.
   */
  const handleOnboardingComplete = async () => {
    setIsSubmitting(true);
    try {
      await completeOnboarding();
      router.push("/dashboard");
    } catch (err) {
      setError(getUserMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  /**
   * Handle skip for voice/email and preferences steps.
   */
  const handleSkip = useCallback(async () => {
    try {
      if (currentStep === "voice") {
        setCompletedSteps((prev) => new Set(prev).add("voice"));
        await updateOnboardingStep("preferences");
        goToStep("preferences");
      } else if (currentStep === "preferences") {
        setCompletedSteps((prev) => new Set(prev).add("preferences"));
        await updateOnboardingStep("complete");
        goToStep("complete");
      }
    } catch (err) {
      setError(getUserMessage(err));
    }
  }, [currentStep, goToStep]);

  /**
   * Handle back navigation.
   */
  const handleBack = useCallback(() => {
    const prevStep = STEPS[currentStepIndex - 1];
    if (prevStep) {
      goToStep(prevStep.id);
    }
  }, [currentStepIndex, goToStep]);

  /**
   * Render step indicator (progress bar).
   */
  const renderStepIndicator = () => (
    <nav aria-label="Setup progress" className="mb-8">
      <ol className="flex items-center justify-center gap-2">
        {STEPS.map((step, index) => (
          <li key={step.id} className="flex items-center">
            <div
              className={cn(
                "flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium",
                completedSteps.has(step.id) &&
                  "bg-success text-text-inverse",
                currentStep === step.id &&
                  "bg-primary text-text-inverse",
                !completedSteps.has(step.id) &&
                  currentStep !== step.id &&
                  "bg-bg-tertiary text-text-secondary",
              )}
              aria-current={currentStep === step.id ? "step" : undefined}
            >
              {completedSteps.has(step.id) ? (
                <svg
                  className="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              ) : (
                step.number
              )}
            </div>
            <span
              className={cn(
                "ml-2 text-sm",
                currentStep === step.id
                  ? "font-medium text-text-primary"
                  : "text-text-tertiary",
              )}
            >
              {step.label}
            </span>
            {index < STEPS.length - 1 && (
              <div
                className={cn(
                  "mx-2 h-0.5 w-8",
                  completedSteps.has(step.id)
                    ? "bg-success"
                    : "bg-bg-tertiary",
                )}
                aria-hidden="true"
              />
            )}
          </li>
        ))}
      </ol>
    </nav>
  );

  /**
   * Render current step content.
   */
  const renderStep = () => {
    switch (currentStep) {
      case "account":
        return (
          <form onSubmit={handleAccountSubmit} className="space-y-5" noValidate>
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
                value={account.username}
                onChange={(e) =>
                  setAccount((prev) => ({ ...prev, username: e.target.value }))
                }
                disabled={isSubmitting}
                className="mt-1 block w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
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
                value={account.email}
                onChange={(e) =>
                  setAccount((prev) => ({ ...prev, email: e.target.value }))
                }
                disabled={isSubmitting}
                className="mt-1 block w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
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
                value={account.password}
                onChange={(e) =>
                  setAccount((prev) => ({ ...prev, password: e.target.value }))
                }
                disabled={isSubmitting}
                className="mt-1 block w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
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
                value={account.confirmPassword}
                onChange={(e) =>
                  setAccount((prev) => ({
                    ...prev,
                    confirmPassword: e.target.value,
                  }))
                }
                disabled={isSubmitting}
                className="mt-1 block w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
                placeholder="Repeat your password"
                aria-describedby={error ? "setup-error" : undefined}
              />
            </div>

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
              Create Account &amp; Continue
            </Button>
          </form>
        );

      case "llm":
        return (
          <div>
            {error != null && (
              <div
                role="alert"
                className="mb-4 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
              >
                {error}
              </div>
            )}
            <SetupStepLLMKeys onNext={handleLLMComplete} onBack={handleBack} />
          </div>
        );

      case "voice":
        return (
          <div>
            {error != null && (
              <div
                role="alert"
                className="mb-4 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
              >
                {error}
              </div>
            )}
            <SetupStepVoiceEmail
              onNext={handleVoiceEmailComplete}
              onBack={handleBack}
              onSkip={handleSkip}
            />
          </div>
        );

      case "preferences":
        return (
          <div>
            {error != null && (
              <div
                role="alert"
                className="mb-4 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
              >
                {error}
              </div>
            )}
            <SetupStepPreferences
              onNext={handlePreferencesComplete}
              onBack={handleBack}
              onSkip={handleSkip}
            />
          </div>
        );

      case "complete":
        return (
          <div>
            {error != null && (
              <div
                role="alert"
                className="mb-4 rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark"
              >
                {error}
              </div>
            )}
            <SetupStepComplete onComplete={handleOnboardingComplete} />
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-bg-primary">
      <div className="w-full max-w-lg space-y-6 px-4">
        {/* Brand */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-primary">MyJob</h1>
          <p className="mt-2 text-sm text-text-secondary">
            {currentStep === "complete"
              ? "Setup Complete"
              : "First-Time Setup"}
          </p>
        </div>

        {/* Step indicator */}
        {renderStepIndicator()}

        {/* Step content */}
        <div className="rounded-lg border border-border bg-bg-secondary p-6 shadow-sm">
          {renderStep()}
        </div>

        {/* Help text */}
        <p className="text-center text-xs text-text-tertiary">
          This is a local-first application. Your data stays on your machine.
        </p>
      </div>
    </main>
  );
}
