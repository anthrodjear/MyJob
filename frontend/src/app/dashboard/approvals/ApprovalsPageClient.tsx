/**
 * ApprovalsPageClient — client-side approvals page with URL-synced state.
 *
 * Filters and pagination are stored in URL search params.
 * Handles approve/reject actions with dialog confirmation.
 */

"use client";

import { useState, useCallback } from "react";
import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { useApprovals, useApproveApproval, useRejectApproval } from "@/hooks/useApprovals";
import { ApprovalList } from "@/components/approvals/ApprovalList";
import { RejectDialog } from "@/components/approvals/RejectDialog";
import { Button } from "@/components/shared/Button";
import type { ApprovalStatus } from "@/lib/types/approvals";

/** Status filter options. */
const STATUS_OPTIONS: { value: ApprovalStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
];

const PAGE_SIZE = 20;

export function ApprovalsPageClient() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // Read state from URL
  const statusFromUrl = searchParams.get("status") as ApprovalStatus | "all" | null;
  const offsetFromUrl = Number(searchParams.get("offset") ?? "0");
  const statusFilter = statusFromUrl ?? "all";
  const offset = isNaN(offsetFromUrl) ? 0 : offsetFromUrl;

  // Helper to update URL search params
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

  const { data, isLoading } = useApprovals({
    status: statusFilter === "all" ? undefined : statusFilter,
    limit: PAGE_SIZE,
    offset,
  });

  const approveMutation = useApproveApproval();
  const rejectMutation = useRejectApproval();

  // Reject dialog state
  const [rejectTargetId, setRejectTargetId] = useState<string | null>(null);

  const handleStatusFilter = useCallback(
    (value: ApprovalStatus | "all") => {
      const params = new URLSearchParams(searchParams.toString());
      if (value === "all") {
        params.delete("status");
      } else {
        params.set("status", value);
      }
      params.set("offset", "0");
      router.push(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [router, pathname, searchParams],
  );

  const handleLoadMore = useCallback(() => {
    setParam("offset", String(offset + PAGE_SIZE));
  }, [offset, setParam]);

  const handleApprove = useCallback(
    (id: string) => {
      approveMutation.mutate({ id });
    },
    [approveMutation],
  );

  const handleRejectClick = useCallback((id: string) => {
    setRejectTargetId(id);
  }, []);

  const handleRejectConfirm = useCallback(
    (reason: string) => {
      if (rejectTargetId) {
        rejectMutation.mutate({ id: rejectTargetId, reason });
        setRejectTargetId(null);
      }
    },
    [rejectTargetId, rejectMutation],
  );

  const handleRejectCancel = useCallback(() => {
    setRejectTargetId(null);
  }, []);

  const handleSelect = useCallback(
    (id: string) => {
      router.push(`/dashboard/approvals/${id}`);
    },
    [router],
  );

  const approvals = data?.approvals ?? [];
  const total = data?.total ?? 0;
  const hasMore = offset + PAGE_SIZE < total;
  const isAppending = offset > 0 && isLoading;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Approvals</h1>
          <p className="text-sm text-text-secondary" aria-live="polite">
            {total} request{total !== 1 ? "s" : ""} found
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
      <ApprovalList
        approvals={approvals}
        isLoading={isLoading && !isAppending}
        onApprove={handleApprove}
        onReject={handleRejectClick}
        onSelect={handleSelect}
        isPending={approveMutation.isPending || rejectMutation.isPending}
        hasMore={hasMore}
        onLoadMore={handleLoadMore}
        loadingMore={isAppending}
      />

      {/* Reject Dialog */}
      <RejectDialog
        isOpen={rejectTargetId !== null}
        onConfirm={handleRejectConfirm}
        onCancel={handleRejectCancel}
      />
    </div>
  );
}
