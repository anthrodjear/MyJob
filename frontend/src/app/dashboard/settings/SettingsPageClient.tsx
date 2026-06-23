/**
 * SettingsPageClient — client-side settings page with profile form sections.
 *
 * Fetches the profile via useProfile hook and renders four tab-like sections:
 * Preferences, Skills, Education, and Links. Each section uses its own
 * form component with independent save buttons.
 *
 * Does NOT:
 * - Handle authentication (server middleware handles this)
 * - Manage navigation (AppShell handles sidebar)
 *
 * @see hooks/useProfile.ts — useProfile
 * @see components/settings/ — form components
 */

"use client";

import { useState, useCallback } from "react";
import { useProfile } from "@/hooks/useProfile";
import { PreferencesForm } from "@/components/settings/PreferencesForm";
import { SkillsForm } from "@/components/settings/SkillsForm";
import { EducationForm } from "@/components/settings/EducationForm";
import { LinksForm } from "@/components/settings/LinksForm";
import { Button } from "@/components/shared/Button";

/** Settings page section tabs. */
const SECTIONS = ["preferences", "skills", "education", "links"] as const;
type Section = (typeof SECTIONS)[number];

/** Tab label mapping. */
const SECTION_LABELS: Record<Section, string> = {
  preferences: "Preferences",
  skills: "Skills",
  education: "Education",
  links: "Links",
};

/**
 * Client-side settings page.
 *
 * Renders a tabbed interface for editing different parts of the profile.
 * Each tab shows a form that saves independently via PATCH.
 *
 * Loading state shows skeleton placeholders. Error state shows a retry prompt.
 *
 * @example
 *   <SettingsPageClient />
 */
export function SettingsPageClient() {
  const { data: profile, isLoading, isError, error, refetch } = useProfile();
  const [activeSection, setActiveSection] = useState<Section>("preferences");
  const [saveMessage, setSaveMessage] = useState<string | null>(null);

  /** Called after a successful save — show confirmation then clear. */
  const handleSaved = useCallback(() => {
    setSaveMessage("Saved successfully");
    setTimeout(() => setSaveMessage(null), 3000);
  }, []);

  // ---------------------------------------------------------------------------
  // Loading State
  // ---------------------------------------------------------------------------

  if (isLoading) {
    return (
      <div className="space-y-6" aria-busy="true">
        <div className="h-8 w-48 animate-pulse rounded bg-surface" />
        <div className="h-4 w-96 animate-pulse rounded bg-surface" />
        <div className="mt-6 space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-32 animate-pulse rounded-md border border-border bg-surface" />
          ))}
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Error State
  // ---------------------------------------------------------------------------

  if (isError || !profile) {
    return (
      <div
        className="flex flex-col items-center justify-center py-16 text-center"
        role="alert"
        aria-live="assertive"
      >
        <h2 className="mb-2 text-lg font-semibold text-text-primary">
          Unable to Load Profile
        </h2>
        <p className="mb-6 max-w-md text-sm text-text-secondary">
          {error instanceof Error
            ? "Failed to load your profile. Please check your connection and try again."
            : "Failed to load your profile. Please try again."}
        </p>
        <Button variant="primary" onClick={() => refetch()}>
          Try Again
        </Button>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-text-primary">Settings</h1>
        <p className="mt-1 text-sm text-text-secondary">
          Manage your profile, skills, and application preferences.
        </p>
      </div>

      {/* Save Confirmation */}
      {saveMessage && (
        <div
          className="rounded-md bg-success/10 p-3 text-sm text-success-dark"
          role="status"
          aria-live="polite"
        >
          {saveMessage}
        </div>
      )}

      {/* Section Tabs */}
      <nav aria-label="Settings sections" className="border-b border-border">
        <div className="flex gap-1" role="tablist">
          {SECTIONS.map((section) => (
            <button
              key={section}
              role="tab"
              aria-selected={activeSection === section}
              aria-controls={`panel-${section}`}
              id={`tab-${section}`}
              onClick={() => setActiveSection(section)}
              className={`rounded-t-md px-4 py-2 text-sm font-medium transition-colors ${
                activeSection === section
                  ? "border-b-2 border-primary bg-primary/5 text-primary"
                  : "text-text-secondary hover:bg-surface hover:text-text-primary"
              }`}
            >
              {SECTION_LABELS[section]}
            </button>
          ))}
        </div>
      </nav>

      {/* Active Section Panel */}
      <div
        role="tabpanel"
        id={`panel-${activeSection}`}
        aria-labelledby={`tab-${activeSection}`}
        className="rounded-md border border-border bg-card p-6"
      >
        {activeSection === "preferences" && (
          <PreferencesForm
            preferences={profile.data.preferences}
            onSaved={handleSaved}
          />
        )}
        {activeSection === "skills" && (
          <SkillsForm
            skills={profile.data.skills ?? []}
            onSaved={handleSaved}
          />
        )}
        {activeSection === "education" && (
          <EducationForm
            education={profile.data.education ?? []}
            onSaved={handleSaved}
          />
        )}
        {activeSection === "links" && (
          <LinksForm
            links={profile.data.links ?? {}}
            onSaved={handleSaved}
          />
        )}
      </div>

      {/* Profile Stats Footer */}
      <div className="flex items-center gap-4 text-xs text-text-tertiary">
        <span>{profile.stats.skill_count} skills</span>
        <span>·</span>
        <span>{profile.stats.education_count} education entries</span>
        <span>·</span>
        <span>
          Last updated: {new Date(profile.updated_at).toLocaleDateString()}
        </span>
      </div>
    </div>
  );
}
