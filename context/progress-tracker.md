# Project Progress Tracker

> Auto-updated as milestones complete. Last updated: 2026-06-16

---

## Current Status

| Field | Value |
|-------|-------|
| **Project** | AI Job Search Agent |
| **Active Phase** | Phase 1 — Foundation (implementation in progress) |
| **Phase Progress** | Scaffolding 100% / Implementation ~90% (6/6 domains + 8/10 worker handlers + Ollama + Browser Agent + code review) |
| **Overall Progress** | ~60% (structure built, services compile, 6 domains implemented + wired, Browser Agent fully implemented and reviewed) |
| **Blockers** | None |
| **Next Up** | Voice module implementation (types → livekit → brain → providers → session) |

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

#### 1.5 Worker Task Handlers — COMPLETE

| Task Type | Queue Name | Status | Notes |
|-----------|------------|--------|-------|
| Job discovery | `jobs:discover` | **Complete** ✅ | BrowserAgentClient interface + handler, reviewed |
| Job scoring | `job_scoring` | **Complete** ✅ | LLM scoring pipeline, Ollama HTTP calls, reviewed |
| Resume generation | `resume_generate` | **Complete** ✅ | LLM generation with Ollama, reviewed |
| Cover letter generation | `cover_letter_gen` | **Complete** ✅ | LLM generation with Ollama, reviewed |
| Application submission | `application_submit` | **Complete** ✅ | Browser agent form fill, reviewed |
| Fill form | `fill_form` | **Complete** ✅ | Direct browser agent form fill, reviewed |
| Email check | `email_check` | **Complete** ✅ | Microsoft Graph via browser agent, reviewed |
| Interview prep | `interview_prep` | **Complete** ✅ | Placeholder (LLM pending), reviewed |
| Embedding generation | `embedding_generate` | Stub | Pending Ollama integration |
| Voice session | `voice_session` | Stub | Pending LiveKit integration |

#### 1.6 Browser Agent Scrapers — COMPLETE

| Source | Tier | Adapter | Status | Notes |
|--------|------|---------|--------|-------|
| Greenhouse | 1 (API) | `greenhouse.ts` | **Complete** ✅ | Standalone, paginated JSON API, no LLM, baseUrl validation |
| Lever | 1 (API) | `lever.ts` | **Complete** ✅ | Standalone, JSON API, typed `LeverJob`, throws on bad URL |
| RemoteOK | 1 (API) | `remoteok.ts` | **Complete** ✅ | Standalone, JSON API, salary parser, tags→requirements, dedup |
| Indeed | 3 (Browser) | `indeed.ts` | **Complete** ✅ | BaseScraper, fallback selectors, DOM extraction, SHA-256 IDs, anti-bot |
| CustomScraper | 2/3 (Hybrid) | `custom.ts` | **Complete** ✅ | JSON-LD → link discovery → LLM fallback, noise removal, autoScroll |

**Architecture:** Tiered system — Tier 1 API scrapers (no LLM, no browser) for structured sources; CustomScraper (JSON-LD + link discovery + LLM fallback) for everything else. Config-driven via `config/application.yaml` under `job_sources`.

**Key decisions:**
- API scrapers are standalone classes (no BaseScraper inheritance)
- CustomScraper uses 3-strategy hybrid: JSON-LD → link discovery → LLM
- `retry()` from utils for API scrapers; `scrapeWithRetry()` from BaseScraper for browser scrapers
- Stable IDs: SHA-256 hash (Indeed), deterministic prefix+jobId (API scrapers)
- Deduplication via `Set<string>` on external_id before keyword filtering
- Adding new Tier 2 sites: just add URL to config, no code changes

#### 1.7 Browser Agent Server — COMPLETE

| Component | Status | Notes |
|-----------|--------|-------|
| Express server | **Complete** ✅ | Port 3000, endpoints for scrape/fill/email |
| Scrape endpoint | **Complete** ✅ | POST /api/v1/scrape/jobs with Zod validation, scraper map |
| Fill form endpoint | **Complete** ✅ | POST /api/v1/forms/fill with LLM-based field mapping |
| Check emails endpoint | **Complete** ✅ | POST /api/v1/emails/check (placeholder) |
| Ollama client | **Complete** ✅ | LLM-based job extraction |
| Global error middleware | **Complete** ✅ | Structured error responses `{ code, message }` |
| Request timeout | **Complete** ✅ | 5-min timeout for scrape operations |
| API versioning | **Complete** ✅ | All endpoints under `/api/v1/` |

#### 1.8 Browser Agent Form Filler — COMPLETE

| Component | Status | Notes |
|-----------|--------|-------|
| Field detector | **Complete** ✅ | Playwright-based DOM scanning, CSS.escape for selectors, non-fillable field filtering |
| LLM field mapper | **Complete** ✅ | Uses `form_understanding` prompt via Ollama, Zod-validated output |
| Form submitter | **Complete** ✅ | Fills fields, handles file uploads, clicks submit, logger on screenshot failure |
| Heuristic fallback | **Complete** ✅ | Priority-based matching when LLM parsing fails |
| Code review (fields.ts) | **Complete** ✅ | All BLOCKERs (Zod validation, greedy regex, console.error, any types) fixed |

#### 1.9 Browser Agent Code Review (All Files) — COMPLETE

| File | Review Status | Fixes Applied |
|------|--------------|---------------|
| `config.ts` | **Complete** ✅ | Zod schemas, ConfigError, try-catch, env var overrides |
| `ollama.ts` | **Complete** ✅ | Zod validation, OllamaError/LLMExtractionError, balanced JSON, logger |
| `logger.ts` | **Complete** ✅ | LOG_LEVEL validation, Error serialization, circular refs |
| `server.ts` | **Complete** ✅ | Global error middleware, proper types, scraper map, error envelope |
| `fields.ts` | **Complete** ✅ | Zod validation, balanced JSON, logger, heuristic priority rules |
| `detector.ts` | **Complete** ✅ | CSS.escape for XSS, non-fillable field filtering, JSDoc |
| `submitter.ts` | **Complete** ✅ | logger.warn on screenshot failure, throws on unsupported type, selector validation |
| `base.ts` | **Complete** ✅ | `JobExtractionResult` return type, BrowserContext tracking, exponential backoff |
| `indeed.ts` | **Complete** ✅ | `Locator`/`Page` types (no `any`), structured logger |
| `remoteok.ts` | **Complete** ✅ | `Locator` type, structured logger, salary regex with commas/en-dash |
| `greenhouse.ts` | **Complete** ✅ | `JSON.parse` try/catch, `data.jobs` validation, typed `location` cast |
| `lever.ts` | **Complete** ✅ | `LeverListItem` interface, `JSON.parse` try/catch, `Array.isArray` check |

**Review stats:** 11 BLOCKERs + 9 WARNINGs + 4 NITs → all addressed. Build passes clean.

#### 1.10 Ollama Integration — COMPLETE

| Component | Status | Notes |
|-----------|--------|-------|
| Ollama HTTP client (shared pattern) | **Complete** ✅ | Reusable singleton with 2-min timeout, JSON body parsing |
| `OllamaLLMScorer.ScoreJob()` | **Complete** ✅ | Calls `/api/generate`, parses JSON via `ParseLLMScoreResult()` |
| `OllamaResumeGenerator.GenerateContent()` | **Complete** ✅ | Calls `/api/generate`, parses JSON as `ResumeContent` |
| `OllamaCoverLetterGenerator.GenerateContent()` | **Complete** ✅ | Calls `/api/generate`, parses JSON as `CoverLetterGenResult` |
| Browser Agent OllamaClient | **Complete** ✅ | Zod-validated extraction, custom error classes, balanced JSON extraction |
| Safe template parsing | **Complete** ✅ | No `template.Must` — try-parse with fallback strings |
| Code review (all components) | **Complete** ✅ | All BLOCKERs and WARNINGs addressed |

#### 1.11 Browser Agent Voice Module — DESIGNED (Not Implemented)

**Architecture:** Autonomous Interview Agent with pluggable providers, two modes (Assist + Autonomous).

| Layer | File(s) | Status | Notes |
|-------|---------|--------|-------|
| Types | `voice/types.ts` | Not started | STTProvider, TTSProvider, VoiceProvider, InterviewMode interfaces |
| Transport | `voice/livekit.ts` | Stub | LiveKit room join/leave/publish/subscribe — audio transport only |
| Brain | `voice/brain/planner.ts` | Not started | Decide response strategy (answer, clarify, defer) |
| Brain | `voice/brain/responder.ts` | Not started | Generate answers via Ollama with resume/job/app context |
| Brain | `voice/brain/memory.ts` | Not started | Conversation history + key facts extraction |
| Brain | `voice/brain/retrieval.ts` | Not started | Fetch resume, job, application context from backend API |
| Provider | `voice/providers/openai-realtime.ts` | Not started | OpenAI Realtime API (STT+TTS combined) |
| Provider | `voice/providers/elevenlabs.ts` | Not started | ElevenLabs TTS + Whisper STT |
| Provider | `voice/providers/local.ts` | Not started | Local Whisper + Piper/Kokoro TTS |
| Session | `voice/session.ts` | Not started | Interview session orchestration (both modes) |
| API | `voice/index.ts` | Not started | Public API: startVoiceSession(), stopVoiceSession() |

**Key decisions:**
- Voice is an input channel, not a feature. The asset is Interview Intelligence.
- Providers are pluggable — config `voice.provider` selects which one runs.
- Brain (planner/responder/memory/retrieval) is provider-agnostic.
- No new service — stays inside `browser-agent/voice/`.
- Ollama for reasoning (reuse existing `OllamaClient` from `llm/ollama.ts`).

**Backend changes needed:**
- Add `TypeVoiceSession` constant to `tasks/model.go`
- Add `VoiceSessionPayload` to `tasks/dto.go`
- Add `DispatchVoiceSession()` to `tasks/dispatcher.go`
- Implement `handleVoiceSession` in `handlers_application.go`

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
| **Scoring** | `LLMScorer` + `OllamaLLMScorer` | `prompts.scoring` | ✅ Interface + config + handler wired (async) + **Ollama HTTP working** |
| **Cover Letters** | `CoverLetterGenerator` + `OllamaCoverLetterGenerator` | `prompts.cover_letter` | ✅ Interface + config + handler + StringSliceDB (LLM-first with traceability) + **Ollama HTTP working** |
| **Resume Generation** | `ResumeGenerator` + `OllamaResumeGenerator` | `prompts.resume_generation` | ✅ Interface + config + handler + **Ollama HTTP working** |
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

7. **Worker task handlers** — ✅ **Complete** (8 of 10 implemented + reviewed; 2 stubs remain)
8. **Ollama integration** — ✅ **Complete** (scoring, resume, cover letter generators all making HTTP calls)
9. **Browser Agent scrapers** — ✅ **Complete** (Indeed, RemoteOK, Greenhouse, Lever with LLM extraction)
10. **Browser Agent server** — ✅ **Complete** (Express server with scrape/fill/email endpoints)

### Wave 3: Frontend & Polish

10. **Frontend pages** — Dashboard, jobs, applications, resumes, settings, task monitor.
11. **Integration testing** — End-to-end flow from scan → score → apply.
12. **Docker Compose validation** — Full stack boot, health checks, service connectivity.

---

## Timeline

| Milestone | Target | Actual | Status |
|-----------|--------|--------|--------|
| Phase 1 scaffolding | Week 1 | Week 1 | Done |
| Domain module implementation | Week 2-3 | — | **Done** (6/6 complete + scoring arch) |
| API handler implementations | Week 3 | — | **Done** (6/6 complete + scoring) |
| Worker task handlers | Week 3-4 | — | **Done** (8/10 handlers complete + reviewed) |
| Ollama integration | Week 4 | — | **Done** (3 generators working) |
| Browser agent scrapers | Week 4 | — | **Done** (4 sources with LLM extraction) |
| Browser agent server | Week 4 | — | **Done** (Express + endpoints) |
| Browser agent code review | Week 4 | — | **Done** (all 12 files reviewed, 24 issues fixed) |
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
| 2026-06-16 | **Ollama HTTP integration** | Raw `net/http` POST `/api/generate` with `stream: false`; shared `http.Client` per generator; safe template parsing with fallbacks; no Ollama SDK dependency |
| 2026-06-16 | **Browser Agent LLM-first scrapers** | All 4 scrapers (Indeed, RemoteOK, Greenhouse, Lever) use `job_extraction` prompt via Ollama; no CSS selectors; Express server with Zod validation |
| 2026-06-16 | **Browser Agent form filler** | LLM-based field mapping via `form_understanding` prompt; Playwright for DOM interaction; heuristic fallback for LLM parsing failures |
| 2026-06-16 | **Browser Agent code review (all files)** | 11 BLOCKERs + 9 WARNINGs + 4 NITs fixed across 7 files. Key: CSS.escape for XSS, BrowserContext leak fix, exponential backoff, `any` → proper types, structured logging everywhere, JSON.parse try/catch on API responses |
| 2026-06-17 | **Browser Agent tiered scraper architecture** | Tier 1 (API-native, no LLM): Greenhouse, Lever, RemoteOK — standalone classes, no BaseScraper inheritance. Tier 2/3 (CustomScraper): JSON-LD → link discovery → LLM fallback. Config-driven via YAML. Adding new sites = add URL to config, no code changes. |

---

*This file tracks project state. Update after completing any milestone or making a significant decision.*
