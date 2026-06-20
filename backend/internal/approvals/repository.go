// Repository handles database operations for approval requests.
//
// Responsibilities:
//   - CRUD operations on approval_requests table
//   - Filtering and pagination for list queries
//   - Status transitions (via UpdateStatus)
//
// This file contains NO business logic. It translates SQL errors to
// domain errors (ErrNotFound) for the service layer.
//
// Rules followed:
//   - No SELECT * — columns listed explicitly
//   - Parameterized queries only — no string interpolation
//   - Errors wrapped with context for debugging
package approvals

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

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository handles database operations for approval requests.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new approvals repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Queries: Read
// ---------------------------------------------------------------------------

// GetByID fetches an approval request by ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	var a ApprovalRequest
	err := r.db.GetContext(ctx, &a,
		`SELECT `+approvalRequestColumns+`
		 FROM approval_requests WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get approval by id: %w", err)
	}
	return &a, nil
}

// ListFilter defines filters for listing approval requests.
type ListFilter struct {
	Status        string
	ApplicationID uuid.UUID
	Limit         int
	Offset        int
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
	if f.ApplicationID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("application_id = $%d", argIdx))
		args = append(args, f.ApplicationID)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// List returns approval requests matching the filter with total count.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]ApprovalRequest, int64, error) {
	where, args := filter.buildWhere()

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM approval_requests" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count approvals: %w", err)
	}

	// Fetch page
	query := `SELECT ` + approvalRequestColumns + ` FROM approval_requests` + where + " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var approvals []ApprovalRequest
	if err := r.db.SelectContext(ctx, &approvals, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list approvals: %w", err)
	}
	return approvals, total, nil
}

// ---------------------------------------------------------------------------
// Queries: Write
// ---------------------------------------------------------------------------

// Create inserts a new approval request.
// Defaults status to "pending" if not already set.
func (r *Repository) Create(ctx context.Context, a *ApprovalRequest) error {
	now := time.Now()
	a.CreatedAt = now
	if a.Status == "" {
		a.Status = ApprovalStatusPending
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO approval_requests (id, application_id, job_snapshot, resume_preview_path,
		                            cover_letter_preview, status, rejection_reason, created_at, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.ApplicationID, a.JobSnapshot, a.ResumePreviewPath,
		a.CoverLetterPreview, a.Status, a.RejectionReason, a.CreatedAt, a.ReviewedAt)
	if err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	return nil
}

// UpdateStatus updates the approval status and optional rejection reason.
// Validates the transition is allowed before updating.
// Only pending → approved/rejected transitions are permitted.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, rejectionReason *string) error {
	// Fetch current to validate transition
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !CanTransition(current.Status, status) {
		return fmt.Errorf("%w: %s → %s", ErrInvalidStatus, current.Status, status)
	}

	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE approval_requests
		 SET status = $1, rejection_reason = $2, reviewed_at = $3
		 WHERE id = $4`,
		status, rejectionReason, now, id)
	if err != nil {
		return fmt.Errorf("update approval status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
