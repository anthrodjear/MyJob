/**
 * JobCompare — side-by-side job comparison table.
 *
 * Displays selected jobs in a comparison table with key fields aligned
 * for easy comparison. Supports adding/removing jobs and highlights
 * differences between compared items.
 *
 * Requires `"use client"` for event handlers.
 *
 * @example
 *   <JobCompare jobs={[job1, job2]} onRemove={handleRemove} />
 */

"use client";

import { X, Trash2, BarChart3 } from "lucide-react";
import { cn, formatDate } from "@/lib/utils";
import { Badge } from "@/components/shared/Badge";
import { SourceBadge } from "./SourceBadge";
import { MatchScoreBadge } from "./MatchScoreBadge";
import type { Job } from "@/lib/types/jobs";

/** Comparison row definition. */
interface CompareRow {
  /** Row label for screen readers. */
  label: string;
  /** Extract value from a job. */
  getValue: (job: Job) => string | number | null;
  /** Optional custom render function. */
  render?: (value: string | number | null, job: Job) => React.ReactNode;
}

/** Rows to compare. */
const COMPARE_ROWS: CompareRow[] = [
  {
    label: "Match Score",
    getValue: (job) => job.match_score,
    render: (value) => {
      if (typeof value !== "number") return <Badge>{String(value ?? "—")}</Badge>;
      return <MatchScoreBadge score={value} size="sm" />;
    },
  },
  {
    label: "Status",
    getValue: (job) => job.status,
    render: (value) => <Badge>{String(value)}</Badge>,
  },
  {
    label: "Source",
    getValue: (job) => job.source_name ?? job.source,
    render: (value, job) => <SourceBadge source={job.source} label={String(value)} size="sm" />,
  },
  {
    label: "Company",
    getValue: (job) => job.company,
  },
  {
    label: "Location",
    getValue: (job) => job.location || "—",
  },
  {
    label: "Remote",
    getValue: (job) => job.remote_type || "—",
  },
  {
    label: "Salary",
    getValue: (job) => {
      if (!job.salary_min && !job.salary_max) return "—";
      const fmt = new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: job.salary_currency || "USD",
        maximumFractionDigits: 0,
      });
      if (job.salary_min && job.salary_max) {
        return `${fmt.format(job.salary_min)} – ${fmt.format(job.salary_max)}`;
      }
      if (job.salary_min) return `From ${fmt.format(job.salary_min)}`;
      return `Up to ${fmt.format(job.salary_max)}`;
    },
  },
  {
    label: "Posted",
    getValue: (job) => job.posted_at ?? "—",
    render: (value) => formatDate(String(value)),
  },
];

/** Check if a row's values differ across all jobs. */
function isDifferent(row: CompareRow, jobs: Job[]): boolean {
  const values = jobs.map((job) => row.getValue(job));
  return new Set(values.map(String)).size > 1;
}

interface JobCompareProps {
  /** Jobs to compare (2–4 recommended). */
  jobs: Job[];
  /** Callback when a job is removed from comparison. */
  onRemove?: (jobId: string) => void;
  /** Callback to clear all jobs from comparison. */
  onClear?: () => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobCompare — side-by-side job comparison table.
 *
 * Accessibility:
 * - Table uses `<table>` with `<thead>` and `<tbody>` for semantics
 * - Row headers use `scope="row"` for screen reader association
 * - Column headers use `scope="col"`
 * - Remove buttons have descriptive `aria-label`
 * - Table is scrollable on mobile with visual scroll hint
 */
export function JobCompare({
  jobs,
  onRemove,
  onClear,
  className,
}: JobCompareProps) {
  // Empty state
  if (jobs.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-bg-secondary p-8 text-center">
        <BarChart3 className="mx-auto h-10 w-10 text-text-tertiary" aria-hidden="true" />
        <h3 className="mt-3 text-sm font-semibold text-text-primary">
          No jobs to compare
        </h3>
        <p className="mt-1 text-xs text-text-secondary">
          Select 2 or more jobs from the list to see a side-by-side comparison.
        </p>
      </div>
    );
  }

  return (
    <div className={cn("space-y-3", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 id="comparison-heading" className="text-lg font-semibold text-text-primary">
          Comparing {jobs.length} job{jobs.length !== 1 ? "s" : ""}
        </h2>
        {onClear && (
          <button
            type="button"
            onClick={onClear}
            className="flex items-center gap-1 rounded-md px-3 py-1.5 text-xs font-medium text-text-secondary hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            aria-label="Clear all jobs from comparison"
          >
            <Trash2 className="h-3 w-3" aria-hidden="true" />
            Clear All
          </button>
        )}
      </div>

      {/* Comparison Table */}
      <div
        role="region"
        aria-label="Comparison table — scroll horizontally to see all jobs"
        className="overflow-x-auto rounded-lg border border-border"
        tabIndex={0}
      >
        <table className="w-full min-w-[600px] border-collapse" aria-labelledby="comparison-heading">
          <thead>
            <tr className="border-b border-border bg-bg-tertiary">
              <th
                scope="col"
                className="p-3 text-left text-xs font-medium text-text-secondary"
              >
                Field
              </th>
              {jobs.map((job) => (
                <th
                  key={job.id}
                  scope="col"
                  className="p-3 text-left text-xs font-medium text-text-secondary"
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-text-primary">
                        {job.title}
                      </p>
                      <p className="truncate text-xs text-text-tertiary">
                        {job.company}
                      </p>
                    </div>
                    {onRemove && (
                      <button
                        type="button"
                        onClick={() => onRemove(job.id)}
                        className="flex-shrink-0 rounded p-0.5 text-text-tertiary hover:text-danger focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        aria-label={`Remove ${job.title} from comparison`}
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {COMPARE_ROWS.map((row) => {
              const diff = isDifferent(row, jobs);
              return (
                <tr
                  key={row.label}
                  className={cn(
                    "border-b border-border last:border-b-0",
                    diff && "bg-primary-light/5",
                  )}
                >
                  <th
                    scope="row"
                    className={cn(
                      "p-3 text-xs font-medium",
                      diff ? "text-primary-dark" : "text-text-secondary",
                    )}
                  >
                    {row.label}
                    {diff && (
                      <span
                        role="img"
                        aria-label="Different across jobs"
                        className="ml-1 text-[10px] text-primary"
                      >
                        ★
                      </span>
                    )}
                  </th>
                  {jobs.map((job) => {
                    const value = row.getValue(job);
                    return (
                      <td
                        key={job.id}
                        className="p-3 text-sm text-text-primary"
                      >
                        {row.render
                          ? row.render(value, job)
                          : String(value ?? "—")}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Legend */}
      <p className="text-xs text-text-tertiary">
        <span className="text-primary">★</span> indicates values differ between compared jobs.
      </p>
    </div>
  );
}
