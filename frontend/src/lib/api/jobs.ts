/**
 * Jobs API client — search, list, detail, and mutation endpoints.
 */

import { apiGet, apiPost, apiPatch, apiDelete } from "@/lib/api/client";
import type {
  Job,
  JobListParams,
  JobListResponse,
  JobStatus,
  JobApplicationHistory,
} from "@/lib/types/jobs";

/**
 * Fetch paginated job list with filters.
 *
 * Backend params: status, source_id, min_score, limit, offset.
 *
 * @param params - Query parameters
 * @returns Paginated job list
 */
export async function fetchJobs(params?: JobListParams): Promise<JobListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set("status", params.status);
  if (params?.source_id) searchParams.set("source_id", params.source_id);
  if (params?.min_score != null)
    searchParams.set("min_score", String(params.min_score));
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));

  const queryString = searchParams.toString();
  const path = queryString ? `/jobs?${queryString}` : "/jobs";

  const result = await apiGet<JobListResponse>(path);
  if (result === undefined) {
    throw new Error("Failed to fetch jobs");
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
  const result = await apiGet<Job>(`/jobs/${id}`);
  if (result === undefined) {
    throw new Error(`Job not found: ${id}`);
  }
  return result;
}

/**
 * Submit an application for a job.
 *
 * @param jobId - Job UUID to apply to
 * @returns Created application
 */
export async function applyToJob(jobId: string): Promise<{ application_id: string }> {
  const result = await apiPost<{ application_id: string }>(`/jobs/${jobId}/apply`);
  if (result === undefined) {
    throw new Error("Failed to submit application");
  }
  return result;
}

/**
 * Score a job against the user's profile.
 *
 * @param jobId - Job UUID to score
 * @returns Scoring task info
 */
export async function scoreJob(jobId: string): Promise<{ task_id: string }> {
  const result = await apiPost<{ task_id: string }>(`/jobs/${jobId}/score`);
  if (result === undefined) {
    throw new Error("Failed to queue job scoring");
  }
  return result;
}

/**
 * Save/unsave a job for later.
 *
 * @param jobId - Job UUID
 * @param save - true to save, false to unsave
 * @returns Updated job
 */
export async function saveJob(jobId: string, save: boolean): Promise<Job> {
  const result = await apiPatch<Job>(`/jobs/${jobId}/save`, { save });
  if (result === undefined) {
    throw new Error(save ? "Failed to save job" : "Failed to unsave job");
  }
  return result;
}

/**
 * Update job status (e.g., archive, mark as applied).
 *
 * @param jobId - Job UUID
 * @param status - New status
 * @returns Updated job
 */
export async function updateJobStatus(jobId: string, status: JobStatus): Promise<Job> {
  const result = await apiPatch<Job>(`/jobs/${jobId}`, { status });
  if (result === undefined) {
    throw new Error("Failed to update job status");
  }
  return result;
}

/**
 * Delete a job from the user's list.
 *
 * @param jobId - Job UUID
 * @returns void
 */
export async function deleteJob(jobId: string): Promise<void> {
  const result = await apiDelete<void>(`/jobs/${jobId}`);
  if (result !== undefined) {
    throw new Error("Unexpected response from delete job");
  }
}

/**
 * Fetch similar/comparable jobs for a given job.
 *
 * @param jobId - Job UUID
 * @returns List of similar jobs
 */
export async function fetchSimilarJobs(jobId: string): Promise<Job[]> {
  const result = await apiGet<{ jobs: Job[] }>(`/jobs/${jobId}/similar`);
  if (result === undefined) {
    throw new Error("Failed to fetch similar jobs");
  }
  return result.jobs ?? [];
}

/**
 * Fetch job application history for a job.
 *
 * @param jobId - Job UUID
 * @returns Application history
 */
export async function fetchJobApplications(
  jobId: string,
): Promise<JobApplicationHistory[]> {
  const result = await apiGet<JobApplicationHistory[]>(
    `/jobs/${jobId}/applications`,
  );
  if (result === undefined) {
    throw new Error("Failed to fetch job applications");
  }
  return result;
}