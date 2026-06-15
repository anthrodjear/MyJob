package applications

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateApplicationRequest is the payload for POST /applications.
type CreateApplicationRequest struct {
	JobID         uuid.UUID  `json:"job_id" binding:"required"`
	ResumeID      *uuid.UUID `json:"resume_id"`
	CoverLetterID *uuid.UUID `json:"cover_letter_id"`
	PortalType    *string    `json:"portal_type"`
	PortalURL     *string    `json:"portal_url"`
}

// UpdateStatusRequest is the payload for PUT /applications/:id/status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Notes  string `json:"notes"`
}

// UpdateApplicationNotesRequest is the payload for PATCH /applications/:id/notes.
type UpdateApplicationNotesRequest struct {
	Notes string `json:"notes" binding:"required"`
}

// --- Response DTOs ---

// ApplicationResponse is the API response for a single application.
type ApplicationResponse struct {
	ID            uuid.UUID  `json:"id"`
	JobID         uuid.UUID  `json:"job_id"`
	ResumeID      *uuid.UUID `json:"resume_id,omitempty"`
	CoverLetterID *uuid.UUID `json:"cover_letter_id,omitempty"`
	Status        string     `json:"status"`
	ApprovalTier  string     `json:"approval_tier"`
	AppliedAt     *time.Time `json:"applied_at,omitempty"`
	ResponseAt    *time.Time `json:"response_at,omitempty"`
	InterviewAt   *time.Time `json:"interview_at,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	PortalType    *string    `json:"portal_type,omitempty"`
	PortalURL     *string    `json:"portal_url,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ApplicationListResponse is the API response for listing applications.
type ApplicationListResponse struct {
	Applications []ApplicationResponse `json:"applications"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}

// ApplicationStatsResponse is the API response for GET /applications/stats.
type ApplicationStatsResponse struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
	ByTier   map[string]int64 `json:"by_tier"`
}

// ApplicationEventResponse is the API response for a single audit event.
type ApplicationEventResponse struct {
	ID        uuid.UUID `json:"id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// ApplicationTimelineResponse is the API response for GET /applications/:id/events.
type ApplicationTimelineResponse struct {
	ApplicationID uuid.UUID                 `json:"application_id"`
	Events        []ApplicationEventResponse `json:"events"`
}

// --- Mappers ---

// ToResponse converts a domain model to an API response.
func ToResponse(a *Application) ApplicationResponse {
	return ApplicationResponse{
		ID:            a.ID,
		JobID:         a.JobID,
		ResumeID:      a.ResumeID,
		CoverLetterID: a.CoverLetterID,
		Status:        a.Status,
		ApprovalTier:  a.ApprovalTier,
		AppliedAt:     a.AppliedAt,
		ResponseAt:    a.ResponseAt,
		InterviewAt:   a.InterviewAt,
		Notes:         a.Notes,
		PortalType:    a.PortalType,
		PortalURL:     a.PortalURL,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// ToEventResponse converts an audit event to an API response.
func ToEventResponse(e *ApplicationEvent) ApplicationEventResponse {
	return ApplicationEventResponse{
		ID:        e.ID,
		OldStatus: e.OldStatus,
		NewStatus: e.NewStatus,
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt,
	}
}
