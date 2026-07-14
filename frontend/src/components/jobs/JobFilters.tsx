/**
 * JobFilters — status and score filters for job listings.
 *
 * Provides status dropdown and minimum score slider.
 * All filters are controlled via the onFilterChange callback.
 *
 * Backend supports: status, source_id, min_score, limit, offset.
 * This component exposes the user-friendly subset (status, min_score).
 *
 * Requires `"use client"` for state management and input handling.
 *
 * @example
 *   <JobFilters filters={filters} onFilterChange={setFilters} />
 */

"use client";

import { useCallback, useState } from "react";
import { X, SlidersHorizontal } from "lucide-react";
import { cn } from "@/lib/utils";
import type { JobListParams } from "@/lib/types/jobs";

/** Available status options for filtering. */
const STATUS_OPTIONS = ["discovered", "matched", "applied", "archived"] as const;

interface JobFiltersProps {
  /** Current filter values. */
  filters: JobListParams;
  /** Callback when filters change. */
  onFilterChange: (filters: JobListParams) => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobFilters — filter controls for job listings.
 *
 * Accessibility:
 * - All inputs have associated labels
 * - Filter controls are grouped in a `<fieldset>` with `<legend>`
 * - Clear button has descriptive `aria-label`
 */
export function JobFilters({
  filters,
  onFilterChange,
  className,
}: JobFiltersProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);

  /** Update a single filter and reset to offset 0. */
  const updateFilter = useCallback(
    <K extends keyof JobListParams>(key: K, value: JobListParams[K]) => {
      onFilterChange({ ...filters, [key]: value, offset: 0 });
    },
    [filters, onFilterChange],
  );

  /** Clear all filters. */
  const clearFilters = useCallback(() => {
    onFilterChange({ limit: filters.limit, offset: 0 });
  }, [onFilterChange, filters.limit]);

  /** Check if any filters are active. */
  const hasActiveFilters =
    !!filters.status ||
    (filters.min_score != null && filters.min_score > 0);

  return (
    <form
      className={cn("space-y-3", className)}
      role="search"
      aria-label="Job filters"
      onSubmit={(e) => e.preventDefault()}
    >
      {/* Toggle Advanced */}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className={cn(
            "flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
            showAdvanced
              ? "border-primary bg-primary-light text-primary-dark"
              : "border-border bg-bg-secondary text-text-primary hover:bg-bg-tertiary",
          )}
          aria-expanded={showAdvanced}
          aria-controls="advanced-filters"
        >
          <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
          Filters
          {hasActiveFilters && (
            <span className="ml-1 rounded-full bg-primary px-1.5 text-xs text-white">
              Active
            </span>
          )}
        </button>
      </div>

      {/* Advanced Filters */}
      {showAdvanced && (
        <fieldset
          id="advanced-filters"
          className="rounded-lg border border-border bg-bg-secondary p-3"
        >
          <legend className="sr-only">Advanced job filters</legend>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {/* Status Filter */}
            <div>
              <label
                htmlFor="filter-status"
                className="mb-1 block text-xs font-medium text-text-secondary"
              >
                Status
              </label>
              <select
                id="filter-status"
                value={filters.status ?? ""}
                onChange={(e) => updateFilter("status", e.target.value || undefined)}
                className="w-full rounded-md border border-border bg-surface px-2 py-1.5 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="">All Statuses</option>
                {STATUS_OPTIONS.map((status) => (
                  <option key={status} value={status}>
                    {status.charAt(0).toUpperCase() + status.slice(1)}
                  </option>
                ))}
              </select>
            </div>

            {/* Min Score */}
            <div>
              <label
                htmlFor="filter-score"
                className="mb-1 block text-xs font-medium text-text-secondary"
              >
                Min Match Score: {filters.min_score ?? 0}%
              </label>
              <input
                id="filter-score"
                type="range"
                min={0}
                max={100}
                step={5}
                value={filters.min_score ?? 0}
                onChange={(e) => {
                  const val = Number(e.target.value);
                  updateFilter("min_score", val > 0 ? val : undefined);
                }}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={filters.min_score ?? 0}
                aria-valuetext={`Minimum match score: ${filters.min_score ?? 0}%`}
                className="w-full accent-primary"
              />
            </div>
          </div>

          {/* Clear Filters */}
          {hasActiveFilters && (
            <div className="mt-3 flex justify-end">
              <button
                type="button"
                onClick={clearFilters}
                className="flex items-center gap-1 rounded-md px-3 py-1.5 text-xs font-medium text-text-secondary hover:text-text-primary"
                aria-label="Clear all filters"
              >
                <X className="h-3 w-3" aria-hidden="true" />
                Clear Filters
              </button>
            </div>
          )}
        </fieldset>
      )}
    </form>
  );
}
