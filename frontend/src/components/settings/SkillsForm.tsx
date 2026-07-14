/**
 * SkillsForm — profile skills editor.
 *
 * Displays skills as a list with inline editing (name, proficiency, years).
 * Supports add/remove/reorder operations. On submit, replaces the entire
 * skills array via PATCH.
 *
 * Does NOT:
 * - Handle preferences/education/links (separate forms)
 * - Validate against a fixed skill catalog (free-form entry)
 *
 * @see lib/types/profile.ts — Skill, SkillProficiency
 * @see hooks/useProfile.ts — usePatchProfile
 */

"use client";

import { useState, useCallback } from "react";
import { usePatchProfile } from "@/hooks/useProfile";
import type { Skill, SkillProficiency } from "@/lib/types/profile";
import { SKILL_PROFICIENCIES } from "@/lib/types/profile";
import { Button } from "@/components/shared/Button";

/** Props for SkillsForm. */
interface SkillsFormProps {
  /** Current skills list to populate the form. */
  skills: Skill[];
  /** Called after a successful save. */
  onSaved?: () => void;
}

/**
 * Form for editing the profile skills list.
 *
 * Each skill row has name (text), proficiency (select), and years (number).
 * Users can add new rows, remove existing ones, and reorder via drag handles.
 *
 * @example
 *   <SkillsForm skills={profile.data.skills ?? []} onSaved={handleSaved} />
 */
export function SkillsForm({ skills, onSaved }: SkillsFormProps) {
  const patchMutation = usePatchProfile();
  const [rows, setRows] = useState<Skill[]>(skills);
  const [errors, setErrors] = useState<Record<number, string>>({});
  const [serverError, setServerError] = useState<string | null>(null);

  // ---------------------------------------------------------------------------
  // Row Operations
  // ---------------------------------------------------------------------------

  /** Add an empty skill row at the end. */
  const addRow = useCallback(() => {
    setRows((prev) => [...prev, { name: "", proficiency: undefined, years: undefined }]);
    setServerError(null);
  }, []);

  /** Remove a skill row by index. */
  const removeRow = useCallback((index: number) => {
    setRows((prev) => prev.filter((_, i) => i !== index));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[index];
      return next;
    });
    setServerError(null);
  }, []);

  /** Update a single field in a skill row. */
  const updateRow = useCallback(
    (index: number, field: keyof Skill, value: string | number | SkillProficiency | undefined) => {
      setRows((prev) => prev.map((row, i) => (i === index ? { ...row, [field]: value } : row)));
      // Clear error for this row on edit
      setErrors((prev) => {
        const next = { ...prev };
        delete next[index];
        return next;
      });
      setServerError(null);
    },
    [],
  );

  // ---------------------------------------------------------------------------
  // Validation
  // ---------------------------------------------------------------------------

  const validate = useCallback((): boolean => {
    const newErrors: Record<number, string> = {};
    for (let i = 0; i < rows.length; i++) {
      if (!rows[i].name.trim()) {
        newErrors[i] = "Skill name is required";
      }
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [rows]);

  // ---------------------------------------------------------------------------
  // Submit
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate()) return;

      // Filter out empty rows (user added then cleared)
      const validSkills = rows.filter((r) => r.name.trim());

      patchMutation.mutate(
        { skills: validSkills.length > 0 ? validSkills : [] },
        {
          onSuccess: onSaved,
          onError: (error) => {
            // Show the actual error message from the backend
            const message = error instanceof Error ? error.message : "Failed to save skills. Please try again.";
            setServerError(message);
          },
        },
      );
    },
    [rows, validate, patchMutation, onSaved],
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Error feedback */}
      {serverError && (
        <div className="rounded-md bg-error/10 p-3 text-sm text-error-dark" role="alert">
          {serverError}
        </div>
      )}

      {/* Skills List */}
      <div className="space-y-3">
        {rows.length === 0 && (
          <p className="text-sm text-text-secondary italic">
            No skills added yet. Click &quot;Add Skill&quot; to get started.
          </p>
        )}

        {rows.map((skill, index) => (
          <div
            key={index}
            className="flex flex-col gap-2 rounded-md border border-border p-3 transition-colors hover:border-primary/30 sm:flex-row sm:items-start"
          >
            {/* Skill Name */}
            <div className="flex-1">
              <label
                htmlFor={`skill-name-${index}`}
                className="block text-xs font-medium text-text-secondary"
              >
                Skill Name <span className="text-error">*</span>
              </label>
              <input
                id={`skill-name-${index}`}
                type="text"
                value={skill.name}
                onChange={(e) => updateRow(index, "name", e.target.value)}
                placeholder="e.g., Go, TypeScript, React"
                className={`mt-1 block w-full rounded-md border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                  errors[index] ? "border-error" : "border-border"
                }`}
                aria-invalid={errors[index] != null}
                aria-describedby={errors[index] ? `skill-error-${index}` : undefined}
              />
              {errors[index] && (
                <p id={`skill-error-${index}`} className="mt-1 text-xs text-error-dark" role="alert">
                  {errors[index]}
                </p>
              )}
            </div>

            {/* Proficiency */}
            <div className="w-full sm:w-40">
              <label
                htmlFor={`skill-proficiency-${index}`}
                className="block text-xs font-medium text-text-secondary"
              >
                Proficiency
              </label>
              <select
                id={`skill-proficiency-${index}`}
                value={skill.proficiency ?? ""}
                onChange={(e) =>
                  updateRow(
                    index,
                    "proficiency",
                    (e.target.value || undefined) as SkillProficiency | undefined,
                  )
                }
                className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="">—</option>
                {SKILL_PROFICIENCIES.map((p) => (
                  <option key={p} value={p}>
                    {p.charAt(0).toUpperCase() + p.slice(1)}
                  </option>
                ))}
              </select>
            </div>

            {/* Years */}
            <div className="w-full sm:w-24">
              <label
                htmlFor={`skill-years-${index}`}
                className="block text-xs font-medium text-text-secondary"
              >
                Years
              </label>
              <input
                id={`skill-years-${index}`}
                type="number"
                min={0}
                max={50}
                value={skill.years?.toString() ?? ""}
                onChange={(e) =>
                  updateRow(index, "years", e.target.value ? parseInt(e.target.value, 10) : undefined)
                }
                placeholder="0"
                className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            {/* Remove Button */}
            <div className="flex items-end sm:pt-5">
              <Button
                type="button"
                variant="ghost"
                onClick={() => removeRow(index)}
                className="text-text-secondary hover:text-error"
                aria-label={`Remove ${skill.name || "unnamed skill"}`}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <Button type="button" variant="ghost" onClick={addRow}>
          <svg xmlns="http://www.w3.org/2000/svg" className="mr-1.5 h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Add Skill
        </Button>

        <Button type="submit" loading={patchMutation.isPending} loadingText="Saving...">
          Save Skills
        </Button>
      </div>
    </form>
  );
}
