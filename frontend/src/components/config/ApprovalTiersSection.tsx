/**
 * ApprovalTiersSection — auto/review/reject tier configuration editor.
 *
 * Covers tier definitions with min/max scores, actions, and notification flags.
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle Scoring/LLM/Email settings (separate sections)
 * - Manage authentication state
 *
 * @see lib/types/config.ts — ApprovalTiersSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSetOverride, useDeleteOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type { ApprovalTiersSection as ApprovalTiersSectionType } from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for ApprovalTiersSection. */
interface ApprovalTiersSectionProps {
  /** Current approval tiers config to populate the form. */
  approvalTiers: ApprovalTiersSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing approval tier configuration.
 *
 * Renders fields for auto-apply, review, and reject tiers.
 * Each field saves independently via PATCH.
 *
 * @example
 *   <ApprovalTiersSection approvalTiers={config.approval_tiers} onSaved={() => console.log("saved")} />
 */
export function ApprovalTiersSection({ approvalTiers, onSaved }: ApprovalTiersSectionProps) {
  const { mutateAsync } = useSetOverride();
  const deleteOverride = useDeleteOverride();

  const [autoMin, setAutoMin] = useState(approvalTiers.auto_apply.min_score.toString());
  const [autoMax, setAutoMax] = useState(approvalTiers.auto_apply.max_score?.toString() ?? "");
  const [autoAction, setAutoAction] = useState(approvalTiers.auto_apply.action);
  const [autoNotify, setAutoNotify] = useState(approvalTiers.auto_apply.notify ?? false);

  const [reviewMin, setReviewMin] = useState(approvalTiers.review.min_score.toString());
  const [reviewMax, setReviewMax] = useState(approvalTiers.review.max_score?.toString() ?? "");
  const [reviewAction, setReviewAction] = useState(approvalTiers.review.action);
  const [reviewNotify, setReviewNotify] = useState(approvalTiers.review.notify ?? false);

  const [rejectMin, setRejectMin] = useState(approvalTiers.reject.min_score.toString());
  const [rejectMax, setRejectMax] = useState(approvalTiers.reject.max_score?.toString() ?? "");
  const [rejectAction, setRejectAction] = useState(approvalTiers.reject.action);
  const [rejectNotify, setRejectNotify] = useState(approvalTiers.reject.notify ?? false);
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
    setAutoMin(approvalTiers.auto_apply.min_score.toString());
    setAutoMax(approvalTiers.auto_apply.max_score?.toString() ?? "");
    setAutoAction(approvalTiers.auto_apply.action);
    setAutoNotify(approvalTiers.auto_apply.notify ?? false);

    setReviewMin(approvalTiers.review.min_score.toString());
    setReviewMax(approvalTiers.review.max_score?.toString() ?? "");
    setReviewAction(approvalTiers.review.action);
    setReviewNotify(approvalTiers.review.notify ?? false);

    setRejectMin(approvalTiers.reject.min_score.toString());
    setRejectMax(approvalTiers.reject.max_score?.toString() ?? "");
    setRejectAction(approvalTiers.reject.action);
    setRejectNotify(approvalTiers.reject.notify ?? false);
  }, [approvalTiers]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);
      setError(null);

      const overrides: Array<[string, unknown]> = [];

      // Auto-apply tier
      if (autoMin !== approvalTiers.auto_apply.min_score.toString()) {
        const parsed = parseInt(autoMin, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
          setError("Auto-apply min score must be a valid number between 0 and 100.");
          setIsSaving(false);
          return;
        }
        overrides.push(["approval_tiers.auto_apply.min_score", parsed]);
      }
      if (autoMax !== (approvalTiers.auto_apply.max_score?.toString() ?? "")) {
        if (autoMax !== "") {
          const parsed = parseInt(autoMax, 10);
          if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
            setError("Auto-apply max score must be a valid number between 0 and 100.");
            setIsSaving(false);
            return;
          }
          overrides.push(["approval_tiers.auto_apply.max_score", parsed]);
        } else {
          // Field cleared — delete the override to revert to YAML default
          await deleteOverride.mutateAsync("approval_tiers.auto_apply.max_score");
        }
      }
      if (autoAction !== approvalTiers.auto_apply.action) {
        overrides.push(["approval_tiers.auto_apply.action", autoAction]);
      }
      if (autoNotify !== (approvalTiers.auto_apply.notify ?? false)) {
        overrides.push(["approval_tiers.auto_apply.notify", autoNotify]);
      }

      // Review tier
      if (reviewMin !== approvalTiers.review.min_score.toString()) {
        const parsed = parseInt(reviewMin, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
          setError("Review min score must be a valid number between 0 and 100.");
          setIsSaving(false);
          return;
        }
        overrides.push(["approval_tiers.review.min_score", parsed]);
      }
      if (reviewMax !== (approvalTiers.review.max_score?.toString() ?? "")) {
        if (reviewMax !== "") {
          const parsed = parseInt(reviewMax, 10);
          if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
            setError("Review max score must be a valid number between 0 and 100.");
            setIsSaving(false);
            return;
          }
          overrides.push(["approval_tiers.review.max_score", parsed]);
        } else {
          // Field cleared — delete the override to revert to YAML default
          await deleteOverride.mutateAsync("approval_tiers.review.max_score");
        }
      }
      if (reviewAction !== approvalTiers.review.action) {
        overrides.push(["approval_tiers.review.action", reviewAction]);
      }
      if (reviewNotify !== (approvalTiers.review.notify ?? false)) {
        overrides.push(["approval_tiers.review.notify", reviewNotify]);
      }

      // Reject tier
      if (rejectMin !== approvalTiers.reject.min_score.toString()) {
        const parsed = parseInt(rejectMin, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
          setError("Reject min score must be a valid number between 0 and 100.");
          setIsSaving(false);
          return;
        }
        overrides.push(["approval_tiers.reject.min_score", parsed]);
      }
      if (rejectMax !== (approvalTiers.reject.max_score?.toString() ?? "")) {
        if (rejectMax !== "") {
          const parsed = parseInt(rejectMax, 10);
          if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
            setError("Reject max score must be a valid number between 0 and 100.");
            setIsSaving(false);
            return;
          }
          overrides.push(["approval_tiers.reject.max_score", parsed]);
        } else {
          // Field cleared — delete the override to revert to YAML default
          await deleteOverride.mutateAsync("approval_tiers.reject.max_score");
        }
      }
      if (rejectAction !== approvalTiers.reject.action) {
        overrides.push(["approval_tiers.reject.action", rejectAction]);
      }
      if (rejectNotify !== (approvalTiers.reject.notify ?? false)) {
        overrides.push(["approval_tiers.reject.notify", rejectNotify]);
      }

      try {
        const result = await executeOverrides(overrides, mutateAsync, onSaved);
        if (result.failed > 0) {
          setError(
            result.failed === result.total
              ? "Failed to save approval tier settings. Please try again."
              : `Partially saved: ${result.succeeded} of ${result.total} settings saved. ${result.failed} failed.`,
          );
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to save approval tier settings. Please try again.",
        );
      } finally {
        setIsSaving(false);
      }
    },
    [
      autoMin, autoMax, autoAction, autoNotify,
      reviewMin, reviewMax, reviewAction, reviewNotify,
      rejectMin, rejectMax, rejectAction, rejectNotify,
      approvalTiers, mutateAsync, onSaved,
    ],
  );

  const tierClass = "space-y-4 rounded-md border border-border p-4";

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
          {error}
        </div>
      )}

      {/* Auto-Apply Tier */}
      <fieldset className={tierClass}>
        <legend className="text-sm font-medium text-text-primary">Auto-Apply Tier</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label htmlFor="auto-min" className="block text-sm font-medium text-text-primary">Min Score</label>
            <input
              id="auto-min"
              type="number"
              min="0"
              max="100"
              value={autoMin}
              onChange={(e) => {
                setAutoMin(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="auto-max" className="block text-sm font-medium text-text-primary">Max Score</label>
            <input
              id="auto-max"
              type="number"
              min="0"
              max="100"
              value={autoMax}
              onChange={(e) => {
                setAutoMax(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="auto-action" className="block text-sm font-medium text-text-primary">Action</label>
            <input
              id="auto-action"
              type="text"
              value={autoAction}
              onChange={(e) => {
                setAutoAction(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
        </div>
        <div className="mt-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={autoNotify}
              onChange={(e) => {
                setAutoNotify(e.target.checked);
                clearError();
              }}
              className="rounded border-border text-primary focus:ring-primary"
            />
            <span className="text-sm text-text-primary">Send notification</span>
          </label>
        </div>
      </fieldset>

      {/* Review Tier */}
      <fieldset className={tierClass}>
        <legend className="text-sm font-medium text-text-primary">Review Tier</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label htmlFor="review-min" className="block text-sm font-medium text-text-primary">Min Score</label>
            <input
              id="review-min"
              type="number"
              min="0"
              max="100"
              value={reviewMin}
              onChange={(e) => {
                setReviewMin(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="review-max" className="block text-sm font-medium text-text-primary">Max Score</label>
            <input
              id="review-max"
              type="number"
              min="0"
              max="100"
              value={reviewMax}
              onChange={(e) => {
                setReviewMax(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="review-action" className="block text-sm font-medium text-text-primary">Action</label>
            <input
              id="review-action"
              type="text"
              value={reviewAction}
              onChange={(e) => {
                setReviewAction(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
        </div>
        <div className="mt-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={reviewNotify}
              onChange={(e) => {
                setReviewNotify(e.target.checked);
                clearError();
              }}
              className="rounded border-border text-primary focus:ring-primary"
            />
            <span className="text-sm text-text-primary">Send notification</span>
          </label>
        </div>
      </fieldset>

      {/* Reject Tier */}
      <fieldset className={tierClass}>
        <legend className="text-sm font-medium text-text-primary">Reject Tier</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label htmlFor="reject-min" className="block text-sm font-medium text-text-primary">Min Score</label>
            <input
              id="reject-min"
              type="number"
              min="0"
              max="100"
              value={rejectMin}
              onChange={(e) => {
                setRejectMin(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="reject-max" className="block text-sm font-medium text-text-primary">Max Score</label>
            <input
              id="reject-max"
              type="number"
              min="0"
              max="100"
              value={rejectMax}
              onChange={(e) => {
                setRejectMax(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="reject-action" className="block text-sm font-medium text-text-primary">Action</label>
            <input
              id="reject-action"
              type="text"
              value={rejectAction}
              onChange={(e) => {
                setRejectAction(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
        </div>
        <div className="mt-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={rejectNotify}
              onChange={(e) => {
                setRejectNotify(e.target.checked);
                clearError();
              }}
              className="rounded border-border text-primary focus:ring-primary"
            />
            <span className="text-sm text-text-primary">Send notification</span>
          </label>
        </div>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="primary" disabled={isSaving} loading={isSaving}>
          Save Approval Tier Settings
        </Button>
      </div>
    </form>
  );
}