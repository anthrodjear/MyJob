/**
 * Zod v4 schemas for Jobs domain.
 *
 * Provides runtime validation for API requests/responses.
 * Aligns with types in @/lib/types/jobs.
 */

import { z } from "zod";
import type { Job, JobStatus, JobSource, JobListParams, JobListResponse } from "@/lib/types/jobs";
import type { SortDirection } from "@/lib/types/common";

/** Valid job status values — matches backend ApplicationStatus. */
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

/** Job schema — matches backend Job response. */
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
  status: jobStatusSchema,
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
  source_name: z.string().optional(),
});

/** Job list query parameters. */
export const jobListParamsSchema = z.object({
  page: z.number().int().positive().default(1),
  limit: z.number().int().min(1).max(100).default(20),
  source: z.string().optional(),
  status: z.string().optional(),
  min_score: z.number().int().min(0).max(100).optional(),
  search: z.string().optional(),
  sort_by: z.string().optional(),
  sort_dir: z.enum(["asc", "desc"]).optional(),
});

/** Job list response schema. */
export const jobListResponseSchema = z.object({
  items: z.array(jobSchema),
  total: z.number().int().nonnegative(),
  page: z.number().int().positive(),
  limit: z.number().int().min(1).max(100),
});

/** Type exports using Zod v4 inference. */
export type JobStatusValidated = z.output<typeof jobStatusSchema>;
export type JobSourceValidated = z.output<typeof jobSourceSchema>;
export type JobValidated = z.output<typeof jobSchema>;
export type JobListParamsValidated = z.output<typeof jobListParamsSchema>;
export type JobListResponseValidated = z.output<typeof jobListResponseSchema>;

// Re-export types that align with existing type definitions
export type { Job, JobStatus, JobSource, JobListParams, JobListResponse } from "@/lib/types/jobs";
export type { SortDirection } from "@/lib/types/common";