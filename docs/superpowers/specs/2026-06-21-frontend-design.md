# Frontend Design Spec — MyJob Dashboard

> **Date:** 2026-06-21
> **Status:** Approved — ready for implementation planning
> **Author:** Frontend Developer Agent

---

## 1. Overview

Build the Next.js 16 frontend for the MyJob AI Job Search Agent. The frontend is a dashboard application that provides visibility and control over job discovery, application tracking, AI-generated documents, and approval workflows.

**Stack:**
- Next.js 16 (App Router)
- React 19
- TypeScript (strict mode)
- Tailwind CSS v4
- TanStack Query (React Query)

**Design principles:**
- Server Components by default; Client Components only for interactivity
- Design tokens via CSS custom properties → Tailwind theme (no hardcoded colors)
- Types split by domain (no god files)
- API client split by domain (no god files)
- Feature-level Zod schemas for runtime validation
- Error boundaries per route segment

---

## 2. Folder Structure

```
src/
├── app/
│   ├── layout.tsx                    # Root layout (Geist font, providers)
│   ├── page.tsx                      # Redirect to /dashboard
│   ├── dashboard/
│   │   ├── layout.tsx                # Dashboard shell (AppShell)
│   │   ├── page.tsx                  # Dashboard home (Server Component)
│   │   ├── error.tsx                 # Dashboard error boundary
│   │   ├── jobs/
│   │   │   ├── page.tsx
│   │   │   └── error.tsx
│   │   ├── applications/
│   │   │   ├── page.tsx
│   │   │   └── error.tsx
│   │   ├── approvals/
│   │   │   ├── page.tsx
│   │   │   └── error.tsx
│   │   ├── email/
│   │   │   ├── page.tsx
│   │   │   └── error.tsx
│   │   └── settings/
│   │       ├── page.tsx
│   │       └── error.tsx
│   └── globals.css                   # Tailwind + design tokens
│
├── components/
│   ├── layout/
│   │   ├── AppShell.tsx              # Sidebar + TopBar + content wrapper
│   │   ├── Sidebar.tsx               # Navigation sidebar
│   │   ├── TopBar.tsx                # Search, notifications, avatar
│   │   └── MobileNav.tsx             # Hamburger menu for mobile
│   │
│   ├── shared/
│   │   ├── Button.tsx
│   │   ├── Card.tsx
│   │   ├── Badge.tsx
│   │   ├── Modal.tsx
│   │   ├── DataTable.tsx
│   │   ├── LoadingSkeleton.tsx
│   │   ├── EmptyState.tsx
│   │   ├── SearchInput.tsx
│   │   ├── Pagination.tsx
│   │   ├── Tooltip.tsx
│   │   ├── Avatar.tsx
│   │   └── ProgressBar.tsx
│   │
│   ├── providers/
│   │   ├── QueryProvider.tsx         # TanStack Query provider
│   │   ├── ThemeProvider.tsx         # Dark/light mode
│   │   └── ToastProvider.tsx         # Toast notifications
│   │
│   ├── errors/
│   │   └── ErrorBoundary.tsx         # Reusable error boundary
│   │
│   ├── dashboard/
│   │   ├── DashboardStats.tsx        # KPI cards (Server)
│   │   ├── PipelineSummary.tsx       # Horizontal funnel (Server)
│   │   ├── ActivityFeed.tsx          # Recent activity (Server)
│   │   ├── QuickActions.tsx          # Action buttons (Server)
│   │   └── UpcomingTasks.tsx         # Pending items (Server)
│   │
│   ├── jobs/
│   │   ├── JobCard.tsx               # Job summary card (Server)
│   │   ├── JobList.tsx               # Grid of JobCards (Server)
│   │   ├── JobDetail.tsx             # Full job view (Server)
│   │   ├── MatchScoreBadge.tsx       # Color-coded score (Server)
│   │   ├── SourceBadge.tsx           # Source indicator (Server)
│   │   ├── JobFilters.tsx            # Filter controls (Client)
│   │   └── JobCompare.tsx            # Side-by-side compare (Server)
│   │
│   ├── applications/
│   │   ├── ApplicationTable.tsx      # Sortable table (Server)
│   │   ├── ApplicationRow.tsx        # Single row (Server)
│   │   ├── PipelineBoard.tsx         # Kanban view (Client)
│   │   ├── PipelineCard.tsx          # Draggable card (Server)
│   │   ├── ApplicationDetail.tsx     # Full detail (Server)
│   │   └── StatusBadge.tsx           # Status indicator (Server)
│   │
│   ├── approvals/
│   │   ├── ApprovalQueue.tsx         # Pending items list (Server)
│   │   ├── ApprovalCard.tsx          # Single pending item (Server)
│   │   ├── ApprovalActions.tsx       # Approve/reject/edit (Client)
│   │   ├── ApprovalFilters.tsx       # Filter controls (Client)
│   │   └── BulkApproval.tsx          # Batch operations (Client)
│   │
│   ├── email/
│   │   ├── EmailList.tsx             # Email list (Server)
│   │   ├── EmailDetail.tsx           # Full email view (Server)
│   │   ├── EmailNotification.tsx     # Real-time toast (Client)
│   │   └── EmailActions.tsx          # Reply/archive/flag (Client)
│   │
│   ├── interview/
│   │   ├── InterviewCard.tsx         # Upcoming interview (Server)
│   │   ├── PrepChecklist.tsx         # Prep items (Server)
│   │   ├── CoachingSession.tsx       # Live coaching (Client)
│   │   └── FeedbackPanel.tsx         # Post-session feedback (Server)
│   │
│   └── resume/
│       ├── ResumePreview.tsx         # PDF preview (Server)
│       ├── CoverLetterPreview.tsx    # Cover letter preview (Server)
│       ├── ResumeEditor.tsx          # Rich text editor (Client)
│       ├── ResumeDiff.tsx            # Side-by-side compare (Server)
│       └── TemplateSelector.tsx      # Template picker (Server)
│
├── lib/
│   ├── api/
│   │   ├── client.ts                 # Base fetch wrapper, error handling
│   │   ├── jobs.ts                   # Job endpoints
│   │   ├── applications.ts           # Application endpoints
│   │   ├── resumes.ts                # Resume endpoints
│   │   ├── emails.ts                 # Email endpoints
│   │   ├── approvals.ts              # Approval endpoints
│   │   ├── scoring.ts                # Scoring endpoints
│   │   ├── tasks.ts                  # Task status polling
│   │   ├── profile.ts                # Profile endpoints
│   │   └── dashboard.ts              # Dashboard aggregate endpoints
│   │
│   ├── types/
│   │   ├── jobs.ts                   # Job, JobSource, MatchScore
│   │   ├── applications.ts           # Application, Status, Pipeline
│   │   ├── resumes.ts                # Resume, CoverLetter
│   │   ├── emails.ts                 # Email, Classification
│   │   ├── approvals.ts              # ApprovalRequest
│   │   ├── interviews.ts             # InterviewSession, Transcript
│   │   ├── tasks.ts                  # Task, TaskStatus
│   │   ├── user.ts                   # Profile, Settings
│   │   └── common.ts                 # Pagination, ApiResponse, Error
│   │
│   ├── schemas/
│   │   ├── jobs.ts                   # Zod schemas for job data
│   │   ├── applications.ts           # Zod schemas for applications
│   │   └── settings.ts               # Zod schemas for settings forms
│   │
│   ├── utils.ts                      # cn(), formatDate(), etc.
│   └── constants.ts                  # Status maps, source colors, limits
│
└── hooks/
    ├── useJobs.ts                    # Job query hooks
    ├── useApplications.ts            # Application query/mutation hooks
    ├── useApprovals.ts               # Approval query/mutation hooks
    ├── useEmails.ts                  # Email query hooks
    ├── useDebounce.ts                # Debounce hook
    └── usePagination.ts              # Pagination state hook
```

---

## 3. Data Fetching Architecture

### Server Components (initial render)

Server Components fetch data directly from the Go backend API. No `useEffect`, no loading spinners — the component renders with data or shows a loading state via `loading.tsx`.

```typescript
// app/dashboard/page.tsx
import { getDashboardStats } from "@/lib/api/dashboard";

export default async function DashboardPage() {
  const stats = await getDashboardStats();
  return <DashboardStats data={stats} />;
}
```

### TanStack Query (client-side)

Used for:
- Filtering and pagination (Jobs, Applications)
- Mutations (approve, reject, update settings)
- Polling (email notifications, task status)
- Optimistic updates (approval workflow)
- Background refresh (dashboard stats)

```typescript
// hooks/useJobs.ts
export function useJobsQuery(filters: JobFilters) {
  return useQuery({
    queryKey: ["jobs", filters],
    queryFn: () => fetchJobs(filters),
    placeholderData: keepPreviousData,
  });
}
```

### API Client Pattern

```typescript
// lib/api/client.ts
const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    ...options,
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...options?.headers },
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: { message: "Request failed" } }));
    throw new ApiError(res.status, error.error?.code, error.error?.message);
  }
  return res.json();
}
```

---

## 4. Design Token Flow

```
globals.css (:root)
    ↓ CSS custom properties
Tailwind theme (@theme inline)
    ↓ Theme classes
Components (bg-primary, text-success-dark, rounded-card)
    ↓ No hardcoded colors
```

Tokens defined in `context/ui-tokens.md` — applied to `globals.css` and Tailwind theme. Components consume via utility classes only.

---

## 5. Server/Client Component Rules

### Server Components (default)

Use for:
- Data display (tables, cards, badges, lists)
- Layout and structure
- Formatting and static content
- Initial data fetching

### Client Components (opt-in)

Add `"use client"` ONLY when the component needs:
- `useState`, `useReducer`, `useEffect`, `useContext`
- Browser APIs (`window`, `document`, `localStorage`)
- Event handlers (`onClick`, `onChange`, `onSubmit`)
- Real-time updates (WebSocket, polling, intervals)
- Drag and drop
- Forms with validation
- Animations requiring state

**Rule: Push Client Components to the leaf level. Keep the parent tree as Server Components.**

---

## 6. Error Handling

### Error Boundaries per Route Segment

Each route under `dashboard/` gets an `error.tsx`:

```typescript
// app/dashboard/jobs/error.tsx
"use client";

export default function JobsError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <h2 className="text-xl font-semibold text-text-primary">Jobs failed to load</h2>
      <p className="text-text-secondary mt-2">{error.message}</p>
      <Button onClick={reset} className="mt-4">Try again</Button>
    </div>
  );
}
```

### API Error Handling

- API errors: Inline error message within the component
- Form validation: Field-level errors below the input
- Network errors: Toast notification with "Retry" button
- Never render a broken component — always handle loading/error/empty

---

## 7. Build Phases

### Phase 0: Foundation

**Goal:** Everything Dashboard depends on exists.

**Files to create:**
- `globals.css` — design tokens + Tailwind theme
- `lib/utils.ts` — `cn()` helper, `formatDate()`, `formatScore()`
- `lib/constants.ts` — status maps, source colors, status badges
- `lib/types/*` — all domain types
- `lib/api/client.ts` — base fetch wrapper
- `lib/api/dashboard.ts` — dashboard endpoints
- `lib/schemas/*` — Zod validation schemas
- `components/shared/*` — Button, Card, Badge, LoadingSkeleton, EmptyState, Modal, DataTable, Pagination, SearchInput, Tooltip, Avatar, ProgressBar
- `components/layout/*` — AppShell, Sidebar, TopBar, MobileNav
- `components/providers/*` — QueryProvider, ThemeProvider, ToastProvider
- `components/errors/ErrorBoundary.tsx`
- `app/layout.tsx` — root layout with providers
- `app/globals.css` — tokens + theme

**Validates:** Design tokens work, components render, layout is responsive.

### Phase 1: Dashboard

**Goal:** Complete vertical slice — one page end-to-end.

**Files to create:**
- `app/dashboard/layout.tsx` — dashboard shell
- `app/dashboard/page.tsx` — main dashboard (Server Component)
- `app/dashboard/error.tsx` — error boundary
- `components/dashboard/DashboardStats.tsx` — 4 KPI cards
- `components/dashboard/PipelineSummary.tsx` — horizontal funnel
- `components/dashboard/ActivityFeed.tsx` — recent activity
- `components/dashboard/QuickActions.tsx` — action buttons
- `components/dashboard/UpcomingTasks.tsx` — pending items
- `lib/api/dashboard.ts` — stats/activity/tasks endpoints
- `hooks/useJobs.ts` — initial query hooks

**Validates:** UI system, server/client boundary, responsive layout, API integration.

### Phase 2: Jobs

**Goal:** Core product flow — search, filter, paginate, detail view.

**Files to create:**
- `app/dashboard/jobs/page.tsx`
- `app/dashboard/jobs/error.tsx`
- `components/jobs/*` — JobCard, JobList, JobDetail, MatchScoreBadge, SourceBadge, JobFilters, JobCompare
- `lib/api/jobs.ts`
- `lib/types/jobs.ts`
- `lib/schemas/jobs.ts`
- `hooks/useJobs.ts` (expand)

**Validates:** Search, filtering, pagination, API querying, reusable cards.

### Phase 3: Applications

**Goal:** Hardest page — table + Kanban + detail drawer + status changes.

**Files to create:**
- `app/dashboard/applications/page.tsx`
- `app/dashboard/applications/error.tsx`
- `components/applications/*` — ApplicationTable, ApplicationRow, PipelineBoard, PipelineCard, ApplicationDetail, StatusBadge
- `lib/api/applications.ts`
- `lib/types/applications.ts`
- `hooks/useApplications.ts` — query + mutation hooks

**Validates:** Table/Kanban toggle, state management, hooks pattern, optimistic updates.

### Phase 4: Approvals

**Goal:** AI generation → human review → submit flow.

**Files to create:**
- `app/dashboard/approvals/page.tsx`
- `app/dashboard/approvals/error.tsx`
- `components/approvals/*` — ApprovalQueue, ApprovalCard, ApprovalActions, ApprovalFilters, BulkApproval
- `components/resume/ResumePreview.tsx`
- `components/resume/CoverLetterPreview.tsx`
- `lib/api/approvals.ts`
- `lib/types/approvals.ts`
- `hooks/useApprovals.ts`

**Validates:** Approval workflow, preview components, bulk operations.

### Phase 5: Email + Interview

**Goal:** Isolated subsystems.

**Files to create:**
- `app/dashboard/email/page.tsx` + `error.tsx`
- `components/email/*` — EmailList, EmailDetail, EmailNotification, EmailActions
- `lib/api/emails.ts`, `lib/types/emails.ts`, `hooks/useEmails.ts`
- `app/dashboard/interviews/page.tsx` + `error.tsx` (if interview route exists)
- `components/interview/*` — InterviewCard, PrepChecklist, CoachingSession, FeedbackPanel

**Validates:** Email monitoring, interview coaching (WebRTC/LiveKit).

### Phase 6: Settings

**Goal:** Forms, config management.

**Files to create:**
- `app/dashboard/settings/page.tsx` + `error.tsx`
- Settings forms (profile, scraper config, matching criteria, notifications)
- `lib/api/profile.ts`, `lib/types/user.ts`, `lib/schemas/settings.ts`

**Validates:** Form handling, settings persistence.

---

## 8. Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data fetching | TanStack Query | Async jobs, polling, mutations, optimistic updates |
| Styling | Tailwind CSS v4 | Utility-first, design tokens via CSS vars, no CSS modules |
| State management | React Query + local state | No global store needed — server state via RQ, UI state via hooks |
| Forms | Native form + Zod validation | No form library needed for this scope |
| Routing | Next.js App Router | Server Components, streaming, parallel routes |
| Error handling | ErrorBoundary per route segment | Jobs crash doesn't take down Dashboard |
| Type safety | TypeScript strict + Zod schemas | Compile-time + runtime validation |
| Dark mode | Tailwind `dark:` + CSS vars | Token system supports it, user preference |
| Icons | Lucide React | Tree-shakeable, consistent, modern |

---

## 9. What This Spec Does NOT Cover

- Backend API changes (all endpoints exist)
- Authentication flow (single-user JWT, already implemented)
- Voice coaching WebRTC details (Phase 5 concern)
- LaTeX resume rendering (backend concern)
- Docker Compose changes (frontend is already a service)

---

**Spec written and committed. Ready for implementation planning via `writing-plans` skill.**
