/**
 * Zod schemas for application-related API request/response validation.
 *
 * Validates application filter params, status transitions, and
 * event data at the API boundary. Runtime validation catches type
 * mismatches that TypeScript compile-time checks cannot guarantee
 * (e.g., data from the backend or user input).
 *
 * Usage:
 *   import { applicationFilterSchema } from "@/lib/schemas/applications";
 *   const filter = applicationFilterSchema.parse(rawQuery);
 */
import { z } from "zod";

/**
 * Valid application status values.
 * Mirrors backend/internal/applications/model.go Status* constants.
 *
 * Status flow: draft → queued → applied → assessment/phone_screen/technical/final → offer/rejected
 */
export const applicationStatusSchema = z.enum([
  "draft",
  "queued",
  "applied",
  "assessment",
  "phone_screen",
  "technical",
  "final",
  "offer",
  "rejected",
]);

/**
 * Valid approval tier values.
 * Mirrors backend/internal/applications/model.go Tier* constants.
 *
 * - auto: score >= AutoThreshold → submit without review
 * - review: score between ReviewThreshold and AutoThreshold → needs human approval
 * - reject: score < ReviewThreshold → rejected automatically
 */
export const approvalTierSchema = z.enum(["auto", "review", "reject"]);

/**
 * Schema for application list filter/query parameters.
 *
 * Validates incoming query params from the Applications page.
 * Narrower than jobFilter — only application-specific fields.
 *
 * @example
 *   const filter = applicationFilterSchema.parse({ status: "applied", min_score: 80 });
 *   // => { status: "applied", min_score: 80, page: 1, limit: 20 }
 */
export const applicationFilterSchema = z.object({
  /** Filter by application status. Only one status at a time. */
  status: applicationStatusSchema.optional(),
  /** Minimum match score threshold (0–100). */
  min_score: z.number().min(0).max(100).optional(),
  /** Page number (1-indexed). Defaults to 1. */
  page: z.number().int().positive().default(1),
  /** Items per page. Clamped to 1–100. Defaults to 20. */
  limit: z.number().int().min(1).max(100).default(20),
});

/**
 * Input type for application filter — what the form/API consumer provides.
 */
export type ApplicationFilterInput = z.input<typeof applicationFilterSchema>;

/**
 * Output type for application filter — fully resolved after parsing.
 */
export type ApplicationFilter = z.output<typeof applicationFilterSchema>;

/**
 * Schema for validating a status transition request.
 * Used when the user approves/rejects an application or moves it through the pipeline.
 *
 * @example
 *   const transition = statusTransitionSchema.parse({ status: "applied", notes: "Ready to submit" });
 */
export const statusTransitionSchema = z.object({
  /** Target status to transition to. Must be a valid status. */
  status: applicationStatusSchema,
  /** Optional notes for the status change (stored in application_events). */
  notes: z.string().max(1000).optional(),
});

/** Type inferred from statusTransitionSchema. */
export type StatusTransition = z.output<typeof statusTransitionSchema>;
