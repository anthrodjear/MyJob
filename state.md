# State
_Updated: 2026-06-14 16:45_

## Current Goal
`auth` domain complete with session versioning. Next: `jobs` domain (job listing CRUD).

## Decisions
- Go module path: `backend` (not github.com/myjob/backend) — avoids git fetch errors
- Next.js frontend created via `npx create-next-app@latest` with TypeScript + Tailwind + App Router
- Skeleton Go files use `package <name>` only — no implementation yet
- Skeleton TS files use minimal exports — no implementation yet
- Docker Compose not yet updated with frontend service (pending)
- **Auth model**: Single-user local-first, password hash in Postgres, JWT with session version
- **Config**: All required vars validated at startup via `config.Validate()`
- **Session versioning**: JWT includes session_version; password change increments version → invalidates all tokens

## Plan Status
Phase 1: Foundation — Implementation in progress
- [x] docker-compose.yml (8 services)
- [x] .env.example
- [x] Makefile (with frontend commands + hash-password target)
- [x] backend/ (Go module, all domain packages, migrations)
- [x] browser-agent/ (TypeScript, Playwright, scrapers)
- [x] frontend/ (Next.js 16 + Tailwind)
- [x] config/ (application.yaml, jobsites/*.yaml, prometheus.yml)
- [x] profile/master.yaml
- [x] templates/resumes/base.tex, cover-letters/base.tex
- [x] scripts/ (setup.sh, migrate.sh, seed.sh, hash_password.go)
- [x] All services compile/build successfully

## Domain Implementation Status
- [x] `tasks` domain — model, repository, service, handler, dispatcher, DTO
- [x] `auth` domain — model, repository, service, handler, DTO, middleware
  - Login (POST /auth/login) → JWT with session_version
  - Change password (POST /auth/change-password) → increments session_version
  - JWT validation middleware checks session_version matches DB
  - UpdateLastLogin on successful login
  - Config validation at startup
  - Postgres persistence (survives restart)
  - Mutex for thread safety

## Evidence
- `go build ./cmd/api` — success
- `go build ./cmd/worker` — success
- `npx tsc --noEmit` (browser-agent) — success
- `npm run build` (frontend) — success
- `go build ./internal/tasks/...` — success
- `go build ./internal/auth/...` — success

## Open Issues
- Docker Compose needs frontend service added
- `jobs`, `applications`, `resumes`, `scoring` domains still skeleton
- No tests written yet
- No actual scraping, form filling, or LLM integration yet
- Migration 002_users needs to run on first boot