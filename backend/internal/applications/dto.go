package applications

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateApplicationRequest is the payload for POST /applications.
type CreateApplicationRequest struct {
	JobID         uuid.UUID  `json:"job_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ResumeID      *uuid.UUID `json:"resume_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	CoverLetterID *uuid.UUID `json:"cover_letter_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	PortalType    *string    `json:"portal_type,omitempty" example:"greenhouse"`
	PortalURL     *string    `json:"portal_url,omitempty" example:"https://boards.greenhouse.io/openai/jobs/12345"`
}

// UpdateStatusRequest is the payload for PUT /applications/:id/status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=draft queued applied assessment phone_screen technical final offer rejected" example:"applied" enums:"draft,queued,applied,assessment,phone_screen,technical,final,offer,rejected"`
	Notes  string `json:"notes" example:"Applied via Greenhouse portal"`
}

// UpdateApplicationNotesRequest is the payload for PATCH /applications/:id/notes.
type UpdateApplicationNotesRequest struct {
	Notes string `json:"notes" binding:"required" example:"Follow up with recruiter on Monday"`
}

// --- Response DTOs ---

// ApplicationResponse is the API response for a single application.
type ApplicationResponse struct {
	ID            uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	JobID         uuid.UUID  `json:"job_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ResumeID      *uuid.UUID `json:"resume_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	CoverLetterID *uuid.UUID `json:"cover_letter_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	Status        string     `json:"status" example:"applied" enums:"pending,approved,applied,interviewed,offered,rejected,archived"`
	ApprovalTier  string     `json:"approval_tier" example:"AUTO" enums:"AUTO,REVIEW,REJECT"`
	AppliedAt     *time.Time `json:"applied_at,omitempty" example:"2026-01-20T10:00:00Z"`
	ResponseAt    *time.Time `json:"response_at,omitempty" example:"2026-01-22T14:30:00Z"`
	InterviewAt   *time.Time `json:"interview_at,omitempty" example:"2026-01-25T10:00:00Z"`
	Notes         *string    `json:"notes,omitempty" example:"Applied via Greenhouse portal"`
	PortalType    *string    `json:"portal_type,omitempty" example:"greenhouse"`
	PortalURL     *string    `json:"portal_url,omitempty" example:"https://boards.greenhouse.io/openai/jobs/12345"`
	CreatedAt     time.Time  `json:"created_at" example:"2026-01-15T08:00:00Z"`
	UpdatedAt     time.Time  `json:"updated_at" example:"2026-01-20T10:00:00Z"`
}

// ApplicationListResponse is the API response for listing applications.
type ApplicationListResponse struct {
	Applications []ApplicationResponse `json:"applications"`
	Total        int64                 `json:"total" example:"42"`
	Limit        int                   `json:"limit" example:"20"`
	Offset       int                   `json:"offset" example:"0"`
}

// ApplicationStatsResponse is the API response for GET /applications/stats.
type ApplicationStatsResponse struct {
	Total    int64            `json:"total" example:"42"`
	ByStatus map[string]int64 `json:"by_status"`
	ByTier   map[string]int64 `json:"by_tier"`
}

// ApplicationEventResponse is the API response for a single audit event.
type ApplicationEventResponse struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440004"`
	OldStatus string    `json:"old_status" example:"pending"`
	NewStatus string    `json:"new_status" example:"applied"`
	Notes     string    `json:"notes" example:"Applied via Greenhouse portal"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-20T10:00:00Z"`
}

// ApplicationTimelineResponse is the API response for GET /applications/:id/events.
type ApplicationTimelineResponse struct {
	ApplicationID uuid.UUID                  `json:"application_id" example:"550e8400-e29b-41d4-a716-446655440000"`
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
