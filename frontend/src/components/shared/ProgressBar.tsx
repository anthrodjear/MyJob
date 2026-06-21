/**
 * ProgressBar — horizontal progress indicator.
 *
 * Displays a labeled bar with percentage text and semantic color coding.
 * Value is clamped to 0–100 automatically.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <ProgressBar value={75} label="Match Score" color="success" />
 *   <ProgressBar value={45} color="warning" />
 */

import { cn } from "@/lib/utils";

interface ProgressBarProps {
  /** Progress value (0–100). Automatically clamped. */
  value: number;
  /** Optional label displayed above the bar. */
  label?: string;
  /** Bar color variant. Default: "primary". */
  color?: "primary" | "success" | "warning" | "danger" | "info";
  /** Additional CSS classes. */
  className?: string;
}

/**
 * Color-to-Tailwind class mapping.
 */
const colorStyles: Record<"primary" | "success" | "warning" | "danger" | "info", string> = {
  primary: "bg-primary",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/**
 * ProgressBar — horizontal progress indicator.
 *
 * Accessibility:
 * - `role="progressbar"` on the fill element
 * - `aria-valuenow`, `aria-valuemin`, `aria-valuemax` for screen readers
 * - `aria-label` from the `label` prop
 * - Percentage displayed as monospace tabular-nums for alignment
 * - `transition-all duration-300` for smooth animation
 */
export function ProgressBar({
  value,
  label,
  color = "primary",
  className,
}: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));

  return (
    <div className={cn("w-full", className)}>
      {label != null && (
        <div className="mb-1 flex items-center justify-between text-sm">
          <span className="text-text-secondary">{label}</span>
          <span className="font-mono tabular-nums text-text-primary">
            {clamped}%
          </span>
        </div>
      )}
      <div className="h-2 w-full overflow-hidden rounded-full bg-bg-tertiary">
        <div
          className={cn(
            "h-full rounded-full transition-all duration-300",
            colorStyles[color],
          )}
          style={{ width: `${clamped}%` }}
          role="progressbar"
          aria-valuenow={clamped}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={label}
        />
      </div>
    </div>
  );
}
