/**
 * EmailInterviewSection — email polling and interview memory configuration editor.
 *
 * Covers email provider, polling interval, folders, and interview memory/responder/planner settings.
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle Scoring/LLM/Voice settings (separate sections)
 * - Manage authentication state
 *
 * @see lib/types/config.ts — EmailSection, InterviewSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback } from "react";
import { useSetOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type {
  EmailSection as EmailSectionType,
  InterviewSection as InterviewSectionType,
} from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for EmailInterviewSection. */
interface EmailInterviewSectionProps {
  /** Current email config to populate the form. */
  email: EmailSectionType;
  /** Current interview config to populate the form. */
  interview: InterviewSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing email and interview configuration.
 *
 * Renders fields for email polling and interview memory settings.
 * Each field saves independently via PATCH.
 *
 * @example
 *   <EmailInterviewSection email={config.email} interview={config.interview} onSaved={handleSaved} />
 */
export function EmailInterviewSection({
  email,
  interview,
  onSaved,
}: EmailInterviewSectionProps) {
  const { mutateAsync } = useSetOverride();

  const [emailProvider, setEmailProvider] = useState(email.provider);
  const [checkInterval, setCheckInterval] = useState(email.check_interval);
  const [folders, setFolders] = useState(email.folders.join(", "));

  const [maxRecentSegments, setMaxRecentSegments] = useState(
    interview.memory.max_recent_segments.toString(),
  );
  const [keepAfterSummarize, setKeepAfterSummarize] = useState(
    interview.memory.keep_after_summarize.toString(),
  );
  const [llmTimeoutMs, setLlmTimeoutMs] = useState(
    interview.responder.llm.timeout_ms.toString(),
  );
  const [llmRetries, setLlmRetries] = useState(
    interview.responder.llm.retries.toString(),
  );
  const [duplicateThreshold, setDuplicateThreshold] = useState(
    interview.planner.duplicate_threshold.toString(),
  );
  const [minSubstantiveLength, setMinSubstantiveLength] = useState(
    interview.planner.min_substantive_length.toString(),
  );
  const [isSaving, setIsSaving] = useState(false);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);

      const overrides: Array<[string, unknown]> = [];

      // Email overrides
      if (emailProvider !== email.provider) {
        overrides.push(["email.provider", emailProvider]);
      }
      if (checkInterval !== email.check_interval) {
        overrides.push(["email.check_interval", checkInterval]);
      }
      const parsedFolders = folders
        .split(",")
        .map((f) => f.trim())
        .filter(Boolean);
      if (JSON.stringify(parsedFolders) !== JSON.stringify(email.folders)) {
        overrides.push(["email.folders", parsedFolders]);
      }

      // Interview memory overrides
      if (maxRecentSegments !== interview.memory.max_recent_segments.toString()) {
        overrides.push(["interview.memory.max_recent_segments", parseInt(maxRecentSegments, 10)]);
      }
      if (keepAfterSummarize !== interview.memory.keep_after_summarize.toString()) {
        overrides.push(["interview.memory.keep_after_summarize", parseInt(keepAfterSummarize, 10)]);
      }

      // Interview responder overrides
      if (llmTimeoutMs !== interview.responder.llm.timeout_ms.toString()) {
        overrides.push(["interview.responder.llm.timeout_ms", parseInt(llmTimeoutMs, 10)]);
      }
      if (llmRetries !== interview.responder.llm.retries.toString()) {
        overrides.push(["interview.responder.llm.retries", parseInt(llmRetries, 10)]);
      }

      // Interview planner overrides
      if (duplicateThreshold !== interview.planner.duplicate_threshold.toString()) {
        overrides.push(["interview.planner.duplicate_threshold", parseFloat(duplicateThreshold)]);
      }
      if (minSubstantiveLength !== interview.planner.min_substantive_length.toString()) {
        overrides.push(["interview.planner.min_substantive_length", parseInt(minSubstantiveLength, 10)]);
      }

      await executeOverrides(overrides, mutateAsync, onSaved);
      setIsSaving(false);
    },
    [
      emailProvider, checkInterval, folders,
      maxRecentSegments, keepAfterSummarize,
      llmTimeoutMs, llmRetries,
      duplicateThreshold, minSubstantiveLength,
      email, interview, mutateAsync, onSaved,
    ],
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
        Failed to save email/interview settings. Please try again.
      </div>

      {/* Email Settings */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Email Polling</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="email-provider" className="block text-sm font-medium text-text-primary">
              Provider
            </label>
            <input
              id="email-provider"
              type="text"
              value={emailProvider}
              onChange={(e) => setEmailProvider(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="check-interval" className="block text-sm font-medium text-text-primary">
              Check Interval
            </label>
            <input
              id="check-interval"
              type="text"
              value={checkInterval}
              onChange={(e) => setCheckInterval(e.target.value)}
              className={INPUT_CLASS}
            />
            <p className="mt-1 text-xs text-text-secondary">
              Duration string (e.g., 5m, 30s, 1h)
            </p>
          </div>
        </div>
        <div>
          <label htmlFor="email-folders" className="block text-sm font-medium text-text-primary">
            Folders
          </label>
          <input
            id="email-folders"
            type="text"
            value={folders}
            onChange={(e) => setFolders(e.target.value)}
            className={INPUT_CLASS}
          />
          <p className="mt-1 text-xs text-text-secondary">
            Comma-separated list (e.g., INBOX, Spam, Promotions)
          </p>
        </div>
      </fieldset>

      {/* Interview Memory */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Interview Memory</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="max-recent-segments" className="block text-sm font-medium text-text-primary">
              Max Recent Segments
            </label>
            <input
              id="max-recent-segments"
              type="number"
              min="1"
              max="100"
              value={maxRecentSegments}
              onChange={(e) => setMaxRecentSegments(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="keep-after-summarize" className="block text-sm font-medium text-text-primary">
              Keep After Summarize
            </label>
            <input
              id="keep-after-summarize"
              type="number"
              min="0"
              max="50"
              value={keepAfterSummarize}
              onChange={(e) => setKeepAfterSummarize(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      {/* Interview Responder */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Interview Responder</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="llm-timeout" className="block text-sm font-medium text-text-primary">
              LLM Timeout (ms)
            </label>
            <input
              id="llm-timeout"
              type="number"
              min="1000"
              max="120000"
              value={llmTimeoutMs}
              onChange={(e) => setLlmTimeoutMs(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="llm-retries" className="block text-sm font-medium text-text-primary">
              LLM Retries
            </label>
            <input
              id="llm-retries"
              type="number"
              min="0"
              max="10"
              value={llmRetries}
              onChange={(e) => setLlmRetries(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      {/* Interview Planner */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Interview Planner</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="duplicate-threshold" className="block text-sm font-medium text-text-primary">
              Duplicate Threshold
            </label>
            <input
              id="duplicate-threshold"
              type="number"
              step="0.1"
              min="0"
              max="1"
              value={duplicateThreshold}
              onChange={(e) => setDuplicateThreshold(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="min-substantive-length" className="block text-sm font-medium text-text-primary">
              Min Substantive Length
            </label>
            <input
              id="min-substantive-length"
              type="number"
              min="1"
              max="500"
              value={minSubstantiveLength}
              onChange={(e) => setMinSubstantiveLength(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="primary" disabled={isSaving} loading={isSaving}>
          Save Email &amp; Interview Settings
        </Button>
      </div>
    </form>
  );
}
