/**
 * JobCard — individual job listing display.
 *
 * Shows job title, company, location, match score, source, status, and salary.
 * Provides action buttons for apply, score, save, and archive operations.
 * Mobile-responsive with proper ARIA labels and keyboard navigation.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 * Action callbacks are optional; buttons only render when callbacks are provided.
 *
 * @example
 *   <JobCard job={job} onApply={handleApply} onSave={handleSave} />
 */

import { cn } from "@/lib/utils";
import { formatDate, scoreLevel } from "@/lib/utils";
import { Badge } from "@/components/shared/Badge";
import { ProgressBar } from "@/components/shared/ProgressBar";
import { SOURCE_COLORS } from "@/lib/constants";
import type { Job, JobStatus } from "@/lib/types/jobs";

/** Remote type union for type-safe lookups. */
type RemoteType = "remote" | "hybrid" | "onsite" | "unknown";

/** Remote type display labels. */
const REMOTE_LABELS: Record<RemoteType, string> = {
  remote: "Remote",
  hybrid: "Hybrid",
  onsite: "On-site",
  unknown: "",
};

/** Job status → Badge variant mapping. */
const STATUS_VARIANT: Record<JobStatus, "default" | "success" | "warning" | "info"> = {
  discovered: "default",
  matched: "info",
  applied: "success",
  archived: "default",
};

/** Score level → ProgressBar color mapping. */
const SCORE_COLOR: Record<"high" | "medium" | "low", "success" | "warning" | "danger"> = {
  high: "success",
  medium: "warning",
  low: "danger",
};

/** Shared button styles for secondary actions. */
const secondaryBtn =
  "rounded-md border border-border bg-bg-secondary px-3 py-1.5 text-xs font-medium text-text-primary transition-colors hover:bg-bg-tertiary focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2";

/**
 * Format salary range for display.
 * Returns empty string if no salary data is available.
 */
function formatSalary(
  salaryMin: number,
  salaryMax: number,
  currency: string,
): string {
  if (!salaryMin && !salaryMax) return "";
  const fmt = new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
    maximumFractionDigits: 0,
  });
  if (salaryMin && salaryMax) {
    return `${fmt.format(salaryMin)} – ${fmt.format(salaryMax)}`;
  }
  if (salaryMin) return `From ${fmt.format(salaryMin)}`;
  return `Up to ${fmt.format(salaryMax)}`;
}

interface JobCardProps {
  /** Job data to display. */
  job: Job;
  /** Callback when Apply button is clicked. */
  onApply?: (jobId: string) => void;
  /** Callback when Score button is clicked. */
  onScore?: (jobId: string) => void;
  /** Callback when Save/Unsave button is clicked. */
  onSave?: (jobId: string, saved: boolean) => void;
  /** Callback when Archive button is clicked. */
  onArchive?: (jobId: string) => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobCard — individual job listing display.
 *
 * Accessibility:
 * - `role="group"` with `aria-label` identifies the card
 * - Action buttons have descriptive `aria-label` attributes
 * - Match score has both visual bar and text percentage
 * - Salary range is announced by screen readers
 */
export function JobCard({
  job,
  onApply,
  onScore,
  onSave,
  onArchive,
  className,
}: JobCardProps) {
  const level = scoreLevel(job.match_score);
  const sourceKey = job.source as keyof typeof SOURCE_COLORS;
  const sourceColor = SOURCE_COLORS[sourceKey] ?? SOURCE_COLORS.custom;
  const remoteType = job.remote_type as RemoteType;
  const remoteLabel = REMOTE_LABELS[remoteType] ?? "";
  const statusVariant = STATUS_VARIANT[job.status as JobStatus] ?? "default";
  const scoreColor = SCORE_COLOR[level];
  const salary = formatSalary(job.salary_min, job.salary_max, job.salary_currency);

  return (
    <div
      className={cn(
        "group rounded-lg border border-border bg-bg-secondary p-4 transition-colors hover:border-border-hover",
        className,
      )}
      role="group"
      aria-label={`Job: ${job.title} at ${job.company}`}
    >
      {/* Header: Title + Company + Status */}
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-text-primary">
            {job.title}
          </h3>
          <p className="truncate text-sm text-text-secondary">{job.company}</p>
        </div>
        <Badge variant={statusVariant}>{job.status}</Badge>
      </div>

      {/* Meta: Location, Remote, Source, Posted Date */}
      <div className="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-tertiary">
        {job.location && <span>{job.location}</span>}
        {remoteLabel && <span className="rounded bg-bg-tertiary px-1.5 py-0.5">{remoteLabel}</span>}
        <span className={cn("rounded px-1.5 py-0.5 font-medium", sourceColor)}>
          {job.source_name || job.source}
        </span>
        {job.posted_at && <span>Posted {formatDate(job.posted_at)}</span>}
      </div>

      {/* Salary */}
      {salary && (
        <p className="mb-3 text-sm font-medium text-text-primary">{salary}</p>
      )}

      {/* Match Score */}
      <div className="mb-4">
        <ProgressBar
          value={job.match_score}
          label="Match Score"
          color={scoreColor}
        />
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-2">
        {onApply && job.status !== "applied" && (
          <button
            type="button"
            onClick={() => onApply(job.id)}
            className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-primary-hover focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
            aria-label={`Apply to ${job.title} at ${job.company}`}
          >
            Apply
          </button>
        )}
        {onScore && (
          <button
            type="button"
            onClick={() => onScore(job.id)}
            className={secondaryBtn}
            aria-label={`Re-score ${job.title}`}
          >
            Re-score
          </button>
        )}
        {onSave && (
          <button
            type="button"
            onClick={() => onSave(job.id, !job.match_details?.saved)}
            className={secondaryBtn}
            aria-label={job.match_details?.saved ? `Unsave ${job.title}` : `Save ${job.title}`}
          >
            {job.match_details?.saved ? "Unsave" : "Save"}
          </button>
        )}
        {onArchive && job.status !== "archived" && (
          <button
            type="button"
            onClick={() => onArchive(job.id)}
            className={secondaryBtn}
            aria-label={`Archive ${job.title}`}
          >
            Archive
          </button>
        )}
      </div>
    </div>
  );
}
