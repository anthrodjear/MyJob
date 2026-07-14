// DTOs (Data Transfer Types) for the emails domain.
//
// API surface:
//   - GET    /emails                  → List emails with filters
//   - GET    /emails/:id              → Get single email
//   - POST   /emails                  → Store incoming email
//   - PATCH  /emails/:id              → Update read/draft status
//   - POST   /emails/:id/classify     → Re-classify email via LLM
//
// The emails domain stores incoming emails and provides classification.
// The worker stores emails after fetching from the browser agent.
// The API provides read access and manual re-classification.
package emails

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidApplicationID indicates the application_id query parameter is not a valid UUID.
	ErrInvalidApplicationID = errors.New("invalid application_id")
)

// Request DTOs

// StoreEmailRequest is the payload for POST /emails.
// Used by the worker to store incoming emails from the browser agent.
type StoreEmailRequest struct {
	ApplicationID  *uuid.UUID `json:"application_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageID      string     `json:"message_id" binding:"required" example:"<abc123@company.com>"`
	FromAddress    string     `json:"from_address" binding:"required" example:"recruiter@company.com"`
	ToAddress      *string    `json:"to_address,omitempty" example:"candidate@example.com"`
	Subject        *string    `json:"subject,omitempty" example:"Interview Invitation - Senior Go Engineer"`
	Body           *string    `json:"body,omitempty" example:"Hi John, we'd like to invite you..."`
	ReceivedAt     time.Time  `json:"received_at" binding:"required" example:"2026-06-20T10:00:00Z"`
	Classification *string    `json:"classification,omitempty" example:"interview_invitation" enums:"interview_invitation,rejection,offer,recruiter_outreach,other"`
}

// UpdateEmailRequest is the payload for PATCH /emails/:id.
// Updates read status or reply draft.
type UpdateEmailRequest struct {
	IsRead     *bool   `json:"is_read,omitempty" example:"true"`
	ReplyDraft *string `json:"reply_draft,omitempty" example:"Thank you for the invitation. I'd love to schedule..."`
}

// ClassifyRequest is the payload for POST /emails/:id/classify.
// Empty body — triggers LLM re-classification.
// Uses empty struct since no request body is expected.
type ClassifyRequest struct{}

// ListFilterRequest is for GET /emails query params.
type ListFilterRequest struct {
	ApplicationID  string `form:"application_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Classification string `form:"classification" example:"interview_invitation" enums:"interview_invitation,rejection,offer,recruiter_outreach,other"`
	Limit          int    `form:"limit" example:"50" default:"50" minimum:"1" maximum:"100"`
	Offset         int    `form:"offset" example:"0" minimum:"0"`
}

// ToListFilter converts the request to a domain ListFilter.
// Returns ErrInvalidApplicationID if application_id is not a valid UUID.
func (r *ListFilterRequest) ToListFilter() (ListFilter, error) {
	var appID uuid.UUID
	if r.ApplicationID != "" {
		var err error
		appID, err = uuid.Parse(r.ApplicationID)
		if err != nil {
			return ListFilter{}, ErrInvalidApplicationID
		}
	}
	return ListFilter{
		ApplicationID:  appID,
		Classification: r.Classification,
		Limit:          r.Limit,
		Offset:         r.Offset,
	}, nil
}

// Response DTOs

// EmailResponse is the API response for a single email.
type EmailResponse struct {
	ID             uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	MessageID      string     `json:"message_id" example:"<abc123@company.com>"`
	FromAddress    string     `json:"from_address" example:"recruiter@company.com"`
	ToAddress      *string    `json:"to_address,omitempty" example:"candidate@example.com"`
	Subject        *string    `json:"subject,omitempty" example:"Interview Invitation - Senior Go Engineer"`
	Body           *string    `json:"body,omitempty" example:"Hi John, we'd like to invite you..."`
	ReceivedAt     time.Time  `json:"received_at" example:"2026-06-20T10:00:00Z"`
	Classification *string    `json:"classification,omitempty" example:"interview_invitation"`
	IsRead         bool       `json:"is_read" example:"false"`
	ReplyDraft     *string    `json:"reply_draft,omitempty" example:"Thank you for the invitation..."`
	CreatedAt      time.Time  `json:"created_at" example:"2026-06-20T10:05:00Z"`
}

// EmailListResponse is the response for GET /emails.
type EmailListResponse struct {
	Emails []EmailResponse `json:"emails"`
	Total  int64           `json:"total" example:"25"`
	Limit  int             `json:"limit" example:"50"`
	Offset int             `json:"offset" example:"0"`
}

// ClassifyResponse is the response for POST /emails/:id/classify.
type ClassifyResponse struct {
	EmailID        uuid.UUID `json:"email_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Classification string    `json:"classification" example:"interview_invitation"`
	Confidence     float64   `json:"confidence" example:"0.95" minimum:"0" maximum:"1"`
	Reasoning      string    `json:"reasoning" example:"Email contains interview scheduling language and calendar invite"`
}

// Mappers

// ToEmailResponse converts a domain Email to an API EmailResponse.
func ToEmailResponse(e *Email) EmailResponse {
	return EmailResponse{
		ID:             e.ID,
		ApplicationID:  e.ApplicationID,
		MessageID:      e.MessageID,
		FromAddress:    e.FromAddress,
		ToAddress:      e.ToAddress,
		Subject:        e.Subject,
		Body:           e.Body,
		ReceivedAt:     e.ReceivedAt,
		Classification: e.Classification,
		IsRead:         e.IsRead,
		ReplyDraft:     e.ReplyDraft,
		CreatedAt:      e.CreatedAt,
	}
}
