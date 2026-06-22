/**
 * Zod schemas for Interviews domain.
 *
 * Provides runtime validation for interview filter params, mutations, and responses.
 * Aligns with types in @/lib/types/interviews.
 */

import { z } from "zod";

export const interviewStatusSchema = z.enum([
  "pending",
  "starting",
  "active",
  "completed",
  "failed",
  "cancelled",
]);

export const interviewModeSchema = z.enum(["assist", "autonomous"]);

export const interviewFilterSchema = z.object({
  application_id: z.string().uuid().optional(),
  status: interviewStatusSchema.optional(),
  limit: z.number().int().min(1).max(100).optional().default(50),
  offset: z.number().int().min(0).optional().default(0),
});

export const createInterviewSchema = z.object({
  application_id: z.string().uuid(),
  mode: interviewModeSchema,
});

export const startInterviewSchema = z.object({
  provider: z.string().optional(),
  model: z.string().optional(),
});

export const stopInterviewSchema = z.object({
  reason: z.string().optional(),
});

/** Transcript entry schema — matches backend interviews.TranscriptEntry. */
export const transcriptEntrySchema = z.object({
  id: z.string(),
  speaker: z.enum(["candidate", "ai", "system"]),
  content: z.string(),
  timestamp: z.string(),
});

/** Interview session response schema — matches backend interviews.InterviewResponse. */
export const interviewSessionSchema = z.object({
  id: z.string().uuid(),
  application_id: z.string().uuid(),
  mode: interviewModeSchema,
  status: interviewStatusSchema,
  external_session_id: z.string().nullable(),
  provider: z.string(),
  model: z.string(),
  transcript: z.array(transcriptEntrySchema),
  score: z.number().nullable(),
  feedback: z.record(z.string(), z.unknown()).nullable(),
  started_at: z.string().nullable(),
  ended_at: z.string().nullable(),
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
});

/** Interview list response schema. */
export const interviewListResponseSchema = z.object({
  interviews: z.array(interviewSessionSchema),
  total: z.number().int().nonnegative(),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().min(0),
});

export type InterviewFilterInput = z.input<typeof interviewFilterSchema>;
export type CreateInterviewInput = z.input<typeof createInterviewSchema>;
export type StartInterviewInput = z.input<typeof startInterviewSchema>;
export type StopInterviewInput = z.input<typeof stopInterviewSchema>;

// Re-export types for consumers
export type {
  InterviewSession,
  InterviewStatus,
  InterviewMode,
  TranscriptSpeaker,
  TranscriptEntry,
  InterviewListParams,
  InterviewListResponse,
} from "@/lib/types/interviews";
