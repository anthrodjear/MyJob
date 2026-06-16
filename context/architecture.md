# Architecture — AI Job Search Agent

## System Overview

An AI-powered job search automation platform that discovers, scores, and applies to jobs on behalf of users. The system combines a Go backend (API + async workers), a TypeScript browser agent for web interactions, and a Next.js frontend — all orchestrated via Docker Compose.

**Core design principles:**

- **Task-based async processing** — API calls return `{taskId}`, clients poll for completion. Keeps request latency low and work retryable.
- **Score-gated automation** — Every application passes a scoring pipeline. High-confidence applications (≥95) are auto-submitted; mid-range (80–94) enter a human review queue; low scores are rejected.
- **Domain isolation within a monolith** — Each bounded context (jobs, applications, resumes, etc.) owns its own handler/service/repository/model/dto. No cross-domain service calls.
- **Local-first AI** — Ollama runs LLM inference and embedding generation on-premise, avoiding external API dependencies for the core scoring pipeline.
- **LLM-first architecture** — All semantic understanding (scoring, email classification, job extraction, cover letters, resume tailoring, interviews, form filling) is delegated to LLMs via centralized prompt configuration. Hand-written heuristics are eliminated.

---

## Component Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Docker Compose                             │
│                                                                     │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────┐    │
│  │   Frontend   │   │  API Server  │   │     Worker Service    │    │
│  │   (Next.js)  │   │    (Gin)     │   │     (Asynq processor) │    │
│  │   port 3000  │   │  port 8080   │   │                       │    │
│  │              │──▶│              │──▶│  - Job discovery       │    │
│  │  React/Tail  │   │  REST API    │   │  - Resume scoring      │    │
│  │  shadcn/ui   │   │  Middleware   │   │  - Application submit  │    │
│  └──────────────┘   │  Rate limit  │   │  - Embedding generation│    │
│                     └──────┬───────┘   └───────────┬────────────┘    │
│                            │                       │                  │
│                     ┌──────▼───────┐       ┌───────▼────────────┐    │
│                     │  PostgreSQL  │       │      Redis 7       │    │
│                     │     16       │       │                    │    │
│                     │  + pgvector  │       │  - Asynq queue     │    │
│                     │              │       │  - Rate limiter    │    │
│                     │  Jobs, apps  │       │  - Session store   │    │
│                     │  Resumes,    │       │  - Cache           │    │
│                     │  embeddings  │       └────────────────────┘    │
│                     └──────────────┘                                │
│                                                                     │
│  ┌──────────────┐   ┌──────────────┐                                │
│  │ Browser Agent│   │   Ollama     │                                │
│  │ (Playwright) │   │              │                                │
│  │  port 3000*  │   │  LLM infer.  │                                │
│  │              │   │  Embeddings  │                                │
│  │  Web scraping│   └──────────────┘                                │
│  │  Form submit │                                                   │
│  └──────────────┘                                                   │
│                                                                     │
│  ┌──────────────┐                                                   │
│  │   LiveKit     │                                                   │
│  │   (WebRTC)    │                                                   │
│  │   Voice I/O   │                                                   │
│  └──────────────┘                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Runtime | Port | Responsibility |
|-----------|---------|------|----------------|
| **API Server** | Go (Gin) | 8080 | REST endpoints, auth, input validation, task dispatch, result polling |
| **Worker Service** | Go (Asynq) | — | Background task processor: job discovery, scoring, application submission, embedding generation |
| **Browser Agent** | TypeScript (Playwright) | 3000 | Headless browser automation: scrape job listings, fill application forms, handle CAPTCHAs |
| **Frontend** | Next.js 16 + Tailwind | 3000* | User dashboard, application review queue, resume management, settings |
| **PostgreSQL 16** | SQL + pgvector | 5432 | Persistent storage for all domain data + vector similarity search |
| **Redis 7** | In-memory | 6379 | Asynq task queue, rate limiting counters, session storage, result caching |
| **Ollama** | Local LLM server | 11434 | LLM inference for scoring/analysis, embedding generation for semantic search |
| **LiveKit** | WebRTC server | 7880 | Real-time voice interviews, speech-to-text/text-to-speech pipelines |
| **Browser Agent** | TypeScript (Playwright) | 3000 | Web scraping, form filling, job application automation |

*Frontend and Browser Agent both use port 3000 but run in separate containers.

---

## Data Flow

### 1. Job Discovery Pipeline

```
User triggers scan
       │
       ▼
  POST /api/jobs/scan ──▶ API validates input
       │
       ▼
  Enqueue Asynq task ──▶ Worker picks up task
       │
       ▼
  Worker calls Browser Agent ──▶ Playwright scrapes job boards
       │
       ▼
  Raw listings stored in PostgreSQL
       │
       ▼
  Worker scores each listing via Ollama LLM
       │
       ▼
  Score >= 95  ──▶ AUTO: enqueue application task
  Score 80-94  ──▶ REVIEW: add to review queue
  Score < 80   ──▶ REJECT: discard with reason
       │
       ▼
  Results cached in Redis, task status updated
```

### 2. Application Submission Pipeline

```
Application task enqueued (auto or manual approval)
       │
       ▼
  Worker retrieves resume + job details from PostgreSQL
       │
       ▼
  Worker generates tailored cover letter via Ollama
       │
       ▼
  Worker calls Browser Agent ──▶ Playwright fills application form
       │
       ▼
  Submission result stored ──▶ Status updated to submitted/failed
       │
       ▼
  If failed: retry with exponential backoff (max 3 attempts)
```

### 3. Resume Scoring & Embedding Flow

```
Resume uploaded via Frontend
       │
       ▼
  POST /api/resumes ──▶ API stores file in PostgreSQL
       │
       ▼
  Enqueue embedding task ──▶ Worker extracts text
       │
       ▼
  Worker calls Ollama for embedding generation
       │
       ▼
  Vector stored in pgvector column
       │
       ▼
  Enables semantic search: "find resumes similar to this job description"
```

### 4. Real-Time Voice Flow (LiveKit)

```
User joins voice session from Frontend
       │
       ▼
  LiveKit WebRTC connection established
       │
       ▼
  User speaks ──▶ LiveKit STT ──▶ Transcript sent to Ollama
       │
       ▼
  LLM response ──▶ LiveKit TTS ──▶ Audio played back to user
       │
       ▼
  Session transcript stored in PostgreSQL
```

---

## Domain Model

The backend follows a modular monolith pattern. Each domain is fully self-contained:

```
internal/
├── jobs/
│   ├── handler.go      # HTTP handlers (Gin)
│   ├── service.go      # Business logic
│   ├── repository.go   # Database queries
│   ├── model.go        # Domain entities
│   └── dto.go          # Request/response types
├── applications/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   └── dto.go
├── resumes/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   ├── dto.go
│   └── llm.go          # ResumeGenerator + CoverLetterGenerator interfaces
├── scoring/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   └── dto.go
├── tasks/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   └── dto.go
├── auth/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   ├── dto.go
│   └── middleware/
├── jobs/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── model.go
│   └── dto.go
└── applications/
    ├── handler.go
    ├── service.go
    ├── repository.go
    ├── model.go
    └── dto.go
```

### Domain Relationships

```
┌─────────┐     ┌──────────────┐     ┌───────────┐
│  Users  │────▶│  Resumes     │────▶│ Embeddings│
└────┬────┘     └──────────────┘     └───────────┘
     │
     ▼
┌─────────┐     ┌──────────────┐     ┌───────────┐
│  Jobs   │────▶│ Applications │────▶│  Scoring  │
└─────────┘     └──────────────┘     └───────────┘
     │
     ▼
┌─────────┐
│  Tasks  │  (async job tracking)
└─────────┘
```

**Key invariants:**

- A `User` has many `Resumes`; each `Resume` has one embedding vector.
- A `Job` can have many `Applications`; each `Application` belongs to one `User` and references one `Resume`.
- Every `Application` must pass through `Scoring` before submission. Score is immutable once set.
- `Tasks` track async work: one task per API call that returns `{taskId}`. Task status transitions: `pending → running → completed | failed`.

### Current API Routes

```
# Public (no auth)
GET  /health                                    → health check
POST /api/v1/auth/login                         → JWT login
POST /api/v1/auth/change-password               → change password (increments session_version)

# Protected (JWT required)
GET    /api/v1/tasks/:id                        → get task status
GET    /api/v1/tasks                            → list tasks

GET    /api/v1/jobs                             → list jobs (filters: status, company, source_id, min_score)
GET    /api/v1/jobs/:id                         → get job
PATCH  /api/v1/jobs/:id                         → update job status
POST   /api/v1/job-discovery/scan               → trigger scan (returns task IDs)

GET    /api/v1/applications                     → list applications (filters: status, job_id, portal_type)
GET    /api/v1/applications/stats               → dashboard statistics
GET    /api/v1/applications/:id                 → get application
POST   /api/v1/applications                     → create application
PUT    /api/v1/applications/:id/status          → update status (with audit trail)
PATCH  /api/v1/applications/:id/notes           → update permanent notes
GET    /api/v1/applications/:id/events          → audit timeline

GET    /api/v1/resumes                          → list resumes
GET    /api/v1/resumes/:id                      → get resume
POST   /api/v1/resumes                          → create resume
PUT    /api/v1/resumes/:id                      → update resume
DELETE /api/v1/resumes/:id                      → delete resume
GET    /api/v1/resumes/:id/content              → get resume content
PUT    /api/v1/resumes/:id/content              → update resume content
POST   /api/v1/resumes/:id/generate             → generate resume content via LLM
GET    /api/v1/resumes/:id/versions             → list resume versions
GET    /api/v1/resumes/:id/versions/:version    → get specific version

GET    /api/v1/cover-letters                    → list cover letters
GET    /api/v1/cover-letters/:id                → get cover letter
POST   /api/v1/cover-letters                    → create cover letter placeholder
POST   /api/v1/cover-letters/:id/generate       → generate cover letter via LLM
PUT    /api/v1/cover-letters/:id/content        → update cover letter content
DELETE /api/v1/cover-letters/:id                → delete cover letter
```

---

## Technology Choices

### Go Backend (API + Worker)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| HTTP framework | Gin | Fast, minimal, battle-tested for REST APIs |
| Task queue | Asynq | Redis-backed, Go-native, supports retries/scheduling/unique jobs |
| Database driver | pgx | Pure Go,高性能, native PostgreSQL type support |
| Config management | envconfig | 12-factor compliant, no YAML dependency |

**Why Go for the backend?**

- Single binary deployment (API and Worker are separate binaries from the same codebase)
- Goroutines handle concurrent task processing without thread overhead
- Strong type safety catches domain logic errors at compile time
- Low memory footprint for Worker pods processing many concurrent tasks

### TypeScript Browser Agent

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Browser automation | Playwright | Cross-browser, built-in waiting, network interception |
| Runtime | Node.js | Native Playwright support, async/await for sequential form flows |
| Communication | HTTP REST | Worker calls Agent via internal Docker network |

**Why a separate Browser Agent?**

- Isolates browser process crashes from the backend
- Allows independent scaling (many agent instances for high scraping throughput)
- Playwright's Node.js SDK is more mature than Go alternatives
- Separation of concerns: Go handles business logic, TS handles browser DOM interaction

### Next.js Frontend

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Framework | Next.js 16 (App Router) | Server components, streaming, API routes |
| Styling | Tailwind CSS | Utility-first, fast iteration, consistent design |
| Component library | shadcn/ui | Accessible, customizable, copy-paste components |
| State management | React Server Components + client hooks | Minimal client JS, server-first rendering |

### Data Layer

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary database | PostgreSQL 16 | ACID, JSONB for flexible schemas, mature ecosystem |
| Vector search | pgvector extension | Co-locate vectors with relational data, no separate vector DB |
| Cache / Queue | Redis 7 | Asynq dependency, rate limiting, session store — one tool for three jobs |
| AI inference | Ollama | Local LLM execution, no API costs, data stays on-premise |
| Prompt management | YAML config (`config/application.yaml`) | All prompts centralized, user-tunable without redeployment, version-controlled |

**Why pgvector over a dedicated vector DB (Pinecone/Weaviate)?**

- Operational simplicity: one fewer service to run and monitor
- Vector search is a supplementary feature (resume similarity), not the primary query pattern
- PostgreSQL's hybrid queries (relational + vector in one query) are powerful for our scoring pipeline
- If vector search becomes a bottleneck, we can extract to a dedicated DB later without changing the API surface

---

## LLM Prompt Management

All LLM prompts are centralized in `config/application.yaml` under the `prompts` section. This enables:

- **User customization** — Prompts can be edited without code changes or redeployment
- **Version control** — Prompt evolution tracked in git alongside code
- **A/B testing** — Multiple prompt variants can be configured and tested
- **Provider agnostic** — Same prompt structure works with Ollama, OpenAI, Anthropic

### Prompt Categories

| Prompt | Domain | Input | Output |
|--------|--------|-------|--------|
| `scoring` | Scoring | Job + Profile | Score (0-100), tier, reasoning |
| `email_classifier` | Emails | From, Subject, Body | Category, confidence, reasoning |
| `cover_letter` | Cover Letters | Job + Candidate | Cover letter text |
| `resume_tailor` | Resumes | Job + Base Resume | Tailored resume content |
| `interview_prep` | Interviews | Job + Candidate | Mock questions (JSON) |
| `job_extraction` | Jobs/Scraping | Raw HTML/Text | Structured job data (JSON) |
| `form_understanding` | Browser Agent | Form fields + Candidate data | Field mappings (JSON) |

### Prompt Template Syntax

Prompts use Go template syntax (`{{.Field}}`) for variable interpolation. The `OllamaLLMScorer` and similar implementations perform simple string replacement for template variables.

Example (from config):
```yaml
prompts:
  scoring:
    user: |
      ## Job
      Title: {{.Title}}
      Company: {{.Company}}
      ...
```

---

## Deployment Topology

### Docker Compose (Development / Single-Node)

```yaml
services:
  api:        # Go API server, port 8080
  worker:     # Go Asynq worker
  frontend:   # Next.js, port 3000
  browser-agent:  # Playwright, port 3000 (internal)
  postgres:   # PostgreSQL 16 + pgvector, port 5432
  redis:      # Redis 7, port 6379
  ollama:     # Ollama LLM server, port 11434
  livekit:    # LiveKit WebRTC, port 7880
```

**Network topology (Docker internal):**

```
                    ┌─────────────────────────────┐
                    │       docker network         │
                    │                               │
  Host :3000 ─────▶│  frontend                     │
  Host :8080 ─────▶│  api ────▶ postgres (:5432)   │
                    │    │────▶ redis (:6379)        │
                    │    │────▶ ollama (:11434)      │
                    │                               │
                    │  worker ──▶ postgres          │
                    │    │────▶ redis               │
                    │    │────▶ ollama              │
                    │    │────▶ browser-agent (:3000)│
                    │                               │
                    │  livekit (:7880)              │
                    └─────────────────────────────┘
```

### Production Considerations

| Concern | Development | Production |
|---------|-------------|------------|
| Orchestration | Docker Compose | Kubernetes / Docker Swarm |
| Database | Single node | Read replicas + pgvector on primary |
| Redis | Single instance | Redis Sentinel or Cluster |
| Ollama | Single GPU instance | Model-specific GPU nodes |
| Browser Agent | Single instance | Horizontal scaling (stateless) |
| TLS termination | None (localhost) | Ingress controller / load balancer |
| Secrets | `.env` file | Kubernetes Secrets / Vault |

---

## Task-Based API Pattern

All long-running operations follow the same async pattern:

```
POST /api/{resource}/{action}
  Request:  { ... }
  Response: { "taskId": "abc-123", "status": "pending" }

GET /api/tasks/{taskId}
  Response: {
    "taskId": "abc-123",
    "status": "running" | "completed" | "failed",
    "result": { ... },     // present when completed
    "error": "..."         // present when failed
  }
```

**Implementation:**

1. API handler validates input, creates a `Task` record in PostgreSQL (status: `pending`)
2. API enqueues an Asynq job with the task ID as payload
3. API immediately returns `{taskId}` to the client
4. Worker picks up the job, updates task status to `running`
5. Worker executes the work (scrape, score, apply, etc.)
6. Worker updates task status to `completed` (with result) or `failed` (with error)
7. Client polls `GET /api/tasks/{taskId}` until terminal state

**Why async?**

- Job discovery and application submission can take 30s–5min per listing
- Sync requests would timeout or block threads
- Retries are built into Asynq with exponential backoff
- Workers can be scaled independently of API servers

---

## Scoring Pipeline

Every job listing passes through a multi-stage scoring pipeline before an application is considered:

```
Raw Job Listing
        │
        ▼
┌─────────────┐
│  Relevance  │  LLM analyzes job description vs user profile
│   Score     │  Output: 0-100 relevance score
└──────┬──────┘
        │
        ▼
┌─────────────┐
│  Qualification│  LLM checks required skills vs resume
│   Score     │  Output: 0-100 qualification score
└──────┬──────┘
        │
        ▼
┌─────────────┐
│   Final     │  Weighted combination of all sub-scores
│   Score     │  Output: 0-100 final score
└──────┬──────┘
        │
        ├── 95+  → AUTO   (application submitted automatically)
        ├── 80-94 → REVIEW (queued for human approval)
        └── <80  → REJECT (discarded, reason logged)
```

**Score components:**

| Component | Weight | Evaluated By |
|-----------|--------|-------------|
| Role relevance | 30% | LLM: job title vs desired role |
| Skill match | 25% | LLM: required skills vs resume |
| Experience level | 20% | LLM: years/seniority required vs experience |
| Location/remote | 15% | Rule-based: remote policy vs user preference |
| Salary range | 10% | Rule-based: posted range vs user expectation |

### Hybrid Scoring Modes

The scoring service supports three modes via `SCORING_MODE` config:

| Mode | Flow | Cost | Latency |
|------|------|------|---------|
| `heuristic` | Keyword matching only (no LLM) | $0 | ~1ms |
| `llm` | LLM semantic scoring only | ~$0.01/job | ~2-5s |
| `hybrid` | Heuristic pre-filter → LLM for final | ~$0.01/borderline | ~1-2s |

**Default: `hybrid`** — Heuristics fast-filter obvious mismatches (score < 60), LLM scores borderline and good candidates for accuracy.

---

## Error Handling & Resilience

| Failure Mode | Mitigation |
|--------------|------------|
| Worker crash mid-task | Asynq auto-retries with exponential backoff; task status stays `running` until TTL expires |
| Browser Agent timeout | Worker enqueues a new Agent call with fresh browser context; max 3 retries |
| Ollama unavailability | Tasks fail with explicit error; no partial scoring; user notified |
| PostgreSQL down | API returns 503; Worker pauses task pickup; health check fails |
| Redis down | API returns 503 (rate limiter / session unavailable); Asynq queue stalls |
| Duplicate submissions | Asynq unique jobs feature prevents duplicate task enqueuing per user+job |

---

## Security Boundaries

- **Internal services** (API ↔ Worker ↔ Agent ↔ DB) communicate over Docker network — no TLS needed in dev
- **External-facing** (Frontend → API) requires authentication middleware
- **Ollama** runs locally — no data leaves the network for LLM inference
- **Browser Agent** handles cookies/sessions for job boards — isolated per-task browser contexts prevent session leakage
- **Rate limiting** via Redis: per-user, per-endpoint, with sliding window counters

---

*Last updated: 2026-06-14*
