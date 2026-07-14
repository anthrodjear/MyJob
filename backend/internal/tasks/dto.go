package tasks

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateTaskRequest is the payload for creating a new task.
type CreateTaskRequest struct {
	Type        string          `json:"type" binding:"required" example:"job_discovery" enums:"job_discovery,job_scoring,resume_generate,cover_letter_gen,application_submit,fill_form,email_check,interview_prep,embedding_generate,voice_session,resume_tailor"`
	Params      json.RawMessage `json:"params" swaggertype:"object"`
	Priority    int             `json:"priority" example:"0"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty" example:"2026-01-15T10:30:00Z"`
}

// --- Response DTOs ---

// TaskResponse is the API response for a single task.
type TaskResponse struct {
	ID          uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type        string          `json:"type" example:"job_scoring"`
	Status      string          `json:"status" example:"completed" enums:"pending,running,completed,failed,cancelled"`
	Params      json.RawMessage `json:"params,omitempty" swaggertype:"object"`
	Result      json.RawMessage `json:"result,omitempty" swaggertype:"object"`
	Error       *string         `json:"error,omitempty" example:"task timeout"`
	Attempts    int             `json:"attempts" example:"1"`
	MaxAttempts int             `json:"max_attempts" example:"3"`
	Priority    int             `json:"priority" example:"0"`
	ScheduledAt time.Time       `json:"scheduled_at" example:"2026-01-15T10:30:00Z"`
	StartedAt   *time.Time      `json:"started_at,omitempty" example:"2026-01-15T10:30:05Z"`
	CompletedAt *time.Time      `json:"completed_at,omitempty" example:"2026-01-15T10:31:00Z"`
	CreatedAt   time.Time       `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time       `json:"updated_at" example:"2026-01-15T10:31:00Z"`
}

// TaskListResponse is the API response for a list of tasks.
type TaskListResponse struct {
	Tasks  []TaskResponse `json:"tasks"`
	Total  int            `json:"total" example:"42"`
	Limit  int            `json:"limit" example:"20"`
	Offset int            `json:"offset" example:"0"`
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
