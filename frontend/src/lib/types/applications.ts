import type { SortDirection } from "./common";

export type ApplicationStatus =
  | "draft"
  | "queued"
  | "applied"
  | "assessment"
  | "phone_screen"
  | "technical"
  | "final"
  | "offer"
  | "rejected";

export type ApprovalTier = "auto" | "review" | "reject";

export interface Application {
  id: string;
  job_id: string;
  resume_id: string | null;
  cover_letter_id: string | null;
  status: ApplicationStatus;
  approval_tier: ApprovalTier;
  applied_at: string | null;
  response_at: string | null;
  interview_at: string | null;
  notes: string | null;
  portal_type: string | null;
  portal_url: string | null;
  form_data: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface ApplicationEvent {
  id: string;
  application_id: string;
  old_status: string;
  new_status: string;
  notes: string;
  created_at: string;
}

export interface ApplicationStats {
  total: number;
  by_status: Partial<Record<ApplicationStatus, number>>;
}

export interface ApplicationListParams {
  page?: number;
  limit?: number;
  status?: ApplicationStatus;
  min_score?: number;
  sort_by?: string;
  sort_dir?: SortDirection;
}
