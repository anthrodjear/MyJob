# Phase 0: Frontend Foundation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the complete foundation layer — design tokens, shared primitives, layout shell, API client, types, providers, and error handling — so Phase 1 (Dashboard) can build directly on top.

**Architecture:** Next.js 16 App Router with Server Components by default. Design tokens flow from CSS custom properties → Tailwind theme → component classes. Types split by domain. API client split by domain. TanStack Query for client-side data fetching.

**Tech Stack:** Next.js 16, React 19, TypeScript (strict), Tailwind CSS v4, TanStack Query, Zod, Lucide React, `clsx` + `tailwind-merge` for class merging.

---

## File Map

| File | Responsibility |
|------|---------------|
| `src/app/globals.css` | CSS custom properties (design tokens) + Tailwind theme |
| `src/lib/utils.ts` | `cn()` class merger, `formatDate()`, `formatScore()` |
| `src/lib/constants.ts` | Status maps, source colors, API limits |
| `src/lib/types/common.ts` | Shared types: ApiResponse, PaginatedResponse, ApiError |
| `src/lib/types/jobs.ts` | Job, JobSource, MatchScore types |
| `src/lib/types/applications.ts` | Application, ApplicationStatus, Pipeline types |
| `src/lib/types/resumes.ts` | Resume, CoverLetter types |
| `src/lib/types/emails.ts` | Email, Classification types |
| `src/lib/types/approvals.ts` | ApprovalRequest types |
| `src/lib/types/interviews.ts` | InterviewSession, Transcript types |
| `src/lib/types/tasks.ts` | Task, TaskStatus types |
| `src/lib/types/user.ts` | Profile, Settings types |
| `src/lib/types/index.ts` | Re-export barrel for all types |
| `src/lib/schemas/jobs.ts` | Zod schemas for job data |
| `src/lib/schemas/applications.ts` | Zod schemas for applications |
| `src/lib/schemas/settings.ts` | Zod schemas for settings forms |
| `src/lib/api/client.ts` | Base fetch wrapper, error handling, auth headers |
| `src/components/shared/Button.tsx` | Button with variants (primary, secondary, ghost, danger) |
| `src/components/shared/Card.tsx` | Card container with optional header/footer |
| `src/components/shared/Badge.tsx` | Status/label badge with color variants |
| `src/components/shared/LoadingSkeleton.tsx` | Skeleton placeholder for loading states |
| `src/components/shared/EmptyState.tsx` | Empty state with icon, message, action |
| `src/components/shared/Modal.tsx` | Dialog/modal overlay (Client Component) |
| `src/components/shared/DataTable.tsx` | Generic sortable, paginated table |
| `src/components/shared/Pagination.tsx` | Page navigation |
| `src/components/shared/SearchInput.tsx` | Debounced search input (Client Component) |
| `src/components/shared/Tooltip.tsx` | Hover tooltip (Client Component) |
| `src/components/shared/Avatar.tsx` | User avatar with fallback initials |
| `src/components/shared/ProgressBar.tsx` | Linear progress indicator |
| `src/components/layout/AppShell.tsx` | Sidebar + TopBar + content wrapper |
| `src/components/layout/Sidebar.tsx` | Navigation sidebar |
| `src/components/layout/TopBar.tsx` | Search, notifications, avatar (Client Component) |
| `src/components/layout/MobileNav.tsx` | Hamburger menu (Client Component) |
| `src/components/providers/QueryProvider.tsx` | TanStack Query provider (Client Component) |
| `src/components/providers/ThemeProvider.tsx` | Dark/light mode (Client Component) |
| `src/components/providers/ToastProvider.tsx` | Toast notifications (Client Component) |
| `src/components/errors/ErrorBoundary.tsx` | Reusable error boundary (Client Component) |
| `src/app/layout.tsx` | Root layout with providers + Geist font |

---

## Task 1: Install Dependencies

**Files:**
- Modify: `package.json`

- [ ] **Step 1: Install required packages**

```bash
cd frontend
npm install @tanstack/react-query clsx tailwind-merge lucide-react zod
npm install -D @types/node
```

Expected: packages added to `package.json` and `node_modules/`.

- [ ] **Step 2: Verify installation**

```bash
npm ls @tanstack/react-query clsx tailwind-merge lucide-react zod 2>&1 | head -10
```

Expected: all packages listed with version numbers, no `ERR!` messages.

- [ ] **Step 3: Commit**

```bash
cd ..
git add frontend/package.json frontend/package-lock.json
git commit -m "chore(frontend): install foundation dependencies"
```

---

## Task 2: Design Tokens — globals.css

**Files:**
- Create/Modify: `src/app/globals.css`

- [ ] **Step 1: Write globals.css with design tokens and Tailwind theme**

```css
@import "tailwindcss";

:root {
  /* ── Brand ── */
  --color-primary: #2563eb;
  --color-primary-hover: #1d4ed8;
  --color-primary-light: #eff6ff;
  --color-primary-dark: #1e40af;

  /* ── Semantic: Success / Auto-Apply ── */
  --color-success: #16a34a;
  --color-success-hover: #15803d;
  --color-success-light: #f0fdf4;
  --color-success-dark: #166534;

  /* ── Semantic: Warning / Review Required ── */
  --color-warning: #d97706;
  --color-warning-hover: #b45309;
  --color-warning-light: #fffbeb;
  --color-warning-dark: #92400e;

  /* ── Semantic: Danger / Rejected ── */
  --color-danger: #dc2626;
  --color-danger-hover: #b91c1c;
  --color-danger-light: #fef2f2;
  --color-danger-dark: #991b1b;

  /* ── Semantic: Info / Applied ── */
  --color-info: #0891b2;
  --color-info-hover: #0e7490;
  --color-info-light: #ecfeff;
  --color-info-dark: #155e75;

  /* ── Neutrals ── */
  --color-bg: #ffffff;
  --color-bg-secondary: #f8fafc;
  --color-bg-tertiary: #f1f5f9;
  --color-surface: #ffffff;
  --color-surface-hover: #f8fafc;
  --color-border: #e2e8f0;
  --color-border-strong: #cbd5e1;
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-tertiary: #94a3b8;
  --color-text-inverse: #ffffff;

  /* ── Score Colors ── */
  --color-score-high: #16a34a;
  --color-score-high-bg: #f0fdf4;
  --color-score-mid: #d97706;
  --color-score-mid-bg: #fffbeb;
  --color-score-low: #dc2626;
  --color-score-low-bg: #fef2f2;

  /* ── Source Badge Colors ── */
  --color-source-indeed: #2164f3;
  --color-source-greenhouse: #24a800;
  --color-source-lever: #428bca;
  --color-source-remoteok: #00b67a;
  --color-source-linkedin: #0a66c2;
  --color-source-custom: #64748b;

  /* ── Shadows ── */
  --shadow-xs: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-sm: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);

  /* ── Typography ── */
  --font-sans: var(--font-geist-sans), ui-sans-serif, system-ui, sans-serif;
  --font-mono: var(--font-geist-mono), ui-monospace, monospace;

  /* ── Border Radius ── */
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
  --radius-xl: 0.75rem;
  --radius-2xl: 1rem;
  --radius-full: 9999px;

  /* ── Z-Index ── */
  --z-dropdown: 50;
  --z-sticky: 100;
  --z-modal-backdrop: 200;
  --z-modal: 300;
  --z-toast: 400;
}

@theme inline {
  --color-background: var(--color-bg);
  --color-foreground: var(--color-text-primary);

  --color-primary: var(--color-primary);
  --color-primary-hover: var(--color-primary-hover);
  --color-primary-light: var(--color-primary-light);
  --color-primary-dark: var(--color-primary-dark);

  --color-success: var(--color-success);
  --color-success-light: var(--color-success-light);
  --color-success-dark: var(--color-success-dark);
  --color-warning: var(--color-warning);
  --color-warning-light: var(--color-warning-light);
  --color-warning-dark: var(--color-warning-dark);
  --color-danger: var(--color-danger);
  --color-danger-light: var(--color-danger-light);
  --color-danger-dark: var(--color-danger-dark);
  --color-info: var(--color-info);
  --color-info-light: var(--color-info-light);
  --color-info-dark: var(--color-info-dark);

  --color-bg-secondary: var(--color-bg-secondary);
  --color-bg-tertiary: var(--color-bg-tertiary);
  --color-surface: var(--color-surface);
  --color-surface-hover: var(--color-surface-hover);
  --color-border: var(--color-border);
  --color-border-strong: var(--color-border-strong);
  --color-text-secondary: var(--color-text-secondary);
  --color-text-tertiary: var(--color-text-tertiary);
  --color-text-inverse: var(--color-text-inverse);

  --font-sans: var(--font-sans);
  --font-mono: var(--font-mono);
}

@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #0a0a0a;
    --color-bg-secondary: #111111;
    --color-bg-tertiary: #1a1a1a;
    --color-surface: #141414;
    --color-surface-hover: #1a1a1a;
    --color-border: #262626;
    --color-border-strong: #404040;
    --color-text-primary: #fafafa;
    --color-text-secondary: #a1a1a1;
    --color-text-tertiary: #737373;
    --color-primary-light: #172554;
    --color-success-light: #14532d;
    --color-warning-light: #451a03;
    --color-danger-light: #450a0a;
    --color-info-light: #164e63;
  }
}

body {
  font-family: var(--font-sans);
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

* {
  border-color: var(--color-border);
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd frontend
npm run build 2>&1 | tail -5
```

Expected: build succeeds (may have warnings, no errors).

- [ ] **Step 3: Commit**

```bash
git add src/app/globals.css
git commit -m "feat(frontend): add design tokens and Tailwind theme"
```

---

## Task 3: Utility Functions

**Files:**
- Create: `src/lib/utils.ts`

- [ ] **Step 1: Write utils.ts**

```typescript
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Merges Tailwind CSS classes with clsx.
 * Handles conflicting classes (e.g., "p-2 p-4" → "p-4").
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/**
 * Formats a date string to a human-readable format.
 * Returns "—" for null/undefined dates.
 */
export function formatDate(date: string | Date | null | undefined): string {
  if (!date) return "—";
  const d = typeof date === "string" ? new Date(date) : date;
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

/**
 * Formats a match score as a percentage string.
 * Returns "—" for null/undefined scores.
 */
export function formatScore(score: number | null | undefined): string {
  if (score == null) return "—";
  return `${Math.round(score)}%`;
}

/**
 * Returns the color class for a match score.
 * Green (≥80%), yellow (50-79%), red (<50%).
 */
export function scoreColorClass(score: number): string {
  if (score >= 80) return "bg-success-light text-success-dark";
  if (score >= 50) return "bg-warning-light text-warning-dark";
  return "bg-danger-light text-danger-dark";
}

/**
 * Truncates a string to maxLen characters, appending "..." if truncated.
 */
export function truncate(s: string, maxLen: number): string {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen) + "...";
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/lib/utils.ts
git commit -m "feat(frontend): add utility functions (cn, formatDate, formatScore)"
```

---

## Task 4: Constants

**Files:**
- Create: `src/lib/constants.ts`

- [ ] **Step 1: Write constants.ts**

```typescript
/** Application status → display config mapping. */
export const APPLICATION_STATUS = {
  discovered: { label: "Discovered", color: "bg-bg-tertiary text-text-secondary" },
  applied: { label: "Applied", color: "bg-info-light text-info-dark" },
  responded: { label: "Responded", color: "bg-warning-light text-warning-dark" },
  interview: { label: "Interview", color: "bg-primary-light text-primary-dark" },
  offer: { label: "Offer", color: "bg-success-light text-success-dark" },
  rejected: { label: "Rejected", color: "bg-danger-light text-danger-dark" },
} as const;

export type ApplicationStatusKey = keyof typeof APPLICATION_STATUS;

/** Email classification → display config mapping. */
export const EMAIL_CLASSIFICATION = {
  interview_invite: { label: "Interview Invite", color: "bg-primary-light text-primary-dark" },
  rejection: { label: "Rejection", color: "bg-danger-light text-danger-dark" },
  offer: { label: "Offer", color: "bg-success-light text-success-dark" },
  follow_up: { label: "Follow Up", color: "bg-warning-light text-warning-dark" },
  spam: { label: "Spam", color: "bg-bg-tertiary text-text-tertiary" },
  phishing: { label: "Phishing", color: "bg-danger-light text-danger-dark" },
  other: { label: "Other", color: "bg-bg-tertiary text-text-secondary" },
} as const;

/** Job source → color config for source badges. */
export const SOURCE_COLORS: Record<string, string> = {
  indeed: "bg-[#2164f3] text-white",
  greenhouse: "bg-[#24a800] text-white",
  lever: "bg-[#428bca] text-white",
  remoteok: "bg-[#00b67a] text-white",
  linkedin: "bg-[#0a66c2] text-white",
  custom: "bg-[#64748b] text-white",
};

/** Default pagination limits. */
export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;

/** Polling intervals (ms). */
export const POLL_INTERVAL = {
  tasks: 5000,
  emails: 30000,
  dashboard: 60000,
} as const;

/** API path prefix. */
export const API_PREFIX = "/api/v1";
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/lib/constants.ts
git commit -m "feat(frontend): add constants (status maps, source colors, limits)"
```

---

## Task 5: Domain Types

**Files:**
- Create: `src/lib/types/common.ts`
- Create: `src/lib/types/jobs.ts`
- Create: `src/lib/types/applications.ts`
- Create: `src/lib/types/resumes.ts`
- Create: `src/lib/types/emails.ts`
- Create: `src/lib/types/approvals.ts`
- Create: `src/lib/types/interviews.ts`
- Create: `src/lib/types/tasks.ts`
- Create: `src/lib/types/user.ts`
- Create: `src/lib/types/index.ts`

- [ ] **Step 1: Write common.ts**

```typescript
/** Standard API error response. */
export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

/** Standard API success response wrapper. */
export interface ApiResponse<T> {
  data: T;
}

/** Paginated list response. */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

/** Pagination query parameters. */
export interface PaginationParams {
  page?: number;
  limit?: number;
}

/** Sort direction. */
export type SortDirection = "asc" | "desc";
```

- [ ] **Step 2: Write jobs.ts**

```typescript
export interface Job {
  id: string;
  title: string;
  company: string;
  location: string | null;
  description: string | null;
  url: string | null;
  source: string;
  external_id: string;
  match_score: number | null;
  status: string;
  salary_min: number | null;
  salary_max: number | null;
  remote: boolean | null;
  tags: string[];
  posted_at: string | null;
  discovered_at: string;
  updated_at: string;
}

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
  sort_dir?: "asc" | "desc";
}
```

- [ ] **Step 3: Write applications.ts**

```typescript
export type ApplicationStatus =
  | "discovered"
  | "applied"
  | "responded"
  | "interview"
  | "offer"
  | "rejected";

export interface Application {
  id: string;
  job_id: string;
  resume_id: string | null;
  cover_letter_id: string | null;
  status: ApplicationStatus;
  match_score: number | null;
  notes: string | null;
  applied_at: string | null;
  responded_at: string | null;
  created_at: string;
  updated_at: string;
  // Joined fields
  job_title?: string;
  company?: string;
  resume_name?: string;
}

export interface ApplicationStats {
  total: number;
  by_status: Record<ApplicationStatus, number>;
  response_rate: number;
  interview_rate: number;
}

export interface ApplicationListParams {
  page?: number;
  limit?: number;
  status?: ApplicationStatus;
  min_score?: number;
  sort_by?: string;
  sort_dir?: "asc" | "desc";
}
```

- [ ] **Step 4: Write resumes.ts**

```typescript
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
  location: string | null;
  start_date: string;
  end_date: string | null;
  description: string;
  highlights: string[];
}

export interface ProjectEntry {
  name: string;
  description: string;
  technologies: string[];
  url: string | null;
}

export interface EducationEntry {
  institution: string;
  degree: string;
  field: string;
  start_date: string;
  end_date: string | null;
}

export interface LanguageEntry {
  name: string;
  proficiency: string;
}

export interface LinkEntry {
  label: string;
  url: string;
}

export interface CoverLetter {
  id: string;
  application_id: string | null;
  resume_id: string | null;
  job_id: string | null;
  content: string;
  model: string | null;
  prompt_version: string | null;
  resume_version: number | null;
  strengths: string[];
  gaps: string[];
  created_at: string;
  updated_at: string;
}
```

- [ ] **Step 5: Write emails.ts**

```typescript
export type EmailClassification =
  | "interview_invite"
  | "rejection"
  | "offer"
  | "follow_up"
  | "spam"
  | "phishing"
  | "other";

export interface Email {
  id: string;
  application_id: string | null;
  message_id: string;
  from_address: string;
  to_address: string | null;
  subject: string | null;
  body: string | null;
  received_at: string;
  classification: EmailClassification | null;
  is_read: boolean;
  reply_draft: string | null;
  created_at: string;
}

export interface EmailListParams {
  page?: number;
  limit?: number;
  classification?: EmailClassification;
  is_read?: boolean;
}
```

- [ ] **Step 6: Write approvals.ts**

```typescript
export type ApprovalStatus = "pending" | "approved" | "rejected";

export type ApprovalType = "resume" | "cover_letter" | "form_submission";

export interface ApprovalRequest {
  id: string;
  type: ApprovalType;
  status: ApprovalStatus;
  job_id: string | null;
  application_id: string | null;
  resume_id: string | null;
  cover_letter_id: string | null;
  payload: Record<string, unknown>;
  review_notes: string | null;
  created_at: string;
  updated_at: string;
  // Joined fields
  job_title?: string;
  company?: string;
}

export interface ApprovalListParams {
  page?: number;
  limit?: number;
  type?: ApprovalType;
  status?: ApprovalStatus;
}
```

- [ ] **Step 7: Write interviews.ts**

```typescript
export type InterviewStatus = "scheduled" | "in_progress" | "completed" | "cancelled";

export interface InterviewSession {
  id: string;
  application_id: string | null;
  job_id: string | null;
  status: InterviewStatus;
  mode: string;
  provider: string;
  started_at: string | null;
  ended_at: string | null;
  duration_seconds: number | null;
  transcript: TranscriptEntry[];
  feedback: InterviewFeedback | null;
  created_at: string;
}

export interface TranscriptEntry {
  id: string;
  speaker: string;
  content: string;
  timestamp: string;
}

export interface InterviewFeedback {
  overall_score: number;
  strengths: string[];
  improvements: string[];
  summary: string;
}
```

- [ ] **Step 8: Write tasks.ts**

```typescript
export type TaskStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

export type TaskType =
  | "job_scoring"
  | "resume_generate"
  | "cover_letter_gen"
  | "application_submit"
  | "fill_form"
  | "email_check"
  | "interview_prep"
  | "embedding_generate"
  | "voice_session"
  | "resume_tailor"
  | "jobs:discover";

export interface Task {
  id: string;
  type: TaskType;
  status: TaskStatus;
  payload: Record<string, unknown>;
  result: Record<string, unknown> | null;
  error: string | null;
  created_at: string;
  updated_at: string;
}
```

- [ ] **Step 9: Write user.ts**

```typescript
export interface Profile {
  id: string;
  name: string;
  email: string;
  phone: string | null;
  location: string | null;
  summary: string | null;
  skills: string[];
  preferences: UserPreferences;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface UserPreferences {
  auto_apply: boolean;
  min_match_score: number;
  max_applications_per_day: number;
  preferred_sources: string[];
  notification_email: boolean;
}

export interface Settings {
  profile: Profile;
  scraping_enabled: boolean;
  scoring_model: string;
  llm_provider: string;
}
```

- [ ] **Step 10: Write index.ts barrel**

```typescript
export type { ApiError, ApiResponse, PaginatedResponse, PaginationParams, SortDirection } from "./common";
export type { Job, JobSource, JobListParams } from "./jobs";
export type { Application, ApplicationStatus, ApplicationStats, ApplicationListParams } from "./applications";
export type { Resume, ResumeContent, CoverLetter } from "./resumes";
export type { Email, EmailClassification, EmailListParams } from "./emails";
export type { ApprovalRequest, ApprovalStatus, ApprovalType, ApprovalListParams } from "./approvals";
export type { InterviewSession, InterviewStatus, TranscriptEntry, InterviewFeedback } from "./interviews";
export type { Task, TaskStatus, TaskType } from "./tasks";
export type { Profile, UserPreferences, Settings } from "./user";
```

- [ ] **Step 11: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 12: Commit**

```bash
git add src/lib/types/
git commit -m "feat(frontend): add domain types (jobs, applications, resumes, emails, approvals, interviews, tasks, user)"
```

---

## Task 6: Zod Schemas

**Files:**
- Create: `src/lib/schemas/jobs.ts`
- Create: `src/lib/schemas/applications.ts`
- Create: `src/lib/schemas/settings.ts`

- [ ] **Step 1: Write jobs.ts schemas**

```typescript
import { z } from "zod";

export const jobFilterSchema = z.object({
  search: z.string().optional(),
  source: z.string().optional(),
  status: z.string().optional(),
  min_score: z.number().min(0).max(100).optional(),
  page: z.number().int().positive().default(1),
  limit: z.number().int().min(1).max(100).default(20),
});

export type JobFilterInput = z.input<typeof jobFilterSchema>;
export type JobFilter = z.output<typeof jobFilterSchema>;
```

- [ ] **Step 2: Write applications.ts schemas**

```typescript
import { z } from "zod";

export const applicationStatusSchema = z.enum([
  "discovered",
  "applied",
  "responded",
  "interview",
  "offer",
  "rejected",
]);

export const applicationFilterSchema = z.object({
  status: applicationStatusSchema.optional(),
  min_score: z.number().min(0).max(100).optional(),
  page: z.number().int().positive().default(1),
  limit: z.number().int().min(1).max(100).default(20),
});

export type ApplicationFilterInput = z.input<typeof applicationFilterSchema>;
export type ApplicationFilter = z.output<typeof applicationFilterSchema>;
```

- [ ] **Step 3: Write settings.ts schemas**

```typescript
import { z } from "zod";

export const profileSchema = z.object({
  name: z.string().min(1, "Name is required"),
  email: z.string().email("Invalid email address"),
  phone: z.string().optional(),
  location: z.string().optional(),
  summary: z.string().optional(),
  skills: z.array(z.string()).optional(),
});

export const preferencesSchema = z.object({
  auto_apply: z.boolean(),
  min_match_score: z.number().min(0).max(100),
  max_applications_per_day: z.number().int().positive(),
  preferred_sources: z.array(z.string()),
  notification_email: z.boolean(),
});

export type ProfileInput = z.input<typeof profileSchema>;
export type PreferencesInput = z.input<typeof preferencesSchema>;
```

- [ ] **Step 4: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add src/lib/schemas/
git commit -m "feat(frontend): add Zod schemas for jobs, applications, settings"
```

---

## Task 7: API Client

**Files:**
- Create: `src/lib/api/client.ts`

- [ ] **Step 1: Write client.ts**

```typescript
import { API_PREFIX } from "@/lib/constants";

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/** Custom error class for API failures. */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/**
 * Base fetch wrapper for all backend API calls.
 * Handles JSON serialization, error parsing, and auth headers.
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = `${BACKEND_URL}${API_PREFIX}${path}`;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };

  // Attach JWT from localStorage if available (client-side only)
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const res = await fetch(url, {
    ...options,
    headers,
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    const code = body?.error?.code ?? "UNKNOWN_ERROR";
    const message = body?.error?.message ?? `Request failed with status ${res.status}`;
    throw new ApiError(res.status, code, message);
  }

  // Handle 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}

/** GET request helper. */
export function apiGet<T>(path: string): Promise<T> {
  return apiFetch<T>(path, { method: "GET" });
}

/** POST request helper. */
export function apiPost<T>(path: string, data?: unknown): Promise<T> {
  return apiFetch<T>(path, {
    method: "POST",
    body: data ? JSON.stringify(data) : undefined,
  });
}

/** PATCH request helper. */
export function apiPatch<T>(path: string, data: unknown): Promise<T> {
  return apiFetch<T>(path, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

/** DELETE request helper. */
export function apiDelete<T>(path: string): Promise<T> {
  return apiFetch<T>(path, { method: "DELETE" });
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/lib/api/client.ts
git commit -m "feat(frontend): add base API client with error handling"
```

---

## Task 8: Shared Primitives — Button

**Files:**
- Create: `src/components/shared/Button.tsx`

- [ ] **Step 1: Write Button.tsx**

```typescript
import { type ButtonHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  children: ReactNode;
  loading?: boolean;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-text-inverse hover:bg-primary-hover focus-visible:ring-primary",
  secondary:
    "bg-bg-tertiary text-text-primary border border-border hover:bg-border focus-visible:ring-primary",
  ghost:
    "bg-transparent text-text-secondary hover:bg-bg-tertiary hover:text-text-primary focus-visible:ring-primary",
  danger:
    "bg-danger text-text-inverse hover:bg-danger-hover focus-visible:ring-danger",
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
  lg: "px-6 py-3 text-base",
};

export function Button({
  variant = "primary",
  size = "md",
  className,
  children,
  loading,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-md font-medium",
        "transition-colors duration-150",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
        "disabled:pointer-events-none disabled:opacity-50",
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading}
      {...props}
    >
      {loading && (
        <svg
          className="mr-2 h-4 w-4 animate-spin"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      )}
      {children}
    </button>
  );
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/components/shared/Button.tsx
git commit -m "feat(frontend): add Button component with variants"
```

---

## Task 9: Shared Primitives — Card, Badge, EmptyState, LoadingSkeleton

**Files:**
- Create: `src/components/shared/Card.tsx`
- Create: `src/components/shared/Badge.tsx`
- Create: `src/components/shared/EmptyState.tsx`
- Create: `src/components/shared/LoadingSkeleton.tsx`

- [ ] **Step 1: Write Card.tsx**

```typescript
import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface CardProps {
  children: ReactNode;
  className?: string;
  padding?: boolean;
}

export function Card({ children, className, padding = true }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface shadow-sm",
        padding && "p-6",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("mb-4 border-b border-border pb-4", className)}>
      {children}
    </div>
  );
}

export function CardContent({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn(className)}>{children}</div>;
}
```

- [ ] **Step 2: Write Badge.tsx**

```typescript
import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface BadgeProps {
  children: ReactNode;
  className?: string;
  variant?: "default" | "success" | "warning" | "danger" | "info";
}

const variantStyles = {
  default: "bg-bg-tertiary text-text-secondary",
  success: "bg-success-light text-success-dark",
  warning: "bg-warning-light text-warning-dark",
  danger: "bg-danger-light text-danger-dark",
  info: "bg-info-light text-info-dark",
};

export function Badge({ children, className, variant = "default" }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        variantStyles[variant],
        className,
      )}
    >
      {children}
    </span>
  );
}
```

- [ ] **Step 3: Write EmptyState.tsx**

```typescript
import { type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Button } from "./Button";

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center justify-center py-12 text-center", className)}>
      {icon && (
        <div className="mb-4 text-text-tertiary" aria-hidden="true">
          {icon}
        </div>
      )}
      <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-text-secondary">{description}</p>
      {action && (
        <Button onClick={action.onClick} className="mt-4">
          {action.label}
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Write LoadingSkeleton.tsx**

```typescript
import { cn } from "@/lib/utils";

interface SkeletonProps {
  className?: string;
}

export function Skeleton({ className }: SkeletonProps) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-bg-tertiary", className)}
      aria-hidden="true"
    />
  );
}

/** Card-shaped skeleton for job cards, stat cards, etc. */
export function CardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("rounded-xl border border-border bg-surface p-6 shadow-sm", className)}>
      <Skeleton className="h-4 w-3/4 mb-3" />
      <Skeleton className="h-3 w-1/2 mb-2" />
      <Skeleton className="h-3 w-2/3" />
    </div>
  );
}

/** Table row skeleton. */
export function TableRowSkeleton({ columns = 5 }: { columns?: number }) {
  return (
    <tr>
      {Array.from({ length: columns }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <Skeleton className="h-4 w-full" />
        </td>
      ))}
    </tr>
  );
}
```

- [ ] **Step 5: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 6: Commit**

```bash
git add src/components/shared/Card.tsx src/components/shared/Badge.tsx src/components/shared/EmptyState.tsx src/components/shared/LoadingSkeleton.tsx
git commit -m "feat(frontend): add Card, Badge, EmptyState, LoadingSkeleton primitives"
```

---

## Task 10: Shared Primitives — Modal, DataTable, Pagination, SearchInput, Avatar, ProgressBar, Tooltip

**Files:**
- Create: `src/components/shared/Modal.tsx`
- Create: `src/components/shared/DataTable.tsx`
- Create: `src/components/shared/Pagination.tsx`
- Create: `src/components/shared/SearchInput.tsx`
- Create: `src/components/shared/Avatar.tsx`
- Create: `src/components/shared/ProgressBar.tsx`
- Create: `src/components/shared/Tooltip.tsx`

- [ ] **Step 1: Write Modal.tsx (Client Component)**

```typescript
"use client";

import { type ReactNode, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  className?: string;
}

export function Modal({ open, onClose, title, children, className }: ModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    if (open) {
      document.addEventListener("keydown", handleEscape);
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-[--z-modal] flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-label={title}
    >
      <div className="fixed inset-0 bg-black/50" onClick={onClose} aria-hidden="true" />
      <div
        className={cn(
          "relative z-10 w-full max-w-lg rounded-xl bg-surface p-6 shadow-lg",
          className,
        )}
      >
        {title && (
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-text-primary">{title}</h2>
            <button
              onClick={onClose}
              className="rounded-md p-1 text-text-tertiary hover:text-text-primary"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Write DataTable.tsx**

```typescript
import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface Column<T> {
  key: string;
  header: string;
  render?: (item: T) => ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  onRowClick?: (item: T) => void;
  emptyMessage?: string;
}

export function DataTable<T extends { id?: string }>({
  columns,
  data,
  onRowClick,
  emptyMessage = "No data available",
}: DataTableProps<T>) {
  return (
    <div className="overflow-hidden rounded-xl border border-border">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border bg-bg-secondary">
            {columns.map((col) => (
              <th
                key={col.key}
                className={cn(
                  "px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-secondary",
                  col.className,
                )}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center text-text-tertiary">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            data.map((item, i) => (
              <tr
                key={item.id ?? i}
                className={cn(
                  "border-b border-border last:border-0",
                  onRowClick && "cursor-pointer hover:bg-surface-hover",
                )}
                onClick={() => onRowClick?.(item)}
              >
                {columns.map((col) => (
                  <td key={col.key} className={cn("px-4 py-3 text-sm", col.className)}>
                    {col.render ? col.render(item) : String((item as Record<string, unknown>)[col.key] ?? "—")}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 3: Write Pagination.tsx**

```typescript
import { cn } from "@/lib/utils";
import { Button } from "./Button";

interface PaginationProps {
  page: number;
  total: number;
  limit: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export function Pagination({ page, total, limit, onPageChange, className }: PaginationProps) {
  const totalPages = Math.ceil(total / limit);

  if (totalPages <= 1) return null;

  return (
    <nav className={cn("flex items-center justify-between", className)} aria-label="Pagination">
      <span className="text-sm text-text-secondary">
        Page {page} of {totalPages}
      </span>
      <div className="flex gap-2">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
        >
          Previous
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
        >
          Next
        </Button>
      </div>
    </nav>
  );
}
```

- [ ] **Step 4: Write SearchInput.tsx (Client Component)**

```typescript
"use client";

import { type InputHTMLAttributes, useCallback } from "react";
import { cn } from "@/lib/utils";
import { Search } from "lucide-react";

interface SearchInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  onSearch: (value: string) => void;
  debounceMs?: number;
}

export function SearchInput({ onSearch, debounceMs = 300, className, ...props }: SearchInputProps) {
  const debouncedSearch = useCallback(
    debounce((value: string) => onSearch(value), debounceMs),
    [onSearch, debounceMs],
  );

  return (
    <div className={cn("relative", className)}>
      <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-tertiary" />
      <input
        type="text"
        className={cn(
          "w-full rounded-md border border-border bg-surface py-2 pl-9 pr-3 text-sm",
          "placeholder:text-text-tertiary",
          "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
        )}
        onChange={(e) => debouncedSearch(e.target.value)}
        {...props}
      />
    </div>
  );
}

function debounce<T extends (...args: unknown[]) => unknown>(fn: T, ms: number): T {
  let timer: ReturnType<typeof setTimeout>;
  return ((...args: unknown[]) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  }) as T;
}
```

- [ ] **Step 5: Write Avatar.tsx**

```typescript
import { cn } from "@/lib/utils";

interface AvatarProps {
  name: string;
  src?: string | null;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeStyles = {
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-12 w-12 text-base",
};

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

function stringToColor(name: string): string {
  const colors = [
    "bg-primary-light text-primary-dark",
    "bg-success-light text-success-dark",
    "bg-warning-light text-warning-dark",
    "bg-info-light text-info-dark",
    "bg-danger-light text-danger-dark",
  ];
  let hash = 0;
  for (const ch of name) hash = ch.charCodeAt(0) + ((hash << 5) - hash);
  return colors[Math.abs(hash) % colors.length];
}

export function Avatar({ name, src, size = "md", className }: AvatarProps) {
  if (src) {
    return (
      <img
        src={src}
        alt={name}
        className={cn("rounded-full object-cover", sizeStyles[size], className)}
      />
    );
  }

  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-full font-medium",
        sizeStyles[size],
        stringToColor(name),
        className,
      )}
      aria-label={name}
    >
      {getInitials(name)}
    </div>
  );
}
```

- [ ] **Step 6: Write ProgressBar.tsx**

```typescript
import { cn } from "@/lib/utils";

interface ProgressBarProps {
  value: number; // 0-100
  label?: string;
  color?: "primary" | "success" | "warning" | "danger";
  className?: string;
}

const colorStyles = {
  primary: "bg-primary",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
};

export function ProgressBar({ value, label, color = "primary", className }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));

  return (
    <div className={cn("w-full", className)}>
      {label && (
        <div className="mb-1 flex items-center justify-between text-sm">
          <span className="text-text-secondary">{label}</span>
          <span className="font-mono tabular-nums text-text-primary">{clamped}%</span>
        </div>
      )}
      <div className="h-2 w-full overflow-hidden rounded-full bg-bg-tertiary">
        <div
          className={cn("h-full rounded-full transition-all duration-300", colorStyles[color])}
          style={{ width: `${clamped}%` }}
          role="progressbar"
          aria-valuenow={clamped}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={label}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 7: Write Tooltip.tsx (Client Component)**

```typescript
"use client";

import { type ReactNode, useState } from "react";
import { cn } from "@/lib/utils";

interface TooltipProps {
  content: string;
  children: ReactNode;
  className?: string;
}

export function Tooltip({ content, children, className }: TooltipProps) {
  const [show, setShow] = useState(false);

  return (
    <div
      className={cn("relative inline-block", className)}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children}
      {show && (
        <div
          className="absolute bottom-full left-1/2 z-[--z-tooltip] mb-2 -translate-x-1/2 rounded-md bg-text-primary px-2 py-1 text-xs text-text-inverse whitespace-nowrap"
          role="tooltip"
        >
          {content}
          <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-text-primary" />
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 8: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 9: Commit**

```bash
git add src/components/shared/
git commit -m "feat(frontend): add remaining shared primitives (Modal, DataTable, Pagination, SearchInput, Avatar, ProgressBar, Tooltip)"
```

---

## Task 11: Layout Components — AppShell, Sidebar, TopBar, MobileNav

**Files:**
- Create: `src/components/layout/AppShell.tsx`
- Create: `src/components/layout/Sidebar.tsx`
- Create: `src/components/layout/TopBar.tsx`
- Create: `src/components/layout/MobileNav.tsx`

- [ ] **Step 1: Write Sidebar.tsx**

```typescript
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Briefcase,
  FileText,
  CheckCircle,
  Mail,
  Settings,
} from "lucide-react";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/dashboard/jobs", label: "Jobs", icon: Briefcase },
  { href: "/dashboard/applications", label: "Applications", icon: FileText },
  { href: "/dashboard/approvals", label: "Approvals", icon: CheckCircle },
  { href: "/dashboard/email", label: "Email", icon: Mail },
  { href: "/dashboard/settings", label: "Settings", icon: Settings },
];

interface SidebarProps {
  className?: string;
}

export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname();

  return (
    <aside
      className={cn(
        "hidden w-64 flex-col border-r border-border bg-bg-secondary lg:flex",
        className,
      )}
    >
      <div className="flex h-16 items-center px-6">
        <span className="text-xl font-bold text-primary">MyJob</span>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navItems.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary-light text-primary-dark"
                  : "text-text-secondary hover:bg-bg-tertiary hover:text-text-primary",
              )}
            >
              <item.icon className="h-5 w-5" aria-hidden="true" />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
```

- [ ] **Step 2: Write TopBar.tsx (Client Component)**

```typescript
"use client";

import { cn } from "@/lib/utils";
import { Avatar } from "@/components/shared/Avatar";
import { Bell } from "lucide-react";

interface TopBarProps {
  className?: string;
}

export function TopBar({ className }: TopBarProps) {
  return (
    <header
      className={cn(
        "flex h-16 items-center justify-between border-b border-border bg-surface px-6",
        className,
      )}
    >
      <div className="flex-1" />
      <div className="flex items-center gap-4">
        <button
          className="relative rounded-md p-2 text-text-secondary hover:bg-bg-tertiary"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" />
        </button>
        <Avatar name="User" size="sm" />
      </div>
    </header>
  );
}
```

- [ ] **Step 3: Write MobileNav.tsx (Client Component)**

```typescript
"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Menu, X } from "lucide-react";

const navItems = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/dashboard/jobs", label: "Jobs" },
  { href: "/dashboard/applications", label: "Applications" },
  { href: "/dashboard/approvals", label: "Approvals" },
  { href: "/dashboard/email", label: "Email" },
  { href: "/dashboard/settings", label: "Settings" },
];

export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  return (
    <div className="lg:hidden">
      <button
        onClick={() => setOpen(!open)}
        className="rounded-md p-2 text-text-secondary hover:bg-bg-tertiary"
        aria-label={open ? "Close menu" : "Open menu"}
        aria-expanded={open}
      >
        {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
      </button>
      {open && (
        <nav className="absolute left-0 top-16 z-[--z-dropdown] w-full border-b border-border bg-surface shadow-lg">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setOpen(false)}
              className={cn(
                "block px-6 py-3 text-sm font-medium",
                pathname === item.href
                  ? "bg-primary-light text-primary-dark"
                  : "text-text-secondary hover:bg-bg-tertiary",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Write AppShell.tsx**

```typescript
import { type ReactNode } from "react";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";
import { MobileNav } from "./MobileNav";

interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="flex items-center border-b border-border lg:hidden">
          <MobileNav />
          <span className="ml-3 text-lg font-bold text-primary">MyJob</span>
        </div>
        <TopBar className="hidden lg:flex" />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 6: Commit**

```bash
git add src/components/layout/
git commit -m "feat(frontend): add layout components (AppShell, Sidebar, TopBar, MobileNav)"
```

---

## Task 12: Providers — QueryProvider, ThemeProvider, ToastProvider

**Files:**
- Create: `src/components/providers/QueryProvider.tsx`
- Create: `src/components/providers/ThemeProvider.tsx`
- Create: `src/components/providers/ToastProvider.tsx`

- [ ] **Step 1: Write QueryProvider.tsx**

```typescript
"use client";

import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000, // 1 minute
            retry: 1,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}
```

- [ ] **Step 2: Write ThemeProvider.tsx**

```typescript
"use client";

import { createContext, useContext, useEffect, useState } from "react";

type Theme = "light" | "dark" | "system";

const ThemeContext = createContext<{
  theme: Theme;
  setTheme: (t: Theme) => void;
}>({ theme: "system", setTheme: () => {} });

export function useTheme() {
  return useContext(ThemeContext);
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>("system");

  useEffect(() => {
    const saved = localStorage.getItem("theme") as Theme | null;
    if (saved) setTheme(saved);
  }, []);

  useEffect(() => {
    localStorage.setItem("theme", theme);
    const root = document.documentElement;
    root.classList.remove("light", "dark");

    if (theme === "system") {
      const preferred = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
      root.classList.add(preferred);
    } else {
      root.classList.add(theme);
    }
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}
```

- [ ] **Step 3: Write ToastProvider.tsx**

```typescript
"use client";

import { createContext, useCallback, useContext, useState } from "react";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";

interface Toast {
  id: string;
  message: string;
  type: "success" | "error" | "info";
}

interface ToastContextValue {
  toast: (message: string, type?: Toast["type"]) => void;
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

const typeStyles: Record<Toast["type"], string> = {
  success: "bg-success text-text-inverse",
  error: "bg-danger text-text-inverse",
  info: "bg-primary text-text-inverse",
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const toast = useCallback((message: string, type: Toast["type"] = "info") => {
    const id = Math.random().toString(36).slice(2);
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 5000);
  }, []);

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-4 right-4 z-[--z-toast] flex flex-col gap-2">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={cn(
              "flex items-center gap-3 rounded-lg px-4 py-3 text-sm shadow-lg",
              typeStyles[t.type],
            )}
            role="alert"
          >
            <span>{t.message}</span>
            <button onClick={() => dismiss(t.id)} className="ml-2 opacity-70 hover:opacity-100" aria-label="Dismiss">
              <X className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
```

- [ ] **Step 4: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add src/components/providers/
git commit -m "feat(frontend): add providers (QueryProvider, ThemeProvider, ToastProvider)"
```

---

## Task 13: Error Boundary

**Files:**
- Create: `src/components/errors/ErrorBoundary.tsx`

- [ ] **Step 1: Write ErrorBoundary.tsx**

```typescript
"use client";

import { type ReactNode } from "react";
import { Button } from "@/components/shared/Button";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div className="flex flex-col items-center justify-center py-12">
          <h2 className="text-xl font-semibold text-text-primary">Something went wrong</h2>
          <p className="mt-2 max-w-md text-center text-sm text-text-secondary">
            {this.state.error?.message ?? "An unexpected error occurred."}
          </p>
          <Button onClick={() => this.setState({ hasError: false, error: null })} className="mt-4">
            Try again
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}

import React from "react";
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -5
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/components/errors/ErrorBoundary.tsx
git commit -m "feat(frontend): add reusable ErrorBoundary component"
```

---

## Task 14: Root Layout

**Files:**
- Modify: `src/app/layout.tsx`

- [ ] **Step 1: Write root layout.tsx**

```typescript
import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { ThemeProvider } from "@/components/providers/ThemeProvider";
import { ToastProvider } from "@/components/providers/ToastProvider";

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] });
const geistMono = Geist_Mono({ variable: "--font-geist-mono", subsets: ["latin"] });

export const metadata: Metadata = {
  title: "MyJob — AI Job Search Agent",
  description: "AI-powered job search automation platform",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <ThemeProvider>
          <QueryProvider>
            <ToastProvider>{children}</ToastProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
npm run build 2>&1 | tail -10
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add src/app/layout.tsx
git commit -m "feat(frontend): wire root layout with providers and Geist font"
```

---

## Task 15: Dashboard Placeholder Page

**Files:**
- Modify: `src/app/page.tsx`
- Create: `src/app/dashboard/layout.tsx`
- Create: `src/app/dashboard/page.tsx`

- [ ] **Step 1: Write app/page.tsx redirect**

```typescript
import { redirect } from "next/navigation";

export default function Home() {
  redirect("/dashboard");
}
```

- [ ] **Step 2: Write dashboard/layout.tsx**

```typescript
import { AppShell } from "@/components/layout/AppShell";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}
```

- [ ] **Step 3: Write dashboard/page.tsx (placeholder)**

```typescript
export default function DashboardPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold text-text-primary">Dashboard</h1>
      <p className="mt-2 text-text-secondary">
        Phase 0 complete. Foundation ready for Phase 1 implementation.
      </p>
    </div>
  );
}
```

- [ ] **Step 4: Verify build succeeds**

```bash
npm run build 2>&1 | tail -10
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add src/app/page.tsx src/app/dashboard/
git commit -m "feat(frontend): add dashboard placeholder with AppShell layout"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** All files from Phase 0 file map are accounted for in tasks
- [x] **Placeholder scan:** No TBD/TODO steps — all code blocks are complete
- [x] **Type consistency:** Types in `lib/types/` match the structures used in schemas and API client
- [x] **Import paths:** All `@/` imports are consistent
- [x] **Component exports:** Named exports only (no default exports except page/layout files)
- [x] **Client Component boundaries:** Only components with hooks, events, or browser APIs get `"use client"`
- [x] **Commit granularity:** Each task ends with a focused commit

---

## After Phase 0

With Phase 0 complete, the foundation is established:
- Design tokens in `globals.css`
- Shared primitives in `components/shared/`
- Layout shell in `components/layout/`
- Types split by domain in `lib/types/`
- API client in `lib/api/client.ts`
- Providers wired in root layout

**Phase 1 (Dashboard)** can now build directly on these foundations.
