# AI Job Search Agent

> A local-first AI agent that automates 80–95% of the job application pipeline — from discovery to submission — running entirely on your machine via Docker Compose.

## Why This Exists

Job searching is repetitive, time-consuming, and error-prone. You copy-paste the same info into dozens of forms, tailor resumes manually for each posting, and lose track of which companies you applied to. This agent handles the tedious 80% so you can focus on the 20% that actually matters: interview prep and career decisions.

**Everything stays on your machine.** No cloud storage, no third-party analytics, no data leaving your network. Your job search data is yours.

## Features

- **Job Discovery** — Scrapes configurable sources (Indeed, RemoteOK, Greenhouse, Lever, and more) on a schedule. New postings are scored and ranked automatically.
- **AI-Powered Scoring** — Each job is evaluated against your profile and preferences. Scores fall into three tiers: AUTO (95+, submit immediately), REVIEW (80–94, human approval), REJECT (<80, skip).
- **Resume Generation** — Produces ATS-friendly PDF resumes from LaTeX templates, tailored to each job's requirements.
- **Cover Letter Generation** — Writes personalized cover letters that reference specific role requirements and your experience.
- **Automatic Form Filling** — Browser automation fills and submits application forms via Playwright, handling dynamic JS-heavy career pages.
- **Email Monitoring** — Tracks inbound email via Microsoft Graph API (confirmations, interview invitations, recruiter outreach).
- **Voice Interview Coaching** — Real-time AI conversation practice using LiveKit and OpenAI's realtime voice model.
- **RAG Knowledge Base** — Store and retrieve information from your career history, past applications, and research using pgvector embeddings.

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Backend API** | Go 1.22, Gin, sqlx, zap | REST API server on `:8080` |
| **Task Worker** | Go 1.22, Asynq, Redis | Async job processing (scraping, generation, form filling) |
| **Browser Agent** | TypeScript, Playwright, Express | Headless browser automation on `:3000` |
| **Frontend** | Next.js 16, React 19, Tailwind CSS v4 | Dashboard UI on `:3001` |
| **Database** | PostgreSQL 16 + pgvector | Persistent storage and vector embeddings |
| **Queue/Cache** | Redis | Asynq task queue and application cache |
| **LLM Inference** | Ollama (local) | Local embedding and text generation |
| **Voice** | LiveKit + OpenAI Realtime | Real-time voice interview coaching |
| **Orchestration** | Docker Compose | 8-service local deployment |

## Quick Start

### Prerequisites

- [Docker Desktop](https://docs.docker.com/get-docker/) (includes Docker Compose v2)
- 16 GB RAM recommended (Ollama + PostgreSQL + Redis + 4 app services)
- ~10 GB disk for Docker images and Ollama models

### 1. Clone and configure

```bash
git clone <your-repo-url> MyJob
cd MyJob
cp .env.example .env
```

Edit `.env` and add your API keys (optional features work without them):

```env
# Required for cloud LLM generation (optional if using Ollama only)
OPENAI_API_KEY=sk-...

# Required for email monitoring (optional)
MS_TENANT_ID=...
MS_CLIENT_ID=...
MS_CLIENT_SECRET=...

# Required for voice coaching (optional)
LIVEKIT_API_KEY=devkey
LIVEKIT_API_SECRET=devsecret
```

### 2. First-time setup

```bash
make setup
```

This will:
- Create `.env` from `.env.example` if missing
- Start PostgreSQL, Redis, and Ollama
- Pull required Ollama models (`mxbai-embed-large` for embeddings, `qwen2.5:latest` for generation)

### 3. Initialize the database

```bash
make migrate
```

### 4. Start all services

```bash
make start
```

### 5. Access the dashboard

Open [http://localhost:3001](http://localhost:3001) in your browser.

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3001 | Dashboard UI |
| API | http://localhost:8080 | REST API |
| Browser Agent | http://localhost:3000 | Automation service |
| Ollama | http://localhost:11434 | Local LLM API |

## Development Setup

Run services individually for faster iteration:

```bash
# Start only infrastructure (PostgreSQL, Redis, Ollama)
docker compose up -d postgres redis ollama

# Run each service locally in separate terminals
make dev-api        # Go API on :8080
make dev-worker     # Go Worker (async task processor)
make dev-frontend   # Next.js on :3001
make dev-browser    # Browser Agent on :3000
```

### Building locally

```bash
make build-api        # Go API binary → bin/api.exe
make build-worker     # Go Worker binary → bin/worker.exe
make build-frontend   # Next.js production build
make build-browser    # TypeScript compilation
```

### Running tests

```bash
make test          # All tests
make test-api      # Go backend only
make test-frontend # Next.js only
```

### Shell access

```bash
make shell-postgres    # psql CLI
make shell-redis       # redis-cli
make shell-api         # Shell into API container
make shell-worker      # Shell into Worker container
```

## Project Structure

```
MyJob/
├── backend/                    # Go backend (API + Worker)
│   ├── cmd/
│   │   ├── api/               # API server entrypoint
│   │   └── worker/            # Async worker entrypoint
│   ├── internal/
│   │   ├── api/               # HTTP handlers and routes
│   │   ├── applications/      # Application tracking domain
│   │   ├── approvals/         # Score-tier approval workflow
│   │   ├── config/            # YAML config loader
│   │   ├── coverletters/      # Cover letter generation
│   │   ├── database/          # Migrations and DB setup
│   │   ├── emails/            # Microsoft Graph email sync
│   │   ├── interviews/        # Interview coaching
│   │   ├── jobs/              # Job discovery and scoring
│   │   ├── profile/           # User profile management
│   │   ├── rag/               # RAG knowledge base
│   │   ├── resumes/           # LaTeX resume generation
│   │   └── tasks/             # Asynq task definitions
│   ├── Dockerfile.api
│   ├── Dockerfile.worker
│   └── go.mod
├── browser-agent/              # TypeScript browser automation
│   ├── src/                   # Playwright-based automation
│   ├── Dockerfile
│   └── package.json
├── frontend/                   # Next.js dashboard
│   ├── src/                   # App Router pages and components
│   ├── package.json
│   └── tsconfig.json
├── config/
│   └── application.yaml       # Scoring tiers, LLM providers, queue config
├── templates/
│   ├── resumes/               # LaTeX resume templates
│   └── cover-letters/         # LaTeX cover letter templates
├── storage/                   # Generated files (gitignored)
├── scripts/                   # Setup and utility scripts
├── Makefile                   # Dev workflow commands
├── docker-compose.yml         # 8-service orchestration
└── .env.example               # Environment variable template
```

## Architecture

### Task-Based API Pattern

All mutation endpoints return a `{ taskId }` immediately. Clients poll `GET /tasks/:id` for results. This keeps HTTP requests fast and moves heavy work (scraping, PDF generation, form submission) to the background worker.

```
Client → POST /api/jobs/apply → { taskId: "abc123" }
Client → GET /tasks/abc123 → { status: "completed", result: {...} }
```

### Scoring Tiers

Jobs are scored 0–100 against your profile. Tiers are configured in `config/application.yaml`:

| Tier | Score Range | Action |
|------|-------------|--------|
| AUTO | 95+ | Auto-submit application |
| REVIEW | 80–94 | Pause, notify for human approval |
| REJECT | <80 | Skip, log for reference |

Tiers are **immutable policy** — thresholds live in config, not code. Edit the YAML to tune them without redeployment.

### Domain Modules

The Go backend is organized as domain modules in `backend/internal/`. Each domain is self-contained:

```
internal/<domain>/
  handler.go      # HTTP handlers (Gin)
  service.go      # Business logic
  repository.go   # Database queries (sqlx)
  model.go        # DB entities
  dto.go          # Request/response types
```

### Services Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose                        │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │ Frontend │  │   API    │  │     Worker           │  │
│  │ Next.js  │  │   Go     │  │  Go (Asynq)          │  │
│  │  :3001   │  │  :8080   │  │  (async processor)   │  │
│  └────┬─────┘  └────┬─────┘  └──────────┬───────────┘  │
│       │              │                    │              │
│       └──────────────┼────────────────────┘              │
│                      │                                   │
│              ┌───────┴───────┐                           │
│              │     Redis     │                           │
│              │  (queue+cache)│                           │
│              └───────┬───────┘                           │
│                      │                                   │
│  ┌───────────┐  ┌────┴─────┐  ┌──────────────────┐     │
│  │ PostgreSQL │  │  Ollama  │  │  Browser Agent   │     │
│  │ +pgvector  │  │ (local   │  │  Playwright      │     │
│  │            │  │   LLM)   │  │  :3000           │     │
│  └────────────┘  └──────────┘  └──────────────────┘     │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              LiveKit (voice coaching)             │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Configuration

All runtime configuration lives in `config/application.yaml`. Environment-specific secrets (API keys, passwords) go in `.env`.

### Scoring Tiers

```yaml
application:
  approval_tiers:
    auto_apply:
      min_score: 95
      action: "auto_submit"
      notify: true
    review:
      min_score: 80
      max_score: 94
      action: "pause_for_approval"
    reject:
      max_score: 79
      action: "skip"
      log: true
```

### LLM Providers

```yaml
llm:
  primary:
    provider: "openai"
    model: "gpt-4o"
  local:
    provider: "ollama"
    model: "qwen2.5:latest"
  embeddings:
    provider: "ollama"
    model: "mxbai-embed-large"
```

### Queue & Workers

```yaml
queue:
  redis_url: "redis://localhost:6379"
  concurrency: 5
  retryAttempts: 3
```

### Email Sync

```yaml
email:
  provider: "microsoft_graph"
  check_interval: "30m"
  folders: ["Inbox"]
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `POSTGRES_PASSWORD` | Yes | PostgreSQL password |
| `OPENAI_API_KEY` | No | Cloud LLM for generation (Ollama used if absent) |
| `ANTHROPIC_API_KEY` | No | Alternative cloud LLM provider |
| `MS_TENANT_ID` | No | Microsoft Graph for email monitoring |
| `MS_CLIENT_ID` | No | Microsoft Graph for email monitoring |
| `MS_CLIENT_SECRET` | No | Microsoft Graph for email monitoring |
| `LIVEKIT_API_KEY` | No | Voice interview coaching |
| `LIVEKIT_API_SECRET` | No | Voice interview coaching |
| `GRAFANA_PASSWORD` | No | Grafana dashboard (observability) |

## Makefile Commands

```bash
make help           # Show all available commands
make setup          # First-time setup (infra + models)
make start          # Start all services
make stop           # Stop all services
make build          # Build all Docker images
make clean          # Remove containers and volumes
make migrate        # Run database migrations
make logs           # Tail all service logs
make test           # Run all tests
```

See `make help` for the full list.

## Data Storage

All generated data stays local:

```
storage/
├── resumes/           # Generated PDF resumes
├── coverletters/      # Generated PDF cover letters
├── screenshots/       # Browser automation screenshots
├── job_descriptions/  # Scraped job postings
├── interview_prep/    # Interview coaching data
└── voice_recordings/  # Voice coaching sessions
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

**Go (`backend/`):**
- Domain modules in `internal/<domain>/` with handler, service, repository, model, dto files
- Return errors up the stack — never `log.Fatal` in handlers
- Use `zap.Logger` for structured logging — no `fmt.Println`
- Raw SQL in repository layer only (sqlx) — no ORM
- All async work goes through Asynq task queue — no inline processing in handlers

**TypeScript (`browser-agent/`):**
- Strict `tsconfig.json` — no `any`, no `@ts-ignore`
- One class per file, class name matches filename
- Playwright for all browser interaction
- Zod for input validation

**Next.js (`frontend/`):**
- App Router only — no `pages/` directory
- Server Components by default — add `"use client"` only when needed
- Tailwind CSS v4 — no `tailwind.config.js`
- API calls via `fetch` with `NEXT_PUBLIC_API_URL` env var

## License

MIT
