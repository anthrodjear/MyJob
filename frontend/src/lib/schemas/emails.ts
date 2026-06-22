/**
 * Zod schemas for Emails domain.
 *
 * Provides runtime validation for email filter params and mutations.
 * Aligns with types in @/lib/types/emails.
 */

import { z } from "zod";

export const emailClassificationSchema = z.enum([
  "interview_invite",
  "rejection",
  "offer",
  "follow_up",
  "spam",
  "phishing",
  "other",
]);

export const emailFilterSchema = z.object({
  application_id: z.string().uuid().optional(),
  classification: emailClassificationSchema.optional(),
  limit: z.number().int().min(1).max(100).optional().default(50),
  offset: z.number().int().min(0).optional().default(0),
});

export const updateEmailSchema = z.object({
  is_read: z.boolean().optional(),
  reply_draft: z.string().optional(),
});

/** Email response schema — matches backend emails.EmailResponse. */
export const emailResponseSchema = z.object({
  id: z.string().uuid(),
  application_id: z.string().uuid().nullable(),
  message_id: z.string(),
  from_address: z.string(),
  to_address: z.string().nullable(),
  subject: z.string().nullable(),
  body: z.string().nullable(),
  received_at: z.string().datetime(),
  classification: emailClassificationSchema.nullable(),
  is_read: z.boolean(),
  reply_draft: z.string().nullable(),
  created_at: z.string().datetime(),
});

/** Email list response schema. */
export const emailListResponseSchema = z.object({
  emails: z.array(emailResponseSchema),
  total: z.number().int().nonnegative(),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().min(0),
});

export type EmailFilterInput = z.input<typeof emailFilterSchema>;
export type UpdateEmailInput = z.input<typeof updateEmailSchema>;

// Re-export types for consumers
export type { Email, EmailClassification, EmailListParams, EmailListResponse, ClassifyResponse } from "@/lib/types/emails";
