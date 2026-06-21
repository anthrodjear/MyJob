/**
 * Loading skeleton — animated placeholder for content being loaded.
 *
 * Provides Skeleton (base), CardSkeleton, and TableRowSkeleton
 * for consistent loading states across the dashboard.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Skeleton className="h-4 w-3/4" />
 *   <CardSkeleton />
 *   <TableRowSkeleton columns={5} />
 */

import { cn } from "@/lib/utils";

interface SkeletonProps {
  /** Additional CSS classes (width, height, etc.). */
  className?: string;
}

/**
 * Base skeleton block — animated pulse placeholder.
 * Use specific width/height classes to match the expected content shape.
 *
 * Accessibility:
 * - `aria-hidden="true"` — decorative, hidden from screen readers
 * - `animate-pulse` — subtle animation indicates loading state
 */
export function Skeleton({ className }: SkeletonProps) {
  return (
    <div
      className={cn("motion-safe:animate-pulse rounded-md bg-bg-tertiary", className)}
      aria-hidden="true"
    />
  );
}

/**
 * Card-shaped skeleton — matches Card component dimensions.
 * Shows 3 placeholder lines of varying widths for a realistic layout.
 */
export function CardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-6 shadow-sm",
        className,
      )}
    >
      <Skeleton className="mb-3 h-4 w-3/4" />
      <Skeleton className="mb-2 h-3 w-1/2" />
      <Skeleton className="h-3 w-2/3" />
    </div>
  );
}

/**
 * Table row skeleton — matches DataTable row dimensions.
 * Renders N placeholder cells (default: 5) for a single table row.
 *
 * @param columns - Number of columns to render. Default: 5.
 */
export function TableRowSkeleton({ columns = 5 }: { columns?: number }) {
  return (
    <tr>
      {Array.from({ length: columns }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <Skeleton className="h-4 w-full" />
        </td>
      ))}
    </tr>
  );
}
