/**
 * ActivityFeed — recent activity list for the dashboard.
 *
 * Shows the 10 most recent activity log entries with icons and timestamps.
 * Pure Server Component — receives data as props.
 */

import { type ActivityResponse, type ActivityEventType } from "@/lib/types/activity";
import { Card } from "@/components/shared/Card";
import { Badge } from "@/components/shared/Badge";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { ClipboardList } from "lucide-react";
import {
  FileText,
  GitCompare,
  Send,
  SearchCheck,
  Target,
  Mail,
  Tag,
  Settings,
  CheckCircle,
  XCircle,
  User,
} from "lucide-react";

interface ActivityFeedProps {
  /** Activity entries from GET /activity-logs?limit=10 */
  activities: ActivityResponse[];
}

/** Icon mapping for activity event types. */
const eventIcons: Record<ActivityEventType, React.ReactNode> = {
  application_created: <FileText className="h-5 w-5" aria-hidden="true" />,
  application_status_changed: <GitCompare className="h-5 w-5" aria-hidden="true" />,
  application_submitted: <Send className="h-5 w-5" aria-hidden="true" />,
  job_discovered: <SearchCheck className="h-5 w-5" aria-hidden="true" />,
  job_scored: <Target className="h-5 w-5" aria-hidden="true" />,
  email_received: <Mail className="h-5 w-5" aria-hidden="true" />,
  email_classified: <Tag className="h-5 w-5" aria-hidden="true" />,
  task_created: <Settings className="h-5 w-5" aria-hidden="true" />,
  task_completed: <CheckCircle className="h-5 w-5" aria-hidden="true" />,
  task_failed: <XCircle className="h-5 w-5" aria-hidden="true" />,
  profile_updated: <User className="h-5 w-5" aria-hidden="true" />,
};

/** Human-readable labels for event types. */
const eventLabels: Record<ActivityEventType, string> = {
  application_created: "Application created",
  application_status_changed: "Status changed",
  application_submitted: "Application submitted",
  job_discovered: "Job discovered",
  job_scored: "Job scored",
  email_received: "Email received",
  email_classified: "Email classified",
  task_created: "Task created",
  task_completed: "Task completed",
  task_failed: "Task failed",
  profile_updated: "Profile updated",
};

/** Variant mapping for badge colors. */
const eventVariants: Record<ActivityEventType, "default" | "success" | "warning" | "danger" | "info"> = {
  application_created: "info",
  application_status_changed: "warning",
  application_submitted: "success",
  job_discovered: "info",
  job_scored: "success",
  email_received: "info",
  email_classified: "default",
  task_created: "info",
  task_completed: "success",
  task_failed: "danger",
  profile_updated: "info",
};

/** Format activity details for display (no raw JSON). */
function formatActivityDetails(eventType: ActivityEventType, details: Record<string, unknown>): string {
  switch (eventType) {
    case "application_status_changed":
      return `Changed from ${details.old_status ?? ""} to ${details.new_status ?? ""}`;
    case "job_scored":
      return details.score != null ? `Score: ${details.score}%` : "";
    case "email_received":
      return details.from ? `From: ${details.from}` : "";
    case "application_submitted":
      return details.portal ? `Via ${details.portal}` : "";
    case "job_discovered":
      return details.source ? `Source: ${details.source}` : "";
    case "task_completed":
      return details.result ? `Result: ${JSON.stringify(details.result)}` : "";
    case "task_failed":
      return details.error ? `Error: ${details.error}` : "";
    case "profile_updated":
      return details.field ? `Updated: ${details.field}` : "";
    default:
      return "";
  }
}

/**
 * ActivityFeed — recent activity timeline.
 *
 * Renders a list of activity entries with icon, label, timestamp, and details.
 * Shows "No recent activity" when empty.
 */
export function ActivityFeed({ activities }: ActivityFeedProps) {
  if (activities.length === 0) {
    return (
      <Card>
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <ClipboardList className="h-12 w-12 text-text-tertiary mb-2" aria-hidden="true" />
          <p className="text-text-secondary">No recent activity</p>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-text-primary">Recent Activity</h3>
        <p className="text-sm text-text-secondary">
          Latest events across your job search
        </p>
      </div>

      <div className="space-y-3" role="list" aria-label="Recent activity">
        {activities.map((activity) => {
          const icon = eventIcons[activity.event_type] ?? <Settings className="h-5 w-5" aria-hidden="true" />;
          const label = eventLabels[activity.event_type] ?? activity.event_type;
          const variant = eventVariants[activity.event_type] ?? "default";
          const formattedDetails = formatActivityDetails(activity.event_type, activity.details ?? {});

          return (
            <div
              key={activity.id}
              role="listitem"
              className="flex items-start gap-3 p-3 rounded-lg bg-bg-secondary hover:bg-bg-tertiary transition-colors"
            >
              <span className="flex-shrink-0 text-text-tertiary" aria-hidden="true">
                {icon}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-medium text-text-primary">{label}</span>
                  <Badge variant={variant} className="text-xs">
                    {activity.entity_type}
                  </Badge>
                  <span className="text-xs text-text-tertiary ml-auto whitespace-nowrap font-mono tabular-nums">
                    {formatDate(activity.created_at)}
                  </span>
                </div>
                {formattedDetails && (
                  <p className="mt-1 text-sm text-text-secondary">
                    {formattedDetails}
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
}