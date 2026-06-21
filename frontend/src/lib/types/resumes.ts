export interface Resume {
  id: string;
  name: string;
  specialization: string;
  template_path: string;
  focus_skills: string[];
  highlight_experience: string[];
  content: ResumeContent;
  pdf_key: string | null;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ResumeContent {
  summary: string;
  skills: string[];
  experience: ExperienceEntry[];
  projects: ProjectEntry[];
  education: EducationEntry[];
  certifications: string[];
  languages: LanguageEntry[];
  links: LinkEntry[];
}

export interface ExperienceEntry {
  title: string;
  company: string;
  location: string;
  start_date: string;
  end_date: string;
  description: string;
  skills_used: string[];
  highlights: string[];
}

export interface ProjectEntry {
  name: string;
  description: string;
  technologies: string[];
  link?: string;
  start_date?: string;
  end_date?: string;
}

export interface EducationEntry {
  institution: string;
  degree: string;
  field: string;
  start_date: string;
  end_date: string;
  gpa?: string;
  honors?: string[];
}

export interface LanguageEntry {
  language: string;
  proficiency: string;
}

export interface LinkEntry {
  type: string;
  url: string;
  label?: string;
}

export interface CoverLetter {
  id: string;
  job_id: string | null;
  resume_id: string | null;
  job_title: string | null;
  content: string;
  model: string | null;
  prompt_version: string | null;
  resume_version: number | null;
  pdf_key: string | null;
  strengths: string[] | null;
  gaps: string[] | null;
  word_count: number | null;
  version: number;
  created_at: string;
  updated_at: string;
}
