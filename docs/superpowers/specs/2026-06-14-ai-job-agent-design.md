# AI Job Search Agent — System Design

**Date:** 2026-06-14
**Status:** Proposed
**Version:** 1.0

---

## 1. Overview

An autonomous job-search agent that automates 80-95% of the job application workflow while keeping the user in the loop for critical decisions (final submission, interview attendance).

**Core Value:** Find, match, apply, track, prepare — all from a single system running locally on your machine.

### 1.1 Goals

- Scrape job listings from configurable sources
- Match jobs against your skills and preferences
- Generate tailored resumes (LaTeX → PDF) and cover letters per job
- Automate form filling via browser (with human-in-the-loop approval)
- Track application status across the pipeline
- Monitor email for recruiter responses
- Prepare for interviews with research, study plans, and mock interviews
- Provide voice-based interview coaching via OpenAI Realtime API

### 1.2 Non-Goals (Phase 1)

- Multi-user authentication system
- Cloud deployment (local Docker Compose only)
- Mobile app
- Job posting / recruiter portal
- Automated interview attendance (voice coach only)

---

## 2. Architecture

### 2.1 Pattern: Modular Monolith + Browser Agent

A single Go application with clean module boundaries handles all backend logic. A separate TypeScript service handles browser automation (Playwright). They communicate via HTTP/gRPC over a local Docker network.

```
Frontend (Future: React/Next.js Dashboard)
        |
        v HTTP
Go Backend (Modular Monolith)
  |-- Job Search Module
  |-- Resume Module
  |-- Application Module
  |-- Email Module
  |-- Interview Module
  |-- RAG Module
        |
        v
PostgreSQL + Local File Storage
        |
        v HTTP (internal)
TypeScript Browser Agent
  |-- Playwright Controller
  |-- Form Fill Engine
  |-- Document Uploader
  |-- Stealth Plugin
  |-- Voice (LiveKit + OpenAI Realtime)
```

### 2.2 Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Backend API | Go 1.22+ / Gin / Fiber | REST API, orchestration |
| Browser Agent | TypeScript / Playwright | Job scraping, form filling |
| Database | PostgreSQL 16 + pgvector | Application tracking, RAG |
| Object Storage | Local filesystem | PDFs, resumes, cover letters |
| Embeddings | Ollama (mxbai-embed-large) | RAG, semantic search |
| LLM (Generation) | OpenAI / Anthropic API | Resume tailoring, cover letters |
| LLM (Local) | Ollama (Qwen 2.5) | Quick tasks, privacy-sensitive ops |
| Message Queue | Redis (Bull/BullMQ) | Job queue for async tasks |
| Document Gen | LaTeX / pdflatex | Professional PDF generation |
| Voice | OpenAI Realtime API + LiveKit | Interview simulation |
| Monitoring | Prometheus + Grafana | Metrics, dashboards |
| Tracing | OpenTelemetry + Jaeger | Distributed tracing |
| Email | Microsoft Graph API | Outlook/Office 365 integration |

### 2.3 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Go + TypeScript split | Go for backend, TS for browser | Go excels at concurrency/scheduling; TS has best Playwright bindings |
| Modular monolith | Not microservices | Simpler to develop/deploy locally; can extract services later |
| Local-first | All data stays on machine | Privacy for job search, resume data, interview prep |
| YAML config for job sites | Not hardcoded selectors | Easy to add new sites without code changes |
| LaTeX for resumes | Not HTML/Word | ATS-friendly, precise formatting, professional output |
| Score-based approval | AUTO/REVIEW/REJECT tiers | Eliminates review fatigue at 200+ jobs/day — only best matches auto-apply, good fits get quick review, noise is skipped |

---

## 3. Core Components

### 3.1 Job Search Module (Go)

**Responsibility:** Scrape, parse, rank, and store job listings from multiple sources.

**Flow:**
```
Job Site Config (YAML) -> Scraper -> Parser -> Ranker -> Database
                                                        |
                                                 LLM (optional: enhance match scoring)
```

**Scraping Strategy:**
1. Read job source config (YAML files in config/jobsites/)
2. Dispatch scraper tasks to Redis queue (rate-limited per site)
3. TypeScript browser agent handles actual scraping (Playwright)
4. Go backend parses HTML, extracts structured data
5. LLM enhances match scoring (optional)
6. Store in PostgreSQL + generate match report

**Rate Limiting:**
- 1 request per 2 seconds per site (configurable)
- Respect robots.txt
- Stealth mode: rotate user agents, random delays
- Daily caps per site (configurable)

### 3.2 Resume Module (Go)

**Responsibility:** Generate tailored resumes and cover letters per job application.

**Resume Versioning System:**
```
MasterProfile (JSON/YAML)
        |
        v
Resume Generator
  |-- Base Resume
  |-- Backend Developer Resume
  |-- AI/ML Engineer Resume
  |-- DevOps/SRE Resume
  |-- Security Engineer Resume
  |-- Full Stack Resume
        |
        v
LaTeX Templates (per specialization)
        |
        v
pdflatex -> ATS-optimized PDF
```

**Master Profile Structure:**
```yaml
# profile/master.yaml
name: "Your Name"
email: "you@example.com"
phone: "+1-555-0100"
location: "City, Country"
linkedin: "linkedin.com/in/yourname"
github: "github.com/yourname"

experience:
  - title: "Senior Developer"
    company: "Company A"
    duration: "2022-2024"
    highlights:
      - "Built microservices handling 10k req/s"
      - "Reduced infrastructure costs by 40%"
    skills_used: ["Go", "Kubernetes", "PostgreSQL"]

education:
  - degree: "BSc Computer Science"
    university: "University X"
    year: 2020

skills:
  backend: ["Go", "Python", "Node.js", "PostgreSQL", "Redis"]
  frontend: ["React", "TypeScript", "Next.js"]
  devops: ["Docker", "Kubernetes", "AWS", "Terraform"]
  ai_ml: ["PyTorch", "HuggingFace", "RAG", "LLMs"]

projects:
  - name: "Job Agent"
    description: "Autonomous job search AI agent"
    tech: ["Go", "TypeScript", "Playwright", "PostgreSQL"]
    url: "github.com/yourname/job-agent"

specializations:
  backend:
    focus_skills: ["Go", "Python", "PostgreSQL", "Redis", "microservices"]
    resume_template: "templates/backend.tex"
    highlight_experience: ["Company A", "Company B"]
  ai:
    focus_skills: ["PyTorch", "LLMs", "RAG", "HuggingFace"]
    resume_template: "templates/ai.tex"
    highlight_experience: ["Company C"]
  devops:
    focus_skills: ["Kubernetes", "Docker", "Terraform", "AWS"]
    resume_template: "templates/devops.tex"
    highlight_experience: ["Company A"]
```

**Tailoring Flow:**
1. Read job description
2. Extract required skills, keywords, responsibilities
3. Select best specialization (or create new variant)
4. Inject relevant experience/projects at top
5. Optimize for ATS keywords (LLM-powered)
6. Generate cover letter (OpenAI/Anthropic)
7. Compile LaTeX -> PDF
8. Store in applications/{company}/Resume.pdf

### 3.3 Application Module (Go + TypeScript)

**Responsibility:** Fill application forms, upload documents, track status.

**Score-Based Approval Tiers:**

Every matched job gets a score (0-100). The score determines the approval path:

```
Match Score >= 95  -->  AUTO-APPLY
  System generates resume, cover letter, fills form, submits automatically.
  User sees notification after submission. No manual review needed.

Match Score 80-94  -->  REVIEW REQUIRED
  System generates resume, cover letter, fills form.
  Pauses for user approval before submitting.
  User sees summary: job details, tailored resume preview, cover letter preview.

Match Score < 80   -->  REJECT
  Job is skipped. Not applied to. Logged in database for reference.
  User can override and manually apply if desired.
```

**Flow:**
```
Job Found
    |
    v
Match Scoring (skills, experience, requirements)
    |
    v
Score >= 95? --YES--> AUTO-APPLY
    |                   |
    NO                  Generate Resume + Cover Letter
    |                   |
    v                   Fill Form
Score >= 80?            |
    |                   Submit Automatically
YES |                   |
    v                   Notify User
REVIEW REQUIRED         |
    |                   v
    Generate Resume + Cover Letter     Update Status
    |
    Fill Form
    |
    Pause for User Approval
    |
    User Reviews Summary
    |
    +-- Approve --> Submit --> Update Status
    |
    +-- Reject --> Skip --> Log Rejection Reason
    |
    v
Score < 80
    |
    REJECT (skip, log, move on)
```

**Why this works:** At 200+ jobs/day, reviewing every single application is impossible. The 95% threshold means only the best matches auto-apply, and the 80-94% range catches good-but-not-perfect fits for quick human review. Below 80% is noise.

**ATS-Specific Integrations:**

| ATS | Method | Notes |
|-----|--------|-------|
| Greenhouse | API + Form | Has public job API, form filling for apply |
| Lever | API + Form | Similar to Greenhouse |
| Workday | Form only | Complex JS-heavy forms |
| Ashby | Form only | Newer ATS, simpler forms |
| Custom | Form only | Generic form filler |

**Application Status Pipeline:**
```
discovered -> viewed -> applied -> assessment -> phone_screen -> technical -> final -> offer -> rejected
```

### 3.4 Email Module (Go)

**Responsibility:** Monitor recruiter emails, categorize, draft replies, extract data.

**Microsoft Graph Integration:**
```
Outlook Inbox
    |
    v
Email Classifier (LLM)
    |-- Recruiter outreach -> Draft reply
    |-- Interview invitation -> Schedule + prep
    |-- Assessment link -> Extract + notify
    |-- Rejection -> Update status
    |-- Other -> Log and ignore
```

### 3.5 Interview Preparation Module (Go)

**Responsibility:** Research companies, generate questions, create study plans.

**Trigger:** When interview_invite email is detected.

**Flow:**
```
Interview Detected
    |
    |-- Research Company
    |   |-- Company website scraping
    |   |-- Glassdoor reviews
    |   |-- LinkedIn company page
    |   |-- News/recent funding
    |
    |-- Analyze Role
    |   |-- Job description
    |   |-- Required skills
    |   |-- Team/tech stack research
    |
    |-- Generate Study Plan
    |   |-- Technical topics (LLM-generated)
    |   |-- System design problems
    |   |-- Behavioral questions (STAR format)
    |   |-- Company-specific questions
    |
    |-- Create Materials
        |-- 50+ practice questions
        |-- Code challenge problems
        |-- Mock interview script
        |-- Reference document (PDF)
```

### 3.6 Voice Interview Coach (TypeScript + LiveKit)

**Responsibility:** Simulate interviews via voice conversation.

**Technology:**
- OpenAI Realtime API for voice conversation
- LiveKit for WebRTC infrastructure
- Custom prompt system for different interview types

**Interview Types:**
- hr_screen — behavioral, culture fit
- technical_coding — coding problems, DSA
- system_design — architecture discussions
- behavioral — STAR method questions
- technical_deep_dive — domain expertise

### 3.7 RAG Knowledge Base (Go + Ollama)

**Responsibility:** Learn from past applications, improve over time.

**Data Sources:**
- Past job applications and outcomes
- Resume versions and their effectiveness
- Interview questions asked
- Recruiter feedback
- Study notes and learnings

**Architecture:**
```
Documents (Applications, Resumes, Notes)
    |
    v
Ollama Embeddings (mxbai-embed-large)
    |
    v
Vector Store (pgvector in PostgreSQL)
    |
    v
RAG Query -> Context -> LLM -> Improved Output
```

---

## 4. Data Models

### 4.1 Core Tables

```sql
-- Master profile (single row, updated by user)
CREATE TABLE master_profile (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data JSONB NOT NULL,
    version INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Resume versions
CREATE TABLE resume_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    specialization VARCHAR(100) NOT NULL,
    template_path VARCHAR(500) NOT NULL,
    focus_skills TEXT[] NOT NULL,
    highlight_experience UUID[],
    generated_at TIMESTAMPTZ,
    pdf_path VARCHAR(500),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job sources (configurable scraping targets)
CREATE TABLE job_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_scraped_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job listings
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES job_sources(id),
    external_id VARCHAR(200),
    title VARCHAR(300) NOT NULL,
    company VARCHAR(200) NOT NULL,
    location VARCHAR(200),
    remote_type VARCHAR(50),
    salary_min INT,
    salary_max INT,
    salary_currency VARCHAR(10),
    description TEXT NOT NULL,
    requirements TEXT,
    url VARCHAR(1000) NOT NULL,
    company_url VARCHAR(500),
    posted_at TIMESTAMPTZ,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    match_score FLOAT,
    match_details JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'discovered',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source_id, external_id)
);

-- Applications
CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id) NOT NULL,
    resume_version_id UUID REFERENCES resume_versions(id),
    cover_letter_path VARCHAR(500),
    resume_path VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    applied_at TIMESTAMPTZ,
    response_at TIMESTAMPTZ,
    interview_at TIMESTAMPTZ,
    notes TEXT,
    portal_type VARCHAR(50),
    portal_url VARCHAR(1000),
    form_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Email messages
CREATE TABLE email_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id),
    message_id VARCHAR(200) NOT NULL,
    from_address VARCHAR(200) NOT NULL,
    to_address VARCHAR(200),
    subject VARCHAR(500),
    body TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    classification VARCHAR(50),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    reply_draft TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Interview prep
CREATE TABLE interview_preps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id) NOT NULL,
    company_research TEXT,
    study_plan JSONB,
    practice_questions JSONB,
    mock_interview_log JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RAG embeddings (pgvector)
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    metadata JSONB,
    embedding vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

### 4.2 Application Status Flow

```
discovered
    |
    v
viewed
    |
    v
applied
    |
    +--> assessment
    |         |
    |         v
    |    phone_screen
    |         |
    |         v
    |    technical
    |         |
    |         v
    |      final
    |         |
    |         v
    +--> rejected         offer
```

---

## 5. Configuration

### 5.1 Job Sources Config

```yaml
# config/jobsites/indeed.yaml
name: indeed
type: indeed
base_url: "https://www.indeed.com"
search_url: "https://www.indeed.com/jobs?q={query}&l={location}&sort=date"
rate_limit:
  requests_per_second: 0.5
  daily_cap: 500
  respect_robots_txt: true
selectors:
  job_list: ".jobsearch-ResultsList .result"
  title: ".jobTitle a"
  company: ".companyName"
  location: ".companyLocation"
  snippet: ".job-snippet"
  link: ".jobTitle a@href"
pagination:
  type: "offset"
  param: "start"
  per_page: 15
  max_pages: 10
```

```yaml
# config/jobsites/remoteok.yaml
name: remoteok
type: remoteok
base_url: "https://remoteok.com"
api_url: "https://remoteok.com/api"
rate_limit:
  requests_per_second: 0.5
  daily_cap: 200
```

```yaml
# config/jobsites/custom_sites.yaml
name: custom
type: custom
sites:
  - name: "myjob-mags"
    url: "https://example.com/jobs"
    selectors:
      job_list: ".job-listing"
      title: "h2.title"
      company: ".company"
      link: "a@href"
    pagination:
      type: "page"
      param: "page"
```

### 5.2 Application Config

```yaml
# config/application.yaml
application:
  # Score-based approval tiers
  approval_tiers:
    auto_apply:
      min_score: 95
      action: "auto_submit"
      notify: true              # send notification after auto-submit
    review:
      min_score: 80
      max_score: 94
      action: "pause_for_approval"
      approval_methods:         # user can approve via any of these
        - "web_dashboard"
        - "cli"
        - "email"
    reject:
      max_score: 79
      action: "skip"
      log: true                 # log skipped jobs for reference

  # Document generation
  auto_generate:
    resume: true
    cover_letter: true
  preview_before_submit: true   # show preview even for auto-apply (logged)

  resume:
    engine: "latex"
    template_dir: "./templates/resumes"
    output_dir: "./applications"
  cover_letter:
    engine: "latex"
    template_dir: "./templates/cover-letters"
    max_length: 400

  status_check_interval: "1h"
  email_check_interval: "30m"

llm:
  primary:
    provider: "openai"
    model: "gpt-4o"
    api_key_env: "OPENAI_API_KEY"
  local:
    provider: "ollama"
    model: "qwen2.5:latest"
    base_url: "http://localhost:11434"
  embeddings:
    provider: "ollama"
    model: "mxbai-embed-large"
    base_url: "http://localhost:11434"
  fallback:
    provider: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key_env: "ANTHROPIC_API_KEY"

voice:
  provider: "openai_realtime"
  model: "gpt-4o-realtime-preview"
  api_key_env: "OPENAI_API_KEY"
  livekit:
    url: "ws://localhost:7880"
    api_key: "devkey"
    api_secret: "devsecret"

email:
  provider: "microsoft_graph"
  tenant_id_env: "MS_TENANT_ID"
  client_id_env: "MS_CLIENT_ID"
  client_secret_env: "MS_CLIENT_SECRET"
  check_interval: "30m"
  folders: ["Inbox"]
```

---

## 6. API Contracts

### 6.1 Go Backend API (Port 8080)

```
GET    /api/v1/jobs                    # List jobs with filters
GET    /api/v1/jobs/:id                # Get job details
POST   /api/v1/jobs/search             # Trigger job search
GET    /api/v1/jobs/:id/match          # Get match score

GET    /api/v1/applications            # List applications
GET    /api/v1/applications/:id        # Get application details
POST   /api/v1/applications            # Create application
PUT    /api/v1/applications/:id/status # Update status
GET    /api/v1/applications/stats      # Dashboard statistics

GET    /api/v1/resumes                 # List resume versions
GET    /api/v1/resumes/:id             # Get resume details
POST   /api/v1/resumes/generate        # Generate new resume version
GET    /api/v1/resumes/:id/pdf         # Download PDF

POST   /api/v1/cover-letters/generate  # Generate cover letter

GET    /api/v1/emails                  # List email messages
POST   /api/v1/emails/sync             # Trigger email sync
GET    /api/v1/emails/:id              # Get email details
POST   /api/v1/emails/:id/reply        # Draft reply

GET    /api/v1/interviews/:id/prep     # Get interview prep
POST   /api/v1/interviews/:id/prep     # Generate interview prep
POST   /api/v1/interviews/:id/mock     # Start mock interview session

GET    /api/v1/profile                 # Get master profile
PUT    /api/v1/profile                 # Update master profile

GET    /api/v1/config/jobsites         # List job sources
POST   /api/v1/config/jobsites         # Add job source
PUT    /api/v1/config/jobsites/:id     # Update job source
DELETE /api/v1/config/jobsites/:id     # Remove job source

GET    /api/v1/dashboard/summary       # Dashboard summary
GET    /api/v1/dashboard/timeline      # Application timeline
```

### 6.2 Browser Agent API (TypeScript, Port 3000)

```
POST   /api/v1/scrape                  # Start scraping job source
GET    /api/v1/scrape/:taskId          # Get scraping task status
POST   /api/v1/scrape/stop/:taskId     # Stop scraping

POST   /api/v1/browser/fill-form       # Fill application form
POST   /api/v1/browser/upload          # Upload document
GET    /api/v1/browser/screenshot      # Get current page screenshot
POST   /api/v1/browser/navigate        # Navigate to URL

POST   /api/v1/browser/voice/start     # Start voice session
POST   /api/v1/browser/voice/stop      # Stop voice session
WS     /ws/voice                       # WebSocket for voice stream
```

---

## 7. File Structure

```
MyJob/
├── docker-compose.yml
├── .env.example
├── Makefile
│
├── backend/                          # Go Backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go
│   │   │   ├── handlers/
│   │   │   │   ├── jobs.go
│   │   │   │   ├── applications.go
│   │   │   │   ├── resumes.go
│   │   │   │   ├── emails.go
│   │   │   │   ├── interviews.go
│   │   │   │   └── profile.go
│   │   │   └── middleware/
│   │   ├── models/
│   │   │   ├── job.go
│   │   │   ├── application.go
│   │   │   ├── resume.go
│   │   │   ├── email.go
│   │   │   └── profile.go
│   │   ├── services/
│   │   │   ├── job_scraper.go
│   │   │   ├── match_engine.go
│   │   │   ├── resume_generator.go
│   │   │   ├── cover_letter.go
│   │   │   ├── email_monitor.go
│   │   │   ├── interview_prep.go
│   │   │   └── rag.go
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   ├── migrations/
│   │   │   └── queries/
│   │   └── config/
│   │       └── config.go
│   ├── go.mod
│   └── go.sum
│
├── browser-agent/                    # TypeScript Browser Agent
│   ├── src/
│   │   ├── index.ts
│   │   ├── scrapers/
│   │   │   ├── base.ts
│   │   │   ├── indeed.ts
│   │   │   ├── greenhouse.ts
│   │   │   ├── lever.ts
│   │   │   ├── remoteok.ts
|   |   |   |──fuzu.ts
|   |   |   |──myjobmag.ts
│   │   │   └── custom.ts
│   │   ├── form-filler/
│   │   │   ├── detector.ts
│   │   │   ├── fields.ts
│   │   │   └── submitter.ts
│   │   ├── voice/
│   │   │   ├── realtime.ts
│   │   │   └── livekit.ts
│   │   └── utils/
│   │       ├── stealth.ts
│   │       └── retry.ts
│   ├── package.json
│   ├── tsconfig.json
│   └── Dockerfile
│
├── config/                           # Configuration
│   ├── jobsites/
│   │   ├── indeed.yaml
│   │   ├── greenhouse.yaml
│   │   ├── lever.yaml
│   │   ├── remoteok.yaml
|   |   ├── fuzu.yaml
|   |   ├── myjobmag.yaml
│   │   └── custom_sites.yaml
│   ├── application.yaml
│   └── llm.yaml
│
├── profile/                          # User Profile
│   ├── master.yaml
│   └── preferences.yaml
│
├── templates/                        # Document Templates
│   ├── resumes/
│   │   ├── base.tex
│   │   ├── backend.tex
│   │   ├── ai.tex
│   │   ├── devops.tex
│   │   └── security.tex
│   └── cover-letters/
│       ├── base.tex
│       └── generic.tex
│
├── applications/                     # Generated Documents
│   └── {CompanyA}/
│       ├── Resume.pdf
│       ├── CoverLetter.pdf
│       └── JobDescription.txt
│
├── docs/
│   └── superpowers/
│       └── specs/
│           └── 2026-06-14-ai-job-agent-design.md
│
└── scripts/
    ├── setup.sh
    ├── migrate.sh
    └── seed.sh
```

---

## 8. Deployment

### 8.1 Docker Compose Services

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| backend | Custom Go build | 8080 | REST API, orchestration |
| browser-agent | Custom TS build | 3000 | Playwright, scraping, voice |
| postgres | pgvector/pgvector:pg16 | 5432 | Database + vector store |
| redis | redis:7-alpine | 6379 | Job queue, caching |
| ollama | ollama/ollama | 11434 | Local LLM + embeddings |
| livekit | livekit/livekit-server | 7880 | WebRTC for voice |
| prometheus | prom/prometheus | 9090 | Metrics |
| grafana | grafana/grafana | 3001 | Dashboards |

### 8.2 Quick Start

```bash
# 1. Clone and setup
git clone <repo>
cd MyJob
cp .env.example .env
# Edit .env with your API keys

# 2. Start services
docker compose up -d

# 3. Setup database
docker compose exec backend ./migrate.sh

# 4. Pull Ollama models
docker compose exec ollama ollama pull mxbai-embed-large
docker compose exec ollama ollama pull qwen2.5:latest

# 5. Configure job sources
# Edit config/jobsites/*.yaml

# 6. Add your profile
# Edit profile/master.yaml

# 7. Start searching
curl -X POST http://localhost:8080/api/v1/jobs/search \
  -H "Content-Type: application/json" \
  -d '{"query": "backend developer", "location": "remote"}'
```

---

## 9. Phased Implementation Plan

### Phase 1: Foundation (Weeks 1-2)
- Go backend scaffolding (Gin, PostgreSQL, migrations)
- TypeScript browser agent scaffolding (Playwright)
- Docker Compose setup
- Master profile model and API
- Job source configuration system
- Basic job scraping (Indeed, Remote OK)
- Job listing storage and retrieval
- Match scoring engine (simple keyword matching)

### Phase 2: Documents (Weeks 3-4)
- Resume versioning system
- Master profile to resume specialization flow
- LaTeX template system
- PDF generation pipeline
- Cover letter generation (LLM-powered)
- ATS keyword optimization
- Document storage in applications/ folder

### Phase 3: Applications (Weeks 5-6)
- Application tracking API
- ATS detection (Greenhouse, Lever, Workday, Ashby)
- Form filling engine (Playwright)
- Document upload automation
- Human-in-the-loop approval flow
- Application status pipeline
- Company career page scraping

### Phase 4: Email and Tracking (Weeks 7-8)
- Microsoft Graph API integration
- Email classification (LLM)
- Recruiter response drafting
- Interview invitation detection
- Assessment link extraction
- Application status updates from email
- Dashboard with application stats

### Phase 5: Interview Prep (Weeks 9-10)
- Company research automation
- Study plan generation (LLM)
- Practice question generation
- Interview prep documents
- LiveKit server setup
- OpenAI Realtime API integration
- Voice interview coach (basic)

### Phase 6: Intelligence (Weeks 11-12)
- RAG knowledge base (pgvector + Ollama)
- Historical application learning
- Resume effectiveness tracking
- Interview question pattern analysis
- Continuous improvement loop
- Analytics dashboard (Grafana)
- Performance monitoring

---

## 10. Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Job site blocks scraping | High | Stealth mode, rate limiting, rotation, multiple sources |
| TOS violation | High | Human-in-the-loop, respect robots.txt, no automated mass-applying |
| LLM hallucination in resume | High | Template-based generation, user review, keyword validation |
| Browser automation detection | Medium | Playwright stealth, user agent rotation, realistic delays |
| API rate limits (OpenAI) | Medium | Local Ollama fallback, caching, batching |
| Data loss | Medium | PostgreSQL backups, local file versioning |
| Email API changes | Low | Microsoft Graph stable API, abstraction layer |

---

## 11. Success Metrics

| Metric | Target | Phase |
|--------|--------|-------|
| Jobs scraped per day | 200+ | 1 |
| Match accuracy | >70% relevant | 1 |
| Resume generation time | <30 seconds | 2 |
| Application completion rate | >90% success | 3 |
| Email classification accuracy | >95% | 4 |
| Interview prep quality | User rated 4/5+ | 5 |
| Overall automation rate | 80-95% of workflow | 6 |

---

## 12. Future Considerations

- Multi-user support: Add authentication, separate profiles
- Cloud deployment: AWS/GCP with managed PostgreSQL, S3
- Browser extension: Real-time job matching while browsing
- LinkedIn integration: Official API (requires approval)
- Salary negotiation coach: LLM-powered negotiation prep
- Offer comparison: Side-by-side offer analysis
- Career path planning: Long-term skill development recommendations
