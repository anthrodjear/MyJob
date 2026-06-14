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

**Phase:** Scaffolding complete, implementation starting.

- Project structure and directory layout established
- Technology stack decisions finalized
- Docker Compose orchestration configured
- Core modules scaffolded with interface definitions
- **Next:** Begin building core services — starting with job scraping engine and PostgreSQL schema

## What's Built

- [x] Project structure and scaffolding
- [x] Docker Compose configuration (Go API, Browser Agent, Next.js frontend, PostgreSQL, Redis)
- [x] Module interface definitions
- [x] Database schema design (draft)

## What's Next

- [ ] Job scraping engine (Indeed, RemoteOK, Greenhouse, Lever adapters)
- [ ] PostgreSQL schema + pgvector for semantic skill matching
- [ ] Resume generation pipeline (LaTeX → PDF)
- [ ] Playwright browser agent for form filling
- [ ] Application tracker UI
- [ ] Recruiter email monitoring (Microsoft Graph API)
- [ ] Voice interview coaching (OpenAI Realtime + LiveKit)
