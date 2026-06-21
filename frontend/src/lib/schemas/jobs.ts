/**
 * Zod schemas for job-related API request/response validation.
 *
 * These schemas validate data at runtime when it crosses the API boundary.
 * TypeScript types are inferred from schemas using z.input (what goes in)
 * and z.output (what comes out after validation).
 *
 * Usage:
 *   import { jobFilterSchema, type JobFilter } from "@/lib/schemas/jobs";
 *   const filter = jobFilterSchema.parse(rawQuery); // throws on invalid
 */
import { z } from "zod";

/**
 * Schema for job list filter/query parameters.
 *
 * Validates incoming query params from the Jobs page search/filter form.
 * Defaults: page=1, limit=20 (matching backend defaults).
 *
 * @example
 *   const filter = jobFilterSchema.parse({ search: "react", min_score: 70 });
 *   // => { search: "react", min_score: 70, page: 1, limit: 20 }
 */
export const jobFilterSchema = z.object({
  /** Free-text search query (matches title, company, description). */
  search: z.string().optional(),
  /** Filter by job source name (e.g., "linkedin", "indeed"). */
  source: z.string().optional(),
  /** Filter by job status (e.g., "discovered", "matched"). */
  status: z.string().optional(),
  /** Minimum match score threshold (0–100). Jobs below this are excluded. */
  min_score: z.number().min(0).max(100).optional(),
  /** Page number (1-indexed). Defaults to 1. */
  page: z.number().int().positive().default(1),
  /** Items per page. Clamped to 1–100. Defaults to 20. */
  limit: z.number().int().min(1).max(100).default(20),
});

/**
 * Input type for job filter — what the form/API consumer provides.
 * May have partial data (optional fields not yet filled in).
 */
export type JobFilterInput = z.input<typeof jobFilterSchema>;

/**
 * Output type for job filter — fully resolved after parsing.
 * All defaults applied, all optional fields resolved.
 */
export type JobFilter = z.output<typeof jobFilterSchema>;

/**
 * Schema for validating a job source object from the API.
 * Used when parsing the list of configured job sources.
 */
export const jobSourceSchema = z.object({
  /** Source identifier (e.g., "linkedin", "indeed"). */
  name: z.string(),
  /** Priority tier (1=highest). Lower tier sources are checked first. */
  tier: z.number().int().min(1).max(5),
  /** Whether this source is enabled for scraping. */
  enabled: z.boolean(),
});

/** Type inferred from jobSourceSchema — the validated output shape. */
export type JobSourceValidated = z.output<typeof jobSourceSchema>;
