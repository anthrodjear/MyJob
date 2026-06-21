/**
 * Pagination — page navigation for list views.
 *
 * Shows "Page X of Y" text with Previous/Next buttons.
 * Automatically hides when there's only 1 page.
 *
 * No `"use client"` — pure presentational. Button handles its own interactivity.
 *
 * @example
 *   <Pagination page={1} total={50} limit={20} onPageChange={setPage} />
 */

import { cn } from "@/lib/utils";
import { Button } from "./Button";

interface PaginationProps {
  /** Current page number (1-indexed). */
  page: number;
  /** Total number of items. */
  total: number;
  /** Items per page. */
  limit: number;
  /** Called when the user clicks Previous or Next. */
  onPageChange: (page: number) => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * Pagination — page navigation for list views.
 *
 * Accessibility:
 * - `<nav>` wrapper with `aria-label="Pagination"`
 * - Page info announced to screen readers
 * - Previous/Next buttons use Button component (keyboard accessible)
 * - Buttons are disabled at boundaries (page 1 / last page)
 */
export function Pagination({
  page,
  total,
  limit,
  onPageChange,
  className,
}: PaginationProps) {
  const totalPages = Math.ceil(total / limit);

  // Hide pagination when there's only 1 page
  if (totalPages <= 1) return null;

  return (
    <nav
      className={cn("flex items-center justify-between", className)}
      aria-label="Pagination"
    >
      <span className="text-sm text-text-secondary">
        Page <span aria-current="page">{page}</span> of {totalPages}
      </span>
      <div className="flex gap-2">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
        >
          Previous
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
        >
          Next
        </Button>
      </div>
    </nav>
  );
}
