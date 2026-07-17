/**
 * Resumes & Cover Letters API — aligned with backend/internal/resumes/handler.go.
 *
 * Backend endpoints:
 * - Resumes: CRUD + content + generate + versions
 * - Cover Letters: CRUD + generate + content update
 *
 * @example
 *   import { fetchResumes, createResume } from "@/lib/api/resumes";
 *   const { resumes, total } = await fetchResumes({ limit: 20, offset: 0 });
 */

import { apiGet, apiPost, apiPut, apiDelete } from "./client";
import type {
  ResumeDetail,
  ResumeListResponse,
  ResumeContentResponse,
  ResumeVersionListResponse,
  CoverLetter,
  CoverLetterListResponse,
  CreateResumeRequest,
  UpdateResumeRequest,
  GenerateResumeContentRequest,
  UpdateResumeContentRequest,
  CreateCoverLetterRequest,
  GenerateCoverLetterRequest,
  UpdateCoverLetterContentRequest,
  ResumeListParams,
  CoverLetterListParams,
} from "@/lib/types/resumes";

// ============================================================================
// Resumes
// ============================================================================

/**
 * Fetch resumes with pagination.
 *
 * @param params - Limit and offset (default: limit=20, offset=0)
 * @returns Paginated resume list
 * @throws ApiError on server error
 */
export async function fetchResumes(
  params: ResumeListParams = {},
): Promise<ResumeListResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit != null) searchParams.set("limit", String(params.limit));
  if (params.offset != null) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  const path = qs.length > 0 ? `/resumes?${qs}` : "/resumes";
  const resp = await apiGet<ResumeListResponse>(path);
  return resp ?? { resumes: [], total: 0, limit: 20, offset: 0 };
}

/**
 * Fetch a single resume with content.
 *
 * @param id - Resume UUID
 * @returns Resume detail (includes content if generated)
 * @throws ApiError on 404 or server error
 */
export async function fetchResume(id: string): Promise<ResumeDetail> {
  const resp = await apiGet<ResumeDetail>(`/resumes/${id}`);
  if (resp == null) {
    throw new Error("Resume not found");
  }
  return resp;
}

/**
 * Create a new resume.
 *
 * @param data - Resume metadata (name, specialization, template, skills)
 * @returns Created resume detail
 * @throws ApiError on validation error or server error
 */
export async function createResume(
  data: CreateResumeRequest,
): Promise<ResumeDetail> {
  const resp = await apiPost<ResumeDetail>("/resumes", data);
  if (resp == null) {
    throw new Error("Failed to create resume");
  }
  return resp;
}

/**
 * Update resume metadata (name, specialization, template, skills).
 *
 * @param id - Resume UUID
 * @param data - Updated resume fields
 * @returns Updated resume detail
 * @throws ApiError on 404, version conflict, or server error
 */
export async function updateResume(
  id: string,
  data: UpdateResumeRequest,
): Promise<ResumeDetail> {
  const resp = await apiPut<ResumeDetail>(`/resumes/${id}`, data);
  if (resp == null) {
    throw new Error("Failed to update resume");
  }
  return resp;
}

/**
 * Delete a resume.
 *
 * @param id - Resume UUID
 * @throws ApiError on 404 or server error
 */
export async function deleteResume(id: string): Promise<void> {
  await apiDelete(`/resumes/${id}`);
}

// ============================================================================
// Resume Content
// ============================================================================

/**
 * Fetch resume content (structured LLM-generated data).
 *
 * @param id - Resume UUID
 * @returns Content with version
 * @throws ApiError on 404 or "no content" error
 */
export async function fetchResumeContent(
  id: string,
): Promise<ResumeContentResponse> {
  const resp = await apiGet<ResumeContentResponse>(`/resumes/${id}/content`);
  if (resp == null) {
    throw new Error("Resume has no content");
  }
  return resp;
}

/**
 * Update resume content manually.
 *
 * @param id - Resume UUID
 * @param content - New structured content
 * @returns Updated content with new version
 * @throws ApiError on 404, version conflict, or server error
 */
export async function updateResumeContent(
  id: string,
  content: ResumeContentResponse["content"],
): Promise<ResumeContentResponse> {
  const resp = await apiPut<ResumeContentResponse>(`/resumes/${id}/content`, {
    content,
  } satisfies UpdateResumeContentRequest);
  if (resp == null) {
    throw new Error("Failed to update content");
  }
  return resp;
}

/**
 * Generate resume content via LLM (synchronous — blocks until done).
 *
 * @param id - Resume UUID
 * @param request - Optional job context for tailoring
 * @returns Generated content with version
 * @throws ApiError on 404, version conflict, or LLM error
 */
export async function generateResumeContent(
  id: string,
  request: GenerateResumeContentRequest = {},
): Promise<ResumeContentResponse> {
  const resp = await apiPost<ResumeContentResponse>(
    `/resumes/${id}/generate`,
    request,
  );
  if (resp == null) {
    throw new Error("Content generation failed");
  }
  return resp;
}

// ============================================================================
// Resume Versions
// ============================================================================

/**
 * Fetch all versions of a resume.
 *
 * @param id - Resume UUID
 * @returns List of historical versions
 * @throws ApiError on 404 or server error
 */
export async function fetchResumeVersions(
  id: string,
): Promise<ResumeVersionListResponse> {
  const resp = await apiGet<ResumeVersionListResponse>(
    `/resumes/${id}/versions`,
  );
  return resp ?? { versions: [] };
}

// ============================================================================
// Cover Letters
// ============================================================================

/**
 * Fetch cover letters with pagination.
 *
 * @param params - Limit and offset (default: limit=20, offset=0)
 * @returns Paginated cover letter list
 * @throws ApiError on server error
 */
export async function fetchCoverLetters(
  params: CoverLetterListParams = {},
): Promise<CoverLetterListResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit != null) searchParams.set("limit", String(params.limit));
  if (params.offset != null) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  const path = qs.length > 0 ? `/cover-letters?${qs}` : "/cover-letters";
  const resp = await apiGet<CoverLetterListResponse>(path);
  return resp ?? { cover_letters: [], total: 0, limit: 20, offset: 0 };
}

/**
 * Fetch a single cover letter.
 *
 * @param id - Cover letter UUID
 * @returns Cover letter detail
 * @throws ApiError on 404 or server error
 */
export async function fetchCoverLetter(id: string): Promise<CoverLetter> {
  const resp = await apiGet<CoverLetter>(`/cover-letters/${id}`);
  if (resp == null) {
    throw new Error("Cover letter not found");
  }
  return resp;
}

/**
 * Create a new cover letter placeholder.
 *
 * @param data - Cover letter metadata (job_id, optional resume_id)
 * @returns Created cover letter
 * @throws ApiError on server error
 */
export async function createCoverLetter(
  data: CreateCoverLetterRequest,
): Promise<CoverLetter> {
  const resp = await apiPost<CoverLetter>("/cover-letters", data);
  if (resp == null) {
    throw new Error("Failed to create cover letter");
  }
  return resp;
}

/**
 * Generate cover letter content via LLM (synchronous).
 *
 * @param id - Cover letter UUID
 * @param request - Job context for generation
 * @returns Generated cover letter with content
 * @throws ApiError on 404, version conflict, or LLM error
 */
export async function generateCoverLetter(
  id: string,
  request: GenerateCoverLetterRequest,
): Promise<CoverLetter> {
  const resp = await apiPost<CoverLetter>(
    `/cover-letters/${id}/generate`,
    request,
  );
  if (resp == null) {
    throw new Error("Cover letter generation failed");
  }
  return resp;
}

/**
 * Update cover letter content manually.
 *
 * @param id - Cover letter UUID
 * @param content - New plain text content
 * @returns Updated cover letter
 * @throws ApiError on 404, version conflict, or server error
 */
export async function updateCoverLetterContent(
  id: string,
  content: string,
): Promise<CoverLetter> {
  const resp = await apiPut<CoverLetter>(`/cover-letters/${id}/content`, {
    content,
  } satisfies UpdateCoverLetterContentRequest);
  if (resp == null) {
    throw new Error("Failed to update cover letter content");
  }
  return resp;
}

/**
 * Delete a cover letter.
 *
 * @param id - Cover letter UUID
 * @throws ApiError on 404 or server error
 */
export async function deleteCoverLetter(id: string): Promise<void> {
  await apiDelete(`/cover-letters/${id}`);
}
