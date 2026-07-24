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

/** Color map for the visual indicator dots and connecting lines. */
const COLOR_MAP: Record<string, string> = {
  default: "bg-gray-400",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  info: "bg-sky-500",
  danger: "bg-red-500",
};

/**
 * PipelineSummary — responsive pipeline visualization.
 *
 * Stacks vertically on mobile (< sm), horizontal on desktop (sm+).
 * Shows each stage with count, percentage, and visual flow indicators.
 * Connects stages with a subtle horizontal line for visual flow.
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
      {/* Header */}
      <div className="mb-5">
        <h3 className="text-lg font-semibold text-text-primary">Application Pipeline</h3>
        <p className="text-sm text-text-secondary mt-1">
          {stats.total} total applications across all stages
        </p>
      </div>

      {/* Funnel steps — vertical on mobile, horizontal on desktop */}
      <div className="relative flex flex-col sm:flex-row items-stretch sm:items-center gap-0 sm:gap-0">
        {stages.map((stage, index) => (
          <div
            key={stage.status}
            className={`relative flex flex-col items-center w-full sm:w-auto group
                transition-all duration-200 ease-out
                hover:bg-primary/10 hover:rounded-lg cursor-default`}
            style={{ animationDelay: `${index * 60}ms` }}
          >
            {/* Step indicator with connecting line to the right */}
            <div className="flex items-center w-full sm:w-auto">
              {/* Colored visual indicator */}
              <div
                className={`
                  w-3 h-3 rounded-full flex-shrink-0
                  ${COLOR_MAP[stage.color] ?? COLOR_MAP["default"]}
                   ring-2 ring-offset-1 ring-offset-white transition-transform duration-200
                  group-hover:scale-125
                `}
                aria-hidden="true"
              />

              {/* Connecting line (not after last stage) */}
              {index < stages.length - 1 && (
                <div
                  className={`hidden sm:block flex-1 h-[2px] mx-1 bg-gradient-to-r from-border to-border/50`}
                  aria-hidden="true"
                />
              )}
            </div>

            {/* Stage content */}
            <div className="mt-3 flex flex-col items-center">
              {/* Count badge */}
              <Badge
                variant={stage.color}
                className="px-3 py-1 text-sm font-semibold"
              >
                {stage.count}
              </Badge>

              {/* Percentage — more prominent */}
              <span
                className={`mt-1.5 text-sm font-bold tabular-nums                 ${stage.percentage > 0 ? "text-text-primary" : "text-text-tertiary"}`}
              >
                {stage.percentage}%
              </span>

              {/* Stage label */}
              <span className="mt-1.5 text-[11px] font-medium text-text-tertiary whitespace-nowrap tracking-wide uppercase">
                {stage.shortLabel}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Summary bar at the bottom */}
      <div
        className={`mt-4 pt-3 flex items-center justify-between border-t border-border/60`}
      >
        <span className="text-xs font-medium text-text-secondary">
          Total Pipeline
        </span>
        <span className="text-sm font-bold text-text-primary tabular-nums">
          {total} applications
        </span>
      </div>
    </Card>
  );
}