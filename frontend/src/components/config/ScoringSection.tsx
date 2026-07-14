/**
 * ScoringSection — scoring thresholds, weights, and mode editor.
 *
 * Covers auto/review thresholds, scoring mode, hybrid reject margin,
 * and individual weights (skill, experience, location, salary, description).
 *
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle LLM/Voice/Email settings (separate sections)
 * - Manage authentication state
 *
 * @see lib/types/config.ts — ScoringSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSetOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type { ScoringSection as ScoringSectionType } from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for ScoringSection. */
interface ScoringSectionProps {
  /** Current scoring config to populate the form. */
  scoring: ScoringSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing scoring configuration.
 *
 * Renders fields for thresholds, mode, and weights. Each field saves
 * independently via PATCH to avoid partial updates.
 *
 * @example
 *   <ScoringSection scoring={config.scoring} onSaved={() => console.log("saved")} />
 */
export function ScoringSection({ scoring, onSaved }: ScoringSectionProps) {
  const { mutateAsync } = useSetOverride();

  const [autoThreshold, setAutoThreshold] = useState(
    scoring.auto_threshold.toString(),
  );
  const [reviewThreshold, setReviewThreshold] = useState(
    scoring.review_threshold.toString(),
  );
  const [mode, setMode] = useState(scoring.mode);
  const [hybridRejectMargin, setHybridRejectMargin] = useState(
    scoring.hybrid_reject_margin.toString(),
  );
  const [skillWeight, setSkillWeight] = useState(
    scoring.weights.Skill.toString(),
  );
  const [experienceWeight, setExperienceWeight] = useState(
    scoring.weights.Experience.toString(),
  );
  const [locationWeight, setLocationWeight] = useState(
    scoring.weights.Location.toString(),
  );
  const [salaryWeight, setSalaryWeight] = useState(
    scoring.weights.Salary.toString(),
  );
  const [descriptionWeight, setDescriptionWeight] = useState(
    scoring.weights.Description.toString(),
  );
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
    setAutoThreshold(scoring.auto_threshold.toString());
    setReviewThreshold(scoring.review_threshold.toString());
    setMode(scoring.mode);
    setHybridRejectMargin(scoring.hybrid_reject_margin.toString());
    setSkillWeight(scoring.weights.Skill.toString());
    setExperienceWeight(scoring.weights.Experience.toString());
    setLocationWeight(scoring.weights.Location.toString());
    setSalaryWeight(scoring.weights.Salary.toString());
    setDescriptionWeight(scoring.weights.Description.toString());
  }, [scoring]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);
      setError(null);

      const overrides: Array<[string, number | string]> = [];

      // Validate and add threshold overrides
      if (autoThreshold !== scoring.auto_threshold.toString()) {
        const parsed = parseInt(autoThreshold, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
          setError("Auto-Apply Threshold must be a valid number between 0 and 100.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.auto_threshold", parsed]);
      }
      if (reviewThreshold !== scoring.review_threshold.toString()) {
        const parsed = parseInt(reviewThreshold, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 100) {
          setError("Review Threshold must be a valid number between 0 and 100.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.review_threshold", parsed]);
      }
      if (mode !== scoring.mode) {
        overrides.push(["scoring.mode", mode]);
      }
      if (hybridRejectMargin !== scoring.hybrid_reject_margin.toString()) {
        const parsed = parseInt(hybridRejectMargin, 10);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 50) {
          setError("Hybrid Reject Margin must be a valid number between 0 and 50.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.hybrid_reject_margin", parsed]);
      }

      // Validate and add weight overrides
      if (skillWeight !== scoring.weights.Skill.toString()) {
        const parsed = parseFloat(skillWeight);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
          setError("Skill weight must be a valid number between 0 and 1.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.weights.skill", parsed]);
      }
      if (experienceWeight !== scoring.weights.Experience.toString()) {
        const parsed = parseFloat(experienceWeight);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
          setError("Experience weight must be a valid number between 0 and 1.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.weights.experience", parsed]);
      }
      if (locationWeight !== scoring.weights.Location.toString()) {
        const parsed = parseFloat(locationWeight);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
          setError("Location weight must be a valid number between 0 and 1.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.weights.location", parsed]);
      }
      if (salaryWeight !== scoring.weights.Salary.toString()) {
        const parsed = parseFloat(salaryWeight);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
          setError("Salary weight must be a valid number between 0 and 1.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.weights.salary", parsed]);
      }
      if (descriptionWeight !== scoring.weights.Description.toString()) {
        const parsed = parseFloat(descriptionWeight);
        if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
          setError("Description weight must be a valid number between 0 and 1.");
          setIsSaving(false);
          return;
        }
        overrides.push(["scoring.weights.description", parsed]);
      }

      try {
        const result = await executeOverrides(overrides, mutateAsync, onSaved);
        if (result.failed > 0) {
          setError(
            result.failed === result.total
              ? "Failed to save scoring settings. Please try again."
              : `Partially saved: ${result.succeeded} of ${result.total} settings saved. ${result.failed} failed.`,
          );
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to save scoring settings. Please try again.",
        );
      } finally {
        setIsSaving(false);
      }
    },
    [
      autoThreshold,
      reviewThreshold,
      mode,
      hybridRejectMargin,
      skillWeight,
      experienceWeight,
      locationWeight,
      salaryWeight,
      descriptionWeight,
      scoring,
      mutateAsync,
      onSaved,
    ],
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
          {error}
        </div>
      )}

      {/* Thresholds */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label
            htmlFor="auto-threshold"
            className="block text-sm font-medium text-text-primary"
          >
            Auto-Apply Threshold
          </label>
          <input
            id="auto-threshold"
            type="number"
            min="0"
            max="100"
            value={autoThreshold}
            onChange={(e) => {
              setAutoThreshold(e.target.value);
              clearError();
            }}
            className={INPUT_CLASS}
          />
          <p className="mt-1 text-xs text-text-secondary">
            Jobs with score ≥ this value are auto-applied
          </p>
        </div>
        <div>
          <label
            htmlFor="review-threshold"
            className="block text-sm font-medium text-text-primary"
          >
            Review Threshold
          </label>
          <input
            id="review-threshold"
            type="number"
            min="0"
            max="100"
            value={reviewThreshold}
            onChange={(e) => {
              setReviewThreshold(e.target.value);
              clearError();
            }}
            className={INPUT_CLASS}
          />
          <p className="mt-1 text-xs text-text-secondary">
            Jobs with score ≥ this value require manual review
          </p>
        </div>
      </div>

      {/* Mode and Margin */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label
            htmlFor="scoring-mode"
            className="block text-sm font-medium text-text-primary"
          >
            Scoring Mode
          </label>
          <select
            id="scoring-mode"
            value={mode}
            onChange={(e) => {
              setMode(e.target.value as ScoringSectionType["mode"]);
              clearError();
            }}
            className={INPUT_CLASS}
          >
            <option value="heuristic">Heuristic (fast, keyword-based)</option>
            <option value="llm">LLM (semantic, slower)</option>
            <option value="hybrid">Hybrid (recommended)</option>
          </select>
        </div>
        <div>
          <label
            htmlFor="hybrid-reject-margin"
            className="block text-sm font-medium text-text-primary"
          >
            Hybrid Reject Margin
          </label>
          <input
            id="hybrid-reject-margin"
            type="number"
            min="0"
            max="50"
            value={hybridRejectMargin}
            onChange={(e) => {
              setHybridRejectMargin(e.target.value);
              clearError();
            }}
            className={INPUT_CLASS}
          />
          <p className="mt-1 text-xs text-text-secondary">
            Margin below review threshold for hybrid reject
          </p>
        </div>
      </div>

      {/* Weights */}
      <div>
        <h3 className="text-sm font-medium text-text-primary mb-3">
          Scoring Weights (must sum to 1.0)
        </h3>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label
              htmlFor="skill-weight"
              className="block text-sm font-medium text-text-primary"
            >
              Skill
            </label>
            <input
              id="skill-weight"
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={skillWeight}
              onChange={(e) => {
                setSkillWeight(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label
              htmlFor="experience-weight"
              className="block text-sm font-medium text-text-primary"
            >
              Experience
            </label>
            <input
              id="experience-weight"
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={experienceWeight}
              onChange={(e) => {
                setExperienceWeight(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label
              htmlFor="location-weight"
              className="block text-sm font-medium text-text-primary"
            >
              Location
            </label>
            <input
              id="location-weight"
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={locationWeight}
              onChange={(e) => {
                setLocationWeight(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label
              htmlFor="salary-weight"
              className="block text-sm font-medium text-text-primary"
            >
              Salary
            </label>
            <input
              id="salary-weight"
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={salaryWeight}
              onChange={(e) => {
                setSalaryWeight(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label
              htmlFor="description-weight"
              className="block text-sm font-medium text-text-primary"
            >
              Description
            </label>
            <input
              id="description-weight"
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={descriptionWeight}
              onChange={(e) => {
                setDescriptionWeight(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </div>

      {/* Submit */}
      <div className="flex justify-end">
        <Button
          type="submit"
          variant="primary"
          disabled={isSaving}
          loading={isSaving}
        >
          Save Scoring Settings
        </Button>
      </div>
    </form>
  );
}