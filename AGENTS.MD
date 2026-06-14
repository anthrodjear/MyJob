# AGENTS.MD — AI Job Search Agent

## ⚠️ Required Reading Before Any Task

**Always load context from `context/` folder before starting work:**
- `context/project-overview.md` — what the project is, who it's for
- `context/Architecture.md` — system design, data flows, component diagrams
- `context/code-standards.md` — Go/TypeScript/Frontend coding conventions
- `context/library-docs.md` — library usage patterns and gotchas
- `context/progress-tracker.md` — current phase, what's done, what's next
- `context/ui-registry.md` — component hierarchy and file structure
- `context/ui-rules.md` — Server/Client component patterns, styling rules
- `context/ui-tokens.md` — design tokens, color palette, typography

**Never assume project state — always check `context/progress-tracker.md` first.**

## ⚡ Skills & Superpowers

This project is **not standalone**. Always use the superpowers skills and the skills available in `.agents/skills/` and `~/.config/opencode/skills/`.

### Mandatory Skills (use via the `skill` tool)

| Skill | When to Use |
|-------|-------------|
| `using-superpowers` | **ALWAYS** — before any code change, debugging, planning, or review |
| `brainstorming` | Before building new features or making architecture decisions |
| `writing-plans` | After design approval, to decompose into executable tasks |
| `executing-plans` | When implementing from an approved plan |
| `test-driven-development` | Before writing any production code |
| `systematic-debugging` | Before attempting any fix for bugs or failures |
| `verification-before-completion` | Before claiming "done" — always verify |
| `requesting-code-review` | After meaningful code changes |
| `context-management` | To save state across sessions |
| `error-recovery` | To track recurring errors and fixes |

### Project-Specific Skills (in `.agents/skills/`)

| Skill | When to Use |
|-------|-------------|
| `architect` | Before starting any non-trivial feature |
| `imprint` | After building UI components — extract patterns to `ui-registry.md` |
| `recover` | When something goes wrong during a build |

### Engineering Skills (in `.config/opencode/skills/`)

| Skill | When to Use |
|-------|-------------|
| `senior-backend` | Go API/Worker implementation |
| `senior-frontend` | Next.js/React implementation |
| `database-designer` | Schema changes or query optimization |
| `docker-development` | Docker Compose changes |
| `ci-cd-pipeline-builder` | CI/CD setup |

**Rule: If a skill exists for the task, use it. Do not improvise workflows when tested skills are available.**

---

## Project Overview

A local-first AI job search agent that automates job discovery, application scoring, resume/cover letter generation, and form submission. Runs entirely on the user's machine via Docker Compose.

**Stack:**
- `backend/` — Go (Gin, sqlx, asynq, go-redis, zap) — API server (`:8080`) + async Worker
- `browser-agent/` — TypeScript (Playwright, Express, ioredis, livekit-client) — browser automation (`:3000`)
- `frontend/` — Next.js 16 (React 19, Tailwind CSS v4, App Router) — dashboard
- `config/` — YAML config for job sites, scoring tiers, LLM providers
- `storage/` — local filesystem for generated resumes, cover letters, scraped data

## Architecture Rules

1. **Task-based API pattern.** All mutation endpoints return `{ taskId }`. Clients poll `GET /tasks/:id` for results. No long-running HTTP requests.
2. **Domain modules in `backend/internal/`.** Each domain (jobs, applications, resumes, etc.) is a self-contained package.
3. **Go module path is `backend`.** Import as `backend/internal/jobs`, not `github.com/...`.
4. **YAML-driven config.** Job site selectors, scoring thresholds, and provider settings live in `config/`. Never hardcode site-specific logic.
5. **Local-first.** All data stays on the user's machine. No external analytics, no telemetry, no cloud storage.
6. **Score tiers are immutable policy.** AUTO (95+), REVIEW (80–94), REJECT (<80). Do not add tier logic outside `config/application.yaml`.

## Go Conventions (`backend/`)

### File Structure per Domain Module

```
internal/<domain>/
  handler.go    — HTTP handlers (Gin)
  service.go    — business logic
  repository.go — database queries (sqlx)
  model.go      — DB entities
  dto.go        — request/response types
```

### Rules

- **Error handling:** Return errors up the stack. Never `log.Fatal` in handlers — return structured errors via `c.JSON(...)`.
- **Logging:** Use `zap.Logger` injected via constructor. No `fmt.Println` or `log.Printf`.
- **SQL:** Use `sqlx`. Raw SQL in repository layer only. No ORM.
- **Queue tasks:** Define task types in `internal/tasks/`. Use asynq for all async work. Task payload goes through `dto.go` types.
- **Config:** Read via `internal/config/` which loads `config/application.yaml`. Never read YAML files directly in handlers/services.
- **Tests:** `_test.go` files in same package. Table-driven tests preferred.

### Imports Order

```go
import (
  // stdlib
  "context"
  "fmt"

  // external
  "github.com/gin-gonic/gin"
  "github.com/jmoiron/sqlx"

  // internal
  "backend/internal/jobs"
)
```

## TypeScript Conventions (`browser-agent/`)

- **`tsconfig.json` is strict.** No `any`, no `@ts-ignore`, no implicit `undefined`.
- **Target: ES2022.** Use modern syntax (top-level await, `Array.at()`, etc.).
- **One class per file.** Class name matches filename.
- **Playwright for all browser interaction.** No raw HTTP for page scraping — render JS, fill forms, wait for selectors.
- **Zod for validation.** All external input (Redis payloads, API requests) validated with schemas before use.
- **No `console.log` in production code.** Use a structured logger. `console.error` for error paths only.

## Next.js Conventions (`frontend/`)

- **App Router only.** No `pages/` directory.
- **Server Components by default.** Add `"use client"` only when needed (interactivity, browser APIs).
- **Tailwind CSS v4.** No `tailwind.config.js` — use CSS-based config.
- **API calls to backend:** Use `fetch` with `NEXT_PUBLIC_API_URL` env var. Never hardcode `http://localhost:8080`.
- **No state management libraries.** Server state via fetch/RSC. Client state via React hooks only.

## What NOT to Do

- **Never hardcode connection strings.** Use env vars (`DATABASE_URL`, `REDIS_URL`, etc.). Example: `config/application.yaml` has `redis_url` as placeholder — actual values come from `.env`.
- **Never bypass the task queue.** All side effects (scraping, form filling, resume generation) go through asynq tasks. No inline processing in handlers.
- **Never add business logic to handlers.** Handlers parse input, call service, return response. That's it.
- **Never use `any` in TypeScript.** If the type is complex, define it. `unknown` + narrowing is acceptable.
- **Never modify `config/application.yaml` programmatically.** Config is read-only at runtime. User edits YAML, app reloads.
- **Never store secrets in code or YAML.** All secrets in `.env` (gitignored). YAML contains only non-sensitive config.
- **Never use `SELECT *`.** Always list columns explicitly in SQL queries.
- **Never create global mutable state in Go.** Pass dependencies via constructor injection.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Go task queue (asynq) over Go channels | Survives restarts, built-in retry, Redis-backed |
| YAML config over database config | Editable without SQL, version-controllable, no migration risk |
| LaTeX resumes over HTML/PDF | ATS parser-friendly, consistent formatting, template-driven |
| Separate browser agent service | Isolates Playwright process (memory-heavy), independent scaling |
| Score tiers in config not code | User-tunable thresholds without redeployment |
| Docker Compose over k8s | Single-machine deployment, zero orchestration overhead |
| Go `backend` module path | Private repo, no vanity import path needed |

## Common Workflows

### Adding a new domain module

1. Create `internal/<domain>/` with the 5 standard files (handler, service, repository, model, dto)
2. Register routes in `cmd/api/main.go`
3. Add queue task type in `internal/tasks/` if async work needed
4. Add migration SQL in `internal/database/`
5. Add config section in `config/application.yaml` if domain has settings

### Adding a new API endpoint

1. Define request/response DTOs in `dto.go`
2. Implement handler method
3. Wire route in `cmd/api/main.go`
4. If it triggers async work → create asynq task, enqueue in handler, return `taskId`

### Modifying scoring logic

Scoring is config-driven. Edit `config/application.yaml` under `application.approval_tiers`. Do NOT add scoring logic to Go code — it should read the tiers from config.
