package tasks

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateTaskRequest is the payload for creating a new task.
type CreateTaskRequest struct {
	Type       string          `json:"type" binding:"required"`
	Params     json.RawMessage `json:"params"`
	Priority   int             `json:"priority"`
	ScheduledAt *time.Time     `json:"scheduled_at"`
}

// --- Response DTOs ---

// TaskResponse is the API response for a single task.
type TaskResponse struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Params      json.RawMessage `json:"params,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       *string         `json:"error,omitempty"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	Priority    int             `json:"priority"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TaskListResponse is the API response for a list of tasks.
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// --- Internal DTOs (for queue payloads) ---

// JobDiscoveryPayload is the params for a job discovery task.
type JobDiscoveryPayload struct {
	SourceID      uuid.UUID `json:"source_id"`
	Keywords      []string  `json:"keywords"`
	Location      string    `json:"location"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// JobScoringPayload is the params for a job scoring task.
type JobScoringPayload struct {
	JobID         uuid.UUID `json:"job_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// ApplicationSubmitPayload is the params for an application submission task.
type ApplicationSubmitPayload struct {
	ApplicationID uuid.UUID       `json:"application_id"`
	FormData      json.RawMessage `json:"form_data"`
	CorrelationID uuid.UUID       `json:"correlation_id"`
}

// EmbeddingPayload is the params for an embedding generation task.
type EmbeddingPayload struct {
	SourceType    string    `json:"source_type"`
	SourceID      uuid.UUID `json:"source_id"`
	Content       string    `json:"content"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// CoverLetterGenPayload is the params for a cover letter generation task.
type CoverLetterGenPayload struct {
	CoverLetterID uuid.UUID `json:"cover_letter_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// ResumeGeneratePayload is the params for a resume generation task.
type ResumeGeneratePayload struct {
	JobID         uuid.UUID `json:"job_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// ResumeTailorPayload is the params for a resume tailoring task.
type ResumeTailorPayload struct {
	JobID         uuid.UUID `json:"job_id"`
	ResumeID      uuid.UUID `json:"resume_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// EmailCheckPayload is the params for an email check task.
type EmailCheckPayload struct {
	ApplicationID uuid.UUID `json:"application_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// InterviewPrepPayload is the params for an interview preparation task.
type InterviewPrepPayload struct {
	ApplicationID uuid.UUID `json:"application_id"`
	CorrelationID uuid.UUID `json:"correlation_id"`
}

// VoiceSessionPayload is the params for starting a voice interview session.
// The browser-agent receives this and joins the LiveKit room.
type VoiceSessionPayload struct {
	InterviewID     uuid.UUID `json:"interview_id"`
	ApplicationID   uuid.UUID `json:"application_id"`
	Mode            string    `json:"mode"`
	ExternalSession string    `json:"external_session"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
}

// ToResponse converts a Task model to a TaskResponse DTO.
func ToResponse(t *Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		Type:        t.Type,
		Status:      t.Status,
		Params:      t.Params,
		Result:      t.Result,
		Error:       t.Error,
		Attempts:    t.Attempts,
		MaxAttempts: t.MaxAttempts,
		Priority:    t.Priority,
		ScheduledAt: t.ScheduledAt,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
