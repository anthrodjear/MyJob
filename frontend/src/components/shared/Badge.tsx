/**
 * Badge — inline status indicator.
 *
 * Displays a short label with semantic color coding.
 * Used for application status, approval tier, email classification, etc.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Badge variant="success">Auto-Apply</Badge>
 *   <Badge variant="warning">Review Required</Badge>
 *   <Badge variant="danger">Rejected</Badge>
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

/** Badge color variant — maps to semantic color tokens. */
type BadgeVariant = "default" | "success" | "warning" | "danger" | "info";

interface BadgeProps {
  /** Badge label. */
  children: ReactNode;
  /** Additional CSS classes. */
  className?: string;
  /** Color variant. Default: "default". */
  variant?: BadgeVariant;
}

/**
 * Variant-to-Tailwind class mapping.
 * Uses light background + dark text for each semantic color.
 */
const variantStyles: Record<BadgeVariant, string> = {
  default: "bg-bg-tertiary text-text-primary",
  success: "bg-success-light text-success-dark",
  warning: "bg-warning-light text-warning-dark",
  danger: "bg-danger-light text-danger-dark",
  info: "bg-info-light text-info-dark",
};

/**
 * Badge — inline status indicator.
 *
 * Accessibility:
 * - Inline element (`<span>`) — does not break text flow
 * - High contrast text on light background
 * - Small but readable (`text-xs font-medium`)
 */
export function Badge({
  children,
  className,
  variant = "default",
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        variantStyles[variant],
        className,
      )}
    >
      {children}
    </span>
  );
}
