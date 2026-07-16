export interface Job {
  id: string;
  source_id: string;
  external_id: string;
  title: string;
  company: string;
  location: string;
  remote_type: string;
  salary_min: number;
  salary_max: number;
  salary_currency: string;
  description: string;
  requirements: string;
  url: string;
  application_url: string;
  company_url: string;
  source: string;
  source_name: string;
  posted_at: string | null;
  scraped_at: string;
  match_score: number;
  match_details: Record<string, unknown> | null;
  score_tier: string | null;
  scored_at: string | null;
  scoring_reasoning: string | null;
  scoring_model: string | null;
  scoring_source: string | null;
  status: string;
  created_at: string;
  updated_at: string;
}

export type JobStatus = "discovered" | "matched" | "applied" | "archived";

export interface JobSource {
  name: string;
  tier: number;
  enabled: boolean;
}

/**
 * Query parameters for GET /jobs.
 * Backend handler: listJobsQuery in jobs/handler.go
 */
export interface JobListParams {
  status?: string;
  source_id?: string;
  min_score?: number;
  limit?: number;
  offset?: number;
}

/**
 * JobListResponse — paginated job list response from GET /jobs.
 * Backend returns { jobs, total, limit, offset }.
 */
export interface JobListResponse {
  jobs: Job[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * JobApplicationHistory — application entry from GET /jobs/:id/applications.
 */
export interface JobApplicationHistory {
  application_id: string;
  status: string;
  applied_at: string | null;
  created_at: string;
}

/**
 * ScoreDetails — per-factor scoring breakdown.
 * Aligned with backend/internal/scoring/model.go ScoreDetails.
 */
export interface ScoreDetails {
  skill_match: number;
  experience_match: number;
  location_match: number;
  salary_match: number;
  description_match: number;
}

/**
 * ScoreResponse — result of scoring a job, returned by the scoring task poll.
 * Aligned with backend/internal/scoring/dto.go ScoreResponse.
 */
export interface ScoreResponse {
  job_id: string;
  score: number;
  tier: "AUTO" | "REVIEW" | "REJECT";
  reasoning?: string;
  source?: string;
  model?: string;
  confidence?: number;
  strengths?: string[];
  gaps?: string[];
  details?: ScoreDetails;
}
