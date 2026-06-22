/**
 * JobFilters — search, source, status, and score filters for job listings.
 *
 * Provides debounced search, dropdown filters for source and status,
 * minimum score slider, and sort controls. All filters are controlled
 * via the onFilterChange callback.
 *
 * Requires `"use client"` for state management and input handling.
 *
 * @example
 *   <JobFilters filters={filters} onFilterChange={setFilters} />
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { Search, X, SlidersHorizontal } from "lucide-react";
import { cn } from "@/lib/utils";
import type { SourceKey } from "@/lib/constants";
import type { JobListParams } from "@/lib/types/jobs";

/** Available source options for filtering. */
const SOURCE_OPTIONS: SourceKey[] = ["indeed", "greenhouse", "lever", "remoteok", "linkedin", "custom"];

/** Available status options for filtering. */
const STATUS_OPTIONS = ["discovered", "matched", "applied", "archived"] as const;

/** Sort field options. */
const SORT_OPTIONS = [
  { value: "created_at", label: "Date Added" },
  { value: "match_score", label: "Match Score" },
  { value: "posted_at", label: "Posted Date" },
  { value: "company", label: "Company" },
] as const;

interface JobFiltersProps {
  /** Current filter values. */
  filters: JobListParams;
  /** Callback when filters change. */
  onFilterChange: (filters: JobListParams) => void;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * JobFilters — search and filter controls for job listings.
 *
 * Accessibility:
 * - All inputs have associated labels
 * - Search input has `aria-label` and `role="searchbox"`
 * - Filter controls are grouped in a `<fieldset>` with `<legend>`
 * - Clear button has descriptive `aria-label`
 */
export function JobFilters({
  filters,
  onFilterChange,
  className,
}: JobFiltersProps) {
  const [search, setSearch] = useState(filters.search ?? "");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const filtersRef = useRef(filters);
  filtersRef.current = filters;

  // Debounce search input — only re-run on search text change
  useEffect(() => {
    const timer = setTimeout(() => {
      if (search !== (filtersRef.current.search ?? "")) {
        onFilterChange({ ...filtersRef.current, search: search || undefined, page: 1 });
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [search, onFilterChange]);

  // Sync search state when filters change externally
  useEffect(() => {
    setSearch(filters.search ?? "");
  }, [filters.search]);

  /** Update a single filter and reset to page 1. */
  const updateFilter = useCallback(
    <K extends keyof JobListParams>(key: K, value: JobListParams[K]) => {
      onFilterChange({ ...filters, [key]: value, page: 1 });
    },
    [filters, onFilterChange],
  );

  /** Clear all filters. */
  const clearFilters = useCallback(() => {
    setSearch("");
    onFilterChange({ page: 1, limit: filters.limit });
  }, [onFilterChange, filters.limit]);

  /** Close advanced panel on Escape key. */
  useEffect(() => {
    if (!showAdvanced) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") setShowAdvanced(false);
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [showAdvanced]);

  /** Check if any filters are active. */
  const hasActiveFilters =
    !!filters.search ||
    !!filters.source ||
    !!filters.status ||
    (filters.min_score != null && filters.min_score > 0) ||
    !!filters.sort_by;

  return (
    <form
      className={cn("space-y-3", className)}
      role="search"
      aria-label="Job filters"
      onSubmit={(e) => e.preventDefault()}
    >
      {/* Search + Toggle */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-tertiary"
            aria-hidden="true"
          />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search jobs..."
            aria-label="Search jobs by title, company, or keyword"
            className="w-full rounded-lg border border-border bg-bg-secondary py-2 pl-9 pr-3 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-text-tertiary hover:text-text-primary"
              aria-label="Clear search"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
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
        </button>
      </div>

      {/* Advanced Filters */}
      {showAdvanced && (
        <fieldset
          id="advanced-filters"
          className="rounded-lg border border-border bg-bg-secondary p-3"
        >
          <legend className="sr-only">Advanced job filters</legend>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {/* Source Filter */}
            <div>
              <label
                htmlFor="filter-source"
                className="mb-1 block text-xs font-medium text-text-secondary"
              >
                Source
              </label>
              <select
                id="filter-source"
                value={filters.source ?? ""}
                onChange={(e) => updateFilter("source", e.target.value || undefined)}
                className="w-full rounded-md border border-border bg-bg-primary px-2 py-1.5 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="">All Sources</option>
                {SOURCE_OPTIONS.map((src) => (
                  <option key={src} value={src}>
                    {src.charAt(0).toUpperCase() + src.slice(1)}
                  </option>
                ))}
              </select>
            </div>

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
                className="w-full rounded-md border border-border bg-bg-primary px-2 py-1.5 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
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

            {/* Sort */}
            <div>
              <label
                htmlFor="filter-sort"
                className="mb-1 block text-xs font-medium text-text-secondary"
              >
                Sort By
              </label>
              <select
                id="filter-sort"
                value={filters.sort_by ?? ""}
                onChange={(e) => {
                  const val = e.target.value;
                  updateFilter("sort_by", val || undefined);
                }}
                className="w-full rounded-md border border-border bg-bg-primary px-2 py-1.5 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="">Default</option>
                {SORT_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
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
