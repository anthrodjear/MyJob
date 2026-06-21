# Frontend Phase 1: Dashboard Implementation Plan

**Goal:** Complete vertical slice — one page end-to-end with Server Components, TanStack Query, and all dashboard widgets.

**Backend API Endpoints:**
- `GET /api/v1/applications/stats` → `ApplicationStatsResponse`
- `GET /api/v1/activity-logs?limit=10` → `ActivityListResponse`
- `GET /api/v1/tasks?status=pending&limit=5` → `TaskListResponse`

---

## Files to Create

### 1. lib/api/dashboard.ts
**Purpose:** API client for dashboard endpoints
**Dependencies:** `@/lib/api/client`, `@/lib/types`

### 2. hooks/useJobs.ts
**Purpose:** TanStack Query hooks for jobs data
**Dependencies:** `@/lib/api/jobs` (will create)

### 3. lib/api/jobs.ts
**Purpose:** API client for jobs endpoints
**Dependencies:** `@/lib/api/client`, `@/lib/types`

### 4. components/dashboard/DashboardStats.tsx
**Purpose:** 4 KPI cards (Total Apps, Pending Approvals, Match Rate, Jobs This Week)
**Type:** Server Component (receives data as props)
**Props:** `stats: ApplicationStatsResponse`

### 5. components/dashboard/PipelineSummary.tsx
**Purpose:** Horizontal funnel showing application pipeline
**Type:** Server Component
**Props:** `stats: ApplicationStatsResponse`

### 6. components/dashboard/ActivityFeed.tsx
**Purpose:** Recent activity list (10 items)
**Type:** Server Component
**Props:** `activities: ActivityResponse[]`

### 7. components/dashboard/QuickActions.tsx
**Purpose:** Action buttons (Discover Jobs, Review Approvals, Check Email, Settings)
**Type:** Client Component (has onClick handlers)

### 8. components/dashboard/UpcomingTasks.tsx
**Purpose:** Pending tasks list (5 items)
**Type:** Server Component
**Props:** `tasks: TaskResponse[]`

### 9. app/dashboard/layout.tsx
**Purpose:** Dashboard shell wrapper
**Type:** Server Component
**Uses:** `AppShell` from layout components

### 10. app/dashboard/page.tsx
**Purpose:** Main dashboard page - Server Component
**Data fetching:** Parallel prefetch of stats, activity, tasks
**Composition:** Renders all dashboard widgets

### 11. app/dashboard/error.tsx
**Purpose:** Error boundary for dashboard route
**Type:** Client Component

---

## Implementation Order

1. `lib/api/dashboard.ts` — API client
2. `lib/api/jobs.ts` — Jobs API client (needed for hooks)
3. `hooks/useJobs.ts` — Query hooks
4. `components/dashboard/DashboardStats.tsx` — KPI cards
5. `components/dashboard/PipelineSummary.tsx` — Funnel
6. `components/dashboard/ActivityFeed.tsx` — Activity list
7. `components/dashboard/QuickActions.tsx` — Action buttons
8. `components/dashboard/UpcomingTasks.tsx` — Task list
9. `app/dashboard/layout.tsx` — Dashboard shell
10. `app/dashboard/page.tsx` — Main page (Server Component with data fetching)
11. `app/dashboard/error.tsx` — Error boundary

---

## Data Flow (Server Components)

```
app/dashboard/page.tsx (Server Component)
  ├── fetch stats: GET /applications/stats
  ├── fetch activity: GET /activity-logs?limit=10
  └── fetch tasks: GET /tasks?status=pending&limit=5
       │
       ├── DashboardStats (stats)
       ├── PipelineSummary (stats)
       ├── ActivityFeed (activity)
       ├── QuickActions (no data)
       └── UpcomingTasks (tasks)
```

All dashboard widgets are Server Components receiving data as props — no client-side fetching needed for the main dashboard view.

---

## TypeScript Types Needed

Add to `lib/types/dashboard.ts`:
- `DashboardStats` (from ApplicationStatsResponse)
- `PipelineStage` (for funnel)
- `ActivityItem` (from ActivityResponse)
- `TaskItem` (from TaskResponse)

---

## Client Components (minimal)

- `QuickActions.tsx` — has onClick navigation
- `app/dashboard/error.tsx` — error boundary

All other dashboard widgets are Server Components.