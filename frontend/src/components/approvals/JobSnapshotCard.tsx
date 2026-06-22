/**
 * JobSnapshotCard — displays the job snapshot from an approval request.
 *
 * Shows the job details at the time of scoring: title, company, location,
 * score, requirements, and link to original listing.
 *
 * @example
 *   <JobSnapshotCard snapshot={approval.job_snapshot} />
 */

import { cn } from "@/lib/utils";
import { Card, CardHeader, CardContent } from "@/components/shared/Card";
import { ExternalLink } from "lucide-react";
import type { JobSnapshot } from "@/lib/types/approvals";

interface JobSnapshotCardProps {
  /** Job snapshot data from approval request. */
  snapshot: JobSnapshot;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobSnapshotCard — job details at time of scoring.
 *
 * Accessibility:
 * - Uses semantic `<dl>` for key-value pairs
 * - External link has `target="_blank"` with `rel="noopener noreferrer"`
 * - Score uses `font-mono tabular-nums` for alignment
 */
export function JobSnapshotCard({ snapshot, className }: JobSnapshotCardProps) {
  const scorePercent = Math.round(snapshot.score);

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <h3 className="text-lg font-semibold text-text-primary">
              {snapshot.title}
            </h3>
            <p className="text-sm text-text-secondary">
              {snapshot.company}
              {snapshot.location && ` · ${snapshot.location}`}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
                "font-mono tabular-nums",
                scorePercent >= 80 && "bg-success-light text-success-dark",
                scorePercent >= 50 && scorePercent < 80 && "bg-warning-light text-warning-dark",
                scorePercent < 50 && "bg-danger-light text-danger-dark",
              )}
              aria-label={`Match score: ${scorePercent}%`}
            >
              {scorePercent}%
            </span>
            {snapshot.url && (
              <a
                href={snapshot.url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center text-text-tertiary hover:text-primary"
                aria-label={`View original listing for ${snapshot.title}`}
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {/* Description */}
        {snapshot.description && (
          <p className="mb-4 line-clamp-3 text-sm text-text-secondary">
            {snapshot.description}
          </p>
        )}

        {/* Requirements */}
        {snapshot.requirements.length > 0 && (
          <div className="mb-4">
            <h4 className="mb-1 text-xs font-medium text-text-tertiary">Requirements</h4>
            <ul className="flex flex-wrap gap-1">
              {snapshot.requirements.map((req: string, i: number) => (
                <li
                  key={i}
                  className="rounded-md bg-bg-tertiary px-2 py-0.5 text-xs text-text-secondary"
                >
                  {req}
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Metadata */}
        <dl className="flex gap-4 text-xs text-text-tertiary">
          <div>
            <dt className="font-medium">Tier</dt>
            <dd>{snapshot.tier}</dd>
          </div>
          <div>
            <dt className="font-medium">Scored</dt>
            <dd>{new Date(snapshot.scored_at).toLocaleDateString()}</dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}
