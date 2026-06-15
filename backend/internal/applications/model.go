package applications

import (
	"time"

	"github.com/google/uuid"
)

// Status constants define the application pipeline.
const (
	StatusDraft       = "draft"
	StatusQueued      = "queued"
	StatusApplied     = "applied"
	StatusAssessment  = "assessment"
	StatusPhoneScreen = "phone_screen"
	StatusTechnical   = "technical"
	StatusFinal       = "final"
	StatusOffer       = "offer"
	StatusRejected    = "rejected"
)

// ApprovalTier constants match config/application.yaml tiers.
const (
	TierAuto   = "auto"
	TierReview = "review"
	TierReject = "reject"
)

// approvalTransitions defines valid status transitions.
// To add a new status: add the constant, add entries to this map.
// IsValidStatus is derived — no separate data structure needed.
var approvalTransitions = map[string][]string{
	StatusDraft:       {StatusQueued, StatusRejected},
	StatusQueued:      {StatusApplied, StatusRejected},
	StatusApplied:     {StatusAssessment, StatusPhoneScreen, StatusTechnical, StatusFinal, StatusOffer, StatusRejected},
	StatusAssessment:  {StatusPhoneScreen, StatusTechnical, StatusFinal, StatusOffer, StatusRejected},
	StatusPhoneScreen: {StatusTechnical, StatusFinal, StatusOffer, StatusRejected},
	StatusTechnical:   {StatusFinal, StatusOffer, StatusRejected},
	StatusFinal:       {StatusOffer, StatusRejected},
	StatusOffer:       {}, // terminal
	StatusRejected:    {}, // terminal
}

// IsValidStatus checks if a status is valid (derived from transitions).
func IsValidStatus(s string) bool {
	_, ok := approvalTransitions[s]
	return ok
}

// CanTransition checks if a status transition is valid.
func CanTransition(from, to string) bool {
	allowed, ok := approvalTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// TableName returns the table name for the Application model.
func (Application) TableName() string {
	return "applications"
}

// Application represents a job application.
type Application struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	JobID         uuid.UUID  `db:"job_id" json:"job_id"`
	ResumeID      *uuid.UUID `db:"resume_id" json:"resume_id,omitempty"`
	CoverLetterID *uuid.UUID `db:"cover_letter_id" json:"cover_letter_id,omitempty"`
	Status        string     `db:"status" json:"status"`
	ApprovalTier  string     `db:"approval_tier" json:"approval_tier"`
	AppliedAt     *time.Time `db:"applied_at" json:"applied_at,omitempty"`
	ResponseAt    *time.Time `db:"response_at" json:"response_at,omitempty"`
	InterviewAt   *time.Time `db:"interview_at" json:"interview_at,omitempty"`
	Notes         *string    `db:"notes" json:"notes,omitempty"`
	PortalType    *string    `db:"portal_type" json:"portal_type,omitempty"`
	PortalURL     *string    `db:"portal_url" json:"portal_url,omitempty"`
	FormData      []byte     `db:"form_data" json:"form_data,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// ApplicationEvent records every status transition for audit trail.
type ApplicationEvent struct {
	ID            uuid.UUID `db:"id" json:"id"`
	ApplicationID uuid.UUID `db:"application_id" json:"application_id"`
	OldStatus     string    `db:"old_status" json:"old_status"`
	NewStatus     string    `db:"new_status" json:"new_status"`
	Notes         string    `db:"notes" json:"notes"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// TableName returns the table name for the ApplicationEvent model.
func (ApplicationEvent) TableName() string {
	return "application_events"
}
