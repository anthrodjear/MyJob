/**
 * Interviews API client — CRUD, start/stop, and list endpoints.
 */

import { apiGet, apiPost } from "@/lib/api/client";
import type {
  InterviewSession,
  InterviewListResponse,
} from "@/lib/types/interviews";
import type {
  CreateInterviewInput,
  StartInterviewInput,
  StopInterviewInput,
} from "@/lib/schemas/interviews";

/**
 * Fetch paginated interview list with optional filters.
 *
 * @param params - Query parameters (application_id, status, limit, offset)
 * @param signal - AbortSignal for request cancellation
 * @returns Paginated interview list
 */
export async function fetchInterviews(
  params?: { application_id?: string; status?: string; limit?: number; offset?: number },
  signal?: AbortSignal
): Promise<InterviewListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.application_id) searchParams.set("application_id", params.application_id);
  if (params?.status) searchParams.set("status", params.status);
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  const result = await apiGet<InterviewListResponse>(`/interviews${qs ? `?${qs}` : ""}`, { signal });
  if (result === undefined) {
    throw new Error("Unexpected empty response from interviews");
  }
  return result;
}

/**
 * Fetch a single interview session by ID.
 *
 * @param id - Interview session UUID
 * @param signal - AbortSignal for request cancellation
 * @returns Interview session record
 */
export async function fetchInterview(
  id: string,
  signal?: AbortSignal
): Promise<InterviewSession> {
  const result = await apiGet<InterviewSession>(`/interviews/${id}`, { signal });
  if (result === undefined) {
    throw new Error("Unexpected empty response from interviews/:id");
  }
  return result;
}

/**
 * Create a new interview session in pending status.
 *
 * @param data - Application ID and interview mode
 * @returns Created interview session
 */
export async function createInterview(
  data: CreateInterviewInput
): Promise<InterviewSession> {
  const result = await apiPost<InterviewSession>("/interviews", data);
  if (result === undefined) {
    throw new Error("Unexpected empty response from interviews POST");
  }
  return result;
}

/**
 * Start an interview session (transitions pending → starting).
 *
 * @param id - Interview session UUID
 * @param data - Optional provider and model overrides
 * @returns Success message
 */
export async function startInterview(
  id: string,
  data?: StartInterviewInput
): Promise<{ message: string }> {
  const result = await apiPost<{ message: string }>(`/interviews/${id}/start`, data ?? {});
  if (result === undefined) {
    throw new Error("Unexpected empty response from interviews/:id/start");
  }
  return result;
}

/**
 * Stop an active interview session.
 *
 * @param id - Interview session UUID
 * @param data - Optional stop reason
 * @returns Success message
 */
export async function stopInterview(
  id: string,
  data?: StopInterviewInput
): Promise<{ message: string }> {
  const result = await apiPost<{ message: string }>(`/interviews/${id}/stop`, data ?? {});
  if (result === undefined) {
    throw new Error("Unexpected empty response from interviews/:id/stop");
  }
  return result;
}
