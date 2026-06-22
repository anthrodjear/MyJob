export type EmailClassification =
  | "interview_invite"
  | "rejection"
  | "offer"
  | "follow_up"
  | "spam"
  | "phishing"
  | "other";

export interface Email {
  id: string;
  application_id: string | null;
  message_id: string;
  from_address: string;
  to_address: string | null;
  subject: string | null;
  body: string | null;
  received_at: string;
  classification: EmailClassification | null;
  is_read: boolean;
  reply_draft: string | null;
  created_at: string;
}

export interface EmailListParams {
  application_id?: string;
  classification?: EmailClassification;
  limit?: number;
  offset?: number;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  limit: number;
  offset: number;
}

export interface ClassifyResponse {
  email_id: string;
  classification: EmailClassification;
  confidence: number;
  reasoning: string;
}
