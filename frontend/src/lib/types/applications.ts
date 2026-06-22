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
  created_at: string;
  updated_at: string;
}

export interface ApplicationEvent {
  id: string;
  application_id: string;
  old_status: ApplicationStatus | null;
  new_status: ApplicationStatus;
  notes: string;
  created_at: string;
}

/**
 * Backend ApplicationStatsResponse from GET /applications/stats
 * Includes by_tier for approval tier breakdown.
 */
export interface ApplicationStatsResponse {
  total: number;
  by_status: Partial<Record<ApplicationStatus, number>>;
  by_tier: Partial<Record<ApprovalTier, number>>;
}

/**
 * Query parameters for GET /applications.
 * Backend handler: listApplicationsQuery in applications/handler.go
 */
export interface ApplicationListParams {
  status?: ApplicationStatus;
  job_id?: string;
  portal_type?: string;
  limit?: number;
  offset?: number;
}
