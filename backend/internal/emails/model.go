// Package emails handles email classification, storage, and retrieval.
//
// The emails domain stores incoming emails from the job search workflow
// and provides LLM-based classification to determine email intent
// (interview invite, rejection, offer, follow-up, spam, phishing, other).
//
// Schema: emails(id, application_id, message_id, from_address, to_address,
//   subject, body, received_at, classification, is_read, reply_draft, created_at)
//
// Flow:
//  1. Worker receives email_check task
//  2. Worker calls browser-agent to fetch emails from Outlook/IMAP
//  3. Browser-agent returns classified emails (or emails to classify)
//  4. Worker stores emails via emails.Service.Store()
//  5. Worker updates application status based on classification
//
// The classifier can also be called directly from the API for
// manual classification or re-classification of existing emails.
package emails

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain Errors

// ErrNotFound indicates the email does not exist.
var ErrNotFound = errors.New("email not found")

// ErrInvalidClassification indicates a classification value is not recognized.
var ErrInvalidClassification = errors.New("invalid classification")

// Classification Constants

const (
	ClassificationInterviewInvite = "interview_invite"
	ClassificationRejection       = "rejection"
	ClassificationOffer           = "offer"
	ClassificationFollowUp        = "follow_up"
	ClassificationSpam            = "spam"
	ClassificationPhishing        = "phishing"
	ClassificationOther           = "other"
)

// validClassifications is the set of known classification values.
var validClassifications = map[string]bool{
	ClassificationInterviewInvite: true,
	ClassificationRejection:       true,
	ClassificationOffer:           true,
	ClassificationFollowUp:        true,
	ClassificationSpam:            true,
	ClassificationPhishing:        true,
	ClassificationOther:           true,
}

// IsValidClassification returns true if the classification is a known value.
func IsValidClassification(c string) bool {
	return validClassifications[c]
}

// Database Row Model

// Email represents a stored email with classification metadata.
// Schema: emails(id, application_id, message_id, from_address, to_address,
//
//	subject, body, received_at, classification, is_read, reply_draft, created_at)
type Email struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	ApplicationID  *uuid.UUID `db:"application_id" json:"application_id,omitempty"`
	MessageID      string     `db:"message_id" json:"message_id"`
	FromAddress    string     `db:"from_address" json:"from_address"`
	ToAddress      *string    `db:"to_address" json:"to_address,omitempty"`
	Subject        *string    `db:"subject" json:"subject,omitempty"`
	Body           *string    `db:"body" json:"body,omitempty"`
	ReceivedAt     time.Time  `db:"received_at" json:"received_at"`
	Classification *string    `db:"classification" json:"classification,omitempty"`
	IsRead         bool       `db:"is_read" json:"is_read"`
	ReplyDraft     *string    `db:"reply_draft" json:"reply_draft,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// Column List

const emailColumns = `
	id, application_id, message_id, from_address, to_address,
	subject, body, received_at, classification, is_read, reply_draft, created_at
`
