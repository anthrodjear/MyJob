/**
 * ApplicationCard — individual application listing display.
 *
 * Shows job title, company, status, tier, and key dates.
 * Provides action buttons for status transitions.
 *
 * @example
 *   <ApplicationCard application={app} onStatusChange={handleStatus} />
 */

"use client";

import { cn, formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import { TierBadge } from "./TierBadge";
import { Button } from "@/components/shared/Button";
import type { Application } from "@/lib/types/applications";

/** Next valid transitions for each status (subset for UI actions). */
const QUICK_ACTIONS: Partial<Record<Application["status"], { label: string; target: Application["status"] }[]>> = {
  draft: [{ label: "Submit", target: "queued" }],
  queued: [{ label: "Mark Applied", target: "applied" }],
};

interface ApplicationCardProps {
  /** Application data to display. */
  application: Application;
  /** Job title (joined from job relation). */
  jobTitle?: string;
  /** Company name (joined from job relation). */
  company?: string;
  /** Callback when a status transition is triggered. */
  onStatusChange?: (id: string, status: Application["status"]) => void;
  /** Callback when the card is clicked (navigate to detail). */
  onClick?: (id: string) => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApplicationCard — application listing display.
 *
 * Accessibility:
 * - `role="group"` with `aria-label` identifies the card
 * - Action buttons have descriptive `aria-label` attributes
 * - Status and tier use semantic badge components
 */
export function ApplicationCard({
  application,
  jobTitle,
  company,
  onStatusChange,
  onClick,
  className,
}: ApplicationCardProps) {
  const actions = QUICK_ACTIONS[application.status] ?? [];

  return (
    <div
      className={cn(
        "group rounded-lg border border-border bg-bg-secondary p-4 transition-colors hover:border-border-hover",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2",
        onClick && "cursor-pointer",
        className,
      )}
      role="group"
      aria-label={`Application: ${jobTitle ?? "Unknown job"} at ${company ?? "Unknown company"}`}
      onClick={onClick ? () => onClick(application.id) : undefined}
      onKeyDown={onClick ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onClick(application.id); } } : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Header: Title + Company */}
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-text-primary">
            {jobTitle ?? "Unknown Job"}
          </h3>
          <p className="truncate text-sm text-text-secondary">
            {company ?? "Unknown Company"}
          </p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <StatusBadge status={application.status} />
          <TierBadge tier={application.approval_tier} />
        </div>
      </div>

      {/* Dates */}
      <div className="mb-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-tertiary">
        {application.applied_at && (
          <span>Applied {formatDate(application.applied_at)}</span>
        )}
        {application.response_at && (
          <span>Response {formatDate(application.response_at)}</span>
        )}
        {application.interview_at && (
          <span>Interview {formatDate(application.interview_at)}</span>
        )}
        {!application.applied_at && (
          <span>Created {formatDate(application.created_at)}</span>
        )}
      </div>

      {/* Notes preview */}
      {application.notes && (
        <p className="mb-3 line-clamp-2 text-sm text-text-secondary">
          {application.notes}
        </p>
      )}

      {/* Actions */}
      {actions.length > 0 && onStatusChange && (
        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <Button
              key={action.target}
              variant="primary"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onStatusChange(application.id, action.target);
              }}
              aria-label={`${action.label} for ${jobTitle ?? "application"}`}
            >
              {action.label}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}
