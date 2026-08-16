/**
 * EmptyState — displayed when a list or section has no data.
 *
 * Shows an optional icon, title, description, optional hint, and action button(s).
 * Used for empty job lists, no applications, no emails, etc.
 *
 * No `"use client"` — pure presentational. Action button handles its own interactivity.
 *
 * @example
 *   <EmptyState
 *     icon={<InboxIcon />}
 *     title="No jobs found"
 *     description="Start a search to discover opportunities."
 *     action={{ label: "Start Search", onClick: handleSearch }}
 *   />
 *
 *   <EmptyState
 *     icon={<MailIcon />}
 *     title="No emails yet"
 *     description="Connect your email to get started."
 *     hint="Go to Settings → Integrations to set up email."
 *     action={{ label: "Open Settings", onClick: handleSettings }}
 *   />
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Button } from "./Button";

interface EmptyStateAction {
  /** Button label. */
  label: string;
  /** Click handler. */
  onClick: () => void;
}

interface EmptyStateProps {
  /** Optional decorative icon (rendered above the title). */
  icon?: ReactNode;
  /** Main heading. */
  title: string;
  /** Supporting description text. */
  description: string;
  /** Optional hint text — shown below the description in a muted style. */
  hint?: string;
  /** Optional call-to-action button. */
  action?: EmptyStateAction;
  /** Optional secondary action button. */
  secondaryAction?: EmptyStateAction;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * EmptyState — centered placeholder for empty data views.
 *
 * Accessibility:
 * - Semantic heading (`h3`) for screen reader navigation
 * - Icon is decorative (`aria-hidden="true"`)
 * - Action button uses Button component (keyboard accessible, focus ring)
 * - Empty `role="status"` for screen reader announcement
 */
export function EmptyState({
  icon,
  title,
  description,
  hint,
  action,
  secondaryAction,
  className,
}: EmptyStateProps) {
  return (
    <div
      role="status"
      aria-label={title}
      className={cn(
        "flex flex-col items-center justify-center py-16 text-center",
        className,
      )}
    >
      {icon != null && (
        <div
          className={cn(
            "relative mb-6 flex items-center justify-center",
            "animate-scale-in",
          )}
          aria-hidden="true"
        >
          {/* Outer glow ring */}
          <div className="absolute inset-0 -m-2 rounded-full bg-gradient-to-br from-primary/15 to-purple-500/15 blur-xl opacity-60" />
          {/* Gradient accent behind the icon */}
          <div className="absolute inset-0 rounded-full bg-gradient-to-br from-primary/20 to-purple-500/20 blur-md" />
          {/* Subtle colored circular background */}
          <div className="relative flex items-center justify-center w-16 h-16 rounded-full bg-gradient-to-br from-primary/10 to-purple-500/10 border border-primary/20 shadow-sm">
            {/* Larger 8×8 icon */}
            <div className="w-8 h-8 text-text-tertiary">{icon}</div>
          </div>
        </div>
      )}
      <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
      <p className="mt-1.5 max-w-sm text-sm text-text-secondary leading-relaxed">
        {description}
      </p>
      {hint != null && (
        <p className="mt-2 max-w-xs text-xs text-text-tertiary italic">
          {hint}
        </p>
      )}
      {(action != null || secondaryAction != null) && (
        <div className="mt-5 flex items-center gap-3">
          {action != null && (
            <Button variant="gradient" onClick={action.onClick}>
              {action.label}
            </Button>
          )}
          {secondaryAction != null && (
            <Button variant="secondary" onClick={secondaryAction.onClick}>
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
