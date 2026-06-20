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
    String()),
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
| **approvals** | Human-in-the-loop approval before auto-apply | approval_requests |
| **rag** | Embedding generation + semantic search | embeddings (pgvector) |
| **activity** | User activity logging | activity_log |

**Priority:** Medium — needed for full feature completeness (Approvals → auto-apply gate, RAG → embeddings, Activity → audit)

---

## 7. Profile Domain — ✅ COMPLETE

**What:** `backend/internal/profile/` implemented with all 5 files + wiring.

**Completed:**
- `profile/model.go` — Profile entity with JSONB `ProfileData` (Preferences, Skills, Education, Links), Skill proficiency constants, `Validate()`, `ApplyPatch()` for PATCH merge logic, `driver.Valuer`/`sql.Scanner` for JSONB serialization
- `profile/dto.go` — `UpdateProfileRequest` (PUT), `PatchProfileRequest` with pointer fields for PATCH semantics, `ProfileResponse` with embedded stats, `ToResponse`/`ToStatsResponse` mappers
- `profile/repository.go` — Singleton pattern (LIMIT 1), `Get`/`Create`/`Update` (OCC via version), `UpdatePartial` with `SELECT FOR UPDATE` + transaction, validation inside tx
- `profile/service.go` — `RepositoryInterface` for DI, `GetOrCreate` (first-run default), `Update` (PUT with If-Match), `UpdatePartial` (PATCH with If-Match), client version from header
- `profile/handler.go` — GET/PUT/PATCH `/profile`, ETag/If-Match headers (RFC 7232), `httpresp` helpers, structured error mapping

**Backend wiring:**
- `internal/api/router.go` — Added `ProfileHandler` to `RouterConfig`, registered in protected routes
- `cmd/api/main.go` — Initialized profileRepo, profileService, profileHandler; added to router

**Why:** User profile is the central data source for job matching, scoring, resume generation, and cover letter personalization. Single-profile (singleton) pattern for local-first single-user system.

**Status:** ✅ **COMPLETE** (2026-06-20)

---

## 7. Emails Domain — ✅ COMPLETE

**What:** `backend/internal/emails/` implemented with all 6 files + wiring.

**Completed:**
- `emails/model.go` — Email entity, Classification constants (interview_invite, rejection, offer, follow_up, spam, phishing, other), Domain Errors (ErrNotFound, ErrInvalidClassification), IsValidClassification
- `emails/dto.go` — Request/Response DTOs (StoreEmailRequest, UpdateEmailRequest, ListFilterRequest with ToListFilter conversion, EmailResponse, EmailListResponse, ClassifyResponse), ToEmailResponse mapper
- `emails/repository.go` — CRUD + Upsert by message_id, List with filters, execAndCheckRows helper, parameterized LIMIT/OFFSET
- `emails/classifier.go` — LLMClient interface, OllamaClient implementing it, Classifier with pre-compiled template, System field used in Ollama request, fallback JSON parsing (direct → code-fence regex → error), truncation logging
- `emails/service.go` — Business logic (Store with StoreEmailParams, GetByID, List, MarkRead, UpdateDraft, Reclassify), RepositoryInterface + ClassifierInterface for DI, getEmail helper for DRY error handling
- `emails/handler.go` — 5 HTTP handlers (Store POST, List GET, GetByID GET, Update PATCH, Reclassify POST), logger injection, httpresp helpers (BadRequest, NotFound, InternalError, Created, OK), structured error logging

**Backend wiring:**
- `internal/api/router.go` — Added EmailsHandler to RouterConfig, registered in protected routes
- `cmd/api/main.go` — Initialized emailsRepo, emailClassifier (NewClassifierFromConfig with dedicated LLM.EmailClassifier config), emailsService, emailsHandler; added to router
- `config.go` — Added EmailClassifier to LLMConfig with separate provider/model/baseURL/timeout

**Why:** Worker stores incoming emails from Browser Agent; API provides read access and manual re-classification; classification updates application status.

**Status:** ✅ **COMPLETE** (2026-06-20)

---

## 8. Resume Tailor Worker Handler — ✅ COMPLETE

**What:** Missing handler for existing `resume_tailor` task type (task type, payload, dispatcher method all existed but no handler).

**Completed:**
- `resumes/llm.go` — `ResumeTailor` interface, `OllamaResumeTailor` implementation with pre-compiled template, System field in Ollama request, fallback JSON parsing, `ResumeTailorResult` with `content` + `summary`
- `resumes/service.go` — Added `ResumeTailor` to Service struct, `NewService` accepts `ResumeTailor`, `TailorResume` method (fetches resume, calls tailor, saves version snapshot, updates content)
- `cmd/worker/handlers_resume.go` — `newHandleTailorResume` handler (fetches job, calls `resumesSvc.TailorResume`, logs summary)
- `cmd/worker/main.go` — Initialized `resumeTailor` via `NewResumeTailorFromConfig`, passed to `NewService`, registered `tasks.TypeResumeTailor` handler

**Backend wiring:**
- `tasks/model.go` — `TypeResumeTailor = "resume_tailor"` (already existed)
- `tasks/dto.go` — `ResumeTailorPayload` with job_id, resume_id, correlation_id (already existed)
- `tasks/dispatcher.go` — `DispatchResumeTailor` method (already existed)
- `config/prompts.go` — `ResumeTailor` PromptPair loaded from YAML (already existed)
- `cmd/api/main.go` — Initialized `resumeTailor`, passed to `resumes.NewService`

**Why:** Resume tailoring adapts an existing resume for a specific job by reordering skills, adjusting experience descriptions to match job requirements, and using keywords from the job description — while preserving factual accuracy.

**Status:** ✅ **COMPLETE** (2026-06-21)