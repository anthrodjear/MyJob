# Comprehensive Test Coverage Plan — 100% Coverage Target

> **Generated**: 2025-07-13  
> **Project**: AI Job Search Agent (Backend + Frontend + Browser Agent)  
> **Current Coverage**: Backend ~15%, Frontend ~3%, Browser Agent ~5%  
> **Target**: 100% across all three services

---

## 📊 Executive Summary

| Service | Source Files | Existing Test Files | Coverage Gap |
|---------|--------------|---------------------|--------------|
| **Backend (Go)** | 78 files | 7 test files | **71 files** |
| **Frontend (Next.js/TS)** | 84 files | 6 test files | **78 files** |
| **Browser Agent (TS)** | 42 files | 3 test files | **39 files** |
| **TOTAL** | **204 files** | **16 test files** | **188 files** |

---

## 🎯 Priority Ordering Strategy

**Order of Implementation** (highest → lowest impact):

| Phase | Category | Rationale |
|-------|----------|-----------|
| **1** | Core Utilities & Config | Foundation; used everywhere; high ROI |
| **2** | Database/Repository Layer | Data integrity; critical for correctness |
| **3** | API Clients & HTTP Layer | Contract testing; integration boundaries |
| **4** | Business Logic / Services | Core domain logic; regression prevention |
| **5** | Handlers / HTTP Layer | Request/response handling; integration tests |
| **6** | Shared UI Components | Reusable; high reuse = high impact |
| **7** | Hooks & Custom Logic | Business logic in frontend |
| **8** | Page Components | Integration/E2E; lower unit test ROI |
| **9** | Browser Agent Scrapers | Complex; integration-heavy; lower unit ROI |

---

## 📦 BACKEND (Go) — Test File Mapping

> **Convention**: Every `file.go` gets `file_test.go` in same package  
> **Location**: Same package (internal tests) or `_test` package (external tests)  
> **Framework**: `testing` + `testify` + `sqlmock` + `httptest` + `testcontainers`

### Phase 1: Core Utilities & Config (Priority 1)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/config/config.go` | `config_test.go` | Unit: YAML parsing, env var interpolation, validation |
| `internal/config/prompts.go` | `prompts_test.go` | Unit: Template rendering, variable substitution |
| `internal/httpresp/response.go` | ✅ `response_test.go` (90.9%) | Already covered |
| `internal/pgvector/format.go` | ✅ `format_test.go` (100%) | Already covered |
| `internal/database/postgres.go` | `postgres_test.go` | **Integration**: testcontainers PostgreSQL; test migrations, connection pooling, health checks |
| `internal/database/redis.go` | `redis_test.go` | **Integration**: testcontainers Redis; test connection, pub/sub, locking |

### Phase 2: Authentication & Auth Middleware (Priority 1-2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/auth/model.go` | `model_test.go` | Unit: struct validation, JWT claims parsing |
| `internal/auth/dto.go` | `dto_test.go` | Unit: request/response serialization, validation |
| `internal/auth/repository.go` | `repository_test.go` | **Integration**: sqlmock for user lookup, session storage |
| `internal/auth/service.go` | `service_test.go` | Unit: password hashing (bcrypt), JWT issuance/validation, token refresh; mock repo |
| `internal/auth/handler.go` | `handler_test.go` | **Integration**: httptest + mock service; test login, register, refresh, logout flows |
| `internal/api/middleware/auth.go` | ✅ `auth_test.go` (52.1%) | **Extend**: add tests for expired tokens, malformed tokens, missing claims |
| `internal/api/middleware/logging.go` | ✅ `logging_test.go` | Extend coverage |
| `internal/api/middleware/ratelimit.go` | ✅ `ratelimit_test.go` | Extend coverage |
| `internal/api/middleware/setup.go` | ✅ `setup_test.go` | Extend coverage |

### Phase 3: Domain Modules — Applications (Priority 2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/applications/model.go` | `model_test.go` | Unit: struct validation, status transitions |
| `internal/applications/dto.go` | `dto_test.go` | Unit: request/response binding, validation |
| `internal/applications/repository.go` | `repository_test.go` | **Integration**: sqlmock for CRUD, status transitions, queries with filters |
| `internal/applications/service.go` | `service_test.go` | Unit: business logic (status transitions, scoring integration, approval workflow); mock repo |
| `internal/applications/handler.go` | `handler_test.go` | **Integration**: httptest + mock service; test all endpoints |

### Phase 4: Domain Modules — Approvals (Priority 2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/approvals/model.go` | `model_test.go` | Unit: state machine validation (AUTO/REVIEW/REJECT) |
| `internal/approvals/dto.go` | `dto_test.go` | Unit: request/response validation |
| `internal/approvals/repository.go` | `repository_test.go` | **Integration**: sqlmock for approval records, workflow state |
| `internal/approvals/service.go` | `service_test.go` | Unit: approval logic, tier evaluation (from config), notifications; mock repo |
| `internal/approvals/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/approvals/workflow.go` | `workflow_test.go` | Unit: state transitions, approval rules, escalation logic |

### Phase 5: Domain Modules — Jobs (Priority 2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/jobs/model.go` | `model_test.go` | Unit: job struct validation, embedding handling |
| `internal/jobs/dto.go` | `dto_test.go` | Unit: search params, filters, pagination |
| `internal/jobs/repository.go` | `repository_test.go` | **Integration**: sqlmock for vector search, full-text search, filters |
| `internal/jobs/service.go` | `service_test.go` | Unit: job sync logic, deduplication, embedding generation; mock repo + embeddings |
| `internal/jobs/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |

### Phase 6: Domain Modules — Resumes (Priority 2-3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/resumes/model.go` | ✅ `model_test.go` (1.1%) | **Extend heavily**: template rendering, LaTeX generation, validation |
| `internal/resumes/dto.go` | `dto_test.go` | Unit: request/response validation |
| `internal/resumes/repository.go` | `repository_test.go` | **Integration**: sqlmock for versioned resumes, templates |
| `internal/resumes/service.go` | `service_test.go` | Unit: resume generation pipeline, template selection, LLM integration; mock repo + LLM |
| `internal/resumes/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/resumes/llm.go` | `llm_test.go` | Unit: prompt building, response parsing, retry logic; mock LLM client |

### Phase 7: Domain Modules — Scoring (Priority 2-3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/scoring/model.go` | `model_test.go` | Unit: score breakdown, tier thresholds |
| `internal/scoring/dto.go` | `dto_test.go` | Unit: request/response validation |
| `internal/scoring/keywords.go` | `keywords_test.go` | **Unit (critical)**: keyword extraction, matching, weighting algorithms; table-driven tests |
| `internal/scoring/llm.go` | `llm_test.go` | Unit: prompt templates, response parsing, scoring logic; mock LLM |
| `internal/scoring/repository.go` | `repository_test.go` | **Integration**: sqlmock for score storage, history |
| `internal/scoring/service.go` | `service_test.go` | Unit: scoring pipeline orchestration, tier assignment (from config); mock deps |
| `internal/scoring/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |

### Phase 8: Domain Modules — Profile (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/profile/model.go` | `model_test.go` | Unit: profile validation, skills normalization |
| `internal/profile/dto.go` | `dto_test.go` | Unit: request/response binding |
| `internal/profile/repository.go` | `repository_test.go` | **Integration**: sqlmock for profile CRUD, skills, preferences |
| `internal/profile/service.go` | `service_test.go` | Unit: profile completion scoring, skill extraction; mock repo |
| `internal/profile/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |

### Phase 9: Domain Modules — Emails (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/emails/model.go` | `model_test.go` | Unit: email parsing, classification labels |
| `internal/emails/dto.go` | `dto_test.go` | Unit: request/response validation |
| `internal/emails/repository.go` | `repository_test.go` | **Integration**: sqlmock for email storage, threading |
| `internal/emails/service.go` | `service_test.go` | Unit: email sync logic, threading, attachment handling; mock repo + IMAP |
| `internal/emails/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/emails/classifier.go` | ✅ `classifier_test.go` (4.7%) | **Extend heavily**: test all classification categories, edge cases |

### Phase 10: Domain Modules — Interviews (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/interviews/model.go` | `model_test.go` | Unit: interview states, scheduling |
| `internal/interviews/dto.go` | `dto_test.go` | Unit: request/response validation |
| `internal/interviews/repository.go` | `repository_test.go` | **Integration**: sqlmock for interview CRUD, calendar sync |
| `internal/interviews/service.go` | `service_test.go` | Unit: scheduling logic, reminders, prep materials; mock repo + calendar |
| `internal/interviews/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |

### Phase 11: Domain Modules — RAG (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/rag/model.go` | `model_test.go` | Unit: document chunks, embeddings |
| `internal/rag/dto.go` | `dto_test.go` | Unit: query/response types |
| `internal/rag/repository.go` | `repository_test.go` | **Integration**: sqlmock + pgvector for vector search |
| `internal/rag/service.go` | `service_test.go` | Unit: retrieval pipeline, reranking, context assembly; mock repo + embeddings |
| `internal/rag/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |

### Phase 12: Domain Modules — Activity, Tasks, SystemConfig, Embeddings (Priority 3-4)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/activity/model.go` | `model_test.go` | Unit: activity types, serialization |
| `internal/activity/dto.go` | `dto_test.go` | Unit: request/response |
| `internal/activity/repository.go` | `repository_test.go` | **Integration**: sqlmock for activity log |
| `internal/activity/service.go` | `service_test.go` | Unit: activity recording, aggregation; mock repo |
| `internal/activity/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/tasks/model.go` | `model_test.go` | Unit: task types, payloads, statuses |
| `internal/tasks/dto.go` | `dto_test.go` | Unit: task payload validation |
| `internal/tasks/repository.go` | `repository_test.go` | **Integration**: sqlmock for task queue (if DB-backed) |
| `internal/tasks/service.go` | `service_test.go` | Unit: task dispatch, retry logic, priority; mock asynq |
| `internal/tasks/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/tasks/dispatcher.go` | `dispatcher_test.go` | Unit: task routing, payload serialization |
| `internal/systemconfig/model.go` | `model_test.go` | Unit: config schema validation |
| `internal/systemconfig/dto.go` | `dto_test.go` | Unit: request/response |
| `internal/systemconfig/repository.go` | `repository_test.go` | **Integration**: sqlmock for config overrides |
| `internal/systemconfig/service.go` | `service_test.go` | Unit: config resolution (YAML → env → DB), caching; mock repo |
| `internal/systemconfig/handler.go` | `handler_test.go` | **Integration**: httptest + mock service |
| `internal/systemconfig/convert.go` | `convert_test.go` | Unit: type conversions, merging |
| `internal/systemconfig/db_overrides.go` | `db_overrides_test.go` | Unit: override precedence |
| `internal/systemconfig/env.go` | `env_test.go` | Unit: env parsing, secrets |
| `internal/systemconfig/resolver.go` | `resolver_test.go` | Unit: config resolution priority chain |
| `internal/systemconfig/yaml_types.go` | `yaml_types_test.go` | Unit: custom YAML unmarshaling |
| `internal/embeddings/ollama.go` | `ollama_test.go` | Unit: embedding generation, batching, retries; mock HTTP |

### Phase 13: API Router & Commands (Priority 4)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `internal/api/router.go` | `router_test.go` | **Integration**: httptest full router; test all routes registered, middleware chain |
| `cmd/api/main.go` | `main_test.go` | **Integration**: test server startup, graceful shutdown, config loading |
| `cmd/worker/main.go` | `main_test.go` | **Integration**: worker startup, task registration, graceful shutdown |
| `cmd/worker/browser_agent.go` | `browser_agent_test.go` | Unit: task handler registration, client config |
| `cmd/worker/handlers_application.go` | `handlers_application_test.go` | Unit: async task handlers for applications |
| `cmd/worker/handlers_job.go` | `handlers_job_test.go` | Unit: async task handlers for jobs |
| `cmd/worker/handlers_resume.go` | `handlers_resume_test.go` | Unit: async task handlers for resumes |

---

## 🌐 FRONTEND (Next.js/TypeScript) — Test File Mapping

> **Convention**: `__tests__/ComponentName.test.tsx` or `ComponentName.test.tsx` alongside source  
> **Framework**: Vitest + React Testing Library + MSW (Mock Service Worker)  
> **Priority**: Utils → API Client → Shared Components → Hooks → Page Components

### Phase 1: Core Utilities & Config (Priority 1)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/lib/utils.ts` | ✅ `utils.test.ts` (26 tests) | **Extend**: add edge cases for cn(), formatting, date helpers |
| `src/lib/api/client.ts` | ✅ `client.test.ts` (18 tests) | **Extend**: add interceptors, error handling, auth refresh |
| `src/lib/api/config.ts` | `__tests__/config.test.ts` | Unit: baseURL construction, env handling, headers |
| `src/lib/types/*.ts` (7 files) | `__tests__/types.test.ts` | Unit: type guards, serialization, validation schemas (Zod) |
| `next.config.ts` | `__tests__/next.config.test.ts` | Unit: config validation, env handling |

### Phase 2: API Client Modules (Priority 2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/lib/api/applications.ts` | `__tests__/applications.test.ts` | Unit: MSW mock handlers; test all CRUD methods, error mapping |
| `src/lib/api/approvals.ts` | `__tests__/approvals.test.ts` | Unit: MSW mocks; test approval actions, pagination |
| `src/lib/api/auth.ts` | `__tests__/auth.test.ts` | Unit: MSW mocks; test login, register, refresh, logout flows |
| `src/lib/api/dashboard.ts` | `__tests__/dashboard.test.ts` | Unit: MSW mocks; test stats aggregation |
| `src/lib/api/emails.ts` | `__tests__/emails.test.ts` | Unit: MSW mocks; test threading, sync |
| `src/lib/api/interviews.ts` | `__tests__/interviews.test.ts` | Unit: MSW mocks; test scheduling, prep materials |
| `src/lib/api/jobs.ts` | `__tests__/jobs.test.ts` | Unit: MSW mocks; test search, filters, vector search params |
| `src/lib/api/profile.ts` | `__tests__/profile.test.ts` | Unit: MSW mocks; test profile CRUD, skills |
| `src/lib/api/resumes.ts` | `__tests__/resumes.test.ts` | Unit: MSW mocks; test generation, templates, versions |
| `src/lib/api/tasks.ts` | `__tests__/tasks.test.ts` | Unit: MSW mocks; test polling, status updates |

### Phase 3: Shared UI Components (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/shared/Badge.tsx` | ✅ `Badge.test.tsx` (9 tests) | **Extend**: variants, sizes, accessibility |
| `src/components/shared/Button.tsx` | ✅ `Button.test.tsx` (14 tests) | **Extend**: loading states, disabled, keyboard nav |
| `src/components/shared/EmptyState.tsx` | ✅ `EmptyState.test.tsx` (8 tests) | **Extend**: action buttons, illustrations |
| `src/components/shared/*` (other) | `__tests__/*.test.tsx` | Unit: RTL render, props, variants, a11y (aria), snapshot |

> **Find all shared components**: `src/components/shared/*.tsx` → each needs test file

### Phase 4: Feature Components (Priority 4)

#### Jobs Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/jobs/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test job cards, filters, pagination, search |

#### Applications Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/applications/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test status badges, actions, detail views |

#### Approvals Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/approvals/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test tier badges, approve/reject modals |

#### Emails Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/emails/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test threading, classification badges |

#### Interviews Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/interviews/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test scheduling UI, prep materials |

#### Cover Letters Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/cover-letters/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test generation, preview, editing |

#### Resumes Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/resumes/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test builder, templates, LaTeX preview |

#### Tasks Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/tasks/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test polling, status, retry |

#### Dashboard Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/dashboard/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test stats cards, charts, activity feed |

#### Layout Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/layout/*` | `__tests__/*.test.tsx` | Unit: RTL; test navigation, sidebar, header, responsive |

#### Providers Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/providers/*` | `__tests__/*.test.tsx` | Unit: RTL; test context providers, theme, auth |

#### Settings Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/settings/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test forms, validation, save |

#### Setup Components
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/setup/*` | `__tests__/*.test.tsx` | Unit: RTL + MSW; test onboarding flow, validation |

#### UI Components (shadcn/ui based)
| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/components/ui/*` | `__tests__/*.test.tsx` | Unit: RTL; test primitives (Dialog, Table, Form, etc.) |

### Phase 5: Hooks (Priority 5)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/hooks/useAuth.ts` | `__tests__/useAuth.test.ts` | Unit: RTL renderHook; test login state, refresh, logout; mock auth context |
| `src/hooks/useJobs.ts` | `__tests__/useJobs.test.ts` | Unit: renderHook + MSW; test search, filters, pagination, infinite scroll |
| `src/hooks/useApplications.ts` | `__tests__/useApplications.test.ts` | Unit: renderHook + MSW; test CRUD, status transitions |
| `src/hooks/useEmails.ts` | `__tests__/useEmails.test.ts` | Unit: renderHook + MSW; test sync, threading |
| `src/hooks/useInterviews.ts` | `__tests__/useInterviews.test.ts` | Unit: renderHook + MSW; test scheduling, reminders |
| `src/hooks/useProfile.ts` | `__tests__/useProfile.test.ts` | Unit: renderHook + MSW; test profile updates, skills |
| `src/hooks/useResumes.ts` | `__tests__/useResumes.test.ts` | Unit: renderHook + MSW; test generation, versions |
| `src/hooks/useTasks.ts` | `__tests__/useTasks.test.ts` | Unit: renderHook + MSW; test polling, retry, status |

### Phase 6: Page Components (Priority 6-7)

> **Strategy**: Integration tests with MSW; test full user flows, not individual components

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/app/dashboard/page.tsx` | `__tests__/dashboard.page.test.tsx` | Integration: MSW; test layout, stats, navigation |
| `src/app/dashboard/applications/page.tsx` | `__tests__/applications.page.test.tsx` | Integration: MSW; test list, filters, create flow |
| `src/app/dashboard/applications/[id]/page.tsx` | `__tests__/application-detail.page.test.tsx` | Integration: MSW; test detail view, actions |
| `src/app/dashboard/approvals/page.tsx` | `__tests__/approvals.page.test.tsx` | Integration: MSW; test tier display, actions |
| `src/app/dashboard/approvals/[id]/page.tsx` | `__tests__/approval-detail.page.test.tsx` | Integration: MSW; test review flow |
| `src/app/dashboard/cover-letters/page.tsx` | `__tests__/cover-letters.page.test.tsx` | Integration: MSW; test list, generation |
| `src/app/dashboard/cover-letters/[id]/page.tsx` | `__tests__/cover-letter-detail.page.test.tsx` | Integration: MSW; test preview, edit |
| `src/app/dashboard/emails/page.tsx` | `__tests__/emails.page.test.tsx` | Integration: MSW; test threading, sync |
| `src/app/dashboard/emails/[id]/page.tsx` | `__tests__/email-detail.page.test.tsx` | Integration: MSW; test thread view |
| `src/app/layout.tsx` | `__tests__/layout.test.tsx` | Integration: RTL; test providers, navigation |
| `src/app/page.tsx` (landing) | `__tests__/landing.page.test.tsx` | Integration: RTL; test marketing content, CTAs |

### Phase 7: Remaining Frontend Files

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/app/error.tsx` | `__tests__/error.test.tsx` | Unit: RTL; test error boundary rendering |
| `src/app/loading.tsx` | `__tests__/loading.test.tsx` | Unit: RTL; test loading states |
| `src/app/not-found.tsx` | `__tests__/not-found.test.tsx` | Unit: RTL; test 404 page |
| `vitest.config.ts` | N/A | Config validation (manual) |
| `vitest.setup.ts` | N/A | Setup validation (manual) |

---

## 🤖 BROWSER AGENT (TypeScript) — Test File Mapping

> **Convention**: `__tests__/filename.test.ts` alongside source  
> **Framework**: Vitest + Playwright (for integration) + MSW (for HTTP mocking)  
> **Priority**: Config → Utils → Core Scrapers → Form Filler → Voice → Server

### Phase 1: Config & Utilities (Priority 1)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/config/config.ts` | ✅ `__tests__/config.test.ts` | **Extend**: env validation, defaults, site configs |
| `src/config/jobsites.ts` | `__tests__/jobsites.test.ts` | Unit: selector validation, site config schema |
| `src/utils/retry.ts` | ✅ `__tests__/retry.test.ts` | **Extend**: backoff strategies, max retries, error filtering |
| `src/utils/browser.ts` | `__tests__/browser.test.ts` | Unit: browser launch options, context config, stealth |
| `src/utils/logger.ts` | `__tests__/logger.test.ts` | Unit: log levels, formatting, redaction |
| `src/utils/stealth.ts` | `__tests__/stealth.test.ts` | Unit: evasion techniques, fingerprint masking |

### Phase 2: Scraper Core & Registry (Priority 2)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/scrapers/base.ts` | `__tests__/base.test.ts` | Unit: abstract methods, navigation, wait strategies; mock Playwright page |
| `src/scrapers/registry.ts` | `__tests__/registry.test.ts` | Unit: registration, lookup, fallback logic |
| `src/scrapers/greenhouse.ts` | `__tests__/greenhouse.test.ts` | **Integration**: Playwright test against test HTML; test selectors, pagination |
| `src/scrapers/lever.ts` | `__tests__/lever.test.ts` | **Integration**: Playwright test against test HTML |
| `src/scrapers/indeed.ts` | `__tests__/indeed.test.ts` | **Integration**: Playwright test against test HTML |
| `src/scrapers/fuzu.ts` | `__tests__/fuzu.test.ts` | **Integration**: Playwright test against test HTML |
| `src/scrapers/myjobmag.ts` | `__tests__/myjobmag.test.ts` | **Integration**: Playwright test against test HTML |
| `src/scrapers/remoteok.ts` | `__tests__/remoteok.test.ts` | **Integration**: Playwright test against test HTML |

### Phase 3: Custom Scrapers (Priority 2-3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/scrapers/custom/base.ts` | `__tests__/custom-base.test.ts` | Unit: shared custom scraper logic |
| `src/scrapers/custom/content-extractor.ts` | ✅ `__tests__/helpers.test.ts` | **Extend**: extraction patterns, fallbacks |
| `src/scrapers/custom/job-page-scraper.ts` | `__tests__/job-page-scraper.test.ts` | Unit: page navigation, data extraction; mock page |
| `src/scrapers/custom/ats-detector.ts` | `__tests__/ats-detector.test.ts` | Unit: ATS detection heuristics, confidence scoring |
| `src/scrapers/custom/apply-link-extractor.ts` | `__tests__/apply-link-extractor.test.ts` | Unit: link discovery, validation |
| `src/scrapers/custom/jsonld-extractor.ts` | `__tests__/jsonld-extractor.test.ts` | Unit: JSON-LD parsing, schema validation |
| `src/scrapers/custom/link-discovery.ts` | `__tests__/link-discovery.test.ts` | Unit: crawl strategies, deduplication |
| `src/scrapers/custom/pagination-discovery.ts` | `__tests__/pagination-discovery.test.ts` | Unit: pagination patterns, next-page detection |
| `src/scrapers/custom/redirect-resolver.ts` | `__tests__/redirect-resolver.test.ts` | Unit: redirect chain resolution, final URL |
| `src/scrapers/custom/deduplicator.ts` | `__tests__/deduplicator.test.ts` | Unit: dedup algorithms, fingerprinting |
| `src/scrapers/custom/helpers.ts` | ✅ `__tests__/helpers.test.ts` | Already covered |

### Phase 4: Form Filler (Priority 3)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/form-filler/detector.ts` | `__tests__/detector.test.ts` | Unit: field detection heuristics, selector strategies; mock page |
| `src/form-filler/fields.ts` | `__tests__/fields.test.ts` | Unit: field type mapping, value generation, validation |
| `src/form-filler/submitter.ts` | `__tests__/submitter.test.ts` | Unit: submission flow, error handling, confirmation detection; mock page |
| `src/form-filler/index.ts` | `__tests__/form-filler.test.ts` | Integration: full form fill + submit flow; Playwright test page |

### Phase 5: LLM & Voice (Priority 3-4)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/llm/ollama.ts` | `__tests__/ollama.test.ts` | Unit: request/response, streaming, retries; mock HTTP |
| `src/voice/types.ts` | `__tests__/voice-types.test.ts` | Unit: type guards, serialization |
| `src/voice/factory.ts` | `__tests__/factory.test.ts` | Unit: provider selection, config validation |
| `src/voice/brain/index.ts` | `__tests__/brain.test.ts` | Unit: planning, memory, retrieval orchestration; mock providers |
| `src/voice/brain/memory.ts` | `__tests__/memory.test.ts` | Unit: memory storage, retrieval, context window |
| `src/voice/brain/planner.ts` | `__tests__/planner.test.ts` | Unit: task decomposition, step planning |
| `src/voice/brain/responder.ts` | `__tests__/responder.test.ts` | Unit: response generation, tool calling |
| `src/voice/brain/retrieval.ts` | `__tests__/retrieval.test.ts` | Unit: knowledge retrieval, ranking |
| `src/voice/livekit.ts` | `__tests__/livekit.test.ts` | Unit: connection, room management; mock LiveKit |
| `src/voice/vad.ts` | `__tests__/vad.test.ts` | Unit: voice activity detection thresholds |
| `src/voice/queue.ts` | `__tests__/queue.test.ts` | Unit: message queue, ordering, retries |
| `src/voice/session.ts` | `__tests__/session.test.ts` | Unit: session lifecycle, state machine |
| `src/voice/providers/elevenlabs.ts` | `__tests__/elevenlabs.test.ts` | Unit: TTS request/response; mock HTTP |
| `src/voice/providers/factory.ts` | `__tests__/voice-provider-factory.test.ts` | Unit: provider selection, fallback |
| `src/voice/providers/local-kokoro.ts` | `__tests__/kokoro.test.ts` | Unit: local TTS integration |
| `src/voice/providers/local-piper.ts` | `__tests__/piper.test.ts` | Unit: local TTS integration |
| `src/voice/providers/local-whisper.ts` | `__tests__/whisper.test.ts` | Unit: local STT integration |
| `src/voice/providers/openai-realtime.ts` | `__tests__/openai-realtime.test.ts` | Unit: realtime API, websocket handling |

### Phase 6: Server & Entry Points (Priority 4)

| Source File | Test File Needed | Test Strategy |
|-------------|------------------|---------------|
| `src/server.ts` | `__tests__/server.test.ts` | Integration: Express routes, Redis queue, task handling; mock dependencies |
| `src/index.ts` | `__tests__/index.test.ts` | Unit: exports, initialization order |

---

## 🛠 TEST STRATEGY NOTES BY CATEGORY

### Backend — Unit Testing Patterns

| Pattern | Tools | When to Use |
|---------|-------|-------------|
| **Pure function testing** | `testing`, `testify/assert` | All pure functions (keywords, scoring, parsing) |
| **Service layer with mocks** | `testify/mock`, `gomock` | Service tests; mock repository interfaces |
| **Repository layer** | `sqlmock` (DATA-DOG), `testcontainers-go` | SQL query verification; integration tests with real DB |
| **Handler layer** | `httptest`, `gin` test mode | Full HTTP request/response testing with mocked services |
| **Table-driven tests** | `testing` | All test functions; multiple cases per function |
| **Golden files** | `testing` | LLM prompt/response, LaTeX templates, email parsing |

### Backend — Integration Testing Strategy

```go
// Pattern for repository tests
func TestRepository_CRUD(t *testing.T) {
    // Use testcontainers for real PostgreSQL
    // Run migrations
    // Test actual SQL execution
}

// Pattern for handler tests
func TestHandler_GetApplication(t *testing.T) {
    // gin.SetMode(gin.TestMode)
    // router := setupRouter(mockService)
    // w := httptest.NewRecorder()
    // req := httptest.NewRequest("GET", "/applications/1", nil)
    // router.ServeHTTP(w, req)
    // assert response
}
```

### Frontend — Testing Patterns

| Pattern | Tools | When to Use |
|---------|-------|-------------|
| **Component unit tests** | Vitest + React Testing Library | All shared/components; test props, rendering, interactions |
| **Hook tests** | `@testing-library/react` `renderHook` | All custom hooks; test state transitions, async flows |
| **API client tests** | Vitest + MSW | All API modules; test request shape, error handling, retries |
| **Page integration tests** | Vitest + RTL + MSW | Page components; test full user flows with mocked backend |
| **Accessibility tests** | `jest-axe` / `vitest-axe` | All interactive components; run in CI |

**MSW Setup Pattern**:
```typescript
// __mocks__/handlers.ts
export const handlers = [
  http.get('/api/applications', () => HttpResponse.json(mockApplications)),
  http.post('/api/applications', () => HttpResponse.json(createdApp, { status: 201 })),
  // ...
]

// test setup
import { setupServer } from 'msw/node'
const server = setupServer(...handlers)
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

### Browser Agent — Testing Patterns

| Pattern | Tools | When to Use |
|---------|-------|-------------|
| **Unit (pure logic)** | Vitest | Extractors, detectors, dedup, retry logic |
| **Playwright component tests** | Playwright Test | Form filler, scraper navigation against test HTML fixtures |
| **Integration (full flow)** | Playwright Test + test server | End-to-end scraping flows against local test pages |
| **Mock Playwright Page** | `playwright-core` mocks | Unit test scraper methods without browser launch |

---

## 📋 EXECUTION CHECKLIST

### Backend (71 test files to create)

- [ ] Phase 1: Config, Database (4 files)
- [ ] Phase 2: Auth (5 files)
- [ ] Phase 3: Applications (5 files)
- [ ] Phase 4: Approvals (6 files)
- [ ] Phase 5: Jobs (5 files)
- [ ] Phase 6: Resumes (6 files)
- [ ] Phase 7: Scoring (7 files)
- [ ] Phase 8: Profile (5 files)
- [ ] Phase 9: Emails (5 files + extend classifier)
- [ ] Phase 10: Interviews (5 files)
- [ ] Phase 11: RAG (5 files)
- [ ] Phase 12: Activity, Tasks, SystemConfig, Embeddings (18 files)
- [ ] Phase 13: Router, Commands (7 files)

### Frontend (78 test files to create)

- [ ] Phase 1: Utils, Config, Types (10 files)
- [ ] Phase 2: API Clients (10 files)
- [ ] Phase 3: Shared Components (~15 files)
- [ ] Phase 4: Feature Components (~30 files)
- [ ] Phase 5: Hooks (8 files)
- [ ] Phase 6: Page Components (~12 files)
- [ ] Phase 7: App-level (3 files)

### Browser Agent (39 test files to create)

- [ ] Phase 1: Config, Utils (5 files + extend 2)
- [ ] Phase 2: Scraper Core & Registry (8 files)
- [ ] Phase 3: Custom Scrapers (10 files + extend 1)
- [ ] Phase 4: Form Filler (4 files)
- [ ] Phase 5: LLM & Voice (18 files)
- [ ] Phase 6: Server & Entry (2 files)

---

## 🚀 RECOMMENDED EXECUTION ORDER

1. **Week 1-2**: Backend Phase 1-2 (Config, DB, Auth) — Foundation
2. **Week 2-3**: Backend Phase 3-5 (Applications, Approvals, Jobs) — Core domain
3. **Week 3-4**: Backend Phase 6-8 (Resumes, Scoring, Profile) — High-value logic
4. **Week 4-5**: Backend Phase 9-13 (Remaining domains, Router, Commands)
5. **Week 5-6**: Frontend Phase 1-2 (Utils, API Clients) — Test infrastructure
6. **Week 6-7**: Frontend Phase 3-4 (Shared + Feature Components) — UI coverage
7. **Week 7-8**: Frontend Phase 5-7 (Hooks, Pages, App) — Integration
8. **Week 8-9**: Browser Agent Phase 1-3 (Config, Scrapers) — Core automation
9. **Week 9-10**: Browser Agent Phase 4-6 (Form filler, Voice, Server)

---

## 📈 COVERAGE TARGETS PER PHASE

| Phase | Target Coverage | Measurement |
|-------|-----------------|-------------|
| Backend Core (1-2) | 90%+ | `go test -coverprofile=coverage.out ./internal/...` |
| Backend Domain (3-11) | 85%+ | Per-package coverage |
| Backend Commands (13) | 70%+ | Integration-focused |
| Frontend Utils/API (1-2) | 95%+ | `vitest run --coverage` |
| Frontend Components (3-4) | 80%+ | Component + hook coverage |
| Frontend Pages (5-7) | 60%+ | Integration test coverage |
| Browser Agent Core (1-3) | 85%+ | Unit + Playwright component |
| Browser Agent Advanced (4-6) | 70%+ | Integration-focused |

---

## 🔧 TOOLING SETUP REQUIRED

### Backend
- [ ] `go install github.com/DATA-DOG/go-sqlmock@latest`
- [ ] `go install github.com/testcontainers/testcontainers-go@latest`
- [ ] Add `testify/mock` for mock generation
- [ ] Configure `go test -coverprofile` in CI

### Frontend
- [ ] `npm install -D vitest @testing-library/react @testing-library/user-event msw @vitest/coverage-v8`
- [ ] Configure `vitest.config.ts` with coverage thresholds
- [ ] Set up MSW handlers for all API endpoints
- [ ] Add `jest-axe` or `vitest-axe` for a11y tests

### Browser Agent
- [ ] `npm install -D vitest @playwright/test`
- [ ] Create test HTML fixtures for scraper testing
- [ ] Set up Playwright config for headless testing

---

## ✅ DEFINITION OF DONE PER FILE

A test file is **complete** when:

- [ ] All exported functions/types have at least one test
- [ ] Error paths tested (not just happy path)
- [ ] Edge cases covered (empty inputs, nil pointers, boundaries)
- [ ] Mocks verified (interactions asserted)
- [ ] No `// TODO: test` comments remaining
- [ ] Coverage for that file > 80% (unit) or integration test exists
- [ ] Tests pass in CI (`go test ./...` / `vitest run` / `playwright test`)

---

*This plan is executable. Start with Phase 1 Backend, create one test file at a time, run tests, verify coverage, then proceed.*