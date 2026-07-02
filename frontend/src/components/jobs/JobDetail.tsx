/**
 * JobDetail — full job detail view.
 *
 * Displays all job information including description, requirements, salary,
 * match score, source, status, and application history. Provides action
 * buttons for apply, score, save, archive, and delete operations.
 * Shows similar jobs when available.
 *
 * Requires `"use client"` for event handlers and hooks.
 *
 * @example
 *   <JobDetail jobId="abc-123" onBack={() => router.back()} />
 */

"use client";

import { useState } from "react";
import {
  ExternalLink,
  Building2,
  MapPin,
  Clock,
  DollarSign,
  Trash2,
  Archive,
  Bookmark,
  BookmarkCheck,
  RotateCcw,
} from "lucide-react";
import { cn, formatDate, scoreLevel } from "@/lib/utils";
import { Badge } from "@/components/shared/Badge";
import { ProgressBar } from "@/components/shared/ProgressBar";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import { SOURCE_COLORS } from "@/lib/constants";
import { useJob, useApplyToJob, useScoreJob, useSaveJob, useUpdateJobStatus, useDeleteJob } from "@/hooks/useJobs";
import type { JobStatus } from "@/lib/types/jobs";

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

/** Format salary range for display. */
function formatSalary(
  salaryMin: number,
  salaryMax: number,
  currency: string,
): string {
  if (!salaryMin && !salaryMax) return "";
  if (salaryMin < 0 || salaryMax < 0) return "";
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

/** Shared button base classes. */
const btnBase = "rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50";
const btnPrimary = cn(btnBase, "bg-primary text-white hover:bg-primary-hover focus:ring-primary");
const btnSecondary = cn(btnBase, "border border-border bg-bg-secondary text-text-primary hover:bg-bg-tertiary focus:ring-primary");
const btnDangerIdle = cn(btnBase, "border border-border bg-bg-secondary text-text-primary hover:bg-bg-tertiary focus:ring-primary");
const btnDangerConfirm = cn(btnBase, "border bg-danger text-white hover:bg-danger-hover focus:ring-danger");

interface JobDetailProps {
  /** Job ID to fetch and display. */
  jobId: string;
  /** Callback when the user navigates back. */
  onBack?: () => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobDetail — full job detail view.
 *
 * Accessibility:
 * - Semantic headings (h1, h2) for document outline
 * - Action buttons have descriptive `aria-label` attributes
 * - External links have `target="_blank"` with `rel="noopener noreferrer"`
 * - Confirmation dialog for destructive actions (delete)
 */
export function JobDetail({ jobId, onBack, className }: JobDetailProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  // Fetch job data
  const { data: job, isLoading, error } = useJob(jobId);

  // Mutations
  const applyMutation = useApplyToJob();
  const scoreMutation = useScoreJob();
  const saveMutation = useSaveJob();
  const statusMutation = useUpdateJobStatus();
  const deleteMutation = useDeleteJob();

  // Loading state
  if (isLoading) {
    return (
      <div className={cn("space-y-4", className)} aria-busy="true">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-6 w-1/4" />
        <div className="flex gap-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-16" />
        </div>
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-4 w-full" />
        <div className="flex gap-3">
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
        </div>
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-4 w-48" />
      </div>
    );
  }

  // Error state
  if (error || !job) {
    return (
      <div
        role="alert"
        className={cn(
          "rounded-lg border border-danger-light bg-danger-light/10 p-6 text-center",
          className,
        )}
      >
        <p className="text-sm text-danger-dark">
          {error?.message ?? "Job not found"}
        </p>
        {onBack && (
          <button
            type="button"
            onClick={onBack}
            className="mt-3 text-sm text-primary hover:underline"
          >
            Go back
          </button>
        )}
      </div>
    );
  }

  const level = scoreLevel(job.match_score);
  const sourceKey = job.source as keyof typeof SOURCE_COLORS;
  const sourceColor = SOURCE_COLORS[sourceKey] ?? SOURCE_COLORS.custom;
  const remoteType = job.remote_type as RemoteType;
  const remoteLabel = REMOTE_LABELS[remoteType] ?? "";
  const statusVariant = STATUS_VARIANT[job.status as JobStatus] ?? "default";
  const scoreColor = SCORE_COLOR[level];
  const salary = formatSalary(job.salary_min, job.salary_max, job.salary_currency);

  /** Handle delete with confirmation. */
  function handleDelete() {
    if (!confirmDelete) {
      setConfirmDelete(true);
      return;
    }
    if (!job) return;
    deleteMutation.mutate(
      { jobId: job.id },
      {
        onSuccess: () => onBack?.(),
      },
    );
  }

  return (
    <article className={cn("space-y-6", className)} aria-label={`Job detail: ${job.title}`}>
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-text-primary">{job.title}</h1>
            <p className="text-lg text-text-secondary">{job.company}</p>
          </div>
          <Badge variant={statusVariant}>{job.status}</Badge>
        </div>

        {/* Meta row */}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-text-tertiary">
          {job.location && (
            <span className="flex items-center gap-1">
              <MapPin className="h-4 w-4" aria-hidden="true" />
              {job.location}
            </span>
          )}
          {remoteLabel && (
            <span className="flex items-center gap-1">
              <Building2 className="h-4 w-4" aria-hidden="true" />
              {remoteLabel}
            </span>
          )}
          <span className={cn("rounded px-1.5 py-0.5 font-medium", sourceColor)}>
            {job.source_name || job.source}
          </span>
          {job.posted_at && (
            <span className="flex items-center gap-1">
              <Clock className="h-4 w-4" aria-hidden="true" />
              Posted {formatDate(job.posted_at)}
            </span>
          )}
        </div>
      </div>

      {/* Salary */}
      {salary && (
        <div className="flex items-center gap-2 text-lg font-semibold text-text-primary">
          <DollarSign className="h-5 w-5 text-text-tertiary" aria-hidden="true" />
          {salary}
        </div>
      )}

      {/* Match Score */}
      <div className="rounded-lg border border-border bg-bg-secondary p-4">
        <ProgressBar
          value={job.match_score}
          label="Match Score"
          color={scoreColor}
        />
      </div>

      {/* Action buttons */}
      <div className="flex flex-wrap gap-3">
        {job.status !== "applied" && (
          <button
            type="button"
            onClick={() => applyMutation.mutate({ jobId: job.id })}
            disabled={applyMutation.isPending}
            className={btnPrimary}
            aria-label={`Apply to ${job.title} at ${job.company}`}
          >
            {applyMutation.isPending ? "Applying..." : "Apply"}
          </button>
        )}
        <button
          type="button"
          onClick={() => scoreMutation.mutate({ jobId: job.id })}
          disabled={scoreMutation.isPending}
          className={btnSecondary}
          aria-label={`Re-score ${job.title}`}
        >
          <RotateCcw className="mr-1.5 inline h-4 w-4" aria-hidden="true" />
          {scoreMutation.isPending ? "Scoring..." : "Re-score"}
        </button>
        <button
          type="button"
          onClick={() => saveMutation.mutate({ jobId: job.id, save: !job.match_details?.saved })}
          disabled={saveMutation.isPending}
          className={btnSecondary}
          aria-label={job.match_details?.saved ? `Unsave ${job.title}` : `Save ${job.title}`}
        >
          {job.match_details?.saved ? (
            <BookmarkCheck className="mr-1.5 inline h-4 w-4" aria-hidden="true" />
          ) : (
            <Bookmark className="mr-1.5 inline h-4 w-4" aria-hidden="true" />
          )}
          {job.match_details?.saved ? "Saved" : "Save"}
        </button>
        {job.status !== "archived" && (
          <button
            type="button"
            onClick={() => statusMutation.mutate({ jobId: job.id, status: "archived" })}
            disabled={statusMutation.isPending}
            className={btnSecondary}
            aria-label={`Archive ${job.title}`}
          >
            <Archive className="mr-1.5 inline h-4 w-4" aria-hidden="true" />
            Archive
          </button>
        )}
        <button
          type="button"
          onClick={handleDelete}
          disabled={deleteMutation.isPending}
          className={confirmDelete ? btnDangerConfirm : btnDangerIdle}
          aria-label={confirmDelete ? `Confirm delete ${job.title}` : `Delete ${job.title}`}
        >
          <Trash2 className="mr-1.5 inline h-4 w-4" aria-hidden="true" />
          {confirmDelete ? "Confirm Delete" : "Delete"}
        </button>
      </div>

      {/* Mutation status for screen readers */}
      <div aria-live="polite" className="sr-only">
        {applyMutation.isPending && "Applying to job..."}
        {applyMutation.isSuccess && "Successfully applied to job"}
        {applyMutation.isError && "Failed to apply to job"}
        {scoreMutation.isPending && "Scoring job..."}
        {scoreMutation.isSuccess && "Job scored successfully"}
        {scoreMutation.isError && "Failed to score job"}
        {saveMutation.isPending && "Saving job..."}
        {saveMutation.isSuccess && "Job saved successfully"}
        {saveMutation.isError && "Failed to save job"}
        {statusMutation.isPending && "Updating job status..."}
        {statusMutation.isSuccess && "Job status updated"}
        {statusMutation.isError && "Failed to update job status"}
        {deleteMutation.isPending && "Deleting job..."}
        {deleteMutation.isSuccess && "Job deleted"}
        {deleteMutation.isError && "Failed to delete job"}
      </div>

      {/* Description */}
      {job.description && (
        <section aria-labelledby="job-description">
          <h2 id="job-description" className="mb-2 text-lg font-semibold text-text-primary">
            Description
          </h2>
          <div className="prose prose-sm max-w-none text-text-secondary">
            {job.description}
          </div>
        </section>
      )}

      {/* Requirements */}
      {job.requirements && (
        <section aria-labelledby="job-requirements">
          <h2 id="job-requirements" className="mb-2 text-lg font-semibold text-text-primary">
            Requirements
          </h2>
          <div className="prose prose-sm max-w-none text-text-secondary">
            {job.requirements}
          </div>
        </section>
      )}

      {/* Links */}
      {(job.url || job.application_url || job.company_url) && (
        <section aria-label="Job links" className="flex flex-wrap gap-3">
          {job.url && (
            <a
              href={job.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
            >
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
              View Original Listing
            </a>
          )}
          {job.application_url && (
            <a
              href={job.application_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
            >
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
              Application Portal
            </a>
          )}
          {job.company_url && (
            <a
              href={job.company_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
            >
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
              Company Website
            </a>
          )}
        </section>
      )}

      {/* Timestamps */}
      <div className="border-t border-border pt-4 text-xs text-text-tertiary">
        <span className="sr-only">Job metadata: </span>
        <span>Scraped {formatDate(job.scraped_at)}</span>
        <span className="mx-2">·</span>
        <span>Updated {formatDate(job.updated_at)}</span>
      </div>
    </article>
  );
}
