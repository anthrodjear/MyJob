/**
 * AutomationSection — queue and auto-generation configuration editor.
 *
 * Covers queue concurrency/retry settings and auto-generation toggles for
 * resumes and cover letters.
 *
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle Scoring/LLM/Email settings (separate sections)
 * - Manage authentication state
 *
 * @see lib/types/config.ts — AutomationSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSetOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type { AutomationSection as AutomationSectionType } from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for AutomationSection. */
interface AutomationSectionProps {
  /** Current automation config to populate the form. */
  automation: AutomationSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing automation configuration.
 *
 * Renders fields for queue settings and auto-generation toggles.
 * Each field saves independently via PATCH.
 *
 * @example
 *   <AutomationSection automation={config.automation} onSaved={() => console.log("saved")} />
 */
export function AutomationSection({ automation, onSaved }: AutomationSectionProps) {
  const { mutateAsync } = useSetOverride();

  const [concurrency, setConcurrency] = useState(automation.queue.concurrency.toString());
  const [retryAttempts, setRetryAttempts] = useState(automation.queue.retry_attempts.toString());
  const [autoResume, setAutoResume] = useState(automation.auto_generate.resume);
  const [autoCoverLetter, setAutoCoverLetter] = useState(automation.auto_generate.cover_letter);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => setError(null), []);

  // Sync state when props change (skip initial mount)
  const isInitialMount = useRef(true);
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    setConcurrency(automation.queue.concurrency.toString());
    setRetryAttempts(automation.queue.retry_attempts.toString());
    setAutoResume(automation.auto_generate.resume);
    setAutoCoverLetter(automation.auto_generate.cover_letter);
  }, [automation]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);
      setError(null);

      const overrides: Array<[string, unknown]> = [];

      if (concurrency !== automation.queue.concurrency.toString()) {
        const parsed = parseInt(concurrency, 10);
        if (Number.isNaN(parsed) || parsed < 1 || parsed > 20) {
          setError("Concurrency must be a valid number between 1 and 20.");
          setIsSaving(false);
          return;
        }
        overrides.push(["automation.queue.concurrency", parsed]);
      }
      if (retryAttempts !== automation.queue.retry_attempts.toString()) {
        const parsed = parseInt(retryAttempts, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 10) {
          setError("Retry Attempts must be a valid number between 0 and 10.");
          setIsSaving(false);
          return;
        }
        overrides.push(["automation.queue.retry_attempts", parsed]);
      }
      if (autoResume !== automation.auto_generate.resume) {
        overrides.push(["automation.auto_generate.resume", autoResume]);
      }
      if (autoCoverLetter !== automation.auto_generate.cover_letter) {
        overrides.push(["automation.auto_generate.cover_letter", autoCoverLetter]);
      }

      try {
        const result = await executeOverrides(overrides, mutateAsync, onSaved);
        if (result.failed > 0) {
          setError(
            result.failed === result.total
              ? "Failed to save automation settings. Please try again."
              : `Partially saved: ${result.succeeded} of ${result.total} settings saved. ${result.failed} failed.`,
          );
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to save automation settings. Please try again.",
        );
      } finally {
        setIsSaving(false);
      }
    },
    [concurrency, retryAttempts, autoResume, autoCoverLetter, automation, mutateAsync, onSaved],
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
          {error}
        </div>
      )}

      {/* Queue Settings */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Queue Settings</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="concurrency" className="block text-sm font-medium text-text-primary">
              Concurrency
            </label>
            <input
              id="concurrency"
              type="number"
              min="1"
              max="20"
              value={concurrency}
              onChange={(e) => {
                setConcurrency(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
            <p className="mt-1 text-xs text-text-secondary">
              Number of concurrent task workers
            </p>
          </div>
          <div>
            <label htmlFor="retry-attempts" className="block text-sm font-medium text-text-primary">
              Retry Attempts
            </label>
            <input
              id="retry-attempts"
              type="number"
              min="0"
              max="10"
              value={retryAttempts}
              onChange={(e) => {
                setRetryAttempts(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
            <p className="mt-1 text-xs text-text-secondary">
              Maximum retry attempts for failed tasks
            </p>
          </div>
        </div>
      </fieldset>

      {/* Auto-Generation Toggles */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Auto-Generation</legend>
        <div className="space-y-3">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={autoResume}
              onChange={(e) => {
                setAutoResume(e.target.checked);
                clearError();
              }}
              className="rounded border-border text-primary focus:ring-primary"
            />
            <span className="text-sm text-text-primary">Auto-generate resumes for new applications</span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={autoCoverLetter}
              onChange={(e) => {
                setAutoCoverLetter(e.target.checked);
                clearError();
              }}
              className="rounded border-border text-primary focus:ring-primary"
            />
            <span className="text-sm text-text-primary">Auto-generate cover letters for new applications</span>
          </label>
        </div>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="primary" disabled={isSaving} loading={isSaving}>
          Save Automation Settings
        </Button>
      </div>
    </form>
  );
}