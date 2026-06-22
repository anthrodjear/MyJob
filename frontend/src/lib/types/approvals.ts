export type ApprovalStatus = "pending" | "approved" | "rejected";

export interface JobSnapshot {
  title: string;
  company: string;
  location: string;
  url: string;
  description: string;
  requirements: string[];
  score: number;
  tier: string;
  scored_at: string;
}

export interface Approval {
  id: string;
  application_id: string;
  job_snapshot: JobSnapshot;
  resume_preview_path: string | null;
  cover_letter_preview: string | null;
  status: ApprovalStatus;
  rejection_reason: string | null;
  created_at: string;
  reviewed_at: string | null;
}

export interface ApprovalListParams {
  status?: ApprovalStatus;
  application_id?: string;
  limit?: number;
  offset?: number;
}

export interface ApprovalListResponse {
  approvals: Approval[];
  total: number;
  limit: number;
  offset: number;
}

export interface ApprovePartialResponse {
  status: string;
  warning: string;
  approval: Approval;
}
