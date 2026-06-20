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
	ApplicationID  *uuid.UUID `json:"application_id,omitempty"`
	MessageID      string     `json:"message_id" binding:"required"`
	FromAddress    string     `json:"from_address" binding:"required"`
	ToAddress      *string    `json:"to_address,omitempty"`
	Subject        *string    `json:"subject,omitempty"`
	Body           *string    `json:"body,omitempty"`
	ReceivedAt     time.Time  `json:"received_at" binding:"required"`
	Classification *string    `json:"classification,omitempty"`
}

// UpdateEmailRequest is the payload for PATCH /emails/:id.
// Updates read status or reply draft.
type UpdateEmailRequest struct {
	IsRead     *bool   `json:"is_read,omitempty"`
	ReplyDraft *string `json:"reply_draft,omitempty"`
}

// ClassifyRequest is the payload for POST /emails/:id/classify.
// Empty body — triggers LLM re-classification.
// Uses empty struct since no request body is expected.
type ClassifyRequest struct{}

// ListFilterRequest is for GET /emails query params.
type ListFilterRequest struct {
	ApplicationID  string `form:"application_id"`
	Classification string `form:"classification"`
	Limit          int    `form:"limit"`
	Offset         int    `form:"offset"`
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
	ID             uuid.UUID  `json:"id"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty"`
	MessageID      string     `json:"message_id"`
	FromAddress    string     `json:"from_address"`
	ToAddress      *string    `json:"to_address,omitempty"`
	Subject        *string    `json:"subject,omitempty"`
	Body           *string    `json:"body,omitempty"`
	ReceivedAt     time.Time  `json:"received_at"`
	Classification *string    `json:"classification,omitempty"`
	IsRead         bool       `json:"is_read"`
	ReplyDraft     *string    `json:"reply_draft,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EmailListResponse is the response for GET /emails.
type EmailListResponse struct {
	Emails []EmailResponse `json:"emails"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ClassifyResponse is the response for POST /emails/:id/classify.
type ClassifyResponse struct {
	EmailID        uuid.UUID `json:"email_id"`
	Classification string    `json:"classification"`
	Confidence     float64   `json:"confidence"`
	Reasoning      string    `json:"reasoning"`
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
