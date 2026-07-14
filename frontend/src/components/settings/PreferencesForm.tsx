/**
 * PreferencesForm — profile preferences editor.
 *
 * Covers job targeting (titles, locations, remote, salary), work authorization,
 * experience, resume generation settings, and application behavior (auto-apply).
 *
 * Uses controlled form state with local React state. On submit, calls the
 * patchProfile mutation which handles ETag concurrency automatically.
 *
 * Does NOT:
 * - Handle skills/education/links (separate forms)
 * - Manage authentication state
 *
 * @see lib/types/profile.ts — ProfilePreferences
 * @see hooks/useProfile.ts — usePatchProfile
 */

"use client";

import { useState, useCallback } from "react";
import { usePatchProfile } from "@/hooks/useProfile";
import type { ProfilePreferences } from "@/lib/types/profile";
import { Button } from "@/components/shared/Button";

/** Props for PreferencesForm. */
interface PreferencesFormProps {
  /** Current preferences to populate the form. */
  preferences: ProfilePreferences;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/**
 * Form for editing profile preferences.
 *
 * Renders a two-column grid of fields grouped by category. Each field
 * uses controlled state initialized from the current profile. On submit,
 * only changed fields are sent via PATCH.
 *
 * @example
 *   <PreferencesForm
 *     preferences={profile.data.preferences}
 *     onSaved={() => console.log("saved")}
 *   />
 */
export function PreferencesForm({ preferences, onSaved }: PreferencesFormProps) {
  const patchMutation = usePatchProfile();

  // ---------------------------------------------------------------------------
  // Form State — initialized from props, updated on change
  // ---------------------------------------------------------------------------

  const [targetTitles, setTargetTitles] = useState(
    preferences.target_titles?.join(", ") ?? "",
  );
  const [targetLocations, setTargetLocations] = useState(
    preferences.target_locations?.join(", ") ?? "",
  );
  const [remoteOnly, setRemoteOnly] = useState(preferences.remote_only ?? false);
  const [minSalary, setMinSalary] = useState(preferences.min_salary?.toString() ?? "");
  const [maxSalary, setMaxSalary] = useState(preferences.max_salary?.toString() ?? "");
  const [workAuthorization, setWorkAuthorization] = useState(
    preferences.work_authorization ?? "",
  );
  const [yearsExperience, setYearsExperience] = useState(
    preferences.years_experience?.toString() ?? "",
  );
  const [resumeTone, setResumeTone] = useState(preferences.resume_tone ?? "");
  const [resumeStyle, setResumeStyle] = useState(preferences.resume_style ?? "");
  const [autoApplyThreshold, setAutoApplyThreshold] = useState(
    preferences.auto_apply_threshold?.toString() ?? "",
  );
  const [coverLetterStyle, setCoverLetterStyle] = useState(
    preferences.cover_letter_style ?? "",
  );
  const [serverError, setServerError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // ---------------------------------------------------------------------------
  // Validation
  // ---------------------------------------------------------------------------

  const validate = useCallback((): boolean => {
    const errors: Record<string, string> = {};

    // Salary range validation
    if (minSalary !== "" && maxSalary !== "") {
      const min = parseInt(minSalary, 10);
      const max = parseInt(maxSalary, 10);
      if (!Number.isNaN(min) && !Number.isNaN(max) && min > max) {
        errors.minSalary = "Minimum salary must be less than maximum";
        errors.maxSalary = "Maximum salary must be greater than minimum";
      }
    }

    // Auto-apply threshold validation
    if (autoApplyThreshold !== "") {
      const threshold = parseInt(autoApplyThreshold, 10);
      if (Number.isNaN(threshold) || threshold < 0 || threshold > 100) {
        errors.autoApplyThreshold = "Threshold must be between 0 and 100";
      }
    }

    // Years experience validation
    if (yearsExperience !== "") {
      const years = parseInt(yearsExperience, 10);
      if (Number.isNaN(years) || years < 0 || years > 50) {
        errors.yearsExperience = "Years must be between 0 and 50";
      }
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }, [minSalary, maxSalary, autoApplyThreshold, yearsExperience]);

  // ---------------------------------------------------------------------------
  // Submit Handler
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      setServerError(null);

      if (!validate()) return;

      // Parse comma-separated lists, trimming whitespace and filtering empties
      const titles = targetTitles
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const locations = targetLocations
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);

      patchMutation.mutate(
        {
          preferences: {
            target_titles: titles.length > 0 ? titles : undefined,
            target_locations: locations.length > 0 ? locations : undefined,
            remote_only: remoteOnly,
            min_salary: minSalary !== "" ? parseInt(minSalary, 10) : undefined,
            max_salary: maxSalary !== "" ? parseInt(maxSalary, 10) : undefined,
            work_authorization: workAuthorization || undefined,
            years_experience: yearsExperience !== "" ? parseInt(yearsExperience, 10) : undefined,
            resume_tone: resumeTone || undefined,
            resume_style: resumeStyle || undefined,
            auto_apply_threshold:
              autoApplyThreshold !== "" ? parseInt(autoApplyThreshold, 10) : undefined,
            cover_letter_style: coverLetterStyle || undefined,
          },
        },
        {
          onSuccess: onSaved,
          onError: (error) => {
            // Show the actual error message from the backend
            const message = error instanceof Error ? error.message : "Failed to save preferences. Please try again.";
            setServerError(message);
          },
        },
      );
    },
    [
      targetTitles,
      targetLocations,
      remoteOnly,
      minSalary,
      maxSalary,
      workAuthorization,
      yearsExperience,
      resumeTone,
      resumeStyle,
      autoApplyThreshold,
      coverLetterStyle,
      validate,
      patchMutation,
      onSaved,
    ],
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Error feedback */}
      {serverError && (
        <div
          className="rounded-md bg-error/10 p-3 text-sm text-error-dark"
          role="alert"
        >
          {serverError}
        </div>
      )}

      {/* Job Targeting Section */}
      <fieldset>
        <legend className="text-lg font-semibold text-text-primary">
          Job Targeting
        </legend>
        <p className="text-sm text-text-secondary mb-4">
          Configure what jobs the agent searches for.
        </p>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {/* Target Titles */}
          <div className="sm:col-span-2">
            <label
              htmlFor="target_titles"
              className="block text-sm font-medium text-text-primary"
            >
              Target Titles
            </label>
            <input
              id="target_titles"
              type="text"
              value={targetTitles}
              onChange={(e) => {
                setTargetTitles(e.target.value);
                setServerError(null);
              }}
              placeholder="Backend Engineer, Platform Engineer"
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <p className="mt-1 text-xs text-text-tertiary">
              Comma-separated job titles to search for
            </p>
          </div>

          {/* Target Locations */}
          <div className="sm:col-span-2">
            <label
              htmlFor="target_locations"
              className="block text-sm font-medium text-text-primary"
            >
              Target Locations
            </label>
            <input
              id="target_locations"
              type="text"
              value={targetLocations}
              onChange={(e) => {
                setTargetLocations(e.target.value);
                setServerError(null);
              }}
              placeholder="Remote, New York, San Francisco"
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <p className="mt-1 text-xs text-text-tertiary">
              Comma-separated preferred locations
            </p>
          </div>

          {/* Remote Only */}
          <div className="flex items-center gap-2">
            <input
              id="remote_only"
              type="checkbox"
              checked={remoteOnly}
              onChange={(e) => {
                setRemoteOnly(e.target.checked);
                setServerError(null);
              }}
              className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
            />
            <label htmlFor="remote_only" className="text-sm text-text-primary">
              Remote only
            </label>
          </div>

          {/* Work Authorization */}
          <div>
            <label
              htmlFor="work_authorization"
              className="block text-sm font-medium text-text-primary"
            >
              Work Authorization
            </label>
            <input
              id="work_authorization"
              type="text"
              value={workAuthorization}
              onChange={(e) => {
                setWorkAuthorization(e.target.value);
                setServerError(null);
              }}
              placeholder="US Citizen, H1B, Green Card"
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          {/* Years Experience */}
          <div>
            <label
              htmlFor="years_experience"
              className="block text-sm font-medium text-text-primary"
            >
              Years of Experience
            </label>
            <input
              id="years_experience"
              type="number"
              min={0}
              max={50}
              value={yearsExperience}
              onChange={(e) => {
                setYearsExperience(e.target.value);
                setServerError(null);
                setFieldErrors((prev) => {
                  const next = { ...prev };
                  delete next.yearsExperience;
                  return next;
                });
              }}
              placeholder="5"
              className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                fieldErrors.yearsExperience ? "border-error" : "border-border"
              }`}
            />
            {fieldErrors.yearsExperience && (
              <p className="mt-1 text-xs text-error-dark" role="alert">
                {fieldErrors.yearsExperience}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Salary Range Section */}
      <fieldset>
        <legend className="text-lg font-semibold text-text-primary">
          Salary Range
        </legend>
        <p className="text-sm text-text-secondary mb-4">
          Filter jobs by salary range (annual, USD).
        </p>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label
              htmlFor="min_salary"
              className="block text-sm font-medium text-text-primary"
            >
              Minimum Salary
            </label>
            <input
              id="min_salary"
              type="number"
              min={0}
              step={1000}
              value={minSalary}
              onChange={(e) => {
                setMinSalary(e.target.value);
                setServerError(null);
                setFieldErrors((prev) => {
                  const next = { ...prev };
                  delete next.minSalary;
                  delete next.maxSalary;
                  return next;
                });
              }}
              placeholder="100000"
              className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                fieldErrors.minSalary ? "border-error" : "border-border"
              }`}
            />
            {fieldErrors.minSalary && (
              <p className="mt-1 text-xs text-error-dark" role="alert">
                {fieldErrors.minSalary}
              </p>
            )}
          </div>
          <div>
            <label
              htmlFor="max_salary"
              className="block text-sm font-medium text-text-primary"
            >
              Maximum Salary
            </label>
            <input
              id="max_salary"
              type="number"
              min={0}
              step={1000}
              value={maxSalary}
              onChange={(e) => {
                setMaxSalary(e.target.value);
                setServerError(null);
                setFieldErrors((prev) => {
                  const next = { ...prev };
                  delete next.minSalary;
                  delete next.maxSalary;
                  return next;
                });
              }}
              placeholder="200000"
              className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                fieldErrors.maxSalary ? "border-error" : "border-border"
              }`}
            />
            {fieldErrors.maxSalary && (
              <p className="mt-1 text-xs text-error-dark" role="alert">
                {fieldErrors.maxSalary}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Resume Generation Section */}
      <fieldset>
        <legend className="text-lg font-semibold text-text-primary">
          Resume Generation
        </legend>
        <p className="text-sm text-text-secondary mb-4">
          Configure how the agent generates resumes and cover letters.
        </p>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label
              htmlFor="resume_tone"
              className="block text-sm font-medium text-text-primary"
            >
              Resume Tone
            </label>
            <select
              id="resume_tone"
              value={resumeTone}
              onChange={(e) => {
                setResumeTone(e.target.value);
                setServerError(null);
              }}
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">Default</option>
              <option value="professional">Professional</option>
              <option value="casual">Casual</option>
              <option value="technical">Technical</option>
              <option value="creative">Creative</option>
            </select>
          </div>

          <div>
            <label
              htmlFor="resume_style"
              className="block text-sm font-medium text-text-primary"
            >
              Resume Style
            </label>
            <select
              id="resume_style"
              value={resumeStyle}
              onChange={(e) => {
                setResumeStyle(e.target.value);
                setServerError(null);
              }}
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">Default</option>
              <option value="chronological">Chronological</option>
              <option value="functional">Functional</option>
              <option value="hybrid">Hybrid</option>
            </select>
          </div>

          <div>
            <label
              htmlFor="cover_letter_style"
              className="block text-sm font-medium text-text-primary"
            >
              Cover Letter Style
            </label>
            <select
              id="cover_letter_style"
              value={coverLetterStyle}
              onChange={(e) => {
                setCoverLetterStyle(e.target.value);
                setServerError(null);
              }}
              className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">Default</option>
              <option value="formal">Formal</option>
              <option value="conversational">Conversational</option>
              <option value="technical">Technical</option>
            </select>
          </div>

          {/* Auto-Apply Threshold */}
          <div>
            <label
              htmlFor="auto_apply_threshold"
              className="block text-sm font-medium text-text-primary"
            >
              Auto-Apply Threshold
            </label>
            <input
              id="auto_apply_threshold"
              type="number"
              min={0}
              max={100}
              value={autoApplyThreshold}
              onChange={(e) => {
                setAutoApplyThreshold(e.target.value);
                setServerError(null);
                setFieldErrors((prev) => {
                  const next = { ...prev };
                  delete next.autoApplyThreshold;
                  return next;
                });
              }}
              placeholder="95"
              className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                fieldErrors.autoApplyThreshold ? "border-error" : "border-border"
              }`}
            />
            <p className="mt-1 text-xs text-text-tertiary">
              Score (0-100) above which applications auto-submit. Leave empty for manual approval.
            </p>
            {fieldErrors.autoApplyThreshold && (
              <p className="mt-1 text-xs text-error-dark" role="alert">
                {fieldErrors.autoApplyThreshold}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Submit */}
      <div className="flex justify-end">
        <Button
          type="submit"
          loading={patchMutation.isPending}
          loadingText="Saving..."
        >
          Save Preferences
        </Button>
      </div>
    </form>
  );
}
