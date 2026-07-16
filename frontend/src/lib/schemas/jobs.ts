/**
 * Zod schemas for Jobs domain.
 *
 * Provides runtime validation for API requests/responses.
 * Aligns with types in @/lib/types/jobs.
 *
 * Schema fields match the Go backend JSON response exactly.
 * TypeScript types are re-exported from @/lib/types/jobs for consistency.
 */

import { z } from "zod";

/** Valid job status values — matches backend job status pipeline. */
export const jobStatusSchema = z.enum([
  "discovered",
  "matched",
  "applied",
  "archived",
]);

/** Valid job source tier values. */
export const jobSourceTierSchema = z.number().int().min(1).max(5);

/** Job source schema — minimal source info for job listings. */
export const jobSourceSchema = z.object({
  name: z.string(),
  tier: jobSourceTierSchema,
  enabled: z.boolean(),
});

/**
 * Job schema — matches backend Job response.
 * Fields align with backend/internal/jobs/model.go Job struct JSON tags.
 */
export const jobSchema = z.object({
  id: z.string().uuid(),
  source_id: z.string().uuid(),
  external_id: z.string(),
  title: z.string(),
  company: z.string(),
  location: z.string(),
  remote_type: z.string(),
  salary_min: z.number().int().nonnegative(),
  salary_max: z.number().int().nonnegative(),
  salary_currency: z.string().length(3),
  description: z.string(),
  requirements: z.string(),
  url: z.string().url(),
  application_url: z.string().url(),
  company_url: z.string().url(),
  source: z.string(),
  posted_at: z.string().datetime().nullable(),
  scraped_at: z.string().datetime(),
  match_score: z.number().int().min(0).max(100),
  match_details: z.record(z.string(), z.unknown()).nullable(),
  score_tier: z.string().nullable(),
  scored_at: z.string().datetime().nullable(),
  scoring_reasoning: z.string().nullable(),
  scoring_model: z.string().nullable(),
  scoring_source: z.string().nullable(),
  status: z.string(),
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
  source_name: z.string(),
});

/**
 * Job list query parameters.
 * Matches backend handler: listJobsQuery in jobs/handler.go.
 * Uses offset-based pagination (not page-based).
 */
export const jobListParamsSchema = z.object({
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().min(0).default(0),
  source_id: z.string().uuid().optional(),
  status: z.string().optional(),
  min_score: z.number().int().min(0).max(100).optional(),
});

/**
 * Job list response schema.
 * Backend returns { jobs, total, limit, offset }.
 */
export const jobListResponseSchema = z.object({
  jobs: z.array(jobSchema),
  total: z.number().int().nonnegative(),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
});

/** Type exports using Zod inference. */
export type JobStatusValidated = z.output<typeof jobStatusSchema>;
export type JobSourceValidated = z.output<typeof jobSourceSchema>;
export type JobValidated = z.output<typeof jobSchema>;
export type JobListParamsValidated = z.output<typeof jobListParamsSchema>;
export type JobListResponseValidated = z.output<typeof jobListResponseSchema>;

// Re-export types that align with existing type definitions
export type { Job, JobStatus, JobSource, JobListParams, JobListResponse } from "@/lib/types/jobs";
export type { SortDirection } from "@/lib/types/common";
