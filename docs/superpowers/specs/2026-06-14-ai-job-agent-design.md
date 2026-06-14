# AI Job Search Agent — System Design

**Date:** 2026-06-14
**Status:** Proposed
**Version:** 2.0

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

### 2.1 Pattern: Modular Monolith + Browser Agent + Worker

Two Go binaries (API + Worker) share domain modules. A separate TypeScript service handles browser automation. Redis provides the task queue connecting them.

```
Frontend (Future: React/Next.js Dashboard)
        |
        v HTTP
Go API Binary (Port 8080)
  |-- Domain Modules (shared)
  |     |-- jobs/
  |     |-- applications/
  |     |-- resumes/
  |     |-- emails/
  |     |-- interviews/
  |     |-- approvals/
  |     |-- profile/
  |     |-- rag/
  |     |-- tasks/
  |     +-- activity/
        |
        v
PostgreSQL + Redis
        ^
        |
Go Worker Binary (async)
  |-- Task consumer
  |-- Resume generator
  |-- Cover letter generator
  |-- Email sync
  |-- Embedding creator
  |-- Interview prep
        |
        v HTTP (task dispatch)
TypeScript Browser Agent (Port 3000)
  |-- Playwright Controller
  |-- Form Fill Engine
  |-- Document Uploader
  |-- Stealth Plugin
  |-- Voice (LiveKit + OpenAI Realtime)
```

### 2.2 Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Backend API | Go 1.22+ / Gin | REST API, orchestration |
| Backend Worker | Go 1.22+ / Asynq | Async task processing |
| Browser Agent | TypeScript / Playwright | Job scraping, form filling |
| Database | PostgreSQL 16 + pgvector | Application tracking, RAG |
| Queue/Cache | Redis 7 | Task queue, rate limiting, session storage |
| Object Storage | Local filesystem | PDFs, resumes, cover letters |
| Embeddings | Ollama (mxbai-embed-large) | RAG, semantic search |
| LLM (Generation) | OpenAI / Anthropic API | Resume tailoring, cover letters |
| LLM (Local) | Ollama (Qwen 2.5) | Quick tasks, privacy-sensitive ops |
| Document Gen | LaTeX / pdflatex | Professional PDF generation |
| Voice | OpenAI Realtime API + LiveKit | Interview simulation |
| Monitoring | Prometheus + Grafana | Metrics, dashboards |
| Tracing | OpenTelemetry + Jaeger | Distributed tracing |
| Email | Microsoft Graph API | Outlook/Office 365 integration |

### 2.3 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Go + TypeScript split | Go for backend, TS for browser | Go excels at concurrency/scheduling; TS has best Playwright bindings |
| API + Worker split | Two Go binaries | API stays responsive; worker handles long-running tasks (scraping, PDF gen, embeddings) |
| Domain modules | Not services/ or models/ | Each domain owns handler/service/repo/model/dto — testable, extractable |
| Redis as backbone | Queue + cache + rate limiter | Central to async task dispatch, rate limiting per job site, session storage |
| Task-based browser agent | Not synchronous HTTP | Playwright operations take minutes — queue task, poll for result |
| Score-based approval | AUTO/REVIEW/REJECT tiers | Eliminates review fatigue at 200+ jobs/day |
| Modular monolith | Not microservices | Simpler to develop/deploy locally; can extract services later |
| Local-first | All data stays on machine | Privacy for job search, resume data, interview prep |
| YAML config for job sites | Not hardcoded selectors | Easy to add new sites without code changes |
| LaTeX for resumes | Not HTML/Word | ATS-friendly, precise formatting, professional output |

---

## 3. Core Components

### 3.1 Job Search Module

**Responsibility:** Scrape, parse, rank, and store job listings from multiple sources.

**Flow:**
```
Job Site Config (YAML)
    |
    v
API receives search request
    |
    v
Enqueue scrape tasks to Redis (one per source)
    |
    v
Worker consumes tasks
    |
    v
Browser Agent scrapes (Playwright)
    |
    v
Worker parses HTML, extracts structured data
    |
    v
Match scoring (skills, experience, LLM-enhanced)
    |
    v
Store in PostgreSQL
    |
    v
Score >= 95? --> AUTO-APPLY (enqueue application task)
Score 80-94? --> REVIEW (create approval request)
Score < 80?  --> REJECT (log and skip)
```

**Rate Limiting (Redis-backed):**
- 1 request per 2 seconds per site (configurable)
- Respect robots.txt
- Stealth mode: rotate user agents, random delays
- Daily caps per site (configurable)
- Redis sliding window counters per source

### 3.2 Resume Module

**Responsibility:** Generate tailored resumes and cover letters per job application.

**Resume Versioning System:**
```
MasterProfile (YAML)
        |
        v
Resume Generator (Worker)
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
        |
        v
Storage: storage/resumes/{company}_{date}.pdf
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

**Tailoring Flow (Worker):**
1. Read job description
2. Extract required skills, keywords, responsibilities
3. Select best specialization (or create new variant)
4. Inject relevant experience/projects at top
5. Optimize for ATS keywords (LLM-powered)
6. Generate cover letter (OpenAI/Anthropic)
7. Compile LaTeX -> PDF
8. Store in `storage/resumes/` and `storage/coverletters/`

### 3.3 Application Module

**Responsibility:** Fill application forms, upload documents, track status.

**Score-Based Approval Tiers:**

Every matched job gets a score (0-100). The score determines the approval path:

```
Match Score >= 95  -->  AUTO-APPLY
  System generates resume, cover letter, fills form, submits automatically.
  User sees notification after submission. No manual review needed.

Match Score 80-94  -->  REVIEW REQUIRED
  System generates resume, cover letter, fills form.
  Creates approval_request in database.
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
    NO                  Enqueue: generate_resume + generate_coverletter + fill_form
    |                   |
    v                   Worker processes tasks
Score >= 80?            |
    |                   Submit Automatically
YES |                   |
    v                   Notify User
REVIEW REQUIRED         |
    |                   v
    Enqueue: generate_resume + generate_coverletter + fill_form
    |
    Create approval_request (status: pending)
    |
    User Reviews Summary
    |
    +-- Approve --> Enqueue submit_task --> Update Status
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

### 3.4 Approval Module

**Responsibility:** Manage human-in-the-loop approval flow for applications.

**Flow:**
```
Application needs review (score 80-94)
    |
    v
Create approval_request
  - job_id
  - application_id
  - snapshot (job details, resume preview, cover letter preview)
  - status: pending
    |
    v
User queries pending approvals
    |
    v
User approves or rejects
    |
    v
  Approve --> Enqueue submit_task
  Reject  --> Update application status, log reason
```

### 3.5 Task Queue System

**Responsibility:** Async task processing for all long-running operations.

**Task Types:**

| Task | Producer | Consumer | Timeout |
|------|----------|----------|---------|
| `scrape_source` | API | Worker | 10min |
| `generate_resume` | Worker | Worker | 2min |
| `generate_coverletter` | Worker | Worker | 1min |
| `fill_form` | Worker | Browser Agent | 5min |
| `submit_application` | Approval | Browser Agent | 5min |
| `sync_emails` | Cron | Worker | 5min |
| `generate_interview_prep` | Email | Worker | 3min |
| `create_embeddings` | Worker | Worker | 1min |
| `voice_session` | API | Browser Agent | 60min |

**Task States:**
```
pending -> active -> completed
                  -> failed (retry up to 3x)
                  -> cancelled
```

**API Response Pattern:**
```json
// POST /api/v1/tasks
{
  "type": "fill_form",
  "application_id": "abc-123",
  "params": { ... }
}

// Response
{
  "taskId": "task_789",
  "status": "queued",
  "estimated_seconds": 120
}

// GET /api/v1/tasks/task_789
{
  "taskId": "task_789",
  "type": "fill_form",
  "status": "active",
  "progress": 0.6,
  "started_at": "2026-06-14T10:30:00Z"
}
```

### 3.6 Email Module

**Responsibility:** Monitor recruiter emails, categorize, draft replies, extract data.

**Microsoft Graph Integration:**
```
Cron (every 30min)
    |
    v
Worker syncs Outlook Inbox via Microsoft Graph
    |
    v
Email Classifier (LLM)
    |-- Recruiter outreach -> Draft reply
    |-- Interview invitation -> Schedule + create interview_prep task
    |-- Assessment link -> Extract + notify
    |-- Rejection -> Update application status
    |-- Other -> Log and ignore
```

### 3.7 Interview Preparation Module

**Responsibility:** Research companies, generate questions, create study plans.

**Trigger:** When interview_invite email is detected (worker creates task).

**Flow (Worker):**
```
Interview Task Created
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
        |
        v
    Store in storage/interview_prep/
```

### 3.8 Voice Interview Coach (TypeScript + LiveKit)

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

### 3.9 RAG Knowledge Base

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

### 3.10 Activity Log

**Responsibility:** Audit trail for all system actions.

**Logged Events:**
- Job scraped
- Match scored
- Resume generated
- Cover letter generated
- Application submitted
- Approval requested / approved / rejected
- Email synced / classified
- Interview prep generated
- Task created / completed / failed
- Voice session started / completed

---

## 4. Data Models

### 4.1 Core Tables

```sql
-- ============================================
-- PROFILES
-- ============================================

-- Master profile (single row, updated by user)
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data JSONB NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- JOB SOURCES
-- ============================================

-- Configurable scraping targets
CREATE TABLE job_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_scraped_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- JOBS
-- ============================================

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

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_match_score ON jobs(match_score DESC);
CREATE INDEX idx_jobs_company ON jobs(company);

-- ============================================
-- RESUMES
-- ============================================

CREATE TABLE resumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    specialization VARCHAR(100) NOT NULL,
    template_path VARCHAR(500) NOT NULL,
    focus_skills TEXT[] NOT NULL,
    highlight_experience UUID[],
    pdf_path VARCHAR(500),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- COVER LETTERS
-- ============================================

CREATE TABLE cover_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id),
    resume_id UUID REFERENCES resumes(id),
    content TEXT NOT NULL,
    pdf_path VARCHAR(500),
    word_count INT,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- APPLICATIONS
-- ============================================

CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id) NOT NULL,
    resume_id UUID REFERENCES resumes(id),
    cover_letter_id UUID REFERENCES cover_letters(id),
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    approval_tier VARCHAR(20) NOT NULL,  -- 'auto', 'review', 'reject'
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

CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_job ON applications(job_id);

-- ============================================
-- APPROVALS
-- ============================================

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id) NOT NULL,
    job_snapshot JSONB NOT NULL,        -- frozen job details at time of request
    resume_preview_path VARCHAR(500),   -- path to resume PDF
    cover_letter_preview TEXT,          -- cover letter text
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, approved, rejected
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_approvals_status ON approval_requests(status);

-- ============================================
-- TASKS
-- ============================================

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, active, completed, failed, cancelled
    params JSONB,
    result JSONB,
    error TEXT,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    priority INT NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_scheduled ON tasks(scheduled_at) WHERE status = 'pending';

-- ============================================
-- EMAILS
-- ============================================

CREATE TABLE emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id),
    message_id VARCHAR(200) NOT NULL UNIQUE,
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

CREATE INDEX idx_emails_application ON emails(application_id);
CREATE INDEX idx_emails_classification ON emails(classification);

-- ============================================
-- INTERVIEWS
-- ============================================

CREATE TABLE interviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id) NOT NULL,
    company_research TEXT,
    study_plan JSONB,
    practice_questions JSONB,
    mock_interview_log JSONB,
    prep_pdf_path VARCHAR(500),
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- RAG EMBEDDINGS
-- ============================================

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type VARCHAR(50) NOT NULL,   -- 'job', 'application', 'resume', 'email', 'interview'
    source_id UUID NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB,
    embedding vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_embeddings_source ON embeddings(source_type, source_id);

-- ============================================
-- ACTIVITY LOG
-- ============================================

CREATE TABLE activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,   -- 'job', 'application', 'task', 'email', etc.
    entity_id UUID NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activity_entity ON activity_log(entity_type, entity_id);
CREATE INDEX idx_activity_created ON activity_log(created_at DESC);
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
      notify: true
    review:
      min_score: 80
      max_score: 94
      action: "pause_for_approval"
      approval_methods:
        - "web_dashboard"
        - "cli"
        - "email"
    reject:
      max_score: 79
      action: "skip"
      log: true

  # Document generation
  auto_generate:
    resume: true
    cover_letter: true
  preview_before_submit: true

  resume:
    engine: "latex"
    template_dir: "./templates/resumes"
  cover_letter:
    engine: "latex"
    template_dir: "./templates/cover-letters"
    max_length: 400

  status_check_interval: "1h"
  email_check_interval: "30m"

# Task Queue Config
queue:
  redis_url: "redis://localhost:6379"
  concurrency: 5
  retryAttempts: 3
  retryDelay: "5s"
  rateLimits:
    scrape: "1/2s"       # 1 per 2 seconds
    llm: "10/1m"         # 10 per minute
    email: "1/5s"        # 1 per 5 seconds

# LLM Config
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

# Voice Config
voice:
  provider: "openai_realtime"
  model: "gpt-4o-realtime-preview"
  api_key_env: "OPENAI_API_KEY"
  livekit:
    url: "ws://localhost:7880"
    api_key: "devkey"
    api_secret: "devsecret"

# Email Config
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
# ============================================
# JOBS
# ============================================
GET    /api/v1/jobs                    # List jobs with filters
GET    /api/v1/jobs/:id                # Get job details
POST   /api/v1/jobs/search             # Trigger job search (returns task ID)
GET    /api/v1/jobs/:id/match          # Get match score

# ============================================
# APPLICATIONS
# ============================================
GET    /api/v1/applications            # List applications
GET    /api/v1/applications/:id        # Get application details
POST   /api/v1/applications            # Create application
PUT    /api/v1/applications/:id/status # Update status
GET    /api/v1/applications/stats      # Dashboard statistics

# ============================================
# APPROVALS
# ============================================
GET    /api/v1/approvals               # List pending approvals
GET    /api/v1/approvals/:id           # Get approval details (with snapshot)
POST   /api/v1/approvals/:id/approve   # Approve application
POST   /api/v1/approvals/:id/reject    # Reject application (with reason)

# ============================================
# RESUMES
# ============================================
GET    /api/v1/resumes                 # List resume versions
GET    /api/v1/resumes/:id             # Get resume details
POST   /api/v1/resumes/generate        # Generate new resume (returns task ID)
GET    /api/v1/resumes/:id/pdf         # Download PDF

# ============================================
# COVER LETTERS
# ============================================
GET    /api/v1/cover-letters           # List cover letters
GET    /api/v1/cover-letters/:id       # Get cover letter details
POST   /api/v1/cover-letters/generate  # Generate cover letter (returns task ID)
GET    /api/v1/cover-letters/:id/pdf   # Download PDF

# ============================================
# EMAILS
# ============================================
GET    /api/v1/emails                  # List email messages
POST   /api/v1/emails/sync             # Trigger email sync (returns task ID)
GET    /api/v1/emails/:id              # Get email details
POST   /api/v1/emails/:id/reply        # Draft reply

# ============================================
# INTERVIEWS
# ============================================
GET    /api/v1/interviews              # List interview preps
GET    /api/v1/interviews/:id          # Get interview prep details
POST   /api/v1/interviews/:id/prep     # Generate interview prep (returns task ID)
POST   /api/v1/interviews/:id/mock     # Start mock interview session

# ============================================
# TASKS
# ============================================
GET    /api/v1/tasks                   # List tasks
GET    /api/v1/tasks/:id               # Get task status/progress
POST   /api/v1/tasks                   # Create task (generic)
DELETE /api/v1/tasks/:id               # Cancel task

# ============================================
# PROFILE
# ============================================
GET    /api/v1/profile                 # Get master profile
PUT    /api/v1/profile                 # Update master profile

# ============================================
# CONFIG
# ============================================
GET    /api/v1/config/jobsites         # List job sources
POST   /api/v1/config/jobsites         # Add job source
PUT    /api/v1/config/jobsites/:id     # Update job source
DELETE /api/v1/config/jobsites/:id     # Remove job source

# ============================================
# RAG
# ============================================
POST   /api/v1/rag/index               # Index documents for RAG
POST   /api/v1/rag/search              # Search RAG knowledge base
GET    /api/v1/rag/stats               # RAG index statistics

# ============================================
# DASHBOARD
# ============================================
GET    /api/v1/dashboard/summary       # Dashboard summary
GET    /api/v1/dashboard/timeline      # Application timeline
GET    /api/v1/dashboard/activity      # Recent activity log
```

### 6.2 Browser Agent API (TypeScript, Port 3000)

```
# Task-based (async) - preferred
POST   /api/v1/tasks/scrape           # Start scraping (returns task ID)
POST   /api/v1/tasks/fill-form        # Fill form (returns task ID)
POST   /api/v1/tasks/upload           # Upload document (returns task ID)
GET    /api/v1/tasks/:taskId          # Get task status

# Direct (sync) - only for quick operations
GET    /api/v1/browser/screenshot     # Get current page screenshot
POST   /api/v1/browser/navigate       # Navigate to URL

# Voice
POST   /api/v1/voice/start            # Start voice session
POST   /api/v1/voice/stop             # Stop voice session
WS     /ws/voice                      # WebSocket for voice stream
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
│   │   ├── api/
│   │   │   └── main.go              # API binary
│   │   └── worker/
│   │       └── main.go              # Worker binary
│   │
│   ├── internal/
│   │   ├── jobs/                     # Job search domain
│   │   │   ├── handler.go           # HTTP handlers
│   │   │   ├── service.go           # Business logic
│   │   │   ├── repository.go        # Database queries
│   │   │   ├── model.go             # Domain models
│   │   │   ├── dto.go               # Request/response types
│   │   │   └── scraper.go           # Scraper orchestration
│   │   │
│   │   ├── applications/             # Application tracking domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── resumes/                  # Resume generation domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   ├── dto.go
│   │   │   └── generator.go         # LaTeX generation logic
│   │   │
│   │   ├── coverletters/             # Cover letter domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── emails/                   # Email monitoring domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   ├── dto.go
│   │   │   └── classifier.go        # LLM email classification
│   │   │
│   │   ├── interviews/               # Interview prep domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── approvals/                # Approval flow domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── tasks/                    # Task queue domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   ├── dto.go
│   │   │   └── dispatcher.go        # Task dispatch to worker/browser
│   │   │
│   │   ├── profile/                  # User profile domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── rag/                      # RAG knowledge base domain
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   │
│   │   ├── activity/                 # Activity logging domain
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   └── model.go
│   │   │
│   │   ├── api/                      # HTTP router and middleware
│   │   │   ├── router.go
│   │   │   └── middleware/
│   │   │       ├── auth.go
│   │   │       ├── logging.go
│   │   │       └── ratelimit.go
│   │   │
│   │   ├── database/                 # Database infrastructure
│   │   │   ├── postgres.go
│   │   │   ├── redis.go
│   │   │   └── migrations/
│   │   │       ├── 001_initial.up.sql
│   │   │       └── 001_initial.down.sql
│   │   │
│   │   └── config/                   # Configuration loading
│   │       └── config.go
│   │
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
│   │   │   ├── fuzu.ts
│   │   │   ├── myjobmag.ts
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
│   │   ├── fuzu.yaml
│   │   ├── myjobmag.yaml
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
├── storage/                          # Generated Files (gitignored)
│   ├── resumes/
│   ├── coverletters/
│   ├── screenshots/
│   ├── job_descriptions/
│   ├── interview_prep/
│   └── voice_recordings/
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
| api | Custom Go build (cmd/api) | 8080 | REST API |
| worker | Custom Go build (cmd/worker) | - | Async task processor |
| browser-agent | Custom TS build | 3000 | Playwright, scraping, voice |
| postgres | pgvector/pgvector:pg16 | 5432 | Database + vector store |
| redis | redis:7-alpine | 6379 | Task queue, cache, rate limiting |
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

# 2. Start infrastructure
docker compose up -d postgres redis ollama

# 3. Setup database
docker compose exec backend ./scripts/migrate.sh

# 4. Pull Ollama models
docker compose exec ollama ollama pull mxbai-embed-large
docker compose exec ollama ollama pull qwen2.5:latest

# 5. Start all services
docker compose up -d

# 6. Configure job sources
# Edit config/jobsites/*.yaml

# 7. Add your profile
# Edit profile/master.yaml

# 8. Start searching
curl -X POST http://localhost:8080/api/v1/jobs/search \
  -H "Content-Type: application/json" \
  -d '{"query": "backend developer", "location": "remote"}'

# 9. Check task status
curl http://localhost:8080/api/v1/tasks/{taskId}

# 10. Review pending approvals
curl http://localhost:8080/api/v1/approvals
```

---

## 9. Phased Implementation Plan

### Phase 1: Foundation (Weeks 1-2)
- Go backend scaffolding (Gin, PostgreSQL, Redis, migrations)
- Worker binary scaffolding (Asynq)
- TypeScript browser agent scaffolding (Playwright)
- Docker Compose setup
- Domain module structure (jobs, applications, resumes, tasks, profile)
- Master profile model and API
- Job source configuration system
- Task queue system (Redis + Asynq)
- Basic job scraping (Indeed, Remote OK)
- Job listing storage and retrieval
- Match scoring engine (simple keyword matching)
- Activity logging

### Phase 2: Documents (Weeks 3-4)
- Resume versioning system
- Master profile to resume specialization flow
- LaTeX template system
- PDF generation pipeline (Worker task)
- Cover letter generation (LLM-powered Worker task)
- ATS keyword optimization
- Document storage in storage/ folder

### Phase 3: Applications (Weeks 5-6)
- Application tracking API
- Approval module (approval_requests table, approve/reject flow)
- Score-based approval tiers (AUTO/REVIEW/REJECT)
- ATS detection (Greenhouse, Lever, Workday, Ashby)
- Form filling engine (Playwright, task-based)
- Document upload automation
- Application status pipeline

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
| Job site blocks scraping | High | Stealth mode, rate limiting (Redis), rotation, multiple sources |
| TOS violation | High | Score-based approval tiers, respect robots.txt, no automated mass-applying |
| LLM hallucination in resume | High | Template-based generation, user review, keyword validation |
| Browser automation detection | Medium | Playwright stealth, user agent rotation, realistic delays |
| API rate limits (OpenAI) | Medium | Local Ollama fallback, caching (Redis), batching |
| Data loss | Medium | PostgreSQL backups, local file versioning |
| Email API changes | Low | Microsoft Graph stable API, abstraction layer |
| Worker crash | Medium | Asynq retry mechanism (3 attempts), task state persistence |
| Redis failure | Medium | Persistence enabled, fallback to synchronous processing |

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
