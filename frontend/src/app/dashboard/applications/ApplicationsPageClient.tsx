/**
 * ApplicationsPageClient — client-side applications page with URL-synced state.
 *
 * Filters and pagination are stored in URL search params (?status=queued&offset=0).
 * Uses useApplications hook for server state management.
 * Deep-linkable and shareable — filter state survives page refresh.
 */

"use client";

import { useCallback } from "react";
import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { useApplications, useUpdateApplicationStatus } from "@/hooks/useApplications";
import { ApplicationList } from "@/components/applications/ApplicationList";
import { Button } from "@/components/shared/Button";
import type { ApplicationStatus } from "@/lib/types/applications";

/** Status filter options. */
const STATUS_OPTIONS: { value: ApplicationStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "draft", label: "Draft" },
  { value: "queued", label: "Queued" },
  { value: "applied", label: "Applied" },
  { value: "assessment", label: "Assessment" },
  { value: "phone_screen", label: "Phone Screen" },
  { value: "technical", label: "Technical" },
  { value: "final", label: "Final" },
  { value: "offer", label: "Offer" },
  { value: "rejected", label: "Rejected" },
];

const PAGE_SIZE = 20;

export function ApplicationsPageClient() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // Read state from URL — survives refresh and deep links
  const statusFromUrl = searchParams.get("status") as ApplicationStatus | "all" | null;
  const offsetFromUrl = Number(searchParams.get("offset") ?? "0");
  const statusFilter = statusFromUrl ?? "all";
  const offset = isNaN(offsetFromUrl) ? 0 : offsetFromUrl;

  // Helper to update URL search params (shallow, no scroll)
  const setParam = useCallback(
    (key: string, value: string) => {
      const params = new URLSearchParams(searchParams.toString());
      if (value) {
        params.set(key, value);
      } else {
        params.delete(key);
      }
      router.push(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [router, pathname, searchParams],
  );

  const { data, isLoading, error, refetch } = useApplications({
    status: statusFilter === "all" ? undefined : statusFilter,
    limit: PAGE_SIZE,
    offset,
  });

  const updateStatusMutation = useUpdateApplicationStatus();

  const handleStatusChange = useCallback(
    (id: string, status: ApplicationStatus) => {
      updateStatusMutation.mutate({ id, status });
    },
    [updateStatusMutation],
  );

  const handleSelect = useCallback(
    (id: string) => {
      router.push(`/dashboard/applications/${id}`);
    },
    [router],
  );

  const handleLoadMore = useCallback(() => {
    setParam("offset", String(offset + PAGE_SIZE));
  }, [offset, setParam]);

  const handleStatusFilter = useCallback(
    (value: ApplicationStatus | "all") => {
      setParam("status", value === "all" ? "" : value);
      setParam("offset", "0"); // Reset pagination on filter change
    },
    [setParam],
  );

  if (error) {
    return (
      <div className="py-12 text-center" aria-live="assertive">
        <p className="text-sm text-danger">Failed to load applications. Please try again.</p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  const applications = data?.applications ?? [];
  const total = data?.total ?? 0;
  const hasMore = offset + PAGE_SIZE < total;
  const isAppending = offset > 0 && isLoading;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Applications</h1>
          <p className="text-sm text-text-secondary" aria-live="polite">
            {total} application{total !== 1 ? "s" : ""} found
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex flex-wrap gap-1" role="radiogroup" aria-label="Status filter">
          {STATUS_OPTIONS.map((opt) => (
            <Button
              key={opt.value}
              variant={statusFilter === opt.value ? "primary" : "secondary"}
              size="sm"
              onClick={() => handleStatusFilter(opt.value)}
              role="radio"
              aria-checked={statusFilter === opt.value}
            >
              {opt.label}
            </Button>
          ))}
        </div>
      </div>

      {/* List */}
      <ApplicationList
        applications={applications}
        isLoading={isLoading && !isAppending}
        onSelect={handleSelect}
        onStatusChange={handleStatusChange}
        hasMore={hasMore}
        onLoadMore={handleLoadMore}
        loadingMore={isAppending}
      />
    </div>
  );
}
