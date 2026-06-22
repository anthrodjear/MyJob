/**
 * Emails API client — list, detail, update, and classify endpoints.
 */

import { apiGet, apiPatch, apiPost } from "@/lib/api/client";
import type { Email, EmailListResponse, ClassifyResponse } from "@/lib/types/emails";
import type { UpdateEmailInput } from "@/lib/schemas/emails";

/**
 * Fetch paginated email list with optional filters.
 *
 * @param params - Filter and pagination parameters
 * @param signal - AbortSignal for request cancellation
 * @returns Paginated email list
 */
export async function fetchEmails(
  params?: { application_id?: string; classification?: string; limit?: number; offset?: number },
  signal?: AbortSignal
): Promise<EmailListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.application_id) searchParams.set("application_id", params.application_id);
  if (params?.classification) searchParams.set("classification", params.classification);
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  const result = await apiGet<EmailListResponse>(`/emails${qs ? `?${qs}` : ""}`, { signal });
  if (result === undefined) {
    throw new Error("Unexpected empty response from emails");
  }
  return result;
}

/**
 * Fetch a single email by ID.
 *
 * @param id - Email UUID
 * @param signal - AbortSignal for request cancellation
 * @returns Email record
 */
export async function fetchEmail(
  id: string,
  signal?: AbortSignal
): Promise<Email> {
  const result = await apiGet<Email>(`/emails/${id}`, { signal });
  if (result === undefined) {
    throw new Error("Unexpected empty response from emails/:id");
  }
  return result;
}

/**
 * Update email read status or reply draft.
 *
 * @param id - Email UUID
 * @param data - Fields to update
 * @returns Updated email record
 */
export async function updateEmail(
  id: string,
  data: UpdateEmailInput
): Promise<Email> {
  const result = await apiPatch<Email>(`/emails/${id}`, data);
  if (result === undefined) {
    throw new Error("Unexpected empty response from emails/:id PATCH");
  }
  return result;
}

/**
 * Trigger email re-classification by the backend.
 *
 * @param id - Email UUID
 * @returns Classification result with confidence and reasoning
 */
export async function classifyEmail(
  id: string
): Promise<ClassifyResponse> {
  const result = await apiPost<ClassifyResponse>(`/emails/${id}/classify`, undefined);
  if (result === undefined) {
    throw new Error("Unexpected empty response from emails/:id/classify");
  }
  return result;
}
