/**
 * ApprovalDetail — full approval request detail view.
 *
 * Shows complete approval data, job snapshot, and approve/reject actions.
 * Used on the [id] page.
 *
 * @example
 *   <ApprovalDetail approval={approval} onApprove={handleApprove} onReject={handleReject} />
 */

"use client";

import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import { JobSnapshotCard } from "./JobSnapshotCard";
import { Button } from "@/components/shared/Button";
import { Card, CardHeader, CardContent } from "@/components/shared/Card";
import type { Approval } from "@/lib/types/approvals";

interface ApprovalDetailProps {
  /** Approval data to display. */
  approval: Approval;
  /** Callback when Approve is clicked. */
  onApprove?: (id: string) => void;
  /** Callback when Reject is clicked. */
  onReject?: (id: string) => void;
  /** Whether mutations are pending. */
  isPending?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * ApprovalDetail — full approval request detail view.
 *
 * Accessibility:
 * - Card sections use proper heading hierarchy
 * - Status announced via StatusBadge
 * - Action buttons have descriptive `aria-label` attributes
 * - aria-live for mutation status feedback
 */
export function ApprovalDetail({
  approval,
  onApprove,
  onReject,
  isPending,
  className,
}: ApprovalDetailProps) {
  return (
    <div className={cn("space-y-6", className)}>
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">Approval Request</h2>
          <p className="mt-1 text-sm text-text-secondary">
            Created {formatDate(approval.created_at)}
            {approval.reviewed_at && ` · Reviewed ${formatDate(approval.reviewed_at)}`}
          </p>
        </div>
        <StatusBadge status={approval.status} />
      </div>

      {/* Job Snapshot */}
      <JobSnapshotCard snapshot={approval.job_snapshot} />

      {/* Documents */}
      {(approval.resume_preview_path || approval.cover_letter_preview) && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold text-text-primary">Documents</h3>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {approval.resume_preview_path && (
                <div>
                  <dt className="text-sm font-medium text-text-tertiary">Resume</dt>
                  <dd className="mt-1 text-sm text-text-primary">
                    {approval.resume_preview_path.startsWith("http") ? (
                      <a
                        href={approval.resume_preview_path}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline"
                      >
                        View Resume
                      </a>
                    ) : (
                      <span className="text-text-tertiary">Local file (not accessible via browser)</span>
                    )}
                  </dd>
                </div>
              )}
              {approval.cover_letter_preview && (
                <div>
                  <dt className="text-sm font-medium text-text-tertiary">Cover Letter</dt>
                  <dd className="mt-1 text-sm text-text-primary">
                    {approval.cover_letter_preview.startsWith("http") ? (
                      <a
                        href={approval.cover_letter_preview}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline"
                      >
                        View Cover Letter
                      </a>
                    ) : (
                      <span className="text-text-tertiary">Local file (not accessible via browser)</span>
                    )}
                  </dd>
                </div>
              )}
            </dl>
          </CardContent>
        </Card>
      )}

      {/* Rejection reason */}
      {approval.rejection_reason && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold text-text-primary">Rejection Reason</h3>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-text-secondary">{approval.rejection_reason}</p>
          </CardContent>
        </Card>
      )}

      {/* Actions — only for pending approvals */}
      {approval.status === "pending" && onApprove && onReject && (
        <Card>
          <CardHeader>
            <h3 className="text-lg font-semibold text-text-primary">Decision</h3>
          </CardHeader>
          <CardContent>
            <p className="mb-4 text-sm text-text-secondary">
              Review the job details and documents above, then approve or reject this application.
            </p>
            <div className="flex gap-3">
              <Button
                variant="primary"
                disabled={isPending}
                onClick={() => onApprove(approval.id)}
                aria-label="Approve this application"
              >
                {isPending ? "Processing..." : "Approve"}
              </Button>
              <Button
                variant="danger"
                disabled={isPending}
                onClick={() => onReject(approval.id)}
                aria-label="Reject this application"
              >
                Reject
              </Button>
            </div>
            {isPending && (
              <p className="mt-2 text-sm text-text-tertiary" aria-live="polite">
                Processing your decision...
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
