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
    zap.String("job_id", id.String()),
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
