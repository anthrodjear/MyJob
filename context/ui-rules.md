# UI Design Rules & Patterns

> Governs all frontend code in the Next.js 16 App Router dashboard.
> Applies to every component, page, and layout.

---

## 1. Component Architecture

### Server Components (default)

- **Every component is a Server Component unless it explicitly needs client interactivity.**
- Server Components handle: data fetching, rendering, layout, static content, tables, cards, badges, status indicators.
- Export `metadata` or `generateMetadata` from page-level Server Components for SEO.

### Client Components (opt-in)

- Add `"use client"` only when the component requires:
  - `useState`, `useReducer`, `useEffect`, `useContext`, or other React hooks
  - Browser APIs (`window`, `document`, `localStorage`)
  - Event handlers (`onClick`, `onChange`, `onSubmit`)
  - Real-time updates (WebSocket, polling, intervals)
- **Push Client Components to the leaf level.** Keep the parent tree as Server Components.
- Name pattern: suffix with `Client` when ambiguity exists (e.g., `JobFiltersClient`).

### Data Fetching

- **Server Components:** Fetch directly from the Go backend API using `fetch()` in the component body or via `async` Server Components. No `useEffect` + loading states needed.
- **Route Handlers (`app/api/`):** Proxy or transform backend responses for client-side consumption.
- **Server Actions:** Use for mutations (approve, reject, update settings, trigger scrape). Mark with `"use server"` in a separate file under `lib/actions/`.
- **Client Components:** Fetch via `fetch()` to route handlers. Never call the Go backend directly from Client Components (exposes internal ports).

### API Client Pattern

```typescript
// lib/api.ts — used by Server Components and Server Actions
const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export async function fetchJobs(params: JobQueryParams) {
  const res = await fetch(`${BACKEND_URL}/api/jobs?${searchParams}`, {
    cache: "no-store", // Always fresh data for dashboard
  });
  if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`);
  return res.json();
}
```

---

## 2. Styling Rules

### Tailwind CSS (primary)

- **All styling via Tailwind utility classes.** No inline styles, no CSS modules, no styled-components.
- **No `@apply`** unless the rule is repeated 5+ times across the codebase (then extract a component instead).
- **No arbitrary values** (`w-[345px]`) unless the design token doesn't exist in the system. Use tokens from `ui-tokens.md`.

### Color Usage

- Use the semantic color tokens defined in `ui-tokens.md` (e.g., `text-primary`, `bg-success`, `border-warning`).
- **Never use raw hex/Tailwind color scales** (`blue-500`, `#3b82f6`) directly in components. Always use the CSS custom property aliases.
- Dark mode: use `dark:` variants mapped to the token system. Test both modes.

### Responsive Breakpoints

```
sm:  640px   — Mobile landscape
md:  768px   — Tablet
lg:  1024px  — Desktop
xl:  1280px  — Wide desktop
2xl: 1536px  — Ultra-wide
```

- **Mobile-first:** Write base styles for mobile, add `md:`, `lg:` overrides.
- **Dashboard layout:** Sidebar collapses to hamburger on `md:` and below. Content goes full-width.
- **Data tables:** Switch to card/stacked layout on `sm:` and below. Never use horizontal scroll on mobile.
- **Job cards:** Single column on mobile, 2 columns on `md:`, 3 columns on `lg:`.

### Layout Patterns

```tsx
// Dashboard page layout pattern
<div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
  {/* Sidebar / filters on left, content on right */}
</div>

// Stats row pattern
<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Card>...</Card>  {/* 4 KPI cards */}
</div>

// Content grid pattern
<div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
  <JobCard />
</div>
```

---

## 3. Accessibility Requirements

### Mandatory

- **Semantic HTML:** Use `<nav>`, `<main>`, `<aside>`, `<section>`, `<header>`, `<footer>`, `<article>` — never `<div>` for landmark regions.
- **Headings:** Maintain proper hierarchy (h1 → h2 → h3). One `h1` per page.
- **Focus management:** All interactive elements must be keyboard-focusable. Visible focus ring on every focusable element (`focus-visible:ring-2 focus-visible:ring-primary`).
- **ARIA labels:** Every icon-only button must have `aria-label`. Every image must have `alt` text (decorative images use `alt=""`).
- **Color contrast:** Minimum 4.5:1 for normal text, 3:1 for large text. The token palette is designed to meet this — don't deviate.
- **Skip link:** Include a "Skip to main content" link as the first element in the layout.
- **Live regions:** Use `aria-live="polite"` for status updates (application submitted, approval granted). Use `aria-live="assertive"` for errors.
- **Form labels:** Every `<input>`, `<select>`, `<textarea>` must have an associated `<label>` (either `htmlFor` or wrapping `<label>`).

### Testing

- Run `axe-core` or `@axe-core/react` in development for automated checks.
- Test with keyboard-only navigation (Tab, Enter, Escape, Arrow keys).
- Test with screen reader (VoiceOver on macOS, NVDA on Windows).

---

## 4. Data Display Patterns

### Job Cards

- Show: job title, company name, match score (%), source badge, posting date, status.
- Match score uses color coding: green (≥ 80%), yellow (50-79%), red (< 50%).
- Source badge shows origin (Indeed, Greenhouse, RemoteOK, etc.) with distinct colors.
- Hover reveals quick actions: View Details, Apply Now, Dismiss.
- Click opens full detail view (drawer or dedicated route).

### Application Pipeline

- **Table view** (default on desktop): Sortable columns — Job, Status, Date Applied, Resume Used, Match Score.
- **Board view** (toggle): Kanban columns — Discovered → Applied → Responded → Interview → Offer.
- Status badges use the semantic colors from tokens: discovered (gray), applied (blue), responded (yellow), interview (green), rejected (red).

### Match Score Display

```tsx
// Pattern for match score rendering
<span className={cn(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
  score >= 80 && "bg-success-light text-success-dark",
  score >= 50 && score < 80 && "bg-warning-light text-warning-dark",
  score < 50 && "bg-danger-light text-danger-dark",
)}>
  {score}% match
</span>
```

### Empty States

- Every list/table view must have a designed empty state with:
  - Illustration or icon
  - Heading explaining what goes here
  - Description of how to populate it
  - Primary action button (e.g., "Start Scraping Jobs")

### Loading States

- Use skeleton placeholders (not spinners) for content areas.
- Skeletons should match the shape of the real content (rectangles for cards, lines for text).
- Show skeleton for the entire content area, not individual elements.

---

## 5. Interaction Patterns

### Approvals

- **Single approve/reject:** Click button → confirmation tooltip (not modal) → immediate action → toast notification.
- **Bulk approve:** Checkbox selection → "Approve Selected" button in toolbar → confirmation modal → batch action → toast.
- **Edit before approve:** Click "Edit" → inline editor or modal with changes → "Approve & Submit" button.

### Real-time Updates

- Recruiter email notifications: WebSocket or polling (every 30s) → toast notification appears → email count updates in sidebar badge.
- Application status changes: Server-Sent Events or polling → table/board updates without full page reload.

### Error Handling

- **API errors:** Show inline error message within the component, not a global error boundary.
- **Form validation:** Show field-level errors below the input, not in a toast.
- **Network errors:** Toast notification with "Retry" button.
- **Empty/error states:** Never render a broken component. Always handle the loading/error/empty trichotomy.

---

## 6. Typography

- **Headings:** Use Geist Sans (already configured in layout). Tight tracking for h1 (`tracking-tight`).
- **Body:** Geist Sans, `text-sm` (14px) as default body size.
- **Data/numbers:** Geist Mono for statistics, scores, and code. Use `tabular-nums` for aligned columns.
- **Monospace:** Geist Mono for timestamps, IDs, and technical data.

---

## 7. Motion & Transitions

- **Minimal, purposeful animation.** Dashboard is data-dense — avoid distraction.
- **Hover states:** `transition-colors duration-150` on buttons, links, and interactive cards.
- **Page transitions:** None (server-rendered navigation).
- **Loading transitions:** Fade-in for skeleton → content swap (`animate-in fade-in duration-200`).
- **Respect `prefers-reduced-motion`:** Wrap non-essential animations in `@media (prefers-reduced-motion: no-preference) { ... }`.

---

## 8. Component Conventions

### Naming

- Files: PascalCase (`JobCard.tsx`)
- Exports: Named exports only, no default exports for components (except page/layout files)
- Props interfaces: `ComponentNameProps` (e.g., `JobCardProps`)

### File Structure

```tsx
// Standard component file structure
import { type FC } from "react";
import { cn } from "@/lib/utils";

interface JobCardProps {
  job: Job;
  onDismiss?: (id: string) => void;
}

export function JobCard({ job, onDismiss }: JobCardProps) {
  return (
    <article className="...">
      {/* content */}
    </article>
  );
}
```

### Composition over Configuration

- Prefer small, composable components over large components with many props.
- Bad: `<Card variant="job" status="active" showScore showSource showActions />`
- Good: `<Card><JobHeader /><MatchScore /><JobActions /></Card>`

---

**Status:** Rules defined. All future frontend code must comply.
**Last updated:** 2026-06-14
