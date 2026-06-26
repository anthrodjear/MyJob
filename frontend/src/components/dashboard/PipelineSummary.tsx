/**
 * PipelineSummary — horizontal funnel showing application pipeline stages.
 *
 * Visualizes the flow from Draft → Queued → Applied → Assessment → Phone Screen → Technical → Final → Offer.
 * Stacks vertically on mobile, horizontal on desktop.
 * Pure Server Component — receives stats as props.
 */

import { type ApplicationStatsResponse, ApplicationStatus } from "@/lib/types/applications";
import { Card } from "@/components/shared/Card";
import { Badge } from "@/components/shared/Badge";
interface PipelineSummaryProps {
  /** Application stats from GET /applications/stats */
  stats: ApplicationStatsResponse;
}

/** Pipeline stage definition with order and label. */
const PIPELINE_STAGES: Array<{
  status: ApplicationStatus;
  label: string;
  shortLabel: string;
  color: "default" | "success" | "warning" | "info" | "danger";
}> = [
  { status: "draft", label: "Draft", shortLabel: "Draft", color: "default" },
  { status: "queued", label: "Queued", shortLabel: "Queued", color: "info" },
  { status: "applied", label: "Applied", shortLabel: "Applied", color: "info" },
  { status: "assessment", label: "Assessment", shortLabel: "Assess", color: "warning" },
  { status: "phone_screen", label: "Phone Screen", shortLabel: "Phone", color: "warning" },
  { status: "technical", label: "Technical", shortLabel: "Tech", color: "warning" },
  { status: "final", label: "Final", shortLabel: "Final", color: "success" },
  { status: "offer", label: "Offer", shortLabel: "Offer", color: "success" },
  { status: "rejected", label: "Rejected", shortLabel: "Rejected", color: "danger" },
];

/**
 * PipelineSummary — responsive pipeline visualization.
 *
 * Stacks vertically on mobile (< sm), horizontal on desktop (sm+).
 * Shows each stage with count and percentage of total.
 * Connects stages with arrows for visual flow.
 */
export function PipelineSummary({ stats }: PipelineSummaryProps) {
  const byStatus = stats.by_status ?? {};
  const total = stats.total ?? 0;

  const stages = PIPELINE_STAGES.map((stage) => {
    const count = byStatus[stage.status] ?? 0;
    const percentage = total > 0 ? Math.round((count / total) * 100) : 0;
    return { ...stage, count, percentage };
  });

  return (
    <Card>
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-text-primary">Application Pipeline</h3>
        <p className="text-sm text-text-secondary">
          {stats.total} total applications across all stages
        </p>
      </div>

      <div className="flex flex-col sm:flex-row items-center gap-2 sm:gap-2 min-w-0">
        {stages.map((stage, index) => (
          <div
            key={stage.status}
            className="flex flex-col items-center w-full sm:w-auto"
          >
            {/* Stage node */}
            <div className="flex flex-col items-center">
              <Badge variant={stage.color} className="w-24 text-center">
                {stage.count}
              </Badge>
              <span className="mt-1 text-xs text-text-tertiary">
                {stage.percentage}%
              </span>
            </div>

            {/* Stage label */}
            <span className="mt-2 text-xs font-medium text-text-secondary whitespace-nowrap">
              {stage.shortLabel}
            </span>

            {/* Arrow connector (not after last stage) */}
            {index < stages.length - 1 && (
              <span
                className="mx-2 sm:mx-2 my-2 sm:my-0 text-text-tertiary"
                aria-hidden="true"
              >
                {index < stages.length - 1 ? "→" : "↓"}
              </span>
            )}
          </div>
        ))}
      </div>
    </Card>
  );
}