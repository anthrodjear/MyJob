/**
 * Approvals API client — list, detail, approve, reject.
 *
 * Backend endpoints (from approvals/handler.go):
 *   GET    /approvals              — list with filters (status, application_id, limit, offset)
 *   GET    /approvals/:id          — single approval
 *   POST   /approvals/:id/approve  — approve (triggers workflow, may return 207)
 *   POST   /approvals/:id/reject   — reject with reason
 */

import { apiGet, apiPost } from "@/lib/api/client";
import type {
  Approval,
  ApprovalListParams,
  ApprovalListResponse,
  ApprovePartialResponse,
} from "@/lib/types/approvals";

/**
 * Fetch paginated approval list with filters.
 *
 * Backend params: status, application_id, limit, offset.
 *
 * @param params - Query parameters
 * @returns Paginated approval list
 */
export async function fetchApprovals(
  params?: ApprovalListParams,
): Promise<ApprovalListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set("status", params.status);
  if (params?.application_id) searchParams.set("application_id", params.application_id);
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));

  const queryString = searchParams.toString();
  const path = queryString ? `approvals?${queryString}` : "approvals";

  const result = await apiGet<ApprovalListResponse>(path);
  if (result === undefined) {
    throw new Error("Failed to fetch approvals");
  }
  return result;
}

/**
 * Fetch a single approval by ID.
 *
 * @param id - Approval UUID
 * @returns Approval detail
 */
export async function fetchApproval(id: string): Promise<Approval> {
  const result = await apiGet<Approval>(`approvals/${id}`);
  if (result === undefined) {
    throw new Error(`Approval not found: ${id}`);
  }
  return result;
}

/**
 * Approve an approval request.
 *
 * Triggers the approve→submit workflow. May return:
 * - 200 OK: full success (approval + application submitted)
 * - 207 Multi-Status: approval succeeded but submission dispatch failed (retry needed)
 *
 * @param id - Approval UUID
 * @returns Approval response (or partial response on 207)
 */
export async function approveApproval(id: string): Promise<Approval | ApprovePartialResponse> {
  const result = await apiPost<Approval | ApprovePartialResponse>(
    `approvals/${id}/approve`,
    {},
  );
  if (result === undefined) {
    throw new Error("Failed to approve request");
  }
  return result;
}

/**
 * Reject an approval request with a reason.
 *
 * @param id - Approval UUID
 * @param reason - Required rejection reason
 * @returns Updated approval
 */
export async function rejectApproval(
  id: string,
  reason: string,
): Promise<Approval> {
  const result = await apiPost<Approval>(`approvals/${id}/reject`, { reason });
  if (result === undefined) {
    throw new Error("Failed to reject request");
  }
  return result;
}
