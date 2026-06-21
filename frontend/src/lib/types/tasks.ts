export type TaskStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

export type TaskType =
  | "job_discovery"
  | "job_scoring"
  | "application_submit"
  | "embedding_generate"
  | "cover_letter_gen"
  | "resume_generate"
  | "resume_tailor"
  | "email_check"
  | "interview_prep"
  | "voice_session"
  | "fill_form";

export interface Task {
  id: string;
  type: TaskType;
  status: TaskStatus;
  params: Record<string, unknown> | null;
  result: Record<string, unknown> | null;
  error: string | null;
  attempts: number;
  max_attempts: number;
  priority: number;
  scheduled_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}
