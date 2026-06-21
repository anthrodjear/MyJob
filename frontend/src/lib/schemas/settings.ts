/**
 * Zod schemas for profile/settings form validation.
 *
 * These schemas validate user input on the Settings page before
 * sending it to the backend API. They enforce business rules
 * (e.g., salary ranges, valid skill proficiencies) that the
 * backend also validates — defense in depth.
 *
 * Usage:
 *   import { profileSchema } from "@/lib/schemas/settings";
 *   const result = profileSchema.safeParse(formData);
 *   if (!result.success) {
 *     // Handle validation errors
 *   }
 */
import { z } from "zod";

/**
 * Valid skill proficiency levels.
 * Mirrors backend/internal/profile/model.go Skill* constants.
 */
const skillProficiencyEnum = z.enum(["beginner", "intermediate", "advanced", "expert"]);

/**
 * Schema for a single skill entry within the profile.
 * Validates name, optional proficiency level, and optional years of experience.
 */
const skillSchema = z.object({
  /** Skill name (e.g., "TypeScript", "React", "Go"). */
  name: z.string().min(1, "Skill name is required"),
  /** Proficiency level. Optional — backend defaults to empty string. */
  proficiency: skillProficiencyEnum.optional(),
  /** Years of experience with this skill. Optional. */
  years: z.number().int().min(0).max(50).optional(),
});

/**
 * Schema for a single education entry within the profile.
 * Validates institution, degree, and optional year ranges.
 */
const educationSchema = z.object({
  /** Institution name (e.g., "MIT", "Stanford"). */
  institution: z.string().min(1, "Institution is required"),
  /** Degree type (e.g., "B.S.", "M.S.", "PhD"). */
  degree: z.string().min(1, "Degree is required"),
  /** Field of study (e.g., "Computer Science"). */
  field: z.string().optional(),
  /** Year the program started. Must be 1900–2100 if provided. */
  start_year: z.number().int().min(1900).max(2100).optional(),
  /** Year the program ended. Must be >= start_year if both provided. */
  end_year: z.number().int().min(1900).max(2100).optional(),
  /** GPA string (kept as string to support "3.8/4.0" format). */
  gpa: z.string().max(20).optional(),
});

/**
 * Schema for external profile links.
 * All fields optional — user may only have some profiles.
 */
const profileLinksSchema = z.object({
  /** LinkedIn profile URL. */
  linkedin: z.string().url("Invalid LinkedIn URL").optional(),
  /** GitHub profile URL. */
  github: z.string().url("Invalid GitHub URL").optional(),
  /** Personal portfolio URL. */
  portfolio: z.string().url("Invalid portfolio URL").optional(),
});

/**
 * Schema for job search preferences within the profile.
 * Groups all preferences that influence how the agent searches,
 * scores, and generates application materials.
 */
const profilePreferencesSchema = z.object({
  /** Job titles the user is targeting (e.g., ["Software Engineer", "Frontend Developer"]). */
  target_titles: z.array(z.string()).optional(),
  /** Target locations (e.g., ["Remote", "San Francisco", "New York"]). */
  target_locations: z.array(z.string()).optional(),
  /** Only apply to remote positions. */
  remote_only: z.boolean().optional(),
  /** Minimum salary expectation. Null = no minimum. */
  min_salary: z.number().int().min(0).nullable().optional(),
  /** Maximum salary expectation. Null = no maximum. */
  max_salary: z.number().int().min(0).nullable().optional(),
  /** Work authorization status (e.g., "US Citizen", "H1B"). Free text. */
  work_authorization: z.string().optional(),
  /** Years of professional experience. Null = not specified. */
  years_experience: z.number().int().min(0).max(50).nullable().optional(),
  /** Resume tone preference (e.g., "professional", "concise", "detailed"). */
  resume_tone: z.string().optional(),
  /** Resume style preference (e.g., "chronological", "functional", "hybrid"). */
  resume_style: z.string().optional(),
  /** Score threshold (0–100) above which applications auto-submit. Null = always require approval. */
  auto_apply_threshold: z.number().int().min(0).max(100).nullable().optional(),
  /** Cover letter style (e.g., "formal", "conversational", "technical"). */
  cover_letter_style: z.string().optional(),
});

/**
 * Schema for validating a profile PATCH request.
 *
 * All fields are optional — only provided fields are updated.
 * This matches the backend's ApplyPatch semantics:
 * - nil pointer → unchanged
 * - non-nil pointer → overwrite
 * - slices: nil = don't change, non-nil (even empty) = replace
 *
 * @example
 *   const patch = profileSchema.parse({
 *     preferences: { target_titles: ["Senior Engineer"] },
 *   });
 */
export const profileSchema = z.object({
  /** Preferences to update. Nested object — only provided fields change. */
  preferences: profilePreferencesSchema.optional(),
  /** Replace entire skills array. Null = don't change, [] = clear, [...] = replace. */
  skills: z.array(skillSchema).nullable().optional(),
  /** Replace entire education array. Null = don't change, [] = clear, [...] = replace. */
  education: z.array(educationSchema).nullable().optional(),
  /** Profile links to update. Nested object — only provided fields change. */
  links: profileLinksSchema.optional(),
}).refine(
  (data) => {
    const prefs = data.preferences;
    if (prefs?.min_salary != null && prefs?.max_salary != null) {
      return prefs.min_salary <= prefs.max_salary;
    }
    return true;
  },
  {
    message: "min_salary must be <= max_salary",
    path: ["preferences", "min_salary"],
  },
);

/** Input type for profile PATCH — what the form provides. */
export type ProfileInput = z.input<typeof profileSchema>;

/** Output type for profile PATCH — fully resolved after validation. */
export type ProfileOutput = z.output<typeof profileSchema>;

/**
 * Input type for profile preferences — what the settings form provides.
 * Extracted from the nested preferences schema for standalone use.
 */
export type PreferencesInput = z.input<typeof profilePreferencesSchema>;

/**
 * Schema for validating a full profile GET response from the API.
 * Used to validate data before rendering on the Settings page.
 *
 * NOTE: The `version` field is sent as an ETag header, not in the JSON body.
 * Extract it from the response headers in the fetch layer.
 */
export const profileResponseSchema = z.object({
  /** Profile ID (UUID). */
  id: z.string().uuid(),
  /** Nested profile data (preferences, skills, education, links). */
  data: z.object({
    preferences: profilePreferencesSchema,
    skills: z.array(skillSchema),
    education: z.array(educationSchema),
    links: profileLinksSchema,
  }),
  /** Computed stats — skill count, education count, has_resume_preferences, has_links. */
  stats: z.object({
    /** Number of skills in the profile. */
    skill_count: z.number().int(),
    /** Number of education entries in the profile. */
    education_count: z.number().int(),
    /** Whether the profile has resume preferences configured. */
    has_resume_preferences: z.boolean(),
    /** Whether the profile has any links configured. */
    has_links: z.boolean(),
  }),
  /** ISO 8601 timestamp. */
  created_at: z.string().datetime(),
  /** ISO 8601 timestamp. */
  updated_at: z.string().datetime(),
});

/** Type inferred from profileResponseSchema — the validated API response shape. */
export type ProfileResponse = z.output<typeof profileResponseSchema>;
