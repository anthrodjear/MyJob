/**
 * Jobs API client — search, list, detail endpoints.
 */

import { apiGet } from "@/lib/api/client";
import type { Job, JobListParams, JobListResponse } from "@/lib/types/jobs";

/**
 * Fetch paginated job list with filters.
 *
 * @param params - Query parameters (search, source, status, min_score, page, limit)
 * @returns Paginated job list
 */
export async function fetchJobs(params?: JobListParams): Promise<JobListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.search) searchParams.set("search", params.search);
  if (params?.source) searchParams.set("source", params.source);
  if (params?.status) searchParams.set("status", params.status);
  if (params?.min_score != null)
    searchParams.set("min_score", String(params.min_score));
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.limit) searchParams.set("limit", String(params.limit));

  const queryString = searchParams.toString();
  const path = queryString ? `jobs?${queryString}` : "jobs";

  const result = await apiGet<JobListResponse>(path);

  if (result === undefined) {
    throw new Error("Unexpected empty response from jobs");
  }
  return result;
}

/**
 * Fetch a single job by ID.
 *
 * @param id - Job UUID
 * @returns Job detail
 */
export async function fetchJob(id: string): Promise<Job> {
  const result = await apiGet<Job>(`jobs/${id}`);
  if (result === undefined) {
    throw new Error(`Job not found: ${id}`);
  }
  return result;
}