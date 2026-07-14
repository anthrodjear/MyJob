// Package interviews implements the voice interview session domain.
//
// Domain model for tracking LiveKit-based interview sessions between
// a candidate and an AI interviewer (or AI copilot).
//
// Key entities:
//   - InterviewSession — aggregate root, owns state machine and transcript
//   - TranscriptEntry — individual speaker turn within a session
//
// State machine (sessionTransitions):
//
//	pending → starting → active → completed
//	        	 ↘       ↘
//	        	  failed    cancelled
//
// All transitions go through CanTransition() or InterviewSession.TransitionTo().
// Terminal states (completed, failed, cancelled) have no outgoing transitions.
//
// This package contains NO HTTP handlers, NO database queries, NO LLM calls.
// It defines types only. Service and repository live in their own files.
package interviews

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Constants: Interview modes
// ---------------------------------------------------------------------------

// Interview mode constants define who drives the conversation.
const (
	// ModeAssist — user attends the interview. AI listens and provides
	// real-time suggestions via a side channel. The user speaks on their
	// own behalf.
	ModeAssist = "assist"

	// ModeAutonomous — user is absent. AI answers interviewer questions
	// directly via LiveKit audio. The AI speaks on the user's behalf.
	ModeAutonomous = "autonomous"
)

// ---------------------------------------------------------------------------
// Constants: Session status lifecycle
// ---------------------------------------------------------------------------

// Interview status constants define the session lifecycle.
//
// Valid transitions are defined in sessionTransitions().
// Do NOT add a status without adding its transition entries.
const (
	// StatusPending — session created, not yet started. Waiting for
	// the voice service to pick up the task.
	StatusPending = "pending"

	// StatusStarting — voice service accepted the task, joining LiveKit
	// room and initializing STT/TTS providers.
	StatusStarting = "starting"

	// StatusActive — session is live. Audio is flowing, brain is
	// generating responses, transcript is being recorded.
	StatusActive = "active"

	// StatusCompleted — interview finished normally. Transcript and
	// scores are finalized. Terminal state.
	StatusCompleted = "completed"

	// StatusFailed — session terminated due to an error (provider crash,
	// timeout, etc.). Terminal state. Check Transcript for partial data.
	StatusFailed = "failed"

	// StatusCancelled — session cancelled by user or system before
	// completion. Terminal state.
	StatusCancelled = "cancelled"
)

// ---------------------------------------------------------------------------
// Constants: Transcript speaker roles
// ---------------------------------------------------------------------------

// Speaker constants identify who is speaking in a transcript entry.
// Use these constants — not raw strings — to ensure consistent analytics
// and UI rendering.
const (
	// SpeakerCandidate is the human being interviewed.
	SpeakerCandidate = "candidate"

	// SpeakerAI is the AI interviewer (autonomous mode) or AI copilot
	// (assist mode) providing suggestions.
	SpeakerAI = "ai"

	// SpeakerSystem is a non-speaking entry (e.g., "interview started",
	// "connection lost", "timeout warning"). Used for metadata events.
	SpeakerSystem = "system"
)

// ---------------------------------------------------------------------------
// State machine: valid transitions
// ---------------------------------------------------------------------------

// sessionTransitions returns the valid status transitions for a session.
// Returns a fresh map on each call to prevent accidental mutation.
//
// Transition rules:
//   - pending → starting (voice service accepted), failed, cancelled
//   - starting → active (room joined, providers ready), failed, cancelled
//   - active → completed (interview finished), failed, cancelled
//   - completed/failed/cancelled → (nothing, terminal states)
func sessionTransitions() map[string][]string {
	return map[string][]string{
		StatusPending:   {StatusStarting, StatusFailed, StatusCancelled},
		StatusStarting:  {StatusActive, StatusFailed, StatusCancelled},
		StatusActive:    {StatusCompleted, StatusFailed, StatusCancelled},
		StatusCompleted: {}, // terminal — no outgoing transitions
		StatusFailed:    {}, // terminal — no outgoing transitions
		StatusCancelled: {}, // terminal — no outgoing transitions
	}
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

// IsValidMode checks if a mode string is a recognized interview mode.
// Use this in service-layer validation before creating a session.
func IsValidMode(m string) bool {
	return m == ModeAssist || m == ModeAutonomous
}

// IsValidStatus checks if a status string is a recognized session status.
// Derived from sessionTransitions() — single source of truth.
func IsValidStatus(s string) bool {
	_, ok := sessionTransitions()[s]
	return ok
}

// CanTransition checks if a status transition is allowed.
// Returns false for unknown source states.
//
// Usage:
//
//	if !CanTransition(session.Status, targetStatus) {
//	    return ErrInvalidStatus
//	}
func CanTransition(from, to string) bool {
	allowed, ok := sessionTransitions()[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Model: TranscriptEntry
// ---------------------------------------------------------------------------

// TranscriptEntry represents a single speaker turn in the interview transcript.
//
// Each entry has a UUID for:
//   - Client-side React keying (no index-based keys)
//   - Per-entry updates or deletions (e.g., redact sensitive content)
//   - Linking audio chunks or evaluation feedback to specific turns
//
// Stored as JSONB in PostgreSQL. The Go layer is responsible for ID
// generation (uuid.New()) — the database stores raw JSONB, it does not
// generate IDs for nested objects.
type TranscriptEntry struct {
	// ID uniquely identifies this transcript entry. Generated by Go code,
	// not by PostgreSQL.
	ID uuid.UUID `json:"id"`

	// Speaker identifies who is talking. Use SpeakerCandidate, SpeakerAI,
	// or SpeakerSystem constants — not raw strings.
	Speaker string `json:"speaker"`

	// Content is the spoken text (for AI) or transcription (for candidate).
	// May be empty for SpeakerSystem metadata entries.
	Content string `json:"content"`

	// Timestamp is when this turn occurred (UTC). Set by the voice service
	// at the moment audio is received or speech is synthesized.
	Timestamp time.Time `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// JSONB helper: Transcript
// ---------------------------------------------------------------------------

// Transcript is a named slice of TranscriptEntry that implements sql.Scanner
// and driver.Valuer for PostgreSQL JSONB round-trips.
//
// Without these methods, database/sql cannot scan JSONB rows into
// []TranscriptEntry — the standard library only knows how to scan into
// types that either have explicit scanner support (e.g., json.RawMessage)
// or implement the sql.Scanner interface. Unlike other common JSONB types,
// unnamed slice types lack built-in scanning, so we provide a named type.
//
// Usage is identical to []TranscriptEntry:
//
//	var t Transcript
//	json.Unmarshal(data, &t)   // works (underlying type is []TranscriptEntry)
//	t[0].Speaker                // works
//	len(t)                       // works
type Transcript []TranscriptEntry

// Scan implements sql.Scanner for Transcript.
func (t *Transcript) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("interviews: scan transcript: unsupported type %T", src)
	}
	if len(data) == 0 {
		*t = Transcript{}
		return nil
	}
	return json.Unmarshal(data, t)
}

// Value implements driver.Valuer for Transcript.
func (t Transcript) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// ---------------------------------------------------------------------------
// JSONB helper: Feedback
// ---------------------------------------------------------------------------

// Feedback is a named json.RawMessage that implements sql.Scanner and
// driver.Valuer for PostgreSQL JSONB round-trips.
//
// Go 1.23 removed the special-case handling of json.RawMessage from
// database/sql, so scanning nil (SQL NULL) into a raw json.RawMessage
// field now fails with "unsupported Scan". This wrapper restores the
// behaviour via the Scanner interface.
//
// Usage is identical to json.RawMessage:
//
//	var f Feedback
//	f = Feedback(`{"score":85}`)
//	json.Unmarshal(data, &f)  // works (underlying type is json.RawMessage)
type Feedback json.RawMessage

// Scan implements sql.Scanner for Feedback.
func (f *Feedback) Scan(src interface{}) error {
	if src == nil {
		*f = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("interviews: scan feedback: unsupported type %T", src)
	}
	if len(data) == 0 {
		*f = Feedback{}
		return nil
	}
	return json.Unmarshal(data, (*json.RawMessage)(f))
}

// Value implements driver.Valuer for Feedback.
func (f Feedback) Value() (driver.Value, error) {
	if len(f) == 0 {
		return nil, nil
	}
	return []byte(f), nil
}

// MarshalJSON implements json.Marshaler for Feedback so that json.Marshal
// emits the raw JSON content rather than base64-encoded bytes.
func (f Feedback) MarshalJSON() ([]byte, error) {
	if len(f) == 0 {
		return []byte("null"), nil
	}
	return []byte(f), nil
}

// UnmarshalJSON implements json.Unmarshaler for Feedback.
func (f *Feedback) UnmarshalJSON(data []byte) error {
	*f = append((*f)[:0], data...)
	return nil
}

// ---------------------------------------------------------------------------
// Model: InterviewSession
// ---------------------------------------------------------------------------

// InterviewSession is the aggregate root for a voice interview.
//
// It owns:
//   - State machine (Status field + TransitionTo method)
//   - Transcript (full conversation history as JSONB)
//   - Scoring (Score + Feedback, populated after interview completes)
//
// It does NOT own:
//   - LiveKit connection state (managed by voice service)
//   - STT/TTS provider instances (managed by voice service)
//   - Application or Job data (referenced by ApplicationID, fetched by service)
//
// Table: interview_sessions
type InterviewSession struct {
	// ID is the primary key (UUID v4, generated on creation).
	ID uuid.UUID `db:"id" json:"id"`

	// ApplicationID links this session to a job application.
	// Deleting the application cascades to delete this session.
	ApplicationID uuid.UUID `db:"application_id" json:"application_id"`

	// Mode is "assist" or "autonomous". Set at creation, immutable.
	Mode string `db:"mode" json:"mode"`

	// Status tracks the session lifecycle. Use TransitionTo() to change
	// status — it validates the transition before applying it.
	Status string `db:"status" json:"status"`

	// ExternalSessionID is the session identifier from the voice provider
	// (e.g., LiveKit room name "RMxxxxx", OpenAI session "sess_xxx").
	// Stored as *string because providers return non-UUID formats.
	ExternalSessionID *string `db:"external_session_id" json:"external_session_id,omitempty"`

	// Provider is the voice backend used (e.g., "openai_realtime",
	// "elevenlabs", "local_whisper+piper"). Set when session starts.
	Provider string `db:"provider" json:"provider"`

	// Model is the specific model used by the provider (e.g.,
	// "gpt-4o-realtime-preview", "whisper-1"). Set when session starts.
	Model string `db:"model" json:"model"`

	// Transcript is the full conversation history. Stored as JSONB.
	// Each entry is a TranscriptEntry with UUID, speaker, content, timestamp.
	//
	// ⚠️  WARNING: Unbounded growth risk. A 90-minute interview can produce
	// 2000+ entries. The voice service MUST enforce a rolling window or
	// periodic summarization. See code-standards.md: "Unbounded Arrays
	// in Memory".
	Transcript Transcript `db:"transcript" json:"transcript"`

	// Score is the AI's evaluation of the interview (0.0–100.0).
	// Populated after interview completes. Nil until scored.
	Score *float64 `db:"score" json:"score,omitempty"`

	// Feedback is the full evaluation payload from the AI (scores by
	// category, summary, recommendations). Stored as JSONB because
	// the schema varies by provider/evaluation model.
	// Populated after interview completes. Nil until scored.
	Feedback json.RawMessage `db:"feedback" json:"feedback,omitempty"`

	// StartedAt is when the session transitioned to "active".
	// Nil until the session actually starts.
	StartedAt *time.Time `db:"started_at" json:"started_at,omitempty"`

	// EndedAt is when the session reached a terminal state
	// (completed/failed/cancelled). Nil while session is live.
	EndedAt *time.Time `db:"ended_at" json:"ended_at,omitempty"`

	// CreatedAt is when the session record was created (UTC).
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// UpdatedAt is the last modification time (UTC). Must be set
	// explicitly on every UPDATE — no database trigger exists.
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// TableName returns the PostgreSQL table name for InterviewSession.
// Implements the sqlx Tabler interface.
func (InterviewSession) TableName() string {
	return "interview_sessions"
}

// TransitionTo attempts a status transition on this session.
// Returns nil if the transition is valid (and applies it).
// Returns an error if the transition is invalid (status unchanged).
//
// Usage:
//
//	if err := session.TransitionTo(StatusActive); err != nil {
//	    return fmt.Errorf("start session: %w", err)
//	}
func (s *InterviewSession) TransitionTo(status string) error {
	if !CanTransition(s.Status, status) {
		return errors.New("invalid transition: " + s.Status + " → " + status)
	}
	s.Status = status
	return nil
}
