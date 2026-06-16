# Project Progress Tracker

> Auto-updated as milestones complete. Last updated: 2026-06-16

---

## Current Status

| Field | Value |
|-------|-------|
| **Project** | AI Job Search Agent |
| **Active Phase** | Phase 1 — Foundation (implementation in progress) |
| **Phase Progress** | Scaffolding 100% / Implementation ~70% (6/6 domains complete + cover letter LLM-first) |
| **Overall Progress** | ~45% (structure built, services compile, 6 domains implemented + wired, LLM-first architecture) |
| **Blockers** | None |
| **Next Up** | Worker task handlers + Browser Agent scrapers |

---

## Milestones

### Phase 1: Foundation

#### 1.1 Project Structure — COMPLETE

| Milestone | Status | Notes |
|-----------|--------|-------|
| Directory layout established | Done | Go backend, TS browser agent, Next.js frontend |
| Module interface definitions | Done | Each domain has handler/service/repository/model/dto scaffold |
| Docker Compose (8 services) | Done | api, worker, frontend, browser-agent, postgres, redis, ollama, livekit |
| Database migrations (14 tables) | Done | 001_initial (12) + 002_users + 003_application_events |
| Config files | Done | YAML configs for scraping sources, matching criteria, generation templates |
| Makefile | Done | All dev commands defined |

#### 1.2 Compilation & Builds — COMPLETE

| Component | Status | Runtime | Notes |
|-----------|--------|---------|-------|
| Go API server | Builds clean | Go (Gin) | Compiles with no errors |
| Go Worker service | Builds clean | Go (Asynq) | Compiles with no errors |
| TypeScript browser agent | Builds clean | Node.js (Playwright) | Compiles with no errors |
| Next.js frontend | Builds clean | Next.js 16 + Tailwind | Compiles with no errors |

#### 1.3 Domain Implementation — IN PROGRESS

| Domain | Handler | Service | Repository | Model | DTO | Status |
|--------|---------|---------|------------|-------|-----|--------|
| `tasks` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete** |
| `auth` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete** |
| `jobs` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete** |
| `applications` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete** |
| `resumes` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete** (with cover letter LLM-first, StringSliceDB) |
| `scoring` | ✅ | ✅ | ✅ | ✅ | ✅ | **Complete (handler + wiring done)** |

#### 1.4 API Handlers — IN PROGRESS

| Endpoint Group | Routes | Status | Notes |
|----------------|--------|--------|-------|
| `/api/v1/auth/*` | login, change-password | **Complete** | JWT authentication |
| `/api/v1/tasks/*` | get, list | **Complete** | Task status polling |
| `/api/v1/jobs/*` | list, get, update, scan | **Complete** | Job discovery + CRUD |
| `/api/v1/applications/*` | list, get, create, update-status, update-notes, stats, events | **Complete** | Application lifecycle + audit trail |
| `/api/v1/resumes/*` | list, get, create, update, delete | **Complete** | Resume CRUD with optimistic locking |
| `/api/v1/cover-letters/*` | list, get, create, generate, update-content, delete | **Complete** | Cover letter with LLM generation + traceability |
| `/api/v1/scoring/*` | score, get, batch | **Complete** | Scoring pipeline |

#### 1.5 Worker Task Handlers — NOT STARTED

| Task Type | Queue Name | Status | Notes |
|-----------|------------|--------|-------|
| Job discovery | `jobs:discover` | Not started | Scrapes sources via Browser Agent |
| Resume scoring | `scoring:resume` | **Architecture ready** | LLM scoring pipeline via Ollama |
| Application submission | `applications:submit` | Not started | Fills forms via Browser Agent |
| Embedding generation | `resumes:embed` | Not started | pgvector embedding via Ollama |

#### 1.6 Browser Agent Scrapers — NOT STARTED

| Source | Adapter | Status | Notes |
|--------|---------|--------|-------|
| Indeed | indeed.go | Not started | Primary job board |
| RemoteOK | remoteok.go | Not started | Remote-first listings |
| Greenhouse | greenhouse.go | Not started | ATS-hosted jobs |
| Lever | lever.go | Not started | ATS-hosted jobs |

**Architecture change:** All scrapers will use LLM-based extraction (`job_extraction` prompt) instead of CSS selectors.

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

## LLM-First Architecture Status

The following domains now have LLM interfaces defined with prompts in `config/application.yaml`:

| Domain | LLM Interface | Prompt in Config | Implementation Status |
|--------|---------------|------------------|----------------------|
| **Scoring** | `LLMScorer` + `OllamaLLMScorer` | `prompts.scoring` | ✅ Interface + config + handler wired (async) |
| **Cover Letters** | `CoverLetterGenerator` + `OllamaCoverLetterGenerator` | `prompts.cover_letter` | ✅ Interface + config + handler + StringSliceDB (LLM-first with traceability) |
| **Email Classifier** | `EmailClassifier` (planned) | `prompts.email_classifier` | 📋 Designed, not coded |
| **Job Extraction** | `JobExtractor` (planned) | `prompts.job_extraction` | 📋 Designed, not coded |
| **Resume Tailor** | `ResumeTailor` (planned) | `prompts.resume_tailor` | 📋 Designed, not coded |
| **Interview Prep** | `InterviewPrep` (planned) | `prompts.interview_prep` | 📋 Designed, not coded |
| **Form Filling** | `FormUnderstander` (planned) | `prompts.form_understanding` | 📋 Designed, not coded |

All prompts use Go template syntax (`{{.Field}}`) and are loaded via `config.LoadPrompts()`.

---

## Upcoming Tasks — Phase 1 Implementation Order

> Recommended implementation sequence based on data flow dependencies.

### Wave 1: Core Domain (blocking everything else)

1. **`tasks` domain** — ✅ Complete
2. **`auth` domain** — ✅ Complete
3. **`jobs` domain** — ✅ Complete (wired into router)
4. **`applications` domain** — ✅ Complete (wired into router, includes audit trail)
5. **`resumes` domain** — ✅ Complete (wired into router, optimistic locking, cover letters with LLM-first)
6. **`scoring` domain** — ✅ Complete (handler + wiring done, LLM scoring architecture)

### Wave 2: Workers & Integration

7. **Worker task handlers** — Wire domain services into Asynq task processors.
8. **Browser Agent scrapers** — Implement source adapters using LLM extraction.
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
| Domain module implementation | Week 2-3 | — | In progress (5/6 done + scoring arch) |
| API handler implementations | Week 3 | — | In progress (5/6 done + scoring) |
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
| 2026-06-15 | Applications audit trail | `application_events` table logs every status transition for timeline UI + debugging |
| 2026-06-15 | Derived IsValidStatus from transitions | Single source of truth — add constant, add to map, done |
| 2026-06-15 | Separate notes from status updates | `PATCH /:id/status` for transitions, `PATCH /:id/notes` for permanent notes |
| 2026-06-15 | OFFSET pagination noted for later | Applications won't hit 100k rows; revisit if jobs table grows |
| 2026-06-15 | Domain models no JSON tags | Domain ≠ API. DTOs handle JSON serialization |
| 2026-06-15 | PdfKey not PdfPath | Storage key, not filesystem path. Service maps to URL |
| 2026-06-15 | Optimistic locking on resumes | `WHERE id = $7 AND version = $8` prevents concurrent overwrites |
| 2026-06-15 | RETURNING on Create/Update | DB handles defaults, returns version/timestamps to caller |
| 2026-06-15 | pq.StringArray for text[] | Safe PostgreSQL array scanning |
| 2026-06-15 | **LLM-first architecture** | All semantic understanding via LLM, prompts in config, no hand-written heuristics |
| 2026-06-15 | **Centralized prompts in config** | `config/application.yaml` holds all prompts, user-tunable, version-controlled |
| 2026-06-16 | **Cover letter LLM-first upgrade** | Added Model, PromptVersion, ResumeVersion, Strengths, Gaps traceability fields |
| 2026-06-16 | **StringSliceDB for JSONB arrays** | Custom driver.Valuer/Scanner for `[]string` ↔ JSONB, avoids pq.StringArray syntax mismatch |
| 2026-06-16 | **Two-phase cover letter creation** | Create placeholder → POST /:id/generate fills content via LLM |

---

*This file tracks project state. Update after completing any milestone or making a significant decision.*
