package jobs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Job represents a job listing from a source.
type Job struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	SourceID        uuid.UUID       `db:"source_id" json:"source_id"`
	ExternalID      string          `db:"external_id" json:"external_id"`
	Title           string          `db:"title" json:"title"`
	Company         string          `db:"company" json:"company"`
	Location        string          `db:"location" json:"location"`
	RemoteType      string          `db:"remote_type" json:"remote_type"`
	SalaryMin       int             `db:"salary_min" json:"salary_min"`
	SalaryMax       int             `db:"salary_max" json:"salary_max"`
	SalaryCurrency  string          `db:"salary_currency" json:"salary_currency"`
	Description     string          `db:"description" json:"description"`
	Requirements    string          `db:"requirements" json:"requirements"`
	URL             string          `db:"url" json:"url"`
	CompanyURL      string          `db:"company_url" json:"company_url"`
	PostedAt        *time.Time      `db:"posted_at" json:"posted_at,omitempty"`
	ScrapedAt       time.Time       `db:"scraped_at" json:"scraped_at"`
	MatchScore      float64         `db:"match_score" json:"match_score"`
	MatchDetails    json.RawMessage `db:"match_details" json:"match_details,omitempty"`
	Status          string          `db:"status" json:"status"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`

	// Populated via JOIN for API responses
	SourceName string `db:"source_name" json:"source_name,omitempty"`
}

// Job status constants.
const (
	StatusDiscovered = "discovered"
	StatusMatched    = "matched"
	StatusApplied    = "applied"
	StatusArchived   = "archived"
)

// ValidStatuses returns all valid job statuses.
func ValidStatuses() []string {
	return []string{
		StatusDiscovered,
		StatusMatched,
		StatusApplied,
		StatusArchived,
	}
}

// IsValidStatus checks if a status is valid.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

// TableName returns the table name for the Job model.
func (Job) TableName() string {
	return "jobs"
}
