// Service handles interview session business logic.
//
// Responsibilities:
//   - Create interview sessions (validates mode, links to application)
//   - Start sessions (dispatches voice_session task to browser-agent)
//   - Stop sessions (notifies voice service, finalizes transcript)
//   - Handle events from voice service (transcript, status, score, feedback)
//   - List and retrieve sessions
//
// This file contains NO HTTP handlers, NO database queries, NO direct
// HTTP calls to external services. It orchestrates repository calls
// and task dispatch.
//
// Error handling:
//   - Returns domain errors (ErrNotFound, ErrInvalidStatus)
//   - Wraps unexpected errors with context
//   - Never logs and returns the same error (handler decides to log)
package interviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/tasks"
)

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// TaskDispatcher abstracts the task dispatch layer. The service does not
// depend on the concrete asynq implementation — only on this interface.
// This makes the service testable with a mock dispatcher.
type TaskDispatcher interface {
	// DispatchVoiceSession enqueues a voice_session task for the browser-agent.
	// Returns the asynq task ID for correlation.
	DispatchVoiceSession(ctx context.Context, payload tasks.VoiceSessionPayload) (string, error)
}

// Service handles interview session business logic.
type Service struct {
	repo       *Repository
	dispatcher TaskDispatcher
	logger     *zap.Logger
}

// NewService creates a new interviews service.
func NewService(repo *Repository, dispatcher TaskDispatcher, logger *zap.Logger) *Service {
	return &Service{
		repo:       repo,
		dispatcher: dispatcher,
		logger:     logger.Named("interviews"),
	}
}

// ---------------------------------------------------------------------------
// Helpers (DRY for common lookups)
// ---------------------------------------------------------------------------

// getSession fetches a session by ID and returns domain errors.
// Use this in every method that operates on an existing session.
func (s *Service) getSession(ctx context.Context, id uuid.UUID) (*InterviewSession, error) {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get session %s: %w", id, err)
	}
	return session, nil
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

// Create creates a new interview session in "pending" status.
// The session is not started until Start() is called.
//
// Validates:
//   - Mode must be "assist" or "autonomous"
//   - ApplicationID must be non-nil (UUID v4)
//
// Returns the created session with generated ID and timestamps.
func (s *Service) Create(ctx context.Context, req CreateInterviewRequest) (*InterviewSession, error) {
	// Validate mode
	if !IsValidMode(req.Mode) {
		return nil, ErrInvalidStatus
	}

	// Validate application ID
	if req.ApplicationID == uuid.Nil {
		return nil, fmt.Errorf("create session: application_id is required")
	}

	session := &InterviewSession{
		ID:            uuid.New(),
		ApplicationID: req.ApplicationID,
		Mode:          req.Mode,
		Status:        StatusPending,
		Provider:      "",
		Model:         "",
		Transcript:    Transcript{},
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.logger.Info("interview session created",
		zap.String("id", session.ID.String()),
		zap.String("application_id", session.ApplicationID.String()),
		zap.String("mode", session.Mode),
	)
	return session, nil
}

// Start transitions a session from "pending" to "starting" and dispatches
// a voice_session task to the browser-agent.
//
// Provider and model are optional — if empty, the voice service uses
// config defaults.
//
// The caller should poll GetByID() or listen for internal events to
// track when the session becomes "active".
func (s *Service) Start(ctx context.Context, id uuid.UUID, req StartInterviewRequest) (*InterviewSession, error) {
	session, err := s.getSession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("start session: %w", err)
	}

	// Generate external session ID for LiveKit room correlation
	externalID := fmt.Sprintf("interview-%s", session.ID.String())

	// Transactional update: status + external_session_id + provider/model
	if err := s.repo.StartSession(ctx, id, externalID, req.Provider, req.Model); err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			return nil, ErrInvalidStatus
		}
		return nil, fmt.Errorf("start session: %w", err)
	}

	// Dispatch voice_session task to browser-agent
	payload := tasks.VoiceSessionPayload{
		InterviewID:     session.ID,
		ApplicationID:   session.ApplicationID,
		Mode:            session.Mode,
		ExternalSession: externalID,
		Provider:        req.Provider,
		Model:           req.Model,
	}

	taskID, err := s.dispatcher.DispatchVoiceSession(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("start session dispatch: %w", err)
	}

	s.logger.Info("interview session started",
		zap.String("id", session.ID.String()),
		zap.String("task_id", taskID),
		zap.String("external_session", externalID),
	)

	// Re-fetch to return updated state
	return s.getSession(ctx, id)
}

// Stop transitions a session to "cancelled" and notifies the voice service.
//
// Only sessions in "starting" or "active" status can be stopped.
// Completed, failed, and cancelled sessions are terminal.
func (s *Service) Stop(ctx context.Context, id uuid.UUID, req StopInterviewRequest) error {
	session, err := s.getSession(ctx, id)
	if err != nil {
		return fmt.Errorf("stop session: %w", err)
	}

	// Validate transition (starting/active → cancelled)
	if err := session.TransitionTo(StatusCancelled); err != nil {
		return ErrInvalidStatus
	}

	// Update status in database (also sets ended_at)
	if err := s.repo.UpdateStatus(ctx, id, StatusCancelled); err != nil {
		return fmt.Errorf("stop session update status: %w", err)
	}

	s.logger.Info("interview session stopped",
		zap.String("id", session.ID.String()),
		zap.String("reason", req.Reason),
	)
	return nil
}

// HandleEvent processes events from the voice service (browser-agent).
// This is the internal callback endpoint handler — NOT exposed to the frontend.
//
// Event types:
//   - "transcript" — appends a speaker turn to the transcript
//   - "status"     — transitions the session to a new status
//   - "score"      — sets the interview score
//   - "feedback"   — sets the evaluation feedback
//
// Returns ErrNotFound if the session doesn't exist.
// Returns ErrInvalidStatus if the status transition is invalid.
func (s *Service) HandleEvent(ctx context.Context, id uuid.UUID, req InterviewEventRequest) error {
	session, err := s.getSession(ctx, id)
	if err != nil {
		return fmt.Errorf("handle event: %w", err)
	}

	switch req.Type {
	case "transcript":
		return s.handleTranscriptEvent(ctx, session, req)
	case "status":
		return s.handleStatusEvent(ctx, session, req)
	case "score":
		return s.handleScoreEvent(ctx, session, req)
	case "feedback":
		return s.handleFeedbackEvent(ctx, session, req)
	default:
		return fmt.Errorf("handle event: unknown type %q", req.Type)
	}
}

// handleTranscriptEvent appends a speaker turn to the session transcript.
func (s *Service) handleTranscriptEvent(ctx context.Context, session *InterviewSession, req InterviewEventRequest) error {
	// Validate speaker
	if req.Speaker != SpeakerCandidate && req.Speaker != SpeakerAI && req.Speaker != SpeakerSystem {
		return fmt.Errorf("handle transcript: invalid speaker %q", req.Speaker)
	}

	// Validate content for non-system speakers
	if (req.Speaker == SpeakerCandidate || req.Speaker == SpeakerAI) && req.Content == "" {
		return fmt.Errorf("handle transcript: empty content for speaker %q", req.Speaker)
	}

	// Default timestamp to now if not provided
	ts := time.Now()
	if req.Timestamp != nil {
		ts = *req.Timestamp
	}

	entry := TranscriptEntry{
		ID:        uuid.New(),
		Speaker:   req.Speaker,
		Content:   req.Content,
		Timestamp: ts,
	}

	if err := s.repo.AppendTranscript(ctx, session.ID, entry); err != nil {
		return fmt.Errorf("handle transcript append: %w", err)
	}

	s.logger.Debug("transcript entry appended",
		zap.String("session_id", session.ID.String()),
		zap.String("speaker", req.Speaker),
	)
	return nil
}

// handleStatusEvent transitions the session to a new status.
func (s *Service) handleStatusEvent(ctx context.Context, session *InterviewSession, req InterviewEventRequest) error {
	if req.Status == "" {
		return fmt.Errorf("handle status: missing status field")
	}

	if err := session.TransitionTo(req.Status); err != nil {
		return ErrInvalidStatus
	}

	if err := s.repo.UpdateStatus(ctx, session.ID, req.Status); err != nil {
		return fmt.Errorf("handle status update: %w", err)
	}

	s.logger.Info("session status updated via event",
		zap.String("session_id", session.ID.String()),
		zap.String("status", req.Status),
	)
	return nil
}

// handleScoreEvent sets the interview score.
// Validates range [0, 100] before persisting.
func (s *Service) handleScoreEvent(ctx context.Context, session *InterviewSession, req InterviewEventRequest) error {
	if req.Score == nil {
		return fmt.Errorf("handle score: missing score field")
	}

	if *req.Score < 0 || *req.Score > 100 {
		return fmt.Errorf("handle score: score %v out of range [0, 100]", *req.Score)
	}

	if err := s.repo.UpdateScore(ctx, session.ID, *req.Score); err != nil {
		return fmt.Errorf("handle score update: %w", err)
	}

	s.logger.Info("session score updated",
		zap.String("session_id", session.ID.String()),
		zap.Float64("score", *req.Score),
	)
	return nil
}

// handleFeedbackEvent sets the evaluation feedback.
// Uses a dedicated UpdateFeedback to avoid overwriting the score.
func (s *Service) handleFeedbackEvent(ctx context.Context, session *InterviewSession, req InterviewEventRequest) error {
	if len(req.Feedback) == 0 {
		return fmt.Errorf("handle feedback: missing feedback field")
	}

	// Validate feedback is valid JSON
	var parsed interface{}
	if err := json.Unmarshal(req.Feedback, &parsed); err != nil {
		return fmt.Errorf("handle feedback: invalid JSON: %w", err)
	}

	if err := s.repo.UpdateFeedback(ctx, session.ID, req.Feedback); err != nil {
		return fmt.Errorf("handle feedback update: %w", err)
	}

	s.logger.Info("session feedback updated",
		zap.String("session_id", session.ID.String()),
	)
	return nil
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// GetByID returns an interview session by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*InterviewSession, error) {
	return s.getSession(ctx, id)
}

// List returns interview sessions matching the filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]InterviewSession, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}
