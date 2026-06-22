/**
 * ApplicationTimeline — audit trail of status transitions.
 *
 * Shows a vertical timeline of application events with timestamps,
 * old/new status, and notes. Used in the application detail view.
 *
 * @example
 *   <ApplicationTimeline events={events} />
 */

import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import type { ApplicationEvent, ApplicationStatus } from "@/lib/types/applications";

interface ApplicationTimelineProps {
  /** Timeline events from GET /applications/:id/events. */
  events: ApplicationEvent[];
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApplicationTimeline — audit trail display.
 *
 * Accessibility:
 * - Uses `<ol>` for ordered timeline
 * - Each event has descriptive text for screen readers
 * - Status transitions are announced via StatusBadge
 */
export function ApplicationTimeline({ events, className }: ApplicationTimelineProps) {
  if (events.length === 0) {
    return (
      <div className={cn("py-8 text-center text-text-tertiary", className)}>
        <p className="text-sm">No status changes recorded yet.</p>
      </div>
    );
  }

  return (
    <ol className={cn("space-y-4", className)} aria-label="Application timeline">
      {events.map((event) => (
        <li key={event.id} className="relative flex gap-3">
          {/* Vertical line — decorative, hidden from screen readers */}
          <div className="flex flex-col items-center" aria-hidden="true">
            <div className="h-2 w-2 rounded-full bg-primary" />
            <div className="mt-1 w-px flex-1 bg-border" />
          </div>

          {/* Event content */}
          <div className="flex-1 pb-4">
            <div className="flex flex-wrap items-center gap-2">
              {event.old_status && (
                <StatusBadge status={event.old_status as ApplicationStatus} />
              )}
              {event.old_status && (
                <span className="text-text-tertiary" aria-hidden="true">→</span>
              )}
              <StatusBadge status={event.new_status as ApplicationStatus} />
            </div>
            {event.notes && (
              <p className="mt-1 text-sm text-text-secondary">{event.notes}</p>
            )}
            <time dateTime={event.created_at} className="mt-1 block text-xs text-text-tertiary">
              {formatDate(event.created_at)}
            </time>
          </div>
        </li>
      ))}
    </ol>
  );
}
