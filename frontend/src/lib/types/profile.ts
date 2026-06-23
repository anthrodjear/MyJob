/**
 * Profile types — mirrors the backend profile domain.
 *
 * The profile is a singleton resource (one per user). The frontend uses
 * PATCH for partial updates and PUT for full replacement. ETag-based
 * optimistic concurrency is handled via the API client.
 *
 * @see backend/internal/profile/model.go
 * @see backend/internal/profile/dto.go
 */

// ---------------------------------------------------------------------------
// Skill Proficiency Constants
// ---------------------------------------------------------------------------

/** Allowed proficiency levels for skills. */
export const SKILL_PROFICIENCIES = ["beginner", "intermediate", "advanced", "expert"] as const;

/** Skill proficiency type derived from constants. */
export type SkillProficiency = (typeof SKILL_PROFICIENCIES)[number];

// ---------------------------------------------------------------------------
// Core Domain Types
// ---------------------------------------------------------------------------

/**
 * Profile preferences — job targeting, resume generation, application behavior.
 * Matches backend ProfilePreferences struct.
 */
export interface ProfilePreferences {
  /** Job titles to search for. */
  target_titles?: string[];
  /** Preferred locations (e.g., "Remote", "New York"). */
  target_locations?: string[];
  /** Only show remote jobs. */
  remote_only?: boolean;
  /** Minimum salary filter (annual, USD). */
  min_salary?: number;
  /** Maximum salary filter (annual, USD). */
  max_salary?: number;
  /** Work authorization status (e.g., "US Citizen", "H1B"). */
  work_authorization?: string;
  /** Years of professional experience. */
  years_experience?: number;
  /** Tone for generated resumes (e.g., "professional", "casual"). */
  resume_tone?: string;
  /** Style for generated resumes (e.g., "chronological", "functional"). */
  resume_style?: string;
  /** Score threshold (0-100) above which applications auto-submit. */
  auto_apply_threshold?: number;
  /** Style for generated cover letters. */
  cover_letter_style?: string;
}

/**
 * External profile links.
 * Matches backend ProfileLinks struct.
 */
export interface ProfileLinks {
  /** LinkedIn profile URL. */
  linkedin?: string;
  /** GitHub profile URL. */
  github?: string;
  /** Portfolio website URL. */
  portfolio?: string;
}

/**
 * A single skill entry.
 * Matches backend Skill struct.
 */
export interface Skill {
  /** Skill name (e.g., "Go", "TypeScript"). */
  name: string;
  /** Proficiency level. */
  proficiency?: SkillProficiency;
  /** Years of experience with this skill. */
  years?: number;
}

/**
 * A single education entry.
 * Matches backend Education struct.
 */
export interface Education {
  /** Institution name. */
  institution: string;
  /** Degree level (e.g., "BS", "MS", "PhD"). */
  degree: string;
  /** Field of study. */
  field?: string;
  /** Year studies began. */
  start_year?: number;
  /** Year studies ended (or expected). */
  end_year?: number;
  /** GPA (string to support "3.8" or "3.8/4.0" formats). */
  gpa?: string;
}

/**
 * Profile data — the JSONB payload stored in the profiles table.
 * Matches backend ProfileData struct.
 */
export interface ProfileData {
  /** Job search and resume preferences. */
  preferences: ProfilePreferences;
  /** List of skills. */
  skills?: Skill[];
  /** Education history. */
  education?: Education[];
  /** External profile links. */
  links?: ProfileLinks;
}

/**
 * Profile stats — computed on every GET.
 * Matches backend ProfileStatsResponse struct.
 */
export interface ProfileStats {
  /** Number of skills in the profile. */
  skill_count: number;
  /** Number of education entries. */
  education_count: number;
  /** Whether resume preferences are configured. */
  has_resume_preferences: boolean;
  /** Whether any external links are provided. */
  has_links: boolean;
}

/**
 * Full profile response from GET /api/v1/profile.
 * Matches backend ProfileResponse struct.
 */
export interface Profile {
  /** Profile UUID. */
  id: string;
  /** Profile data (preferences, skills, education, links). */
  data: ProfileData;
  /** Computed stats. */
  stats: ProfileStats;
  /** Creation timestamp (ISO 8601). */
  created_at: string;
  /** Last update timestamp (ISO 8601). */
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Request Types
// ---------------------------------------------------------------------------

/**
 * PATCH request — partial profile update.
 * All fields are pointers so nil = "don't change".
 * Matches backend PatchProfileRequest struct.
 */
export interface PatchProfileRequest {
  /** Preferences to merge (partial). */
  preferences?: {
    target_titles?: string[];
    target_locations?: string[];
    remote_only?: boolean;
    min_salary?: number;
    max_salary?: number;
    work_authorization?: string;
    years_experience?: number;
    resume_tone?: string;
    resume_style?: string;
    auto_apply_threshold?: number;
    cover_letter_style?: string;
  };
  /** Skills list (nil = don't change, non-nil = replace). */
  skills?: Skill[];
  /** Education list (nil = don't change, non-nil = replace). */
  education?: Education[];
  /** Links to merge (partial). */
  links?: {
    linkedin?: string;
    github?: string;
    portfolio?: string;
  };
}

/**
 * PUT request — full profile replacement.
 * Matches backend UpdateProfileRequest struct.
 */
export interface UpdateProfileRequest {
  /** Full preferences (required). */
  preferences: ProfilePreferences;
  /** Skills list. */
  skills?: Skill[];
  /** Education list. */
  education?: Education[];
  /** Links. */
  links?: ProfileLinks;
}
