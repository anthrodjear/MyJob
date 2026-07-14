"use client";

/**
 * SetupStepPreferences — Step 4 of onboarding wizard.
 *
 * Configures scoring thresholds and job source preferences.
 * Uses sensible defaults — user can skip to use defaults.
 *
 * Accessibility:
 * - Semantic `<fieldset>` with `<legend>` for step context
 * - `aria-describedby` links inputs to help text
 * - Range inputs have `aria-valuemin`, `aria-valuemax`, `aria-valuenow`
 * - Checkbox groups use `role="group"` with `aria-label`
 */

import { useState, type FormEvent } from "react";
import { Button } from "@/components/shared/Button";

interface PreferencesState {
  autoThreshold: number;
  reviewThreshold: number;
  jobSources: string[];
}

interface SetupStepPreferencesProps {
  onNext: (prefs: PreferencesState) => void;
  onBack: () => void;
  onSkip: () => void;
}

/** Available job source options. */
const JOB_SOURCE_OPTIONS = [
  { id: "linkedin", label: "LinkedIn", description: "Professional network job postings" },
  { id: "indeed", label: "Indeed", description: "General job board aggregator" },
  { id: "glassdoor", label: "Glassdoor", description: "Job postings with company reviews" },
  { id: "google", label: "Google Jobs", description: "Aggregated from multiple sources" },
  { id: "remoteok", label: "Remote OK", description: "Remote-first job board" },
  { id: "weworkremotely", label: "We Work Remotely", description: "Remote job listings" },
];

/** Props for the ThresholdSlider component. */
interface ThresholdSliderProps {
  label: string;
  description: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
}

/**
 * ThresholdSlider — range input with label and value display.
 */
function ThresholdSlider({
  label,
  description,
  value,
  min,
  max,
  onChange,
}: ThresholdSliderProps) {
  // Create slug for ID attributes (e.g., "Auto-Apply Threshold" → "auto-apply-threshold")
  const slug = label.toLowerCase().replace(/\s+/g, "-");

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label htmlFor={`threshold-${slug}`} className="text-sm font-medium text-text-primary">
          {label}
        </label>
        <span className="text-sm font-mono text-text-secondary">{value}%</span>
      </div>
      <input
        id={`threshold-${slug}`}
        type="range"
        min={min}
        max={max}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full accent-primary"
        aria-valuemin={min}
        aria-valuemax={max}
        aria-valuenow={value}
        aria-describedby={`threshold-${slug}-help`}
      />
      <p id={`threshold-${slug}-help`} className="text-xs text-text-tertiary">
        {description}
      </p>
    </div>
  );
}

/**
 * SetupStepPreferences — preferences configuration step.
 *
 * @example
 *   <SetupStepPreferences
 *     onNext={(prefs) => savePrefs(prefs)}
 *     onBack={() => goBack()}
 *     onSkip={() => skipStep()}
 *   />
 */
export function SetupStepPreferences({
  onNext,
  onBack,
  onSkip,
}: SetupStepPreferencesProps) {
  const [prefs, setPrefs] = useState<PreferencesState>({
    autoThreshold: 95,
    reviewThreshold: 80,
    jobSources: ["linkedin", "indeed"],
  });

  const toggleSource = (sourceId: string) => {
    setPrefs((prev) => ({
      ...prev,
      jobSources: prev.jobSources.includes(sourceId)
        ? prev.jobSources.filter((s) => s !== sourceId)
        : [...prev.jobSources, sourceId],
    }));
  };

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onNext(prefs);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <fieldset className="space-y-6">
        <legend className="text-lg font-semibold text-text-primary">
          Preferences
        </legend>
        <p className="text-sm text-text-secondary">
          Configure how jobs are scored and which sources to search. These
          settings can be changed later in the dashboard.
        </p>

        {/* Scoring Thresholds */}
        <div className="space-y-4">
          <h3 className="text-sm font-medium text-text-primary">
            Scoring Thresholds
          </h3>
          <p className="text-xs text-text-tertiary">
            Jobs above Auto-Apply are submitted automatically. Jobs between
            Review and Auto-Apply need your approval.
          </p>

          <ThresholdSlider
            label="Auto-Apply Threshold"
            description="Jobs at or above this score are submitted automatically."
            value={prefs.autoThreshold}
            min={80}
            max={100}
            onChange={(v) =>
              setPrefs((prev) => ({ ...prev, autoThreshold: v }))
            }
          />

          <ThresholdSlider
            label="Review Threshold"
            description="Jobs at or above this score appear in your review queue."
            value={prefs.reviewThreshold}
            min={50}
            max={prefs.autoThreshold - 1}
            onChange={(v) =>
              setPrefs((prev) => ({ ...prev, reviewThreshold: v }))
            }
          />
        </div>

        {/* Job Sources */}
        <div className="space-y-3">
          <h3 className="text-sm font-medium text-text-primary">Job Sources</h3>
          <p className="text-xs text-text-tertiary">
            Select which job boards to search. You can add custom sites later.
          </p>

          <div
            role="group"
            aria-label="Job source selection"
            className="space-y-2"
          >
            {JOB_SOURCE_OPTIONS.map((source) => (
              <label
                key={source.id}
                className="flex items-start gap-3 rounded-md border border-border p-3 hover:bg-bg-tertiary focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={prefs.jobSources.includes(source.id)}
                  onChange={() => toggleSource(source.id)}
                  className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary"
                />
                <div>
                  <span className="text-sm font-medium text-text-primary">
                    {source.label}
                  </span>
                  <p className="text-xs text-text-tertiary">
                    {source.description}
                  </p>
                </div>
              </label>
            ))}
          </div>
        </div>
      </fieldset>

      <div className="flex justify-between">
        <Button type="button" variant="ghost" onClick={onBack}>
          Back
        </Button>
        <div className="flex gap-3">
          <Button type="button" variant="ghost" onClick={onSkip}>
            Use defaults
          </Button>
          <Button type="submit" variant="primary" size="lg">
            Continue
          </Button>
        </div>
      </div>
    </form>
  );
}
