/**
 * Approval detail page — fetches single approval and renders detail view.
 *
 * Client Component. Uses useApproval hook.
 */

"use client";

import { use, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useApproval, useApproveApproval, useRejectApproval } from "@/hooks/useApprovals";
import { ApprovalDetail } from "@/components/approvals/ApprovalDetail";
import { RejectDialog } from "@/components/approvals/RejectDialog";
import { Button } from "@/components/shared/Button";

interface ApprovalDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function ApprovalDetailPage({ params }: ApprovalDetailPageProps) {
  const { id } = use(params);
  const router = useRouter();

  const { data: approval, isLoading, error } = useApproval(id);
  const approveMutation = useApproveApproval();
  const rejectMutation = useRejectApproval();

  const [showRejectDialog, setShowRejectDialog] = useState(false);

  const handleApprove = useCallback(
    (approvalId: string) => {
      approveMutation.mutate(
        { id: approvalId },
        {
          onSuccess: () => {
            router.push("/dashboard/approvals");
          },
        },
      );
    },
    [approveMutation, router],
  );

  const handleRejectClick = useCallback(() => {
    setShowRejectDialog(true);
  }, []);

  const handleRejectConfirm = useCallback(
    (reason: string) => {
      rejectMutation.mutate(
        { id, reason },
        {
          onSuccess: () => {
            setShowRejectDialog(false);
            router.push("/dashboard/approvals");
          },
        },
      );
    },
    [rejectMutation, id, router],
  );

  const handleRejectCancel = useCallback(() => {
    setShowRejectDialog(false);
  }, []);

  if (isLoading) {
    return (
      <div className="py-12 text-center">
        <p className="text-sm text-text-tertiary">Loading approval...</p>
      </div>
    );
  }

  if (error || !approval) {
    return (
      <div className="py-12 text-center" aria-live="assertive">
        <p className="text-sm text-danger">Failed to load approval.</p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={() => router.push("/dashboard/approvals")}>
          Back to Approvals
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.push("/dashboard/approvals")}
        aria-label="Back to approvals list"
      >
        ← Back to Approvals
      </Button>
      <ApprovalDetail
        approval={approval}
        onApprove={handleApprove}
        onReject={handleRejectClick}
        isPending={approveMutation.isPending || rejectMutation.isPending}
      />
      <RejectDialog
        isOpen={showRejectDialog}
        onConfirm={handleRejectConfirm}
        onCancel={handleRejectCancel}
      />
    </div>
  );
}
