/**
 * LinksForm — external profile links editor.
 *
 * Simple form for LinkedIn, GitHub, and portfolio URLs.
 * On submit, replaces the entire links object via PATCH.
 *
 * Does NOT:
 * - Handle preferences/skills/education (separate forms)
 * - Validate link ownership (user enters URLs manually)
 *
 * @see lib/types/profile.ts — ProfileLinks
 * @see hooks/useProfile.ts — usePatchProfile
 */

"use client";

import { useState, useCallback } from "react";
import { usePatchProfile } from "@/hooks/useProfile";
import type { ProfileLinks } from "@/lib/types/profile";
import { Button } from "@/components/shared/Button";

/** Props for LinksForm. */
interface LinksFormProps {
  /** Current links to populate the form. */
  links: ProfileLinks;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/**
 * Form for editing external profile links.
 *
 * Three URL fields: LinkedIn, GitHub, Portfolio. All optional.
 * Validates URL format on submit (must start with http/https or be empty).
 *
 * @example
 *   <LinksForm links={profile.data.links ?? {}} onSaved={handleSaved} />
 */
export function LinksForm({ links, onSaved }: LinksFormProps) {
  const patchMutation = usePatchProfile();
  const [linkedin, setLinkedin] = useState(links.linkedin ?? "");
  const [github, setGithub] = useState(links.github ?? "");
  const [portfolio, setPortfolio] = useState(links.portfolio ?? "");
  const [errors, setErrors] = useState<Record<string, string>>({});

  // ---------------------------------------------------------------------------
  // Validation
  // ---------------------------------------------------------------------------

  const validate = useCallback((): boolean => {
    const newErrors: Record<string, string> = {};
    const urlPattern = /^https?:\/\/.+/;

    if (linkedin && !urlPattern.test(linkedin)) {
      newErrors.linkedin = "Must be a valid URL starting with http:// or https://";
    }
    if (github && !urlPattern.test(github)) {
      newErrors.github = "Must be a valid URL starting with http:// or https://";
    }
    if (portfolio && !urlPattern.test(portfolio)) {
      newErrors.portfolio = "Must be a valid URL starting with http:// or https://";
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [linkedin, github, portfolio]);

  // ---------------------------------------------------------------------------
  // Submit
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate()) return;

      patchMutation.mutate(
        {
          links: {
            linkedin: linkedin || undefined,
            github: github || undefined,
            portfolio: portfolio || undefined,
          },
        },
        { onSuccess: onSaved },
      );
    },
    [linkedin, github, portfolio, validate, patchMutation, onSaved],
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Error feedback */}
      {patchMutation.isError && (
        <div className="rounded-md bg-error/10 p-3 text-sm text-error-dark" role="alert">
          Failed to save links. Please try again.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* LinkedIn */}
        <div>
          <label
            htmlFor="linkedin"
            className="block text-sm font-medium text-text-primary"
          >
            LinkedIn
          </label>
          <input
            id="linkedin"
            type="url"
            value={linkedin}
            onChange={(e) => {
              setLinkedin(e.target.value);
              setErrors((prev) => {
                const next = { ...prev };
                delete next.linkedin;
                return next;
              });
            }}
            placeholder="https://linkedin.com/in/yourname"
            className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
              errors.linkedin ? "border-error" : "border-border"
            }`}
            aria-invalid={errors.linkedin != null}
            aria-describedby={errors.linkedin ? "linkedin-error" : undefined}
          />
          {errors.linkedin && (
            <p id="linkedin-error" className="mt-1 text-xs text-error-dark" role="alert">
              {errors.linkedin}
            </p>
          )}
        </div>

        {/* GitHub */}
        <div>
          <label
            htmlFor="github"
            className="block text-sm font-medium text-text-primary"
          >
            GitHub
          </label>
          <input
            id="github"
            type="url"
            value={github}
            onChange={(e) => {
              setGithub(e.target.value);
              setErrors((prev) => {
                const next = { ...prev };
                delete next.github;
                return next;
              });
            }}
            placeholder="https://github.com/yourname"
            className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
              errors.github ? "border-error" : "border-border"
            }`}
            aria-invalid={errors.github != null}
            aria-describedby={errors.github ? "github-error" : undefined}
          />
          {errors.github && (
            <p id="github-error" className="mt-1 text-xs text-error-dark" role="alert">
              {errors.github}
            </p>
          )}
        </div>

        {/* Portfolio */}
        <div className="sm:col-span-2">
          <label
            htmlFor="portfolio"
            className="block text-sm font-medium text-text-primary"
          >
            Portfolio
          </label>
          <input
            id="portfolio"
            type="url"
            value={portfolio}
            onChange={(e) => {
              setPortfolio(e.target.value);
              setErrors((prev) => {
                const next = { ...prev };
                delete next.portfolio;
                return next;
              });
            }}
            placeholder="https://yourname.dev"
            className={`mt-1 block w-full rounded-md border bg-surface px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
              errors.portfolio ? "border-error" : "border-border"
            }`}
            aria-invalid={errors.portfolio != null}
            aria-describedby={errors.portfolio ? "portfolio-error" : undefined}
          />
          {errors.portfolio && (
            <p id="portfolio-error" className="mt-1 text-xs text-error-dark" role="alert">
              {errors.portfolio}
            </p>
          )}
        </div>
      </div>

      {/* Submit */}
      <div className="flex justify-end">
        <Button type="submit" loading={patchMutation.isPending} loadingText="Saving...">
          Save Links
        </Button>
      </div>
    </form>
  );
}
