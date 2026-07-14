/**
 * IntegrationsSection — read-only integration status display.
 *
 * Shows connection health for LiveKit, email, and AI providers.
 * This is a display-only component — status is determined by the backend,
 * not editable via config overrides.
 *
 * Does NOT:
 * - Allow editing integration settings (those are in other sections)
 * - Handle authentication state
 *
 * @see lib/types/config.ts — IntegrationsSection
 */

"use client";

import type { IntegrationsSection as IntegrationsSectionType } from "@/lib/types/config";

/** Props for IntegrationsSection. */
interface IntegrationsSectionProps {
  /** Current integrations status from the config. */
  integrations: IntegrationsSectionType;
}

/** Status badge color mapping. */
const STATUS_STYLES: Record<string, string> = {
  connected: "bg-success-light text-success-dark",
  disconnected: "bg-surface text-text-secondary",
  error: "bg-danger-light text-danger-dark",
};

/** Status label mapping. */
const STATUS_LABELS: Record<string, string> = {
  connected: "Connected",
  disconnected: "Disconnected",
  error: "Error",
};

/**
 * Display component for integration connection status.
 *
 * Renders a list of integrations with their current health status.
 * Read-only — no form controls.
 *
 * @example
 *   <IntegrationsSection integrations={config.integrations} />
 */
export function IntegrationsSection({ integrations }: IntegrationsSectionProps) {
  const renderStatus = (status: string, url?: string) => (
    <span
      className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${STATUS_STYLES[status] ?? STATUS_STYLES.disconnected}`}
    >
      {STATUS_LABELS[status] ?? status}
      {url && (
        <span className="ml-1 text-text-tertiary truncate max-w-[200px]">{url}</span>
      )}
    </span>
  );

  return (
    <div className="space-y-4">
      <p className="text-sm text-text-secondary">
        Connection status for external services. Status is determined automatically by the backend.
      </p>

      <div className="space-y-3">
        {/* LiveKit */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <span className="text-sm font-medium text-text-primary">LiveKit</span>
            <p className="text-xs text-text-secondary">Voice interview server</p>
          </div>
          {renderStatus(integrations.livekit.status, integrations.livekit.url)}
        </div>

        {/* Email */}
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <span className="text-sm font-medium text-text-primary">Email</span>
            <p className="text-xs text-text-secondary">Email polling service</p>
          </div>
          {renderStatus(integrations.email.status, integrations.email.url)}
        </div>

        {/* AI Providers */}
        {Object.entries(integrations.ai_providers).map(([name, info]) => (
          <div key={name} className="flex items-center justify-between rounded-md border border-border p-3">
            <div>
              <span className="text-sm font-medium text-text-primary capitalize">{name}</span>
              <p className="text-xs text-text-secondary">Model: {info.model}</p>
            </div>
            {renderStatus(info.status)}
          </div>
        ))}
      </div>
    </div>
  );
}
