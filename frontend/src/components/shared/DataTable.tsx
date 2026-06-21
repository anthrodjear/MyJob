/**
 * DataTable — generic tabular data display.
 *
 * Renders a styled HTML table with configurable columns, custom renderers,
 * optional row click, and an empty state message.
 *
 * No `"use client"` — pure presentational. Row click is inherited from
 * the parent if needed (add `"use client"` only if interactivity is added).
 *
 * @example
 *   <DataTable
 *     columns={[
 *       { key: "title", header: "Job Title" },
 *       { key: "company", header: "Company" },
 *       { key: "score", header: "Score", render: (item) => `${item.score}%` },
 *     ]}
 *     data={jobs}
 *     onRowClick={(job) => router.push(`/jobs/${job.id}`)}
 *   />
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface Column<T> {
  /** Data key for fallback rendering and column identity. */
  key: string;
  /** Column header label. */
  header: string;
  /** Custom render function. If omitted, uses `String(item[key])`. */
  render?: (item: T) => ReactNode;
  /** Additional CSS classes for the column cells (th + td). */
  className?: string;
}

interface DataTableProps<T> {
  /** Column definitions. */
  columns: Column<T>[];
  /** Row data array. */
  data: T[];
  /** Optional row click handler. Adds hover cursor to rows. */
  onRowClick?: (item: T) => void;
  /** Message shown when data is empty. Default: "No data available". */
  emptyMessage?: string;
  /** Additional CSS classes for the table wrapper. */
  className?: string;
}

/**
 * DataTable — generic tabular data display.
 *
 * Accessibility:
 * - Semantic `<table>`, `<thead>`, `<tbody>`, `<th>`, `<td>` elements
 * - Column headers with `text-xs font-medium uppercase tracking-wider`
 * - Empty state spans full row width via `colSpan`
 * - Row hover feedback when `onRowClick` is provided
 *
 * Type parameter `T` must have an optional `id` field for React keys.
 * Falls back to array index if `id` is missing.
 */
export function DataTable<T extends { id?: string }>({
  columns,
  data,
  onRowClick,
  emptyMessage = "No data available",
  className,
}: DataTableProps<T>) {
  return (
    <div
      className={cn(
        "overflow-hidden rounded-xl border border-border",
        className,
      )}
    >
      <table className="w-full">
        <thead>
          <tr className="border-b border-border bg-bg-secondary">
            {columns.map((col) => (
              <th
                key={col.key}
                scope="col"
                className={cn(
                  "px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-secondary",
                  col.className,
                )}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-8 text-center text-text-tertiary"
              >
                {emptyMessage}
              </td>
            </tr>
          ) : (
            data.map((item, i) => (
              <tr
                key={item.id ?? i}
                className={cn(
                  "border-b border-border last:border-0",
                  onRowClick != null && "cursor-pointer hover:bg-surface-hover",
                )}
                onClick={() => onRowClick?.(item)}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={cn("px-4 py-3 text-sm", col.className)}
                  >
                    {col.render != null
                      ? col.render(item)
                      : String(
                          (item as Record<string, unknown>)[col.key] ?? "—",
                        )}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
