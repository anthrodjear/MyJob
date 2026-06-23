/**
 * EducationForm — profile education editor.
 *
 * Displays education entries as cards with inline editing (institution,
 * degree, field, years, GPA). Supports add/remove operations.
 * On submit, replaces the entire education array via PATCH.
 *
 * Does NOT:
 * - Handle preferences/skills/links (separate forms)
 * - Validate against a fixed institution catalog (free-form entry)
 *
 * @see lib/types/profile.ts — Education
 * @see hooks/useProfile.ts — usePatchProfile
 */

"use client";

import { useState, useCallback } from "react";
import { usePatchProfile } from "@/hooks/useProfile";
import type { Education } from "@/lib/types/profile";
import { Button } from "@/components/shared/Button";

/** Props for EducationForm. */
interface EducationFormProps {
  /** Current education list to populate the form. */
  education: Education[];
  /** Called after a successful save. */
  onSaved?: () => void;
}

/**
 * Form for editing the profile education list.
 *
 * Each education row has institution, degree, field, start/end year, and GPA.
 * Users can add new rows and remove existing ones.
 *
 * @example
 *   <EducationForm education={profile.data.education ?? []} onSaved={handleSaved} />
 */
export function EducationForm({ education, onSaved }: EducationFormProps) {
  const patchMutation = usePatchProfile();
  const [rows, setRows] = useState<Education[]>(education);
  const [errors, setErrors] = useState<Record<number, Record<string, string>>>({});

  // ---------------------------------------------------------------------------
  // Row Operations
  // ---------------------------------------------------------------------------

  /** Add an empty education row at the end. */
  const addRow = useCallback(() => {
    setRows((prev) => [
      ...prev,
      { institution: "", degree: "", field: undefined, start_year: undefined, end_year: undefined, gpa: undefined },
    ]);
  }, []);

  /** Remove an education row by index. */
  const removeRow = useCallback((index: number) => {
    setRows((prev) => prev.filter((_, i) => i !== index));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[index];
      return next;
    });
  }, []);

  /** Update a single field in an education row. */
  const updateRow = useCallback(
    (index: number, field: keyof Education, value: string | number | undefined) => {
      setRows((prev) => prev.map((row, i) => (i === index ? { ...row, [field]: value } : row)));
      // Clear field error on edit
      setErrors((prev) => {
        const rowErrors = prev[index];
        if (!rowErrors) return prev;
        const next = { ...prev };
        const updatedRowErrors = { ...rowErrors };
        delete updatedRowErrors[field];
        if (Object.keys(updatedRowErrors).length === 0) {
          delete next[index];
        } else {
          next[index] = updatedRowErrors;
        }
        return next;
      });
    },
    [],
  );

  // ---------------------------------------------------------------------------
  // Validation
  // ---------------------------------------------------------------------------

  const validate = useCallback((): boolean => {
    const newErrors: Record<number, Record<string, string>> = {};
    for (let i = 0; i < rows.length; i++) {
      const rowErrors: Record<string, string> = {};
      if (!rows[i].institution.trim()) {
        rowErrors.institution = "Institution is required";
      }
      if (!rows[i].degree.trim()) {
        rowErrors.degree = "Degree is required";
      }
      const startYear = rows[i].start_year;
      const endYear = rows[i].end_year;
      if (startYear != null && endYear != null && startYear > endYear) {
        rowErrors.end_year = "End year must be after start year";
      }
      if (Object.keys(rowErrors).length > 0) {
        newErrors[i] = rowErrors;
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

      // Filter out empty rows
      const validEntries = rows.filter((r) => r.institution.trim() && r.degree.trim());

      patchMutation.mutate(
        { education: validEntries.length > 0 ? validEntries : [] },
        { onSuccess: onSaved },
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
      {patchMutation.isError && (
        <div className="rounded-md bg-error/10 p-3 text-sm text-error-dark" role="alert">
          Failed to save education. Please try again.
        </div>
      )}

      {/* Education List */}
      <div className="space-y-3">
        {rows.length === 0 && (
          <p className="text-sm text-text-secondary italic">
            No education entries yet. Click &quot;Add Education&quot; to get started.
          </p>
        )}

        {rows.map((entry, index) => (
          <div
            key={index}
            className="rounded-md border border-border p-4"
          >
            {/* Header row: institution + remove */}
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1">
                <label
                  htmlFor={`edu-institution-${index}`}
                  className="block text-xs font-medium text-text-secondary"
                >
                  Institution
                </label>
                <input
                  id={`edu-institution-${index}`}
                  type="text"
                  value={entry.institution}
                  onChange={(e) => updateRow(index, "institution", e.target.value)}
                  placeholder="e.g., MIT, Stanford"
                  className={`mt-1 block w-full rounded-md border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors[index]?.institution ? "border-error" : "border-border"
                  }`}
                  aria-invalid={errors[index]?.institution != null}
                  aria-describedby={errors[index]?.institution ? `edu-institution-error-${index}` : undefined}
                />
                {errors[index]?.institution && (
                  <p id={`edu-institution-error-${index}`} className="mt-1 text-xs text-error-dark" role="alert">
                    {errors[index].institution}
                  </p>
                )}
              </div>

              <Button
                type="button"
                variant="ghost"
                onClick={() => removeRow(index)}
                className="mt-5 text-text-secondary hover:text-error"
                aria-label={`Remove ${entry.institution || "unnamed institution"}`}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </Button>
            </div>

            {/* Details row */}
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-4">
              {/* Degree */}
              <div>
                <label
                  htmlFor={`edu-degree-${index}`}
                  className="block text-xs font-medium text-text-secondary"
                >
                  Degree
                </label>
                <input
                  id={`edu-degree-${index}`}
                  type="text"
                  value={entry.degree}
                  onChange={(e) => updateRow(index, "degree", e.target.value)}
                  placeholder="BS, MS, PhD"
                  className={`mt-1 block w-full rounded-md border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors[index]?.degree ? "border-error" : "border-border"
                  }`}
                  aria-invalid={errors[index]?.degree != null}
                  aria-describedby={errors[index]?.degree ? `edu-degree-error-${index}` : undefined}
                />
                {errors[index]?.degree && (
                  <p id={`edu-degree-error-${index}`} className="mt-1 text-xs text-error-dark" role="alert">
                    {errors[index].degree}
                  </p>
                )}
              </div>

              {/* Field */}
              <div>
                <label
                  htmlFor={`edu-field-${index}`}
                  className="block text-xs font-medium text-text-secondary"
                >
                  Field
                </label>
                <input
                  id={`edu-field-${index}`}
                  type="text"
                  value={entry.field ?? ""}
                  onChange={(e) => updateRow(index, "field", e.target.value || undefined)}
                  placeholder="Computer Science"
                  className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>

              {/* Start Year */}
              <div>
                <label
                  htmlFor={`edu-start-${index}`}
                  className="block text-xs font-medium text-text-secondary"
                >
                  Start Year
                </label>
                <input
                  id={`edu-start-${index}`}
                  type="number"
                  min={1900}
                  max={2100}
                  value={entry.start_year?.toString() ?? ""}
                  onChange={(e) =>
                    updateRow(index, "start_year", e.target.value ? parseInt(e.target.value, 10) : undefined)
                  }
                  placeholder="2015"
                  className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>

              {/* End Year */}
              <div>
                <label
                  htmlFor={`edu-end-${index}`}
                  className="block text-xs font-medium text-text-secondary"
                >
                  End Year
                </label>
                <input
                  id={`edu-end-${index}`}
                  type="number"
                  min={1900}
                  max={2100}
                  value={entry.end_year?.toString() ?? ""}
                  onChange={(e) =>
                    updateRow(index, "end_year", e.target.value ? parseInt(e.target.value, 10) : undefined)
                  }
                  placeholder="2019"
                  className={`mt-1 block w-full rounded-md border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors[index]?.end_year ? "border-error" : "border-border"
                  }`}
                  aria-invalid={errors[index]?.end_year != null}
                  aria-describedby={errors[index]?.end_year ? `edu-end-error-${index}` : undefined}
                />
                {errors[index]?.end_year && (
                  <p id={`edu-end-error-${index}`} className="mt-1 text-xs text-error-dark" role="alert">
                    {errors[index].end_year}
                  </p>
                )}
              </div>
            </div>

            {/* GPA */}
            <div className="mt-3">
              <label
                htmlFor={`edu-gpa-${index}`}
                className="block text-xs font-medium text-text-secondary"
              >
                GPA
              </label>
              <input
                id={`edu-gpa-${index}`}
                type="text"
                value={entry.gpa ?? ""}
                onChange={(e) => updateRow(index, "gpa", e.target.value || undefined)}
                placeholder="3.8 or 3.8/4.0"
                className="mt-1 block w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary sm:max-w-xs"
              />
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
          Add Education
        </Button>

        <Button type="submit" loading={patchMutation.isPending} loadingText="Saving...">
          Save Education
        </Button>
      </div>
    </form>
  );
}
