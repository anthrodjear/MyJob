# User Preferences

## Development Workflow

### Code Quality
- **Build one file at a time** — never batch files; write one file, review, wait for approval
- **Subagent-driven workflow mandatory** — write code, delegate to specialized(never use a general agent) subagents for review, fix, wait for user approval
- **Never skip review** — even small files get reviewed
- **Never proceed without user approval** — wait for explicit confirmation before next file

### Subagent Usage
- **Use specialized subagents** for code review (e.g., `senior backend`, not `general`)
- **Don't give exact content** — give context, let them decide what to write
- **Use Context7 MCP** for latest library docs before writing any code
-**Always review** make sure the code follows the code-starndards.md rules.

### Code Style
- **Explicit over implicit** — no magic, clear intent
- **Testability over cleverness** — easy to test beats clever solutions
- **Sentinel errors** with `errors.Is()` — not string matching
- **No `SELECT *`** — always list columns
- **Centralize state transitions** — maps, not switch statements
- **Config from env** — use `getEnv` with defaults, not `mustEnv` at startup


### Architecture
- **Modular monolith** with domain packages (`internal/<domain>/`)
- **Task-based API** — all mutations return `{taskId}`, no long-running HTTP
- **Async via Redis** (Asynq) — survives restarts, built-in retry
- **YAML-driven config** — user-tunable without redeployment
- **Local-first** — no cloud, no telemetry, no external analytics
- **LLM-first** — all semantic understanding via LLM, prompts centralized in `config/application.yaml`, no hand-written heuristics for core logic

### Handler Patterns
- **Handler validates** — pagination defaults, filter validation, range checks
- **Service enforces business rules** — input validation, status transitions, config thresholds
- **Repository returns errors** — service translates to domain errors
- **Use `httpresp.*` helpers** for standard responses
- **202 Accepted** for async operations (not 207)
- **Partial failure returns partial results** — caller gets task IDs alongside error

### Service Patterns
- **getJob/getTask helper** — DRY for common lookups
- **canTransition map** — centralized state machine
- **Narrow config dependency** — take `ScoringConfig`, not `*config.Config`
- **Return domain errors** — `ErrNotFound`, `ErrInvalidStatus`, `ErrNoRowsAffected`

### Context Files
- **Always read context folder** before starting work
- **Save deferred items** to `context/deferred-items.md`
- **Update context files** as work progresses
- **Don't duplicate code** in context files — reference files instead

---

## Technical Preferences

### Stack
- **Go backend** — modular monolith, Gin, sqlx, Asynq, go-redis
- **TypeScript browser agent** — Playwright, strict TypeScript
- **Next.js 16 frontend** — App Router, Tailwind CSS v4, React 19
- **PostgreSQL 16** with pgvector
- **Redis** for async task queue
- **Docker Compose** — single machine deployment
- **LaTeX** for resumes (ATS-friendly)

### Libraries
- **Asynq** for task queue (not Go channels)
- **Zap** for logging (not fmt.Println)
- **Zod** for TypeScript validation
- **Context7 MCP** for latest library docs
- **httpresp** for shared HTTP response helpers

### API Design
- **RESTful** — standard HTTP methods, meaningful status codes
- **UUIDs** for all entity IDs
- **Pagination** with limit/offset
- **Filtering** via query parameters
- **Error codes** in responses (e.g., `JOB_NOT_FOUND`, `INVALID_STATUS`)

### Git
- **Conventional commits** — feat:, fix:, refactor:, etc.
- **Worktree isolation** for risky changes
- **Never force push** unless explicitly requested

---

## Communication Preferences

- **Be concise** — don't repeat what I already know
- **Show trade-offs** — when presenting options, name what you're giving up
- **Wait for approval** — don't assume I want to proceed
- **Ask before creating** — don't create files without confirming
- **Respect deferrals** — don't re-raise deferred items unless relevant
- **Save state** — write decisions to context files for future sessions

---

## Review Checklist (Subagent Template)

When reviewing code, check:

1. **Builds without errors**
2. **Proper error propagation** — no ignored errors, sentinel errors used correctly
3. **No security issues** — no hardcoded secrets, proper input validation
4. **Follows patterns** — matches tasks/auth domain patterns and code-standard.md for code standards
5. **All new types documented** — public types have godoc comments
6. **No dead code** — no unused functions/types/imports
7. **Input validation at boundaries** — handler validates, service enforces
8. **Error codes consistent** — matches existing codebase patterns
9. **Domain errors translated** — repo errors become domain errors in service
10. **Build passes** — `go build ./...` succeeds

---

## Wave Implementation Order

| Wave | Domain | Status | Files |
|------|--------|--------|-------|
| 1 | tasks | ✅ Complete | model, dto, repository, service, handler, dispatcher |
| 1 | auth | ✅ Complete | model, dto, repository, service, handler, auth middleware |
| 1 | jobs | ✅ Complete | model, dto, repository, service, handler (wired into router) |
| 1 | applications | ✅ Complete | model, dto, repository, service, handler (with audit trail) |
| 1 | resumes | ✅ Complete | model, dto, repository, service, handler, llm (with cover letter LLM-first) |
| 1 | scoring | ✅ Complete | model, dto, repository, service, handler, llm, keywords |
| 2 | Worker handlers | ⏳ Next | Asynq task processors |
| 2 | Browser Agent | ⏳ Next | Scrapers with LLM extraction |
| 2 | Voice module | DESIGNED | Interview Agent (types → livekit → brain → providers → session) |
| 2 | Backend interviews domain | ⏳ Pending | Model, DTO, repo, service, handler (empty stubs exist) |
| 3 | Frontend | ⏳ Pending | Dashboard, jobs, applications, resumes, settings |

---

## Deferred Items (See: context/deferred-items.md)

1. GET /job-discovery/tasks/:id — Task status polling
2. Route organization — /jobs vs /job-discovery
3. User ID logging — needs JWT wiring
4. Thin list DTO — payload optimization

---

## Last Updated
- Date: 2026-06-18
- Context: Voice module architecture designed (Interview Agent with pluggable providers, two modes)
