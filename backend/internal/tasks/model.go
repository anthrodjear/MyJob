package tasks

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task represents an async task tracked in the database.
type Task struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	Type         string          `db:"type" json:"type"`
	Status       string          `db:"status" json:"status"`
	Params       json.RawMessage `db:"params" json:"params,omitempty"`
	Result       json.RawMessage `db:"result" json:"result,omitempty"`
	Error        *string         `db:"error" json:"error,omitempty"`
	Attempts     int             `db:"attempts" json:"attempts"`
	MaxAttempts  int             `db:"max_attempts" json:"max_attempts"`
	Priority     int             `db:"priority" json:"priority"`
	ScheduledAt  time.Time       `db:"scheduled_at" json:"scheduled_at"`
	StartedAt    *time.Time      `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// Task status constants.
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Task type constants.
const (
	TypeJobDiscovery      = "job_discovery"
	TypeJobScoring        = "job_scoring"
	TypeApplicationSubmit = "application_submit"
	TypeEmbeddingGenerate = "embedding_generate"
	TypeCoverLetterGen    = "cover_letter_gen"
	TypeResumeGenerate    = "resume_generate"
	TypeResumeTailor      = "resume_tailor"
	TypeEmailCheck        = "email_check"
	TypeInterviewPrep     = "interview_prep"
	TypeVoiceSession      = "voice_session"
)

// TableName returns the table name for the Task model.
func (Task) TableName() string {
	return "tasks"
}
