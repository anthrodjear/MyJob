/**
 * EmptyState — displayed when a list or section has no data.
 *
 * Shows an optional icon, title, description, and optional action button.
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
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Button } from "./Button";

interface EmptyStateProps {
  /** Optional decorative icon (rendered above the title). */
  icon?: ReactNode;
  /** Main heading. */
  title: string;
  /** Supporting description text. */
  description: string;
  /** Optional call-to-action button. */
  action?: {
    /** Button label. */
    label: string;
    /** Click handler. */
    onClick: () => void;
  };
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
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-12 text-center",
        className,
      )}
    >
      {icon != null && (
        <div className="mb-4 text-text-tertiary" aria-hidden="true">
          {icon}
        </div>
      )}
      <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-text-secondary">
        {description}
      </p>
      {action != null && (
        <Button onClick={action.onClick} className="mt-4">
          {action.label}
        </Button>
      )}
    </div>
  );
}
