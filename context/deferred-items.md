# Deferred Items

Tracking deferred work items that are not needed now but will be needed later.

---

## 1. GET /job-discovery/tasks/:id — Task Status Polling

**What:** Endpoint to poll async task status.

```http
GET /job-discovery/tasks/:id

{
  "id": "...",
  "status": "running",
  "progress": 42
}
```

**Why:** `TriggerScan` returns task IDs. Currently no way to check if they completed.

**Depends on:** Asynq task inspection API (`client.GetTaskInfo`) or a tasks domain that tracks task state in DB.

**Priority:** Medium — needed when task polling is wired in the frontend.

---

## 2. Route Organization — Jobs vs Job-Discovery

**What:** Currently:
- `/jobs` — CRUD
- `/job-discovery/scan` — trigger scan

Alternative:
- `/jobs` — CRUD
- `/jobs/scan` — trigger scan (scans are a jobs-domain operation)

**Why:** Cleaner REST hierarchy. Scans operate on jobs.

**Priority:** Low — current split works fine. Revisit when more endpoints exist.

---

## 3. User ID Logging for Auditability

**What:** Add `zap.String("user_id", userID)` to handler logs.

```go
userID := middleware.GetUserID(c)
h.logger.Error("update job",
    zap.String("job_id", "job_id", id.String()),
    zap.String("user_id", userID),
    zap.Error(err),
)
```

**Why:** Audit trail. Know who triggered scans, who updated statuses.

**Depends on:** JWT middleware wiring (auth middleware sets user ID in context).

**Priority:** Medium — needed before any production deployment.

---

## 4. Thin List DTO — Payload Optimization

**What:** List endpoint currently returns full `JobResponse` (Description, Requirements, MatchDetails).

```go
type JobListItem struct {
    ID         uuid.UUID `json:"id"`
    Title      string    `json:"title"`
    Company    string    `json:"company"`
    MatchScore float64   `json:"match_score"`
    Status     string    `json:"status"`
    ScrapedAt  time.Time `json:"scraped_at"`
}
```

Full details only on `GET /jobs/:id`.

**Why:** Description + MatchDetails can be large. List responses with 100 jobs = huge payloads.

**Priority:** Low — optimize when payload size becomes a problem.

---

## 5. Backend Interview Domain — ✅ COMPLETE

**What:** `backend/internal/interviews/` implemented with all 5 files + wiring.

**Completed:**
- `interviews/model.go` — `InterviewSession` entity (id, application_id, mode, status, provider, model, external_session_id, transcript, score, feedback, created_at)
- `interviews/dto.go` — Request/response types + internal event types (union type pattern)
- `interviews/repository.go` — CRUD + StartSession (transactional), UpdateStatus, UpdateExternalSessionID, UpdateProvider, AppendTranscript (COALESCE), UpdateScore, UpdateFeedback
- `interviews/service.go` — Business logic (Create, Start, Stop, HandleEvent), TaskDispatcher interface for DI, VoiceSessionPayload defined locally
- `interviews/handler.go` — 5 public + 1 internal endpoints, domain error→HTTP code mapping, structured logging

**Backend wiring:**
- `tasks/model.go` — Added `TypeVoiceSession = "voice_session"`
- `tasks/dto.go` — Added `VoiceSessionPayload` struct
- `tasks/dispatcher.go` — Added `DispatchVoiceSession()` method (1 retry, 30min timeout)
- `handlers_application.go` — Implemented `newHandleVoiceSession` factory function (calls browser-agent)
- `cmd/api/main.go` — Wired interviews domain (repo, service, handler, routes)
- `cmd/worker/main.go` — Registered voice_session handler with graceful shutdown

**Why:** Voice session task needs to be dispatched from the API and processed by the browser-agent.

**Status:** ✅ **COMPLETE** (2026-06-17)

---

## 6. Stub Domains — Need Implementation

The following domains exist as scaffolds but have no implementation:

| Domain | Purpose | DB Tables |
|--------|---------|-----------|
| **profile** | User profile CRUD (JSONB in profiles table) | profiles |
| **approvals** | Human-in-the-loop approval before auto-apply | approval_requests |
| **rag** | Embedding generation + semantic search | embeddings (pgvector) |
| **emails** | Email classification (stub classifier exists) | emails |
| **activity** | User activity logging | activity_log |
| **coverletters** | Duplicated in `resumes` module — remove or consolidate | cover_letters |

**Priority:** Medium — needed for full feature completeness (Profile → API access to profile, Approvals → auto-apply gate, RAG → embeddings, Emails → classifier, Activity → audit)
