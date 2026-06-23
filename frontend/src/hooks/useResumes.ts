/**
 * TanStack Query hooks for resumes and cover letters.
 *
 * Provides useQuery hooks for fetching data with caching,
 * plus useMutation hooks for CRUD and generation operations.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchResumes,
  fetchResume,
  createResume,
  updateResume,
  deleteResume,
  generateResumeContent,
  fetchResumeVersions,
  fetchCoverLetters,
  fetchCoverLetter,
  createCoverLetter,
  generateCoverLetter,
  updateCoverLetterContent,
  deleteCoverLetter,
} from "@/lib/api/resumes";
import type {
  ResumeListParams,
  CreateResumeRequest,
  UpdateResumeRequest,
  GenerateResumeContentRequest,
  CoverLetterListParams,
  CreateCoverLetterRequest,
  GenerateCoverLetterRequest,
} from "@/lib/types/resumes";

/** Stable stringify for query keys — sorts keys for consistent references. */
function stableStringify(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

/** Query keys for resumes — consistent cache invalidation. */
export const resumesKeys = {
  all: ["resumes"] as const,
  lists: () => [...resumesKeys.all, "list"] as const,
  list: (params: ResumeListParams) =>
    [...resumesKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  details: () => [...resumesKeys.all, "detail"] as const,
  detail: (id: string) => [...resumesKeys.details(), id] as const,
  content: (id: string) => [...resumesKeys.detail(id), "content"] as const,
  versions: (id: string) => [...resumesKeys.detail(id), "versions"] as const,
};

/** Query keys for cover letters — consistent cache invalidation. */
export const coverLettersKeys = {
  all: ["cover-letters"] as const,
  lists: () => [...coverLettersKeys.all, "list"] as const,
  list: (params: CoverLetterListParams) =>
    [...coverLettersKeys.lists(), stableStringify(params as Record<string, unknown>)] as const,
  details: () => [...coverLettersKeys.all, "detail"] as const,
  detail: (id: string) => [...coverLettersKeys.details(), id] as const,
};

// ============================================================================
// Resume Hooks
// ============================================================================

/** Fetch resumes with pagination + caching. */
export function useResumes(params: ResumeListParams = {}) {
  return useQuery({
    queryKey: resumesKeys.list(params),
    queryFn: () => fetchResumes(params),
  });
}

/** Fetch a single resume with content. */
export function useResume(id: string) {
  return useQuery({
    queryKey: resumesKeys.detail(id),
    queryFn: () => fetchResume(id),
    enabled: id.length > 0,
  });
}

/** Create resume mutation. Invalidates resume list cache on success. */
export function useCreateResume() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateResumeRequest) => createResume(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: resumesKeys.lists() });
    },
  });
}

/** Update resume mutation. Invalidates list + detail cache on success. */
export function useUpdateResume() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateResumeRequest }) =>
      updateResume(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: resumesKeys.lists() });
      void queryClient.invalidateQueries({ queryKey: resumesKeys.detail(id) });
    },
  });
}

/** Delete resume mutation. Invalidates list cache on success. */
export function useDeleteResume() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteResume(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: resumesKeys.lists() });
    },
  });
}

/** Generate resume content mutation (synchronous LLM call). */
export function useGenerateResumeContent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string;
      request?: GenerateResumeContentRequest;
    }) => generateResumeContent(id, request),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: resumesKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: resumesKeys.content(id) });
    },
  });
}

/** Fetch resume versions. */
export function useResumeVersions(id: string) {
  return useQuery({
    queryKey: resumesKeys.versions(id),
    queryFn: () => fetchResumeVersions(id),
    enabled: id.length > 0,
  });
}

// ============================================================================
// Cover Letter Hooks
// ============================================================================

/** Fetch cover letters with pagination + caching. */
export function useCoverLetters(params: CoverLetterListParams = {}) {
  return useQuery({
    queryKey: coverLettersKeys.list(params),
    queryFn: () => fetchCoverLetters(params),
  });
}

/** Fetch a single cover letter. */
export function useCoverLetter(id: string) {
  return useQuery({
    queryKey: coverLettersKeys.detail(id),
    queryFn: () => fetchCoverLetter(id),
    enabled: id.length > 0,
  });
}

/** Create cover letter mutation. Invalidates list cache on success. */
export function useCreateCoverLetter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCoverLetterRequest) => createCoverLetter(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: coverLettersKeys.lists(),
      });
    },
  });
}

/** Generate cover letter content mutation (synchronous LLM call). */
export function useGenerateCoverLetter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string;
      request: GenerateCoverLetterRequest;
    }) => generateCoverLetter(id, request),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({
        queryKey: coverLettersKeys.detail(id),
      });
    },
  });
}

/** Update cover letter content mutation. */
export function useUpdateCoverLetterContent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, content }: { id: string; content: string }) =>
      updateCoverLetterContent(id, content),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({
        queryKey: coverLettersKeys.detail(id),
      });
    },
  });
}

/** Delete cover letter mutation. Invalidates list cache on success. */
export function useDeleteCoverLetter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCoverLetter(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: coverLettersKeys.lists(),
      });
    },
  });
}
