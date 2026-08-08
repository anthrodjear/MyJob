// Package emails handles email classification, storage, and retrieval.
//
// The emails domain stores incoming emails from the job search workflow
// and provides LLM-based classification to determine email intent
// (interview invite, rejection, offer, follow-up, spam, phishing, other).
//
// Schema: emails(id, application_id, message_id, from_address, to_address,
//
//	subject, body, received_at, classification, is_read, reply_draft, created_at)
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
	"strings"
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

// ClassificationMappings is a transition-window alias map used to normalise
// classifications emitted by external sources (browser-agent, IMAP, etc.)
// into the canonical vocabulary above. Aliases are case-folded before the
// lookup. To remove an alias after the migration window closes, simply
// delete the entry.
var ClassificationMappings = map[string]string{
	// browser-agent currently emits "interview"; canonical is "interview_invite"
	"interview":       ClassificationInterviewInvite,
	"INTERVIEW":       ClassificationInterviewInvite,
	"Interview":       ClassificationInterviewInvite,
	"interview_invite": ClassificationInterviewInvite,
	// any future aliases go here
}

// NormalizeClassification folds case and applies alias mappings so that
// both legacy ("interview") and canonical ("interview_invite") inputs
// resolve to the canonical classification value.
func NormalizeClassification(c string) string {
	lower := strings.ToLower(strings.TrimSpace(c))
	if mapped, ok := ClassificationMappings[lower]; ok {
		return mapped
	}
	if mapped, ok := ClassificationMappings[c]; ok {
		return mapped
	}
	return lower
}

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

// emailListColumns is the lighter subset used by the List query — excludes
// the heavy body column. Keep emailColumns for GetByID.
const emailListColumns = `
	id, application_id, message_id, from_address, to_address,
	subject, received_at, classification, is_read, reply_draft, created_at
`
