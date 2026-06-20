# Project Overview: AI Job Search Agent

## What It Is

An AI-powered job search automation platform that handles 80-95% of the job application workflow end-to-end — from discovering relevant positions to submitting tailored applications with customized resumes, cover letters, and completed forms.

## Who It's For

Job seekers who want to dramatically scale their application volume without sacrificing quality — particularly career changers, those targeting multiple companies simultaneously, or anyone who wants to spend time on interviews rather than repetitive form-filling.

## Key Features

| Capability | Description |
|---|---|
| **Job Scraping** | Configurable multi-source scraping (Indeed, RemoteOK, Greenhouse, Lever, etc.) with skill-based matching and scoring |
| **Resume Generation** | AI-tailored resumes per application, compiled from LaTeX to PDF for consistent professional formatting |
| **Cover Letter Generation** | Context-aware cover letters that align candidate experience with specific job requirements |
| **Application Automation** | Playwright browser agent that navigates and fills application forms across company ATS platforms |
| **Application Tracking** | Status dashboard tracking each application through the pipeline (discovered → applied → responded → interview) |
| **Recruiter Email Monitoring** | Microsoft Graph API integration to detect recruiter responses, schedule updates, and flag action items |
| **Voice Interview Coaching** | Real-time interview practice via OpenAI Realtime API + LiveKit, with feedback on answers and delivery |

## Architecture

| Layer | Technology | Role |
|---|---|---|
| API + Worker | Go | Backend services, job orchestration, scraping, email monitoring |
| Browser Agent | TypeScript + Playwright | Form filling, ATS navigation, application submission |
| Frontend | Next.js 16 | Dashboard, settings, application management UI |
| Database | PostgreSQL + pgvector | Application data, job listings, semantic search for skill matching |
| Cache/Queue | Redis | Job queue, rate limiting, session state |
| LLM Runtime | Ollama (local) | Resume/cover letter generation, skill matching, interview analysis |
| Voice/Realtime | OpenAI Realtime API + LiveKit | Live interview coaching sessions |
| Deployment | Docker Compose | Local-first, single-command startup |

## Design Principles

- **Local-first** — all data stays on user's machine by default, no cloud dependency for core features
- **Privacy-focused** — no third-party analytics, no data leaves the machine except explicitly configured integrations (Microsoft Graph for email)
- **Configurable** — scraping sources, matching criteria, generation templates, and automation behavior are all user-configurable
- **Composable** — each feature works independently (use just scraping, or just resume generation, or the full pipeline)

# Project Overview: AI Job Search Agent

## What It Is

An AI-powered job search automation platform that handles 80-95% of the job application workflow end-to-end — from discovering relevant positions to submitting tailored applications with customized resumes, cover letters, and completed forms.

## Who It's For

Job seekers who want to dramatically scale their application volume without sacrificing quality — particularly career changers, those targeting multiple companies simultaneously, or anyone who wants to spend time on interviews rather than repetitive form-filling.

## Key Features

| Capability | Description |
|---|---|
| **Job Scraping** | Configurable multi-source scraping (Indeed, RemoteOK, Greenhouse, Lever, etc.) with skill-based matching and scoring |
| **Resume Generation** | AI-tailored resumes per application, compiled from LaTeX to PDF for consistent professional formatting |
| **Cover Letter Generation** | Context-aware cover letters that align candidate experience with specific job requirements |
| **Application Automation** | Playwright browser agent that navigates and fills application forms across company ATS platforms |
| **Application Tracking** | Status dashboard tracking each application through the pipeline (discovered → applied → responded → interview) |
| **Recruiter Email Monitoring** | Microsoft Graph API integration to detect recruiter responses, schedule updates, and flag action items |
| **Voice Interview Coaching** | Real-time interview practice via OpenAI Realtime API + LiveKit, with feedback on answers and delivery |

## Architecture

| Layer | Technology | Role |
|---|---|---|
| API + Worker | Go | Backend services, job orchestration, scraping, email monitoring |
| Browser Agent | TypeScript + Playwright | Form filling, ATS navigation, application submission |
| Frontend | Next.js 16 | Dashboard, settings, application management UI |
| Database | PostgreSQL + pgvector | Application data, job listings, semantic search for skill matching |
| Cache/Queue | Redis | Job queue, rate limiting, session state |
| LLM Runtime | Ollama (local) | Resume/cover letter generation, skill matching, interview analysis |
| Voice/Realtime | OpenAI Realtime API + LiveKit | Live interview coaching sessions |
| Deployment | Docker Compose | Local-first, single-command startup |

## Design Principles

- **Local-first** — all data stays on user's machine by default, no cloud dependency for core features
- **Privacy-focused** — no third-party analytics, no data leaves the machine except explicitly configured integrations (Microsoft Graph for email)
- **Configurable** — scraping sources, matching criteria, generation templates, and automation behavior are all user-configurable
- **Composable** — each feature works independently (use just scraping, or just resume generation, or the full pipeline)

## Current Status

**Phase:** Phase 1 Foundation — ~85% complete (11/12 Browser Agent + Voice Module 100% complete, 1 stub domain remaining)

- Project structure and directory layout established
- Technology stack decisions finalized
- Docker Compose orchestration configured (8 services: api, worker, frontend, browser-agent, postgres, redis, ollama, livekit)
- **11 Core domains complete** — jobs, applications, resumes, scoring, auth, tasks, emails, interviews, profile, rag, approvals (all handler/service/repository/model/dto + API + Worker wiring)
- **4 Browser Agent scrapers** — Greenhouse, Lever, RemoteOK (API-native), Indeed (Playwright) + CustomScraper fallback
- **Ollama integration** — 3 LLM generators working (scoring, resume, cover letter) + email classifier + embeddings client
- **Voice Module** — Complete with 4 brain components, 4 providers (OpenAI Realtime, ElevenLabs, Local Whisper+Piper+Kokoro), session orchestration
- **Worker handlers** — 10/11 complete (resume_tailor is the only missing handler for existing task type)
- Database migrations (9 up/down) with pgvector for embeddings

## What's Built

- [x] Project structure and scaffolding
- [x] Docker Compose configuration (8 services)
- [x] Module interface definitions
- [x] Database schema + pgvector (9 migrations)
- [x] Jobs domain — CRUD, scan trigger, tiered scrapers, scoring pipeline
- [x] Applications domain — CRUD, status state machine, audit trail, stats
- [x] Resumes domain — CRUD, LLM generation, cover letters, versioning, PDF keys
- [x] Profile domain — CRUD API for user profile (JSONB in profiles table), ETag/If-Match optimistic locking, PATCH merge logic
- [x] Scoring domain — Heuristic/LLM/Hybrid modes, factor scoring, tier logic
- [x] Auth domain — Single-user JWT, bcrypt, session invalidation
- [x] Tasks domain — Async task queue (Asynq), state machine, HTTP API
- [x] Interviews domain — Session lifecycle, transcript handling, LiveKit, internal events
- [x] Emails domain — Store/list/get/update/classify, LLM classification via Ollama, API + worker integration
- [x] Approvals domain — Human-in-the-loop approval gate, workflow layer, job snapshots
- [x] RAG domain — Embedding generation + semantic search, cosine similarity
- [x] Browser Agent scrapers — 4 sources + fallback CustomScraper
- [x] Browser Agent server — Express + scrape/fill/email endpoints
- [x] Browser Agent form filler — LLM field mapping + heuristic fallback
- [x] Browser Agent Voice Module — Complete brain + providers + session + factory
- [x] Ollama HTTP integration — Raw `/api/generate` + `/api/embeddings`, shared client

## What's Next

### Backend (Phase 1 completion)
- [x] **Profile domain** — CRUD API for user profile (JSONB in profiles table)
- [x] **Approvals domain** — Human-in-the-loop approval before auto-apply (approval_requests table)
- [x] **RAG/Embeddings domain** — Embedding generation + semantic search (embeddings table + pgvector)
- [x] **Emails domain** — Email classifier implementation (emails table + classifier.go, LLMClient/OllamaClient, HTTP handlers wired)
- [ ] **Activity domain** — User activity logging (activity_log table)
- [x] **Resume Tailor worker handler** — Implemented in `handlers_resume.go` + `resumes/llm.go` (ResumeTailor interface + Ollama implementation + Service.TailorResume)
- [ ] **Rate limit middleware** — Implement `internal/api/middleware/ratelimit.go`
- [ ] **Logging middleware** — Implement `internal/api/middleware/logging.go`

### Frontend (Wave 3)
- [ ] Dashboard — Overview + recent activity
- [ ] Jobs — Job listings + scan trigger
- [ ] Applications — Application pipeline view
- [ ] Resumes — Resume upload + management
- [ ] Settings — Config, sources, preferences
- [ ] Task Monitor — Live task progress

### Integration & Polish
- [ ] Docker Compose validation — Full stack boot, health checks, service connectivity
- [ ] End-to-end flow testing — Scan → Score → Apply → Track
- [ ] Error handling hardening — Retry policies, dead letter queues, observability
