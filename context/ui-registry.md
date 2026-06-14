# UI Component Registry

> Component architecture plan for the Next.js 16 frontend (TypeScript + Tailwind CSS + App Router).
> Currently: default `create-next-app` scaffolding. No custom components exist yet.

---

## Directory Structure

```
src/
├── app/
│   ├── layout.tsx              # Root layout (Geist font, providers)
│   ├── page.tsx                # Landing / redirect to dashboard
│   ├── dashboard/
│   │   ├── page.tsx            # Main dashboard (server component)
│   │   ├── layout.tsx          # Dashboard shell (sidebar + topbar)
│   │   ├── jobs/
│   │   │   └── page.tsx        # Job listings view
│   │   ├── applications/
│   │   │   └── page.tsx        # Application pipeline view
│   │   ├── approvals/
│   │   │   └── page.tsx        # Pending approvals queue
│   │   ├── settings/
│   │   │   └── page.tsx        # User configuration
│   │   └── email/
│   │       └── page.tsx        # Recruiter email monitor
│   ├── api/
│   │   └── [...]/route.ts     # API route handlers
│   └── globals.css             # Tailwind + design tokens
├── components/
│   ├── layout/                 # Shell & navigation
│   ├── dashboard/              # Dashboard-specific widgets
│   ├── jobs/                   # Job listing components
│   ├── applications/           # Application pipeline components
│   ├── approvals/              # Approval queue components
│   ├── resume/                 # Resume/cover letter components
│   ├── email/                  # Email notification components
│   ├── interview/              # Interview prep components
│   └── shared/                 # Reusable primitives
├── lib/
│   ├── api.ts                  # Backend API client
│   ├── utils.ts                # Shared utilities
│   └── types.ts                # TypeScript interfaces
└── hooks/                      # Custom React hooks (client-side only)
```

---

## Component Inventory

### Layout Components (`components/layout/`)

| Component | Type | Purpose |
|---|---|---|
| `AppShell` | Server | Main layout wrapper — sidebar + topbar + content area |
| `Sidebar` | Server | Navigation sidebar with route links and status indicators |
| `TopBar` | Client | Search, notifications bell, user avatar dropdown |
| `Breadcrumbs` | Server | Current route breadcrumb trail |
| `MobileNav` | Client | Hamburger menu for mobile viewport |

### Dashboard Components (`components/dashboard/`)

| Component | Type | Purpose |
|---|---|---|
| `DashboardStats` | Server | Top-level KPI cards — total jobs, applications sent, response rate, interviews |
| `ActivityFeed` | Server | Recent activity timeline (new matches, submissions, responses) |
| `QuickActions` | Server | Action buttons — scrape now, generate resumes, review queue |
| `PipelineSummary` | Server | Horizontal funnel showing applications by status |
| `UpcomingTasks` | Server | Pending items requiring user action |

### Job Listing Components (`components/jobs/`)

| Component | Type | Purpose |
|---|---|---|
| `JobCard` | Server | Single job summary — title, company, match score, source badge, status |
| `JobList` | Server | Grid/list of JobCards with pagination |
| `JobDetail` | Server | Full job description view with action buttons |
| `MatchScoreBadge` | Server | Color-coded percentage badge (green ≥ 80%, yellow 50-79%, red < 50%) |
| `SourceBadge` | Server | Icon/text badge for job source (Indeed, Greenhouse, etc.) |
| `JobFilters` | Client | Filter controls — source, match threshold, status, date range |
| `JobCompare` | Server | Side-by-side comparison of 2-3 job listings |

### Application Pipeline Components (`components/applications/`)

| Component | Type | Purpose |
|---|---|---|
| `ApplicationTable` | Server | Sortable table of all applications with status, date, resume link |
| `ApplicationRow` | Server | Single table row with expandable detail |
| `PipelineBoard` | Client | Kanban-style board — columns per status (discovered, applied, responded, interview) |
| `PipelineCard` | Server | Draggable card within the pipeline board |
| `ApplicationDetail` | Server | Full application view — job info, submitted resume, cover letter, status history |
| `StatusBadge` | Server | Application status indicator with color coding |

### Approval Queue Components (`components/approvals/`)

| Component | Type | Purpose |
|---|---|---|
| `ApprovalQueue` | Server | List of items pending user review before submission |
| `ApprovalCard` | Server | Single pending item — resume draft, cover letter, or form submission |
| `ApprovalActions` | Client | Approve / Reject / Edit buttons with confirmation |
| `ApprovalFilters` | Client | Filter by type (resume, cover letter, form), urgency, job |
| `BulkApproval` | Client | Select multiple items for batch approve/reject |

### Resume & Cover Letter Components (`components/resume/`)

| Component | Type | Purpose |
|---|---|---|
| `ResumePreview` | Server | Rendered PDF preview of generated resume |
| `CoverLetterPreview` | Server | Rendered cover letter preview |
| `ResumeEditor` | Client | Rich text editor for manual resume adjustments |
| `ResumeDiff` | Server | Side-by-side comparison of base vs. tailored resume |
| `TemplateSelector` | Server | Resume template picker with previews |

### Email Components (`components/email/`)

| Component | Type | Purpose |
|---|---|---|
| `EmailList` | Server | List of recruiter emails with read/unread status |
| `EmailDetail` | Server | Full email view with action suggestions |
| `EmailNotification` | Client | Real-time toast notification for new recruiter emails |
| `EmailActions` | Client | Reply, archive, flag, schedule follow-up buttons |

### Interview Prep Components (`components/interview/`)

| Component | Type | Purpose |
|---|---|---|
| `InterviewCard` | Server | Upcoming interview with job, date, prep status |
| `PrepChecklist` | Server | Interview preparation checklist items |
| `CoachingSession` | Client | Live voice coaching interface (WebRTC/LiveKit) |
| `FeedbackPanel` | Server | Post-session feedback and improvement suggestions |

### Shared / Primitives (`components/shared/`)

| Component | Type | Purpose |
|---|---|---|
| `Button` | Server | Primary, secondary, ghost, danger variants |
| `Card` | Server | Generic card container with optional header/footer |
| `Badge` | Server | Status/label badge with color variants |
| `DataTable` | Server | Generic sortable, paginated table |
| `EmptyState` | Server | Illustration + message for empty lists |
| `LoadingSkeleton` | Server | Skeleton placeholder during data loading |
| `Modal` | Client | Dialog/modal overlay for confirmations and forms |
| `Toast` | Client | Ephemeral notification toast |
| `SearchInput` | Client | Debounced search input with icon |
| `DropdownMenu` | Client | Context menu / dropdown actions |
| `Tabs` | Client | Tab navigation for segmented views |
| `Pagination` | Server | Page navigation with previous/next and page numbers |
| `Avatar` | Server | User avatar with fallback initials |
| `Tooltip` | Client | Hover tooltip for additional info |
| `ProgressBar` | Server | Linear progress indicator |

---

## Component Hierarchy

```
RootLayout
├── AppShell
│   ├── Sidebar
│   │   ├── Logo
│   │   ├── NavLinks
│   │   └── ConnectionStatus
│   ├── TopBar
│   │   ├── SearchInput
│   │   ├── EmailNotification
│   │   └── Avatar + DropdownMenu
│   └── MainContent (page-specific)
│
├── /dashboard
│   ├── DashboardStats → Card × 4
│   ├── PipelineSummary → ProgressBar × 5
│   ├── ActivityFeed → ActivityItem × N
│   ├── QuickActions → Button × N
│   └── UpcomingTasks → TaskCard × N
│
├── /dashboard/jobs
│   ├── JobFilters → SearchInput + DropdownMenu × N
│   ├── JobList
│   │   └── JobCard → MatchScoreBadge + SourceBadge + StatusBadge
│   └── Pagination
│
├── /dashboard/applications
│   ├── PipelineBoard (toggle view)
│   │   └── PipelineCard × N per column
│   ├── ApplicationTable (toggle view)
│   │   └── ApplicationRow → StatusBadge + Avatar
│   └── ApplicationDetail (expanded/drawer)
│       ├── ResumePreview
│       ├── CoverLetterPreview
│       └── StatusTimeline
│
├── /dashboard/approvals
│   ├── ApprovalFilters → Tabs + SearchInput
│   ├── BulkApproval → Button × 2
│   └── ApprovalQueue
│       └── ApprovalCard → ResumePreview | CoverLetterPreview + ApprovalActions
│
├── /dashboard/email
│   ├── EmailList → EmailRow × N
│   └── EmailDetail → EmailActions
│
└── /dashboard/settings
    ├── ProfileSection
    ├── ScraperConfig
    ├── MatchingCriteria
    └── NotificationPreferences
```

---

## File Naming Conventions

- **Components:** PascalCase (`JobCard.tsx`, `ApplicationTable.tsx`)
- **Route pages:** lowercase (`page.tsx`)
- **Route layouts:** lowercase (`layout.tsx`)
- **Route handlers:** lowercase (`route.ts`)
- **Utility files:** camelCase (`api.ts`, `utils.ts`)
- **Type files:** camelCase (`types.ts`)
- **One component per file**, named export matches filename

---

## Import Rules

```typescript
// 1. React/Next.js
import { useState } from "react";
import Link from "next/link";

// 2. Local components (relative)
import { JobCard } from "@/components/jobs/JobCard";

// 3. Shared components
import { Badge } from "@/components/shared/Badge";

// 4. Lib/utilities
import { api } from "@/lib/api";
import type { Job } from "@/lib/types";
```

---

**Status:** Architecture plan only — no custom components implemented yet.
**Next step:** Create shared primitives (`Button`, `Card`, `Badge`), then layout shell (`AppShell`, `Sidebar`).
