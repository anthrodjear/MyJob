/**
 * SystemConfigPageClient — client-side system configuration page.
 *
 * Fetches the effective config via useSystemConfig hook and renders
 * tabbed sections for Scoring, LLM, Voice, Approval Tiers, Automation,
 * Email & Interview, and Integrations. Each section uses its own
 * form component with independent save buttons.
 *
 * Does NOT:
 * - Handle authentication (server middleware handles this)
 * - Manage navigation (AppShell handles sidebar)
 *
 * @see hooks/useSystemConfig.ts — useSystemConfig
 * @see components/config/ — section form components
 */

"use client";

import { useState, useCallback } from "react";
import { useSystemConfig } from "@/hooks/useSystemConfig";
import { ScoringSection } from "@/components/config/ScoringSection";
import { LLMSection } from "@/components/config/LLMSection";
import { VoiceSection } from "@/components/config/VoiceSection";
import { ApprovalTiersSection } from "@/components/config/ApprovalTiersSection";
import { AutomationSection } from "@/components/config/AutomationSection";
import { EmailInterviewSection } from "@/components/config/EmailInterviewSection";
import { IntegrationsSection } from "@/components/config/IntegrationsSection";
import { Button } from "@/components/shared/Button";

/** System config page section tabs. */
const SECTIONS = [
  "scoring",
  "llm",
  "voice",
  "approval-tiers",
  "automation",
  "email-interview",
  "integrations",
] as const;
type Section = (typeof SECTIONS)[number];

/** Tab label mapping. */
const SECTION_LABELS: Record<Section, string> = {
  scoring: "Scoring",
  llm: "LLM",
  voice: "Voice",
  "approval-tiers": "Approval Tiers",
  automation: "Automation",
  "email-interview": "Email & Interview",
  integrations: "Integrations",
};

/**
 * Client-side system configuration page.
 *
 * Renders a tabbed interface for editing different parts of the system config.
 * Each tab shows a form that saves independently via PATCH.
 *
 * Loading state shows skeleton placeholders. Error state shows a retry prompt.
 *
 * @example
 *   <SystemConfigPageClient />
 */
export function SystemConfigPageClient() {
  const { data, isLoading, isError, error, refetch } = useSystemConfig();
  const [activeSection, setActiveSection] = useState<Section>("scoring");
  const [saveMessage, setSaveMessage] = useState<string | null>(null);

  const handleSaved = useCallback(() => {
    setSaveMessage("Saved successfully");
    setTimeout(() => setSaveMessage(null), 3000);
  }, []);

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

  if (isError || !data) {
    return (
      <div
        className="flex flex-col items-center justify-center py-16 text-center"
        role="alert"
        aria-live="assertive"
      >
        <h2 className="mb-2 text-lg font-semibold text-text-primary">
          Unable to Load Configuration
        </h2>
        <p className="mb-6 max-w-md text-sm text-text-secondary">
          {error instanceof Error
            ? "Failed to load system configuration. Please check your connection and try again."
            : "Failed to load system configuration. Please try again."}
        </p>
        <Button variant="primary" onClick={() => refetch()}>
          Try Again
        </Button>
      </div>
    );
  }

  const { config, version } = data;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">System Configuration</h2>
        <p className="mt-1 text-sm text-text-secondary">
          Configure scoring, LLM providers, voice, approval tiers, and automation settings.
        </p>
      </div>

      {saveMessage && (
        <div
          className="rounded-md bg-success-light p-3 text-sm text-success-dark"
          role="status"
          aria-live="polite"
        >
          {saveMessage}
        </div>
      )}

      <nav aria-label="System configuration sections" className="border-b border-border">
        <div className="flex gap-1 overflow-x-auto" role="tablist">
          {SECTIONS.map((section) => (
            <button
              key={section}
              role="tab"
              aria-selected={activeSection === section}
              aria-controls={`panel-${section}`}
              id={`tab-${section}`}
              onClick={() => setActiveSection(section)}
              className={`whitespace-nowrap rounded-t-md px-4 py-2 text-sm font-medium transition-colors ${
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

      <div
        role="tabpanel"
        id={`panel-${activeSection}`}
        aria-labelledby={`tab-${activeSection}`}
        className="rounded-md border border-border bg-surface p-6"
      >
        {activeSection === "scoring" && (
          <ScoringSection scoring={config.scoring} onSaved={handleSaved} />
        )}
        {activeSection === "llm" && (
          <LLMSection llm={config.llm} onSaved={handleSaved} />
        )}
        {activeSection === "voice" && (
          <VoiceSection voice={config.voice} onSaved={handleSaved} />
        )}
        {activeSection === "approval-tiers" && (
          <ApprovalTiersSection approvalTiers={config.approval_tiers} onSaved={handleSaved} />
        )}
        {activeSection === "automation" && (
          <AutomationSection automation={config.automation} onSaved={handleSaved} />
        )}
        {activeSection === "email-interview" && (
          <EmailInterviewSection
            email={config.email}
            interview={config.interview}
            onSaved={handleSaved}
          />
        )}
        {activeSection === "integrations" && (
          <IntegrationsSection integrations={config.integrations} />
        )}
      </div>

      {version && (
        <div className="text-xs text-text-tertiary">
          Config version: {version}
        </div>
      )}
    </div>
  );
}
