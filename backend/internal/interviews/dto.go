// DTOs (Data Transfer Types) for the interviews domain.
//
// Request DTOs define the API contract for incoming payloads.
// Response DTOs define the API contract for outgoing payloads.
// Mappers convert between domain models and response DTOs.
//
// This file contains NO business logic. Validation happens here
// (binding tags) and in the service layer (business rules).
package interviews

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateInterviewRequest is the payload for POST /api/v1/interviews.
//
// Creates a new interview session in "pending" status. The session
// is not started until POST /api/v1/interviews/:id/start is called.
//
// Example:
//
//	{
//	  "application_id": "550e8400-e29b-41d4-a716-446655440000",
//	  "mode": "autonomous"
//	}
type CreateInterviewRequest struct {
	// ApplicationID links this interview to a job application.
	ApplicationID uuid.UUID `json:"application_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Mode determines who drives the conversation.
	// Valid values: "assist", "autonomous".
	Mode string `json:"mode" binding:"required,oneof=assist autonomous" example:"autonomous" enums:"assist,autonomous"`
}

// StartInterviewRequest is the payload for POST /api/v1/interviews/:id/start.
//
// Starts the interview session. The backend enqueues a voice_session task
// for the browser-agent, which joins the LiveKit room and begins the interview.
//
// Provider and model are optional — if omitted, the voice service uses
// defaults from config/application.yaml.
//
// Example:
//
//	{
//	  "provider": "openai_realtime",
//	  "model": "gpt-4o-realtime-preview"
//	}
type StartInterviewRequest struct {
	// Provider is the voice backend (e.g., "openai_realtime", "elevenlabs").
	// Empty string means "use config default".
	Provider string `json:"provider" example:"openai_realtime"`

	// Model is the specific model for the provider (e.g., "gpt-4o-realtime-preview").
	// Empty string means "use config default".
	Model string `json:"model" example:"gpt-4o-realtime-preview"`
}

// StopInterviewRequest is the payload for POST /api/v1/interviews/:id/stop.
//
// Stops an active interview session. The backend notifies the voice service,
// which gracefully disconnects from the LiveKit room and finalizes the transcript.
//
// Example:
//
//	{
//	  "reason": "user_cancelled"
//	}
type StopInterviewRequest struct {
	// Reason is a free-text explanation for stopping (for audit trail).
	// Optional — empty string is acceptable.
	Reason string `json:"reason" example:"user_cancelled"`
}

// InterviewEventRequest is the payload for POST /internal/interviews/:id/events.
//
// This is an INTERNAL endpoint used by the voice service (browser-agent)
// to report events back to the backend. It is NOT exposed to the frontend.
//
// This is a union type — only fields relevant to Type should be set:
//
//	Type="transcript" → set Speaker, Content, Timestamp
//	Type="status"     → set Status
//	Type="score"      → set Score
//	Type="feedback"   → set Feedback
//
// Example (transcript):
//
//	{
//	  "type": "transcript",
//	  "speaker": "candidate",
//	  "content": "I have 5 years of experience in Go",
//	  "timestamp": "2026-06-19T10:30:00Z"
//	}
//
// Example (status):
//
//	{
//	  "type": "status",
//	  "status": "active"
//	}
type InterviewEventRequest struct {
	// Type identifies the kind of event. Valid values:
	// "transcript", "status", "score", "feedback".
	Type string `json:"type" binding:"required,oneof=transcript status score feedback" example:"transcript" enums:"transcript,status,score,feedback"`

	// Status is the new session status. Only used when Type="status".
	// Must be a valid transition from the current status.
	Status string `json:"status,omitempty" example:"active" enums:"pending,active,completed,cancelled"`

	// --- Transcript fields (Type="transcript") ---

	// Speaker identifies who is talking. Use "candidate", "ai", or "system".
	Speaker string `json:"speaker,omitempty" example:"candidate" enums:"candidate,ai,system"`

	// Content is the spoken text or transcription.
	Content string `json:"content,omitempty" example:"I have 5 years of experience in Go"`

	// Timestamp is when this turn occurred. Use *time.Time so that
	// omitting it produces nil (not "0001-01-01T00:00:00Z").
	Timestamp *time.Time `json:"timestamp,omitempty" example:"2026-06-19T10:30:00Z"`

	// --- Score field (Type="score") ---

	// Score is the AI's evaluation (0.0–100.0). Only used when Type="score".
	Score *float64 `json:"score,omitempty" example:"85.5"`

	// --- Feedback field (Type="feedback") ---

	// Feedback is the full evaluation payload. Only used when Type="feedback".
	// Stored as raw JSON because the schema varies by evaluation model.
	Feedback json.RawMessage `json:"feedback,omitempty" swaggertype:"object"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// InterviewResponse is the API response for a single interview session.
//
// Returned by:
//   - GET /api/v1/interviews/:id
//   - POST /api/v1/interviews (created session)
//   - POST /api/v1/interviews/:id/start (started session)
//   - POST /internal/interviews/:id/events (updated session)
type InterviewResponse struct {
	ID                uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID     uuid.UUID       `json:"application_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Mode              string          `json:"mode" example:"autonomous" enums:"assist,autonomous"`
	Status            string          `json:"status" example:"active" enums:"pending,active,completed,cancelled"`
	ExternalSessionID *string         `json:"external_session_id,omitempty" example:"livekit-session-123"`
	Provider          string          `json:"provider" example:"openai_realtime"`
	Model             string          `json:"model" example:"gpt-4o-realtime-preview"`
	Transcript        Transcript      `json:"transcript"`
	Score             *float64        `json:"score,omitempty" example:"85.5"`
	Feedback          json.RawMessage `json:"feedback,omitempty" swaggertype:"object"`
	StartedAt         *time.Time      `json:"started_at,omitempty" example:"2026-06-19T10:00:00Z"`
	EndedAt           *time.Time      `json:"ended_at,omitempty" example:"2026-06-19T10:30:00Z"`
	CreatedAt         time.Time       `json:"created_at" example:"2026-06-19T09:55:00Z"`
	UpdatedAt         time.Time       `json:"updated_at" example:"2026-06-19T10:30:00Z"`
}

// InterviewListResponse is the API response for listing interview sessions.
//
// Returned by: GET /api/v1/interviews
type InterviewListResponse struct {
	Interviews []InterviewResponse `json:"interviews"`
	Total      int64               `json:"total" example:"15"`
	Limit      int                 `json:"limit" example:"20"`
	Offset     int                 `json:"offset" example:"0"`
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

// ToResponse converts a domain InterviewSession to an API InterviewResponse.
//
// This is a pure data copy — no transformation, no validation.
// The mapper exists to decouple the internal model from the API contract.
func ToResponse(s *InterviewSession) InterviewResponse {
	return InterviewResponse{
		ID:                s.ID,
		ApplicationID:     s.ApplicationID,
		Mode:              s.Mode,
		Status:            s.Status,
		ExternalSessionID: s.ExternalSessionID,
		Provider:          s.Provider,
		Model:             s.Model,
		Transcript:        s.Transcript,
		Score:             s.Score,
		Feedback:          s.Feedback,
		StartedAt:         s.StartedAt,
		EndedAt:           s.EndedAt,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}
