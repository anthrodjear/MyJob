/**
 * Badge — inline status indicator.
 *
 * Displays a short label with semantic color coding.
 * Supports dot indicator for compact status, and size variants.
 * Used for application status, approval tier, email classification, etc.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Badge variant="success">Auto-Apply</Badge>
 *   <Badge variant="warning">Review Required</Badge>
 *   <Badge variant="danger">Rejected</Badge>
 *   <Badge variant="info" dot>New</Badge>
 *   <Badge variant="success" size="sm">Active</Badge>
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

/** Badge color variant — maps to semantic color tokens. */
type BadgeVariant = "default" | "success" | "warning" | "danger" | "info";

/** Badge size — controls padding and font size. */
type BadgeSize = "sm" | "md";

interface BadgeProps {
  /** Badge label. */
  children: ReactNode;
  /** Additional CSS classes. */
  className?: string;
  /** Color variant. Default: "default". */
  variant?: BadgeVariant;
  /** Size. Default: "md". */
  size?: BadgeSize;
  /** Show a dot indicator before the label. Default: false. */
  dot?: boolean;
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
 * Dot color mapping — matches variant border color for visual consistency.
 */
const dotStyles: Record<BadgeVariant, string> = {
  default: "bg-text-secondary",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/**
 * Size-to-Tailwind class mapping.
 */
const sizeStyles: Record<BadgeSize, string> = {
  sm: "px-2 py-0.5 text-[10px]",
  md: "px-2.5 py-0.5 text-xs",
};

/**
 * Badge — inline status indicator.
 *
 * Accessibility:
 * - Inline element (`<span>`) — does not break text flow
 * - High contrast text on light background
 * - `aria-hidden` on decorative dot indicator
 * - Small but readable (`text-xs font-medium` or `text-[10px] font-medium`)
 */
export function Badge({
  children,
  className,
  variant = "default",
  size = "md",
  dot = false,
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full font-medium",
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
    >
      {dot && (
        <span
          className={cn("h-1.5 w-1.5 rounded-full", dotStyles[variant])}
          aria-hidden="true"
        />
      )}
      {children}
    </span>
  );
}
