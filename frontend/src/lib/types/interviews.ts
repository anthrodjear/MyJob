export type InterviewStatus =
  | "pending"
  | "starting"
  | "active"
  | "completed"
  | "failed"
  | "cancelled";

export type InterviewMode = "assist" | "autonomous";

export type TranscriptSpeaker = "candidate" | "ai" | "system";

export interface InterviewSession {
  id: string;
  application_id: string;
  mode: InterviewMode;
  status: InterviewStatus;
  external_session_id: string | null;
  provider: string;
  model: string;
  transcript: TranscriptEntry[];
  score: number | null;
  feedback: Record<string, unknown> | null;
  started_at: string | null;
  ended_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface TranscriptEntry {
  id: string;
  speaker: TranscriptSpeaker;
  content: string;
  timestamp: string;
}
