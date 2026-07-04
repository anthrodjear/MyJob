package applications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Domain errors for applications.
var (
	ErrNotFound        = errors.New("application not found")
	ErrNoRowsAffected  = errors.New("no rows affected")
	ErrInvalidStatus   = errors.New("invalid status transition")
)

// Repository handles database operations for applications.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new applications repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetByID fetches an application by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	var app Application
	err := r.db.GetContext(ctx, &app,
		`SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,
		        applied_at, response_at, interview_at, notes, portal_type, portal_url,
		        form_data, created_at, updated_at
		 FROM applications WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get application by id: %w", err)
	}
	return &app, nil
}

// ListFilter defines filters for listing applications.
type ListFilter struct {
	Status     string
	JobID      uuid.UUID
	PortalType string
	Limit      int
	Offset     int
}

// buildWhere builds the WHERE clause and args from a filter.
func (f ListFilter) buildWhere() (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, f.Status)
		argIdx++
	}
	if f.JobID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("job_id = $%d", argIdx))
		args = append(args, f.JobID)
		argIdx++
	}
	if f.PortalType != "" {
		conditions = append(conditions, fmt.Sprintf("portal_type = $%d", argIdx))
		args = append(args, f.PortalType)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// List returns applications matching the filter with total count.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Application, int64, error) {
	where, args := filter.buildWhere()

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM applications" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}

	// Fetch page
	query := `SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,
	                 applied_at, response_at, interview_at, notes, portal_type, portal_url,
	                 form_data, created_at, updated_at
	          FROM applications` + where + " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var apps []Application
	if err := r.db.SelectContext(ctx, &apps, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	return apps, total, nil
}

// Create inserts a new application.
func (r *Repository) Create(ctx context.Context, app *Application) error {
	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO applications (id, job_id, resume_id, cover_letter_id, status, approval_tier,
		                           portal_type, portal_url, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		app.ID, app.JobID, app.ResumeID, app.CoverLetterID, app.Status, app.ApprovalTier,
		app.PortalType, app.PortalURL, app.Notes, app.CreatedAt, app.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}
	return nil
}

// UpdateStatus updates application status, notes, and timestamps.
// It also logs an audit event in application_events.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, notes string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Fetch current status for audit log
	var oldStatus string
	err = tx.GetContext(ctx, &oldStatus, "SELECT status FROM applications WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get current status: %w", err)
	}

	now := time.Now()

	// Set timestamp based on status
	var timestampCol string
	switch status {
	case StatusApplied:
		timestampCol = "applied_at"
	case StatusRejected:
		timestampCol = "response_at"
	case StatusOffer:
		timestampCol = "response_at"
	default:
		timestampCol = ""
	}

	query := `UPDATE applications SET status = $1, updated_at = $2`
	args := []interface{}{status, now}
	argIdx := 3

	if notes != "" {
		query += fmt.Sprintf(", notes = $%d", argIdx)
		args = append(args, notes)
		argIdx++
	}
	if timestampCol != "" {
		query += fmt.Sprintf(", %s = $%d", timestampCol, argIdx)
		args = append(args, now)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update application status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	// Log audit event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO application_events (id, application_id, old_status, new_status, notes, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), id, oldStatus, status, notes, now)
	if err != nil {
		return fmt.Errorf("log application event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetEvents returns the audit trail for an application.
func (r *Repository) GetEvents(ctx context.Context, applicationID uuid.UUID) ([]ApplicationEvent, error) {
	var events []ApplicationEvent
	err := r.db.SelectContext(ctx, &events,
		`SELECT id, application_id, old_status, new_status, notes, created_at
		 FROM application_events
		 WHERE application_id = $1
		 ORDER BY created_at ASC`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("get application events: %w", err)
	}
	return events, nil
}

// UpdateNotes updates permanent notes on an application.
func (r *Repository) UpdateNotes(ctx context.Context, id uuid.UUID, notes string) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE applications SET notes = $1, updated_at = $2 WHERE id = $3`,
		notes, now, id)
	if err != nil {
		return fmt.Errorf("update notes: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetStats returns aggregate statistics for applications.
func (r *Repository) GetStats(ctx context.Context) (*ApplicationStatsResponse, error) {
	stats := &ApplicationStatsResponse{
		ByStatus: make(map[string]int64),
		ByTier:   make(map[string]int64),
	}

	// Total count
	err := r.db.GetContext(ctx, &stats.Total, "SELECT COUNT(*) FROM applications")
	if err != nil {
		return nil, fmt.Errorf("count applications: %w", err)
	}

	// By status
	rows, err := r.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM applications GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("group by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		stats.ByStatus[status] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status rows: %w", err)
	}

	// By tier
	rows2, err := r.db.QueryContext(ctx, "SELECT approval_tier, COUNT(*) FROM applications GROUP BY approval_tier")
	if err != nil {
		return nil, fmt.Errorf("group by tier: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var tier string
		var count int64
		if err := rows2.Scan(&tier, &count); err != nil {
			return nil, fmt.Errorf("scan tier count: %w", err)
		}
		stats.ByTier[tier] = count
	}
	if err = rows2.Err(); err != nil {
		return nil, fmt.Errorf("iterate tier rows: %w", err)
	}

	return stats, nil
}
