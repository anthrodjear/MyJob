# Code Standards

> Reference for all contributors. Read before writing code in any part of this project.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Go Backend](#go-backend)
3. [TypeScript Browser Agent](#typescript-browser-agent)
4. [Next.js Frontend](#nextjs-frontend)
5. [Shared Patterns](#shared-patterns)
6. [Testing](#testing)
7. [Code Review Checklist](#code-review-checklist)

---

## Project Overview

| Component | Stack | Entry Point |
|---|---|---|
| **Backend** | Go 1.22, Gin, sqlx, PostgreSQL, Redis, asynq | `backend/cmd/api`, `backend/cmd/worker` |
| **Browser Agent** | TypeScript (strict), Playwright | `browser-agent/src/` |
| **Frontend** | Next.js 16, React 19, Tailwind CSS | `frontend/src/` |

**Design principles:**
- **Modular monolith** — domains are isolated by package boundaries, not separate binaries.
- **Local-first** — everything runs with `docker compose up` plus `make dev-*` commands.
- **Testability over cleverness** — if it's hard to test, it's wrong.
- **Explicit over implicit** — no magic, no global state, no hidden side effects.

---

## Go Backend

### Package Structure

```
backend/
  cmd/
    api/main.go        # HTTP server entrypoint
    worker/main.go     # Background job processor entrypoint
  internal/
    config/            # App configuration, env loading
    database/          # Connection setup, migrations
    api/               # Router, middleware, shared handlers
    <domain>/          # One package per domain
      handler.go       # HTTP handlers (request/response)
      service.go       # Business logic
      repository.go    # Database queries
      model.go         # Domain types (DB row structs)
      dto.go           # Request/response types
```

**Rules:**
- Everything lives under `internal/` — no exported packages outside the module.
- One domain per package. Cross-domain calls go through service interfaces, not direct repository access.
- `cmd/` contains only `main.go` — no business logic in entrypoints.
- `api/` package owns the router and middleware. Domain handlers register routes.

### Error Handling

```go
// GOOD: wrap with context, return early
if err != nil {
    return fmt.Errorf("fetching jobs for user %d: %w", userID, err)
}

// BAD: swallowing errors
if err != nil {
    return nil
}

// BAD: returning raw errors without context
return err
```

**Rules:**
- Every returned error must include context about *what failed* and *relevant identifiers* (user ID, resource ID).
- Use `fmt.Errorf("...: %w", err)` to wrap. Never lose the original error.
- Sentinel errors: define as `var ErrNotFound = errors.New("not found")` in the domain package. Compare with `errors.Is()`.
- Handler layer: translate domain errors to HTTP status codes. Business logic never touches HTTP.
- Never log and return the same error. Pick one — the handler decides whether to log.

### Naming

```go
// Packages: short, lowercase, single-word
package jobs        // not job_service, not jobHandler

// Interfaces: noun form of what it does
type JobRepository interface { ... }
type NotificationService interface { ... }

// Structs: descriptive
type CreateJobRequest struct { ... }
type JobResponse struct { ... }

// Variables: camelCase, no stuttering
jobRepo      // not jobRepository
jobService   // not jobServiceInstance
```

**Rules:**
- Package names: lowercase, single-word, no underscores (`jobs` not `job_service`).
- Interface names: noun form — `Repository`, `Service`, `Formatter`. No `I` prefix, no `Interface` suffix.
- No stuttering: `jobs.CreateRequest` not `jobs.JobCreateRequest`.
- Unexported functions/methods: use when possible. Export only what needs to be consumed outside the package.
- Constants: `PascalCase` for exported, `camelCase` for unexported. Group related constants with `iota`.

### Handlers

```go
// Each handler is a method on a struct that holds its dependencies.
type Handler struct {
    svc    *Service
    logger *zap.Logger
}

func (h *Handler) CreateJob(c *gin.Context) {
    var req CreateJobRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid request body", err)
        return
    }

    job, err := h.svc.CreateJob(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Created(c, job)
}
```

**Rules:**
- Handlers bind input, call service, return response. No business logic.
- Use `c.Request.Context()` for context propagation — not `c` itself.
- Validate at the handler boundary (struct tags or manual). Services receive already-validated input.
- Return domain errors. Let a shared response helper translate to HTTP.

### Config Management

- Load config from environment variables. Use `config/` package.
- No hardcoded values — not even "safe" defaults like ports.
- `.env.example` is the source of truth for available config keys. Keep it in sync.
- Secrets never appear in logs, error messages, or commit history.

### Dependencies

- sqlx for database access (raw SQL preferred over ORM).
- zap for structured logging.
- asynq for background job processing via Redis.
- Gin for HTTP routing.
- Add new dependencies only when existing ones can't do the job. Document why in the PR.

---

## TypeScript Browser Agent

### Strict Mode

`strict: true` is non-negotiable. Do not add `@ts-ignore` or `@ts-expect-error` unless there is a linked issue explaining why.

### Project Setup

- Target: ES2022, module: CommonJS.
- Source in `src/`, output in `dist/`.
- Declarations and source maps are generated — keep them enabled.

### Naming

```typescript
// Files: kebab-case
job-scraper.ts
email-sender.ts

// Functions: camelCase, verb-first
async function scrapeJobPage(url: string): Promise<JobData> { ... }
async function sendEmail(to: string, body: string): Promise<void> { ... }

// Types/Interfaces: PascalCase
interface JobData { ... }
type ScrapeResult = { ... }

// Constants: SCREAMING_SNAKE_CASE
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT_MS = 30_000;
```

### Imports

```typescript
// Group order: node builtins → external packages → internal modules
import fs from "node:fs";
import { chromium, Browser } from "playwright";
import { JobData } from "./types";
import { logger } from "./utils/logger";

// Use relative paths for internal imports
import { parseJobPage } from "./parsers/job-page";
```

**Rules:**
- No barrel files (`index.ts` re-exporting everything). Import directly from the module.
- Use `node:` prefix for Node.js builtins.
- Prefer named exports over default exports.
- Sort imports: node builtins first, then external, then internal. Blank line between groups.

### Error Handling

```typescript
// GOOD: typed errors with context
class ScrapeError extends Error {
  constructor(
    public readonly url: string,
    public readonly cause: unknown,
  ) {
    super(`Failed to scrape ${url}`);
    this.name = "ScrapeError";
  }
}

// GOOD: explicit error handling
try {
  await page.goto(url);
} catch (err) {
  logger.error({ url, err }, "Navigation failed");
  throw new ScrapeError(url, err);
}
```

**Rules:**
- Define custom error classes for domain-specific failures.
- Always log before throwing. The logger provides context that the thrower can't.
- Never catch errors silently. If you catch it, handle it or re-throw.
- Playwright operations: always use try/catch with timeout configuration.

### Playwright-Specific

- Always set explicit timeouts. Never rely on defaults for critical operations.
- Close browsers/pages in `finally` blocks. Resource leaks accumulate fast in scraping.
- Use `page.waitForSelector` with `{ state: "attached" }` when querying dynamic content.
- Isolate test data. Never scrape production sites without permission.

---

## Next.js Frontend

### Strict Mode

`strict: true` in `tsconfig.json` is mandatory. Same rules as browser agent — no `@ts-ignore`.

### Project Setup

- App Router (not Pages Router).
- Path alias: `@/*` maps to `./src/*`. Use it consistently.
- React 19 features are available. Prefer them over legacy patterns.

### Component Structure

```
frontend/src/
  app/              # Routes (App Router)
  components/       # Shared UI components
  lib/              # Utilities, API client, hooks
  styles/           # Global styles (Tailwind base)
```

**Rules:**
- One component per file. File name matches component name.
- Co-locate related files: a component and its test, types, and styles live together.
- No inline styles. Use Tailwind classes.
- Server Components by default. Add `"use client"` only when you need interactivity.

### Naming

```typescript
// Files: PascalCase for components, kebab-case for everything else
UserProfile.tsx       // React component
api-client.ts         // Utility module
auth-context.tsx      // Context provider

// Components: PascalCase
function UserProfile({ userId }: { userId: string }) { ... }

// Hooks: camelCase, "use" prefix
function useJobApplications(userId: string) { ... }

// Types: PascalCase
interface Application { ... }
type JobStatus = "pending" | "applied" | "interviewed";
```

### Imports

```typescript
// Group order: react/next → external → internal (@/* alias)
import { use, Suspense } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchJobs } from "@/lib/api";
import { Button } from "@/components/ui/button";
```

**Rules:**
- Use `@/*` alias for all internal imports. No relative paths crossing directory boundaries.
- Relative paths are fine within the same component directory (e.g., `./types`).
- No barrel files in `components/`. Import directly.

### Data Fetching

- Server Components: fetch data directly. No `useEffect` + `useState` for initial data.
- Client Components: use TanStack Query (or equivalent) for server state.
- API routes in `app/api/` for BFF (backend-for-frontend) patterns.
- Never expose internal API URLs to the client. Proxy through Next.js API routes.

### Styling

- Tailwind CSS for all styling.
- Use `cn()` utility (clsx + tailwind-merge) for conditional classes.
- Design tokens: define in `tailwind.config.ts`, not inline.
- Responsive: mobile-first. Use `sm:`, `md:`, `lg:` breakpoints consistently.
- Dark mode: support via Tailwind `dark:` variants.

---

## Shared Patterns

### API Contracts

- Backend exposes JSON APIs. Frontend and browser agent consume them.
- Request/response types are defined in Go (`dto.go`) and must be documented.
- When adding/changing endpoints, update both server and client in the same PR.
- Versioning: use URL path prefix (`/api/v1/`) when making breaking changes.
- Pagination: use cursor-based pagination for lists. Format: `{ items: T[], nextCursor: string | null }`.

### Error Response Format

```json
{
  "error": {
    "code": "JOB_NOT_FOUND",
    "message": "Job with ID 123 not found",
    "details": {}
  }
}
```

- `code`: machine-readable, UPPER_SNAKE_CASE.
- `message`: human-readable, safe to display to users.
- `details`: optional structured data (validation errors, field-level issues).

### Config Management

- All configuration via environment variables.
- `.env.example` is the source of truth. Keep it in sync with actual usage.
- Secrets: never in code, never in git, never in logs.
- Config loading: fail fast on missing required config. Log a clear error message.
- Feature flags: use env vars for simple on/off. Use a feature flag service for complex rollout logic.

### Logging

- Backend: structured logging with zap. Use `logger.Info("msg", zap.String("key", value))`.
- Browser agent: use a logger module, not raw `console.log`.
- Frontend: no server-side logging from client code. Log server-side in API routes and Server Components.
- Log levels: `Debug` for development, `Info` for normal operations, `Warn` for recoverable issues, `Error` for failures.
- Never log secrets, passwords, tokens, or PII.

### Database

- Use raw SQL with sqlx. No ORM.
- Migrations: numbered, forward-only. Never edit a migration that's been applied.
- Migrations live in `backend/internal/database/migrations/`.
- Use parameterized queries. Never interpolate user input into SQL.
- Transactions: wrap multi-step writes in a transaction. Rollback on error.

---

## Testing

### General

- Tests live next to the code they test (`*_test.go`, `*.test.ts`).
- Test behavior, not implementation. Refactoring internals should not break tests.
- Every PR should include tests for new functionality. Bug fixes should include a regression test.
- No flaky tests. If a test is flaky, fix it or delete it — never skip it.

### Go Backend

```go
// Table-driven tests
func TestCreateJob(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateJobRequest
        wantErr bool
    }{
        {"valid job", CreateJobRequest{Title: "Engineer"}, false},
        {"missing title", CreateJobRequest{}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

- Use table-driven tests. One function, multiple cases.
- Test the service layer with a mock repository. Test the repository layer against a real database (use testcontainers or a test database).
- Handlers: test HTTP status codes and response bodies. Use `httptest.NewRecorder`.
- Run `go test ./...` before pushing.

### Browser Agent

- Test scraping logic with saved HTML fixtures, not live sites.
- Mock Playwright page interactions where possible.
- Integration tests (real browser) should be clearly separated and runnable independently.
- Run `npm test` before pushing.

### Frontend

- Unit tests: component behavior with React Testing Library.
- No snapshot tests. They break on every cosmetic change and catch nothing real.
- Test user interactions, not component structure.
- Mock API calls at the network level (MSW), not at the fetch function level.
- Run `npm test` before pushing.

### Coverage

- Aim for high coverage on business logic. Do not chase 100% on boilerplate.
- Critical paths (auth, payments, data mutations) must have tests.
- Coverage reports are informational, not targets. A 90% covered codebase with bad tests is worse than an 80% one with good tests.

---

## Code Review Checklist

Before submitting or approving a PR, verify:

- [ ] **No secrets in code or commit history.** Check `.env`, config files, and hardcoded strings.
- [ ] **Errors are wrapped with context.** No bare `return err`.
- [ ] **No `@ts-ignore` without a linked issue.**
- [ ] **New endpoints have tests.** Both happy path and error cases.
- [ ] **Database changes have migrations.** Forward-only, no edits to applied migrations.
- [ ] **API changes are reflected on the client side.** Server and client updated in the same PR.
- [ ] **Logging is structured and does not leak secrets.**
- [ ] **Dependencies were added intentionally.** No accidental `npm install` or `go get` of unnecessary packages.
- [ ] **Code follows the naming conventions in this document.**
- [ ] **`make test` passes locally.**
