// Package activity provides an append-only audit log for the job search agent.
//
// Activity logs record every significant event in the system: job discoveries,
// application submissions, email classifications, interview completions, etc.
// The package exposes a read-only API for the frontend activity feed and an
// internal LogEvent method for other domains to write events.
//
// The activity_log table is append-only. Events are never updated or deleted.
// Each event captures what happened (event_type), what it affected
// (entity_type + entity_id), and arbitrary context (details JSONB).
//
// Usage:
//
//	// Other domains write events via the service:
//	svc.LogEvent(ctx, activity.EventJobDiscovered, "jobs", jobID, activity.Details{
//	    "title": "Backend Engineer",
//	    "source": "indeed",
//	})
//
//	// Frontend reads the activity feed via GET /api/v1/activity-logs
package activity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event type constants identify the kind of activity recorded.
// Other domains reference these constants when calling LogEvent.
// To add a new event type: define a constant here, use it in LogEvent,
// and optionally add frontend display logic.
const (
	EventJobDiscovered            = "job_discovered"            // New job found by scraper
	EventJobScored                = "job_scored"                // Job scored by LLM
	EventApplicationCreated       = "application_created"       // Application record created
	EventApplicationSubmitted     = "application_submitted"     // Application submitted to employer
	EventApplicationStatusChanged = "application_status_changed" // Status transition (e.g. submitted → interviewed)
	EventResumeGenerated          = "resume_generated"          // Resume content generated via LLM
	EventResumeTailored           = "resume_tailored"           // Resume tailored for specific job
	EventCoverLetterGenerated     = "cover_letter_generated"    // Cover letter generated via LLM
	EventEmailReceived            = "email_received"            // Incoming email stored
	EventEmailClassified          = "email_classified"          // Email classified by LLM
	EventInterviewStarted         = "interview_started"         // LiveKit interview session started
	EventInterviewCompleted       = "interview_completed"       // Interview session ended
	EventApprovalRequested        = "approval_requested"        // Application submitted for human approval
	EventApprovalApproved         = "approval_approved"         // Human approved application
	EventApprovalRejected         = "approval_rejected"         // Human rejected application
	EventEmbeddingCreated         = "embedding_created"         // Vector embedding generated for job/resume
	EventSearchPerformed          = "search_performed"          // RAG similarity search executed
	EventError                    = "error"                     // System error occurred
	EventInfo                     = "info"                      // Informational event
)

// Domain errors for the activity package.
var (
	// ErrNotFound indicates the activity log entry does not exist.
	ErrNotFound = errors.New("activity not found")

	// ErrInvalidEntityID indicates the entity_id query parameter is not a valid UUID.
	ErrInvalidEntityID = errors.New("invalid entity_id")

	// ErrInvalidTimeRange indicates start_time or end_time is not valid RFC3339.
	ErrInvalidTimeRange = errors.New("invalid time range")
)

// ActivityLog represents a single activity log entry.
// Fields map 1:1 to the activity_log table columns.
// The entity_type + entity_id pair identifies what was affected.
// The details field carries event-specific context as JSONB.
type ActivityLog struct {
	ID         uuid.UUID `db:"id" json:"id"`
	EventType  string    `db:"event_type" json:"event_type"`
	EntityType string    `db:"entity_type" json:"entity_type"`
	EntityID   uuid.UUID `db:"entity_id" json:"entity_id"`
	Details    Details   `db:"details" json:"details"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// TableName returns the table name for the ActivityLog model.
// Used by sqlx helpers if generic repository patterns are introduced.
func (ActivityLog) TableName() string {
	return "activity_log"
}

// Details is the JSONB payload for activity log details.
// Each event type populates this with relevant context:
//
//	EventJobDiscovered → {"title": "...", "company": "...", "source": "indeed"}
//	EventEmailClassified → {"category": "interview_invite", "confidence": 0.95}
//	EventError → {"error": "timeout", "component": "scoring"}
//
// The map is serialized to JSONB on write and deserialized on read.
// Nil details are stored as SQL NULL.
type Details map[string]any

// Value implements driver.Valuer for JSONB storage.
// Returns nil for nil Details (stored as SQL NULL).
// Returns marshaled JSON bytes for non-nil Details.
func (d Details) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("activity: marshal details: %w", err)
	}
	return b, nil
}

// Scan implements sql.Scanner for JSONB retrieval.
// Handles both []byte and string sources from PostgreSQL.
// Returns nil Details for NULL or empty values.
func (d *Details) Scan(src interface{}) error {
	if src == nil {
		*d = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("activity: scan details: unsupported type %T", src)
	}
	if len(data) == 0 {
		*d = nil
		return nil
	}
	if err := json.Unmarshal(data, d); err != nil {
		return fmt.Errorf("activity: unmarshal details: %w", err)
	}
	return nil
}

// activityColumns is the column list used in all SELECT queries.
// Matches the activity_log table schema exactly.
// Never use SELECT * — always reference this constant.
const activityColumns = `
	id, event_type, entity_type, entity_id, details, created_at
`
