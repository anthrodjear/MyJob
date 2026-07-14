// Package approvals handles human-in-the-loop approval gates for job applications.
//
// The approval system bridges the gap between automatic scoring and application
// submission. When a job scores in the "review" tier (between ReviewThreshold
// and AutoThreshold), an approval request is created for human review before
// the application is submitted.
//
// Flow:
//  1. Job scored → tier = "review"
//  2. ApprovalRequest created with job snapshot, resume preview, cover letter
//  3. User reviews → approves (auto-apply) or rejects
//  4. If approved → application submitted via task queue
//  5. If rejected → application marked as rejected, no submission
package approvals

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Domain Errors
// ============================================================================

var (
	// ErrNotFound indicates the approval request does not exist.
	ErrNotFound = errors.New("approval request not found")

	// ErrInvalidStatus indicates the requested status transition is not allowed.
	ErrInvalidStatus = errors.New("invalid approval status transition")

	// ErrReasonRequired indicates a rejection reason was not provided.
	ErrReasonRequired = errors.New("approval rejection reason is required")
)

// ============================================================================
// Status Constants
// ============================================================================

const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
)

// approvalTransitions defines valid status transitions.
// Only pending → approved/rejected are allowed.
var approvalTransitions = map[string][]string{
	ApprovalStatusPending:  {ApprovalStatusApproved, ApprovalStatusRejected},
	ApprovalStatusApproved: {}, // terminal
	ApprovalStatusRejected: {}, // terminal
}

// IsValidStatus returns true if the status is a known value.
func IsValidStatus(status string) bool {
	_, ok := approvalTransitions[status]
	return ok
}

// CanTransition returns true if the status transition is allowed.
func CanTransition(from, to string) bool {
	allowed, ok := approvalTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// ============================================================================
// Database Row Model
// ============================================================================

// ApprovalRequest represents a human review gate for a job application.
//
// Schema: approval_requests(id, application_id, job_snapshot, resume_preview_path,
//
//	cover_letter_preview, status, rejection_reason, created_at, reviewed_at)
type ApprovalRequest struct {
	ID                 uuid.UUID   `db:"id"`
	ApplicationID      uuid.UUID   `db:"application_id"`
	JobSnapshot        JobSnapshot `db:"job_snapshot"`
	ResumePreviewPath  *string     `db:"resume_preview_path"`
	CoverLetterPreview *string     `db:"cover_letter_preview"`
	Status             string      `db:"status"`
	RejectionReason    *string     `db:"rejection_reason"`
	CreatedAt          time.Time   `db:"created_at"`
	ReviewedAt         *time.Time  `db:"reviewed_at"`
}

// JobSnapshot is the JSONB payload stored in approval_requests.job_snapshot.
// Captures the job details at the time of scoring so the reviewer sees
// exactly what was scored, even if the job listing changes later.
type JobSnapshot struct {
	Title        string   `json:"title"`
	Company      string   `json:"company"`
	Location     string   `json:"location"`
	URL          string   `json:"url"`
	Description  string   `json:"description"`
	Requirements []string `json:"requirements"`
	Score        float64  `json:"score"`
	Tier         string   `json:"tier"` // "review" — only review-tier jobs need approval
	ScoredAt     string   `json:"scored_at"`
}

// Value implements driver.Valuer so JobSnapshot can be persisted to JSONB.
func (js JobSnapshot) Value() (driver.Value, error) {
	b, err := json.Marshal(js)
	if err != nil {
		return nil, fmt.Errorf("approvals: marshal job snapshot: %w", err)
	}
	return b, nil
}

// Scan implements sql.Scanner so JobSnapshot can be read from JSONB.
func (js *JobSnapshot) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("approvals: scan job snapshot: unsupported type %T", src)
	}
	if err := json.Unmarshal(data, js); err != nil {
		return fmt.Errorf("approvals: unmarshal job snapshot: %w", err)
	}
	return nil
}

// ============================================================================
// Domain Methods
// ============================================================================

// TransitionTo attempts to change the approval status.
// Returns ErrInvalidStatus if the transition is not allowed.
func (a *ApprovalRequest) TransitionTo(newStatus string) error {
	if !CanTransition(a.Status, newStatus) {
		return fmt.Errorf("%w: %s → %s", ErrInvalidStatus, a.Status, newStatus)
	}
	a.Status = newStatus
	now := time.Now()
	a.ReviewedAt = &now
	return nil
}

// Validate checks the approval request for internal consistency.
func (a ApprovalRequest) Validate() error {
	if a.ApplicationID == uuid.Nil {
		return errors.New("approvals: application_id is required")
	}
	if a.JobSnapshot.Title == "" {
		return errors.New("approvals: job_snapshot.title is required")
	}
	if a.JobSnapshot.Company == "" {
		return errors.New("approvals: job_snapshot.company is required")
	}
	if a.JobSnapshot.Score < 0 || a.JobSnapshot.Score > 100 {
		return errors.New("approvals: job_snapshot.score must be 0-100")
	}
	if a.JobSnapshot.Tier != "review" {
		return errors.New("approvals: only review-tier jobs require approval")
	}
	return nil
}

// ============================================================================
// Column List
// ============================================================================

const approvalRequestColumns = `
	id, application_id, job_snapshot, resume_preview_path,
	cover_letter_preview, status, rejection_reason, created_at, reviewed_at
`
