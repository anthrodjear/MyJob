/**
 * Applications API client — list, detail, create, status transitions, timeline.
 *
 * Backend endpoints (from applications/handler.go):
 *   GET    /applications          — list with filters (status, job_id, portal_type, limit, offset)
 *   GET    /applications/stats    — stats (by_status, by_tier)
 *   GET    /applications/:id      — single application
 *   POST   /applications          — create (job_id, resume_id, cover_letter_id, portal_type, portal_url)
 *   PUT    /applications/:id/status — transition status (status, notes)
 *   PATCH  /applications/:id/notes  — update notes
 *   GET    /applications/:id/events — audit trail timeline
 */

import { apiGet, apiPost, apiPut, apiPatch } from "@/lib/api/client";
import type {
  Application,
  ApplicationEvent,
  ApplicationListParams,
  ApplicationStatsResponse,
  ApplicationStatus,
} from "@/lib/types/applications";

/**
 * Response shape from GET /applications.
 * Backend returns { applications, total, limit, offset }.
 */
interface ApplicationListResponse {
  applications: Application[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * Response shape from GET /applications/:id/events.
 * Backend returns { application_id, events }.
 */
interface ApplicationTimelineResponse {
  application_id: string;
  events: ApplicationEvent[];
}

/**
 * Fetch paginated application list with filters.
 *
 * Backend params: status, job_id, portal_type, limit, offset.
 *
 * @param params - Query parameters
 * @returns Paginated application list
 */
export async function fetchApplications(
  params?: ApplicationListParams,
): Promise<ApplicationListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set("status", params.status);
  if (params?.job_id) searchParams.set("job_id", params.job_id);
  if (params?.portal_type) searchParams.set("portal_type", params.portal_type);
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));

  const queryString = searchParams.toString();
  const path = queryString ? `/applications?${queryString}` : "/applications";

  const result = await apiGet<ApplicationListResponse>(path);
  if (result === undefined) {
    throw new Error("Failed to fetch applications");
  }
  return result;
}

/**
 * Fetch application stats (by_status, by_tier).
 *
 * @returns Application stats
 */
export async function fetchApplicationStats(): Promise<ApplicationStatsResponse> {
  const result = await apiGet<ApplicationStatsResponse>("/applications/stats");
  if (result === undefined) {
    throw new Error("Unexpected empty response from application stats");
  }
  return result;
}

/**
 * Fetch a single application by ID.
 *
 * @param id - Application UUID
 * @returns Application detail
 */
export async function fetchApplication(id: string): Promise<Application> {
  const result = await apiGet<Application>(`/applications/${id}`);
  if (result === undefined) {
    throw new Error(`Application not found: ${id}`);
  }
  return result;
}

/**
 * Create a new application.
 *
 * @param payload - Application creation data (job_id required, rest optional)
 * @returns Created application
 */
export async function createApplication(payload: {
  job_id: string;
  resume_id?: string;
  cover_letter_id?: string;
  portal_type?: string;
  portal_url?: string;
}): Promise<Application> {
  const result = await apiPost<Application>("/applications", payload);
  if (result === undefined) {
    throw new Error("Failed to create application");
  }
  return result;
}

/**
 * Transition application status.
 *
 * Backend returns { message: "status updated" } — NOT the full Application.
 * Caller should invalidate or refetch the application after this succeeds.
 *
 * Valid transitions (from backend model.go):
 *   draft → queued, rejected
 *   queued → applied, rejected
 *   applied → assessment, phone_screen, technical, final, offer, rejected
 *   assessment → phone_screen, technical, final, offer, rejected
 *   phone_screen → technical, final, offer, rejected
 *   technical → final, offer, rejected
 *   final → offer, rejected
 *   offer → (terminal)
 *   rejected → (terminal)
 *
 * @param id - Application UUID
 * @param status - Target status (must be valid ApplicationStatus)
 * @param notes - Optional notes for the transition
 * @returns Confirmation message
 */
export async function updateApplicationStatus(
  id: string,
  status: ApplicationStatus,
  notes?: string,
): Promise<{ message: string }> {
  const result = await apiPut<{ message: string }>(`/applications/${id}/status`, {
    status,
    notes: notes ?? "",
  });
  if (result === undefined) {
    throw new Error("Failed to update application status");
  }
  return result;
}

/**
 * Update application notes (non-destructive, does not change status).
 *
 * Backend returns { message: "notes updated" } — NOT the full Application.
 * Caller should invalidate or refetch the application after this succeeds.
 *
 * @param id - Application UUID
 * @param notes - New notes content (replaces existing)
 * @returns Confirmation message
 */
export async function updateApplicationNotes(
  id: string,
  notes: string,
): Promise<{ message: string }> {
  const result = await apiPatch<{ message: string }>(`/applications/${id}/notes`, {
    notes,
  });
  if (result === undefined) {
    throw new Error("Failed to update application notes");
  }
  return result;
}

/**
 * Fetch audit trail timeline for an application.
 *
 * @param id - Application UUID
 * @returns Timeline with events
 */
export async function fetchApplicationTimeline(
  id: string,
): Promise<ApplicationTimelineResponse> {
  const result = await apiGet<ApplicationTimelineResponse>(
    `/applications/${id}/events`,
  );
  if (result === undefined) {
    throw new Error(`Failed to fetch timeline for application: ${id}`);
  }
  return result;
}
