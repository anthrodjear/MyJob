// Repository handles database operations for interview sessions.
//
// Responsibilities:
//   - CRUD operations on interview_sessions table
//   - Status transition enforcement (via model.CanTransition)
//   - Transcript updates (JSONB append)
//   - Filtering and pagination for list queries
//
// This file contains NO business logic. It translates SQL errors to
// domain errors (ErrNotFound, ErrInvalidStatus) for the service layer.
//
// Rules followed:
//   - No SELECT * — columns listed explicitly
//   - Parameterized queries only — no string interpolation
//   - Transactions for multi-step writes
//   - Errors wrapped with context for debugging
package interviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Domain errors
// ---------------------------------------------------------------------------

// Domain errors for the interviews package.
// Use errors.Is() to check — never string matching.
var (
	// ErrNotFound indicates the requested interview session does not exist.
	ErrNotFound = errors.New("interview session not found")

	// ErrInvalidStatus indicates the requested status transition is not allowed.
	ErrInvalidStatus = errors.New("invalid status transition")
)

// ---------------------------------------------------------------------------
// Column list — single source of truth for SELECT queries
// ---------------------------------------------------------------------------

// interviewSessionColumns lists all columns in interview_sessions.
// Use this in every SELECT to prevent drift between queries and struct fields.
const interviewSessionColumns = `
	id, application_id, mode, status, external_session_id,
	provider, model, transcript, score, feedback,
	started_at, ended_at, created_at, updated_at`

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository handles database operations for interview sessions.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new interviews repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Queries: Read
// ---------------------------------------------------------------------------

// GetByID fetches an interview session by ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*InterviewSession, error) {
	var session InterviewSession
	err := r.db.GetContext(ctx, &session,
		`SELECT `+interviewSessionColumns+`
		 FROM interview_sessions WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get interview session by id: %w", err)
	}
	return &session, nil
}

// GetByExternalSessionID fetches an interview session by its external
// session identifier (e.g., LiveKit room name). Returns ErrNotFound
// if no matching row exists.
func (r *Repository) GetByExternalSessionID(ctx context.Context, externalID string) (*InterviewSession, error) {
	var session InterviewSession
	err := r.db.GetContext(ctx, &session,
		`SELECT `+interviewSessionColumns+`
		 FROM interview_sessions WHERE external_session_id = $1`, externalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get interview session by external id: %w", err)
	}
	return &session, nil
}

// ListFilter defines filters for listing interview sessions.
type ListFilter struct {
	ApplicationID uuid.UUID
	Status        string
	Mode          string
	Limit         int
	Offset        int
}

// buildWhere builds the WHERE clause and args from a filter.
func (f ListFilter) buildWhere() (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if f.ApplicationID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("application_id = $%d", len(args)+1))
		args = append(args, f.ApplicationID)
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, f.Status)
	}
	if f.Mode != "" {
		conditions = append(conditions, fmt.Sprintf("mode = $%d", len(args)+1))
		args = append(args, f.Mode)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// List returns interview sessions matching the filter with total count.
// Limit is capped at 100 to prevent unbounded queries.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]InterviewSession, int64, error) {
	// Cap limit to prevent unbounded queries
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	where, args := filter.buildWhere()

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM interview_sessions" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count interview sessions: %w", err)
	}

	// Fetch page
	query := `SELECT ` + interviewSessionColumns + `
	          FROM interview_sessions` + where + " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var sessions []InterviewSession
	if err := r.db.SelectContext(ctx, &sessions, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list interview sessions: %w", err)
	}
	return sessions, total, nil
}

// ---------------------------------------------------------------------------
// Queries: Write
// ---------------------------------------------------------------------------

// Create inserts a new interview session.
// Sets created_at and updated_at to now.
func (r *Repository) Create(ctx context.Context, session *InterviewSession) error {
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	// Marshal transcript to JSONB (empty array by default)
	transcriptJSON, err := json.Marshal(session.Transcript)
	if err != nil {
		return fmt.Errorf("marshal transcript: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO interview_sessions
		    (id, application_id, mode, status, external_session_id,
		     provider, model, transcript, score, feedback,
		     started_at, ended_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		session.ID, session.ApplicationID, session.Mode, session.Status,
		session.ExternalSessionID, session.Provider, session.Model,
		transcriptJSON, session.Score, session.Feedback,
		session.StartedAt, session.EndedAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create interview session: %w", err)
	}
	return nil
}

// UpdateStatus transitions an interview session to a new status.
// It validates the transition using CanTransition before executing.
// Sets started_at when transitioning to "active".
// Sets ended_at when transitioning to a terminal state.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Fetch current status for transition validation
	var currentStatus string
	err = tx.GetContext(ctx, &currentStatus,
		"SELECT status FROM interview_sessions WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get current status: %w", err)
	}

	// Validate transition
	if !CanTransition(currentStatus, status) {
		return ErrInvalidStatus
	}

	now := time.Now()

	// Build dynamic UPDATE query based on status
	query := `UPDATE interview_sessions SET status = $1, updated_at = $2`
	args := []interface{}{status, now}
	argIdx := 3

	// Set timestamps based on status
	switch status {
	case StatusActive:
		query += fmt.Sprintf(", started_at = $%d", argIdx)
		args = append(args, now)
		argIdx++
	case StatusCompleted, StatusFailed, StatusCancelled:
		query += fmt.Sprintf(", ended_at = $%d", argIdx)
		args = append(args, now)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update interview session status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// UpdateExternalSessionID sets the external session identifier (e.g.,
// LiveKit room name) on an interview session.
func (r *Repository) UpdateExternalSessionID(ctx context.Context, id uuid.UUID, externalID string) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET external_session_id = $1, updated_at = $2
		 WHERE id = $3`,
		externalID, now, id)
	if err != nil {
		return fmt.Errorf("update external session id: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateProvider sets the provider and model on an interview session.
func (r *Repository) UpdateProvider(ctx context.Context, id uuid.UUID, provider, model string) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET provider = $1, model = $2, updated_at = $3
		 WHERE id = $4`,
		provider, model, now, id)
	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendTranscript adds a transcript entry to the session's JSONB transcript array.
// Uses PostgreSQL's || operator for atomic append. COALESCE handles nil/NULL
// transcripts (e.g., rows inserted before transcript was initialized).
func (r *Repository) AppendTranscript(ctx context.Context, id uuid.UUID, entry TranscriptEntry) error {
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal transcript entry: %w", err)
	}

	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET transcript = COALESCE(transcript, '[]'::jsonb) || $1::jsonb,
		     updated_at = $2
		 WHERE id = $3`,
		string(entryJSON), now, id)
	if err != nil {
		return fmt.Errorf("append transcript: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateScore sets the interview score on a session.
func (r *Repository) UpdateScore(ctx context.Context, id uuid.UUID, score float64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET score = $1, updated_at = $2
		 WHERE id = $3`,
		score, now, id)
	if err != nil {
		return fmt.Errorf("update score: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateFeedback sets the evaluation feedback on a session.
// Separate from UpdateScore to avoid overwriting the score when
// feedback events arrive independently.
func (r *Repository) UpdateFeedback(ctx context.Context, id uuid.UUID, feedback json.RawMessage) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET feedback = $1, updated_at = $2
		 WHERE id = $3`,
		feedback, now, id)
	if err != nil {
		return fmt.Errorf("update feedback: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// StartSession performs a transactional status transition from "pending"
// to "starting", setting external_session_id, provider, and model in a
// single atomic write. Prevents inconsistent state on partial failure.
func (r *Repository) StartSession(ctx context.Context, id uuid.UUID, externalID, provider, model string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Fetch current status for transition validation
	var currentStatus string
	err = tx.GetContext(ctx, &currentStatus,
		"SELECT status FROM interview_sessions WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get current status: %w", err)
	}

	// Validate transition
	if !CanTransition(currentStatus, StatusStarting) {
		return ErrInvalidStatus
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`UPDATE interview_sessions
		 SET status = $1, external_session_id = $2, provider = $3, model = $4, updated_at = $5
		 WHERE id = $6`,
		StatusStarting, externalID, provider, model, now, id)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	return tx.Commit()
}
