/**
 * DashboardStats — 4 KPI cards for the dashboard header.
 *
 * Displays key metrics: Total Applications, Pending Approvals, Match Rate, Jobs This Week.
 * Pure Server Component — receives data as props.
 */

import { type ApplicationStatsResponse } from "@/lib/types/applications";
import { Card } from "@/components/shared/Card";
import { Badge } from "@/components/shared/Badge";
import { FileText, Clock, Target, XCircle } from "lucide-react";

// Accent color mapping per stat type
const accentColors = {
  blue: {
    border: "border-l-4 border-l-blue-500",
    iconBg: "bg-blue-50 text-blue-500",
    valueGradient: "from-blue-600 to-blue-400",
  },
  amber: {
    border: "border-l-4 border-l-amber-500",
    iconBg: "bg-amber-50 text-amber-500",
    valueGradient: "from-amber-600 to-amber-400",
  },
  green: {
    border: "border-l-4 border-l-green-500",
    iconBg: "bg-green-50 text-green-500",
    valueGradient: "from-green-600 to-green-400",
  },
  red: {
    border: "border-l-4 border-l-red-500",
    iconBg: "bg-red-50 text-red-500",
    valueGradient: "from-red-600 to-red-400",
  },
} as const;

interface DashboardStatsProps {
  /** Application stats from GET /applications/stats */
  stats: ApplicationStatsResponse;
}

/**
 * DashboardStats — 4 KPI cards in a responsive grid.
 *
 * Cards:
 * - Total Applications (all time)
 * - Pending Approvals (review tier)
 * - Match Rate (average score of applied jobs)
 * - Jobs This Week (discovered in last 7 days)
 */
export function DashboardStats({ stats }: DashboardStatsProps) {
  const pendingApprovals = stats.by_tier?.review ?? 0;
  const autoApproved = stats.by_tier?.auto ?? 0;
  const rejected = stats.by_tier?.reject ?? 0;

  // Calculate match rate from applications with scores
  // Note: backend doesn't provide average score directly, so we show placeholder
  // In a real implementation, this would come from a dedicated stats endpoint
  const matchRate = stats.total > 0 ? Math.round((autoApproved / stats.total) * 100) : 0;

  const statCards = [
    {
      label: "Total Applications",
      value: stats.total.toLocaleString(),
      icon: <FileText className="h-8 w-8" aria-hidden="true" />,
      trend: null,
      accent: "blue" as const,
    },
    {
      label: "Pending Approvals",
      value: pendingApprovals.toLocaleString(),
      icon: <Clock className="h-8 w-8" aria-hidden="true" />,
      trend: pendingApprovals > 0 ? `+${pendingApprovals}` : null,
      variant: "warning" as const,
      accent: "amber" as const,
    },
    {
      label: "Auto-Approved %",
      value: `${matchRate}%`,
      icon: <Target className="h-8 w-8" aria-hidden="true" />,
      trend: matchRate > 0 ? `${matchRate}%` : null,
      variant: "success" as const,
      accent: "green" as const,
    },
    {
      label: "Rejected",
      value: rejected.toLocaleString(),
      icon: <XCircle className="h-8 w-8" aria-hidden="true" />,
      trend: rejected > 0 ? `-${rejected}` : null,
      variant: "danger" as const,
      accent: "red" as const,
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {statCards.map((stat, index) => {
        const colors = accentColors[stat.accent];
        return (
          <Card
            key={stat.label}
            className={`min-h-[100px] ${colors.border} transition-all duration-200 hover:-translate-y-2 hover:shadow-lg animate-fade-in-up`}
            style={{ animationDelay: `${index * 0.05}s` }}
          >
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-text-secondary">{stat.label}</p>
                <p
                  className={`mt-1 text-2xl font-bold bg-clip-text bg-gradient-to-r ${colors.valueGradient} text-transparent`}
                >
                  {stat.value}
                </p>
                {stat.trend && (
                  <p className="mt-1 text-xs font-medium">
                    <Badge variant={stat.variant ?? "info"} className="gap-1">
                      <span>{stat.trend}</span>
                    </Badge>
                  </p>
                )}
              </div>
              <span
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${colors.iconBg}`}
                aria-hidden="true"
              >
                {stat.icon}
              </span>
            </div>
          </Card>
        );
      })}
    </div>
  );
}
