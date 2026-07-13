// DTOs (Data Transfer Types) for the approvals domain.
//
// The approval domain handles human review gates for jobs that score
// in the "review" tier. The API surface is simple:
//
//   - GET    /approvals           → List all approval requests (with filters)
//   - GET    /approvals/:id       → Get single approval request
//   - POST   /approvals/:id/approve  → Approve (auto-apply)
//   - POST   /approvals/:id/reject   → Reject with reason
//
// Request DTOs define the API contract for incoming payloads.
// Response DTOs define the API contract for outgoing payloads.
// Mappers convert between domain models and response DTOs.
//
// This file contains NO business logic. Validation happens here
// (binding tags) and in the service layer (business rules).
package approvals

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// ApproveRequest is the payload for POST /approvals/:id/approve.
//
// Approves the application for automatic submission.
// No additional fields needed — the approval decision is in the endpoint.
// The `_` field exists to satisfy linters (empty struct warning).
//
// Example:
//
//	{}  // empty body
type ApproveRequest struct {
	// _ is a no-op field to avoid "empty struct" linter warnings.
	_ struct{} `json:"-"`
}

// RejectRequest is the payload for POST /approvals/:id/reject.
//
// Rejects the application with a required reason.
//
// Example:
//
//	{
//	  "reason": "Salary too low for current market"
//	}
type RejectRequest struct {
	// Reason is required for audit trail.
	Reason string `json:"reason" binding:"required" example:"Salary too low for current market"`
}

// ListFilter is not a DTO — used internally for query params.
// See handler for query binding.

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// ApprovalResponse is the API response for a single approval request.
//
// Returned by:
//   - GET /approvals/:id
//   - GET /approvals (in list)
type ApprovalResponse struct {
	ID                 uuid.UUID   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID      uuid.UUID   `json:"application_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	JobSnapshot        JobSnapshot `json:"job_snapshot"`
	ResumePreviewPath  *string     `json:"resume_preview_path,omitempty" example:"/storage/resumes/550e8400-e29b-41d4-a716-446655440000.pdf"`
	CoverLetterPreview *string     `json:"cover_letter_preview,omitempty" example:"/storage/cover-letters/550e8400-e29b-41d4-a716-446655440001.pdf"`
	Status             string      `json:"status" example:"pending" enums:"pending,approved,rejected"`
	RejectionReason    *string     `json:"rejection_reason,omitempty" example:"Salary too low"`
	CreatedAt          time.Time   `json:"created_at" example:"2026-06-19T10:00:00Z"`
	ReviewedAt         *time.Time  `json:"reviewed_at,omitempty" example:"2026-06-20T14:30:00Z"`
}

// ApprovalListResponse is the API response for listing approval requests.
type ApprovalListResponse struct {
	Approvals []ApprovalResponse `json:"approvals"`
	Total     int64              `json:"total" example:"15"`
	Limit     int                `json:"limit" example:"50"`
	Offset    int                `json:"offset" example:"0"`
}

// ApprovePartialResponse is the API response when approval succeeds but
// task dispatch fails (207 Multi-Status). The approval is persisted;
// the submission task needs retry.
type ApprovePartialResponse struct {
	Status   string           `json:"status" example:"approved"`
	Warning  string           `json:"warning" example:"application submission queued for retry"`
	Approval ApprovalResponse `json:"approval"`
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

// ToResponse converts a domain ApprovalRequest to an API ApprovalResponse.
//
// This is a pure data copy — no transformation, no validation.
// The mapper exists to decouple the internal model from the API contract.
func ToResponse(a *ApprovalRequest) ApprovalResponse {
	return ApprovalResponse{
		ID:                 a.ID,
		ApplicationID:      a.ApplicationID,
		JobSnapshot:        a.JobSnapshot,
		ResumePreviewPath:  a.ResumePreviewPath,
		CoverLetterPreview: a.CoverLetterPreview,
		Status:             a.Status,
		RejectionReason:    a.RejectionReason,
		CreatedAt:          a.CreatedAt,
		ReviewedAt:         a.ReviewedAt,
	}
}
