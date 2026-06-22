/**
 * SourceBadge — job source indicator with color coding.
 *
 * Displays a job source name (e.g., Indeed, LinkedIn, Greenhouse) with
 * semantic color coding. Uses SOURCE_COLORS from constants for consistent
 * branding across the application.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <SourceBadge source="indeed" />
 *   <SourceBadge source="linkedin" size="lg" />
 */

import { cn } from "@/lib/utils";
import { SOURCE_COLORS, type SourceKey } from "@/lib/constants";

/** Badge size variants. */
type BadgeSize = "sm" | "md" | "lg";

interface SourceBadgeProps {
  /** Source identifier. Known sources get branded colors; unknowns get fallback styling. */
  source: SourceKey | string;
  /** Display label override. Falls back to capitalized source name. */
  label?: string;
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

/** Fallback color for unknown sources. */
const FALLBACK_STYLE = "bg-bg-tertiary text-text-primary";

/**
 * SourceBadge — job source indicator with color coding.
 *
 * Accessibility:
 * - `aria-label` announces the source to screen readers
 * - Color is supplementary — text label is always present
 * - High contrast text on colored background
 */
export function SourceBadge({
  source,
  label,
  size = "md",
  className,
}: SourceBadgeProps) {
  const sourceKey = source as SourceKey;
  const colorClass = SOURCE_COLORS[sourceKey] ?? FALLBACK_STYLE;
  const displayLabel = label ?? capitalize(source);

  return (
    <span
      aria-label={`Source: ${displayLabel}`}
      className={cn(
        "inline-flex items-center rounded-full font-medium",
        colorClass,
        sizeStyles[size],
        className,
      )}
    >
      {displayLabel}
    </span>
  );
}

/** Capitalize first letter of a string. */
function capitalize(s: string): string {
  if (!s) return "";
  return s.charAt(0).toUpperCase() + s.slice(1);
}
