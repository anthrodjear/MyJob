export interface Profile {
  id: string;
  data: ProfileData;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ProfileData {
  preferences: ProfilePreferences;
  skills: Skill[];
  education: Education[];
  links: ProfileLinks;
}

export interface ProfilePreferences {
  target_titles?: string[];
  target_locations?: string[];
  remote_only?: boolean;
  min_salary?: number | null;
  max_salary?: number | null;
  work_authorization?: string;
  years_experience?: number | null;
  resume_tone?: string;
  resume_style?: string;
  auto_apply_threshold?: number | null;
  cover_letter_style?: string;
}

export interface ProfileLinks {
  linkedin?: string;
  github?: string;
  portfolio?: string;
}

export type SkillProficiency = "beginner" | "intermediate" | "advanced" | "expert";

export interface Skill {
  name: string;
  proficiency?: SkillProficiency;
  years?: number;
}

export interface Education {
  institution: string;
  degree: string;
  field?: string;
  start_year?: number;
  end_year?: number;
  gpa?: string;
}
