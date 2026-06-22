/**
 * ApplicationDetail — full application detail view.
 *
 * Shows complete application data, status transition buttons,
 * timeline, and notes editor. Used on the [id] page.
 *
 * @example
 *   <ApplicationDetail application={app} />
 */

"use client";

import { useState } from "react";
import { cn, formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import { TierBadge } from "./TierBadge";
import { ApplicationTimeline } from "./ApplicationTimeline";
import { Button } from "@/components/shared/Button";
import { Card, CardHeader, CardContent } from "@/components/shared/Card";
import type { Application, ApplicationEvent, ApplicationStatus } from "@/lib/types/applications";

/** Valid status transitions from the current status. */
const VALID_TRANSITIONS: Record<ApplicationStatus, ApplicationStatus[]> = {
  draft: ["queued", "rejected"],
  queued: ["applied", "rejected"],
  applied: ["assessment", "phone_screen", "technical", "final", "offer", "rejected"],
  assessment: ["phone_screen", "technical", "final", "offer", "rejected"],
  phone_screen: ["technical", "final", "offer", "rejected"],
  technical: ["final", "offer", "rejected"],
  final: ["offer", "rejected"],
  offer: [],
  rejected: [],
};

/** Human-readable status labels. */
const STATUS_LABELS: Record<ApplicationStatus, string> = {
  draft: "Draft",
  queued: "Queued",
  applied: "Applied",
  assessment: "Assessment",
  phone_screen: "Phone Screen",
  technical: "Technical",
  final: "Final Round",
  offer: "Offer",
  rejected: "Rejected",
};

interface ApplicationDetailProps {
  /** Application data to display. */
  application: Application;
  /** Timeline events for the application. */
  timeline?: ApplicationEvent[];
  /** Callback when a status transition is triggered. */
  onStatusChange?: (id: string, status: ApplicationStatus, notes?: string) => void;
  /** Callback when notes are saved. */
  onNotesSave?: (id: string, notes: string) => void;
  /** Whether the application is being updated. */
  isUpdating?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApplicationDetail — full application detail view.
 *
 * Accessibility:
 * - Card sections use proper heading hierarchy
 * - Status transitions use `aria-label` with context
 * - Timeline announced via ApplicationTimeline
 * - aria-live for mutation status feedback
 */
export function ApplicationDetail({
  application,
  timeline,
  onStatusChange,
  onNotesSave,
  isUpdating,
  className,
}: ApplicationDetailProps) {
  const [notes, setNotes] = useState(application.notes ?? "");
  const transitions = VALID_TRANSITIONS[application.status] ?? [];

  return (
    <div className={cn("space-y-6", className)}>
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">Application Details</h2>
          <p className="mt-1 text-sm text-text-secondary">
            Created {formatDate(application.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <StatusBadge status={application.status} />
          <TierBadge tier={application.approval_tier} />
        </div>
      </div>

      {/* Key Information */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold text-text-primary">Information</h3>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <dt className="text-sm font-medium text-text-tertiary">Job ID</dt>
              <dd className="mt-1 text-sm text-text-primary">{application.job_id}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-text-tertiary">Portal</dt>
              <dd className="mt-1 text-sm text-text-primary">{application.portal_type ?? "Not specified"}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-text-tertiary">Portal URL</dt>
              <dd className="mt-1 text-sm text-text-primary">
                {application.portal_url ? (
                  <a href={application.portal_url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                    {application.portal_url}
                  </a>
                ) : "Not specified"}
              </dd>
            </div>
            {application.applied_at && (
              <div>
                <dt className="text-sm font-medium text-text-tertiary">Applied</dt>
                <dd className="mt-1 text-sm text-text-primary">{formatDate(application.applied_at)}</dd>
              </div>
            )}
            {application.response_at && (
              <div>
                <dt className="text-sm font-medium text-text-tertiary">Response</dt>
                <dd className="mt-1 text-sm text-text-primary">{formatDate(application.response_at)}</dd>
              </div>
            )}
            {application.interview_at && (
              <div>
                <dt className="text-sm font-medium text-text-tertiary">Interview</dt>
                <dd className="mt-1 text-sm text-text-primary">{formatDate(application.interview_at)}</dd>
              </div>
            )}
          </dl>
        </CardContent>
      </Card>

      {/* Status Transitions */}
      {transitions.length > 0 && onStatusChange && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold text-text-primary">Update Status</h3>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {transitions.map((target) => (
                <Button
                  key={target}
                  variant={target === "rejected" ? "danger" : "primary"}
                  size="sm"
                  disabled={isUpdating}
                  onClick={() => onStatusChange(application.id, target)}
                  aria-label={`Change status to ${STATUS_LABELS[target]}`}
                >
                  {STATUS_LABELS[target]}
                </Button>
              ))}
            </div>
            {isUpdating && (
              <p className="mt-2 text-sm text-text-tertiary" aria-live="polite">
                Updating status...
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Notes */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold text-text-primary">Notes</h3>
        </CardHeader>
        <CardContent>
          <textarea
            className="w-full rounded-md border border-border bg-bg-primary p-3 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            rows={4}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Add notes about this application..."
            aria-label="Application notes"
          />
          {onNotesSave && notes !== (application.notes ?? "") && (
            <div className="mt-3">
              <Button
                variant="primary"
                size="sm"
                onClick={() => onNotesSave(application.id, notes)}
              >
                Save Notes
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Timeline */}
      {timeline && timeline.length > 0 && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold text-text-primary">Timeline</h3>
          </CardHeader>
          <CardContent>
            <ApplicationTimeline events={timeline} />
          </CardContent>
        </Card>
      )}
    </div>
  );
}
