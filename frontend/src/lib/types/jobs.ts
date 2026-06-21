import type { SortDirection } from "./common";

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
  posted_at: string | null;
  scraped_at: string;
  match_score: number;
  match_details: Record<string, unknown> | null;
  status: string;
  created_at: string;
  updated_at: string;
  source_name?: string;
}

export type JobStatus = "discovered" | "matched" | "applied" | "archived";

export interface JobSource {
  name: string;
  tier: number;
  enabled: boolean;
}

export interface JobListParams {
  page?: number;
  limit?: number;
  source?: string;
  status?: string;
  min_score?: number;
  search?: string;
  sort_by?: string;
  sort_dir?: SortDirection;
}

/**
 * JobListResponse — paginated job list response from GET /jobs.
 */
export interface JobListResponse {
  items: Job[];
  total: number;
  page: number;
  limit: number;
}
