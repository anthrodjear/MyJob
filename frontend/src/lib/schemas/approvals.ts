/**
 * Zod schemas for approval-related API request/response validation.
 *
 * Validates approval filter params, approve/reject actions, and
 * response data at the API boundary.
 *
 * Usage:
 *   import { approvalFilterSchema } from "@/lib/schemas/approvals";
 *   const filter = approvalFilterSchema.parse(rawQuery);
 */
import { z } from "zod";

/**
 * Valid approval status values.
 * Mirrors backend/internal/approvals/model.go ApprovalStatus* constants.
 *
 * Status flow: pending → approved/rejected (both terminal)
 */
export const approvalStatusSchema = z.enum(["pending", "approved", "rejected"]);

/**
 * Schema for approval list filter/query parameters.
 *
 * Validates incoming query params from the Approvals page.
 * Matches backend handler: ListApprovals in approvals/handler.go.
 *
 * @example
 *   const filter = approvalFilterSchema.parse({ status: "pending", limit: 20 });
 */
export const approvalFilterSchema = z.object({
  /** Filter by approval status. Only one status at a time. */
  status: approvalStatusSchema.optional(),
  /** Filter by application UUID. */
  application_id: z.string().uuid().optional(),
  /** Items per page. Clamped to 1–100. Defaults to 50. */
  limit: z.number().int().min(1).max(100).default(50),
  /** Offset from start. Defaults to 0. */
  offset: z.number().int().min(0).default(0),
});

/**
 * Input type for approval filter — what the form/API consumer provides.
 */
export type ApprovalFilterInput = z.input<typeof approvalFilterSchema>;

/**
 * Output type for approval filter — fully resolved after parsing.
 */
export type ApprovalFilter = z.output<typeof approvalFilterSchema>;

/**
 * Schema for reject request body.
 * Used when the user rejects an approval request.
 *
 * @example
 *   const reject = rejectRequestSchema.parse({ reason: "Salary too low" });
 */
export const rejectRequestSchema = z.object({
  /** Required rejection reason for audit trail. */
  reason: z.string().min(1, "Rejection reason is required").max(1000),
});

/** Type inferred from rejectRequestSchema. */
export type RejectRequest = z.output<typeof rejectRequestSchema>;
