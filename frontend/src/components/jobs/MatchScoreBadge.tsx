/**
 * MatchScoreBadge — color-coded match score indicator.
 *
 * Displays a job match score as a percentage with semantic color coding.
 * High (≥80): success, Medium (50–79): warning, Low (<50): danger.
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <MatchScoreBadge score={92} />
 *   <MatchScoreBadge score={65} size="lg" />
 */

import { cn, scoreLevel, formatScore, type ScoreLevel } from "@/lib/utils";

/** Score level → visual variant mapping. */
const LEVEL_CONFIG = {
  high: {
    bg: "bg-success-light",
    text: "text-success-dark",
    border: "border-success",
  },
  medium: {
    bg: "bg-warning-light",
    text: "text-warning-dark",
    border: "border-warning",
  },
  low: {
    bg: "bg-danger-light",
    text: "text-danger-dark",
    border: "border-danger",
  },
} as const;

/** Badge size variants. */
type BadgeSize = "sm" | "md" | "lg";

interface MatchScoreBadgeProps {
  /** Match score value (0–100). */
  score: number;
  /** Size variant. Default: "md". */
  size?: BadgeSize;
  /** Additional CSS classes. */
  className?: string;
}

/** Size-to-Tailwind class mapping. */
const sizeStyles: Record<BadgeSize, string> = {
  sm: "px-2 py-0.5 text-xs",
  md: "px-2.5 py-1 text-sm",
  lg: "px-3 py-1.5 text-base",
};

/**
 * MatchScoreBadge — color-coded match score indicator.
 *
 * Accessibility:
 * - `role="status"` with `aria-label` announces the score to screen readers
 * - Color is not the sole indicator — text label provides context
 * - High contrast text on colored background
 */
export function MatchScoreBadge({
  score,
  size = "md",
  className,
}: MatchScoreBadgeProps) {
  const level = scoreLevel(score);
  const config = LEVEL_CONFIG[level];
  const formatted = formatScore(score);

  /** Human-readable label for screen readers. */
  function levelLabel(l: ScoreLevel): string {
    if (l === "high") return "High match";
    if (l === "medium") return "Medium match";
    return "Low match";
  }

  return (
    <span
      role="status"
      aria-label={`${formatted} match — ${levelLabel(level)}`}
      className={cn(
        "inline-flex items-center rounded-full border font-mono font-semibold tabular-nums",
        config.bg,
        config.text,
        config.border,
        sizeStyles[size],
        className,
      )}
    >
      {formatted}
    </span>
  );
}
