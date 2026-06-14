# Project Progress Tracker

> Auto-updated as milestones complete. Last updated: 2026-06-14

---

## Current Status

| Field | Value |
|-------|-------|
| **Project** | AI Job Search Agent |
| **Active Phase** | Phase 1 — Foundation (scaffolding complete, implementation starting) |
| **Phase Progress** | Scaffolding 100% / Implementation 0% |
| **Overall Progress** | ~15% (structure built, services compile, no business logic yet) |
| **Blockers** | None — ready for implementation |
| **Next Up** | Domain module implementation (jobs, applications, resumes, scoring, tasks, auth) |

---

## Milestones

### Phase 1: Foundation

#### 1.1 Project Structure — COMPLETE

| Milestone | Status | Notes |
|-----------|--------|-------|
| Directory layout established | Done | Go backend, TS browser agent, Next.js frontend |
| Module interface definitions | Done | Each domain has handler/service/repository/model/dto scaffold |
| Docker Compose (8 services) | Done | api, worker, frontend, browser-agent, postgres, redis, ollama, livekit |
| Database migrations (12 tables) | Done | All migration files written and ready |
| Config files | Done | YAML configs for scraping sources, matching criteria, generation templates |
| Makefile | Done | All dev commands defined |

#### 1.2 Compilation & Builds — COMPLETE

| Component | Status | Runtime | Notes |
|-----------|--------|---------|-------|
| Go API server | Builds clean | Go (Gin) | Compiles with no errors |
| Go Worker service | Builds clean | Go (Asynq) | Compiles with no errors |
| TypeScript browser agent | Builds clean | Node.js (Playwright) | Compiles with no errors |
| Next.js frontend | Builds clean | Next.js 16 + Tailwind | Compiles with no errors |

#### 1.3 Domain Implementation — NOT STARTED

| Domain | Handler | Service | Repository | Model | DTO | Status |
|--------|---------|---------|------------|-------|-----|--------|
| `jobs` | — | — | — | — | — | Not started |
| `applications` | — | — | — | — | — | Not started |
| `resumes` | — | — | — | — | — | Not started |
| `scoring` | — | — | — | — | — | Not started |
| `tasks` | — | — | — | — | — | Not started |
| `auth` | — | — | — | — | — | Not started |

#### 1.4 API Handlers — NOT STARTED

| Endpoint Group | Routes | Status | Notes |
|----------------|--------|--------|-------|
| `/api/jobs/*` | scan, list, get, update | Not started | Job discovery + CRUD |
| `/api/applications/*` | create, list, approve, reject | Not started | Application lifecycle |
| `/api/resumes/*` | upload, list, get, match | Not started | Resume management + pgvector |
| `/api/scoring/*` | score, review | Not started | Scoring pipeline |
| `/api/tasks/*` | get, list | Not started | Task status polling |
| `/api/auth/*` | login, register, refresh | Not started | Authentication |

#### 1.5 Worker Task Handlers — NOT STARTED

| Task Type | Queue Name | Status | Notes |
|-----------|------------|--------|-------|
| Job discovery | `jobs:discover` | Not started | Scrapes sources via Browser Agent |
| Resume scoring | `scoring:resume` | Not started | LLM scoring pipeline |
| Application submission | `applications:submit` | Not started | Fills forms via Browser Agent |
| Embedding generation | `resumes:embed` | Not started | pgvector embedding via Ollama |

#### 1.6 Browser Agent Scrapers — NOT STARTED

| Source | Adapter | Status | Notes |
|--------|---------|--------|-------|
| Indeed | indeed.go | Not started | Primary job board |
| RemoteOK | remoteok.go | Not started | Remote-first listings |
| Greenhouse | greenhouse.go | Not started | ATS-hosted jobs |
| Lever | lever.go | Not started | ATS-hosted jobs |

#### 1.7 Frontend Pages — NOT STARTED

| Page | Route | Status | Notes |
|------|-------|--------|-------|
| Dashboard | `/dashboard` | Not started | Overview + recent activity |
| Jobs | `/jobs` | Not started | Job listings + scan trigger |
| Applications | `/applications` | Not started | Application pipeline view |
| Resumes | `/resumes` | Not started | Resume upload + management |
| Settings | `/settings` | Not started | Config, sources, preferences |
| Task Monitor | `/tasks` | Not started | Live task progress |

---

## Upcoming Tasks — Phase 1 Implementation Order

> Recommended implementation sequence based on data flow dependencies.

### Wave 1: Core Domain (blocking everything else)

1. **`tasks` domain** — Task tracking is used by every async operation. Implement model, repository, service, handler, and DTO. Wire up the async task polling pattern (`POST → {taskId}`, `GET → status`).
2. **`auth` domain** — Authentication middleware gates all protected routes. Implement JWT-based auth with register/login/refresh endpoints.
3. **`jobs` domain** — Job listing CRUD. This is the data model that feeds scoring and applications.
4. **`applications` domain** — Application lifecycle management. Depends on jobs domain for foreign key references.
5. **`resumes` domain** — Resume storage and embedding. Depends on auth for ownership, pgvector for semantic search.
6. **`scoring` domain** — Scoring pipeline. Depends on jobs + resumes for input data.

### Wave 2: Workers & Integration

7. **Worker task handlers** — Wire domain services into Asynq task processors.
8. **Browser Agent scrapers** — Implement source adapters (Indeed, RemoteOK, Greenhouse, Lever).
9. **Ollama integration** — LLM calls for scoring, cover letter generation, resume tailoring.

### Wave 3: Frontend & Polish

10. **Frontend pages** — Dashboard, jobs, applications, resumes, settings, task monitor.
11. **Integration testing** — End-to-end flow from scan → score → apply.
12. **Docker Compose validation** — Full stack boot, health checks, service connectivity.

---

## Timeline

| Milestone | Target | Actual | Status |
|-----------|--------|--------|--------|
| Phase 1 scaffolding | Week 1 | Week 1 | Done |
| Domain module implementation | Week 2-3 | — | Next |
| API handler implementations | Week 3 | — | Pending |
| Worker task handlers | Week 3-4 | — | Pending |
| Browser agent scrapers | Week 4 | — | Pending |
| Frontend dashboard pages | Week 4-5 | — | Pending |
| Integration testing | Week 5 | — | Pending |
| Phase 1 complete | Week 5 | — | Pending |

---

## Risk Log

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| ATS anti-scraping (CAPTCHAs, rate limits) | High | Medium | Rotate user agents, add delays, CAPTCHA solving fallback |
| pgvector embedding quality | Medium | Low | Start with OpenAI embeddings, fall back to local models |
| Ollama inference speed | Medium | Medium | Pre-compute embeddings, batch scoring requests |
| Browser Agent form-fill failures | High | Medium | Per-task isolated contexts, retry with exponential backoff |
| Database migration conflicts | Low | Low | Use sequential migration IDs, test up/down |

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-06-14 | Go backend over Python | Single binary deployment, goroutines for concurrent workers, type safety |
| 2026-06-14 | pgvector over dedicated vector DB | Operational simplicity — one fewer service, hybrid queries, extractable later |
| 2026-06-14 | Separate Browser Agent service | Isolates browser crashes from backend, independent scaling, mature Playwright SDK |
| 2026-06-14 | Async task pattern (return taskId) | 30s-5min operations can't block, built-in retry, scalable workers |
| 2026-06-14 | Local-first Ollama over cloud APIs | Privacy, zero API costs, no data leaves machine |

---

*This file tracks project state. Update after completing any milestone or making a significant decision.*
