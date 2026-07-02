/**
 * Zod v4 schemas for Settings/Profile domain.
 *
 * Provides runtime validation for profile forms and API requests.
 * Aligns with types in @/lib/types/profile.
 *
 * @see backend/internal/profile/dto.go
 */

import { z } from "zod";
import { SKILL_PROFICIENCIES } from "@/lib/types/profile";

// ---------------------------------------------------------------------------
// Sub-schemas
// ---------------------------------------------------------------------------

/** Skill proficiency enum — matches backend constants. */
export const skillProficiencySchema = z.enum(SKILL_PROFICIENCIES);

/** Single skill entry schema. */
export const skillSchema = z.object({
  name: z.string().min(1, "Skill name is required"),
  proficiency: skillProficiencySchema.optional(),
  years: z.number().int().min(0).max(50).optional(),
});

/** Single education entry schema. */
export const educationSchema = z.object({
  institution: z.string().min(1, "Institution is required"),
  degree: z.string().min(1, "Degree is required"),
  field: z.string().optional(),
  start_year: z
    .number()
    .int()
    .min(1900, "Year must be >= 1900")
    .max(2100, "Year must be <= 2100")
    .optional(),
  end_year: z
    .number()
    .int()
    .min(1900, "Year must be >= 1900")
    .max(2100, "Year must be <= 2100")
    .optional(),
  gpa: z.string().optional(),
});

/** Profile links schema. */
export const linksSchema = z.object({
  linkedin: z.string().url("Must be a valid URL").optional().or(z.literal("")),
  github: z.string().url("Must be a valid URL").optional().or(z.literal("")),
  portfolio: z.string().url("Must be a valid URL").optional().or(z.literal("")),
});

/** Profile preferences schema. */
export const preferencesSchema = z
  .object({
    target_titles: z.array(z.string().min(1)).optional(),
    target_locations: z.array(z.string().min(1)).optional(),
    remote_only: z.boolean().optional(),
    min_salary: z.number().int().min(0).optional(),
    max_salary: z.number().int().min(0).optional(),
    work_authorization: z.string().optional(),
    years_experience: z.number().int().min(0).max(50).optional(),
    resume_tone: z.string().optional(),
    resume_style: z.string().optional(),
    auto_apply_threshold: z.number().int().min(0).max(100).optional(),
    cover_letter_style: z.string().optional(),
  })
  .refine(
    (data) => {
      if (data.min_salary !== undefined && data.max_salary !== undefined) {
        return data.min_salary <= data.max_salary;
      }
      return true;
    },
    {
      message: "Minimum salary must be less than or equal to maximum salary",
      path: ["max_salary"],
    },
  );

// ---------------------------------------------------------------------------
// Form Schemas (used by React Hook Form or controlled forms)
// ---------------------------------------------------------------------------

/** Preferences form schema — for the preferences section of the settings page. */
export const preferencesFormSchema = preferencesSchema;

/** Skills form schema — wraps skills array with form-level validation. */
export const skillsFormSchema = z.object({
  skills: z.array(skillSchema),
});

/** Education form schema — wraps education array with form-level validation. */
export const educationFormSchema = z.object({
  education: z.array(educationSchema),
});

/** Links form schema — for the links section. */
export const linksFormSchema = linksSchema;

// ---------------------------------------------------------------------------
// API Request Schemas
// ---------------------------------------------------------------------------

/** Full profile PUT request schema. */
export const updateProfileRequestSchema = z.object({
  preferences: preferencesSchema,
  skills: z.array(skillSchema).optional(),
  education: z.array(educationSchema).optional(),
  links: linksSchema.optional(),
});

/** Partial profile PATCH request schema — all fields optional. */
export const patchProfileRequestSchema = z.object({
  preferences: preferencesSchema.partial().optional(),
  skills: z.array(skillSchema).optional(),
  education: z.array(educationSchema).optional(),
  links: linksSchema.partial().optional(),
});

// ---------------------------------------------------------------------------
// API Response Schemas
// ---------------------------------------------------------------------------

/** Profile stats response schema. */
export const profileStatsSchema = z.object({
  skill_count: z.number().int().nonnegative(),
  education_count: z.number().int().nonnegative(),
  has_resume_preferences: z.boolean(),
  has_links: z.boolean(),
});

/** Full profile response schema. */
export const profileSchema = z.object({
  id: z.string().uuid(),
  data: z.object({
    preferences: preferencesSchema,
    skills: z.array(skillSchema).optional(),
    education: z.array(educationSchema).optional(),
    links: linksSchema.optional(),
  }),
  stats: profileStatsSchema,
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
});

// ---------------------------------------------------------------------------
// Type Exports (Zod v4 inference)
// ---------------------------------------------------------------------------

export type SkillProficiencyValidated = z.output<typeof skillProficiencySchema>;
export type SkillValidated = z.output<typeof skillSchema>;
export type EducationValidated = z.output<typeof educationSchema>;
export type LinksValidated = z.output<typeof linksSchema>;
export type PreferencesValidated = z.output<typeof preferencesSchema>;
export type ProfileValidated = z.output<typeof profileSchema>;
export type PatchProfileRequestValidated = z.output<typeof patchProfileRequestSchema>;
export type UpdateProfileRequestValidated = z.output<typeof updateProfileRequestSchema>;

// Re-export types that align with existing type definitions
export type {
  Profile,
  ProfileData,
  Skill,
  Education,
  PatchProfileRequest,
  UpdateProfileRequest,
  SkillProficiency,
  ProfilePreferences,
  ProfileLinks,
} from "@/lib/types/profile";
