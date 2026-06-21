/**
 * Activity log types — from backend activity domain.
 *
 * Used for the dashboard activity feed.
 */

export type ActivityEntityType = "application" | "job" | "email" | "task" | "profile";

export type ActivityEventType =
  | "application_created"
  | "application_status_changed"
  | "application_submitted"
  | "job_discovered"
  | "job_scored"
  | "email_received"
  | "email_classified"
  | "task_created"
  | "task_completed"
  | "task_failed"
  | "profile_updated";

/**
 * Discriminated union for activity details based on event type.
 * Provides type-safe access to event-specific fields.
 */
export type ActivityDetails =
  | { event_type: "application_created"; application_id: string }
  | { event_type: "application_status_changed"; old_status: string; new_status: string; notes?: string }
  | { event_type: "application_submitted"; application_id: string; portal?: string }
  | { event_type: "job_discovered"; job_id: string; source?: string }
  | { event_type: "job_scored"; job_id: string; score: number }
  | { event_type: "email_received"; email_id: string; from: string; subject?: string }
  | { event_type: "email_classified"; email_id: string; classification: string }
  | { event_type: "task_created"; task_id: string; task_type: string }
  | { event_type: "task_completed"; task_id: string; result?: unknown }
  | { event_type: "task_failed"; task_id: string; error: string }
  | { event_type: "profile_updated"; field: string }
  | Record<string, unknown>; // Fallback for unknown/forward-compatible events

export interface ActivityResponse {
  id: string;
  event_type: ActivityEventType;
  entity_type: ActivityEntityType;
  entity_id: string;
  details: ActivityDetails;
  created_at: string;
}

export interface ActivityListResponse {
  activities: ActivityResponse[];
  total: number;
  limit: number;
  offset: number;
}

export interface ActivityListParams {
  entity_type?: ActivityEntityType;
  entity_id?: string;
  event_type?: ActivityEventType;
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
}