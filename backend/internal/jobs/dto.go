package jobs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CreateJobInput is the internal DTO used by scrapers/workers to create jobs.
// Not exposed via public API.
type CreateJobInput struct {
	SourceID       uuid.UUID  `json:"source_id"`
	ExternalID     string     `json:"external_id"`
	Title          string     `json:"title"`
	Company        string     `json:"company"`
	Location       string     `json:"location"`
	RemoteType     string     `json:"remote_type"`
	SalaryMin      int        `json:"salary_min"`
	SalaryMax      int        `json:"salary_max"`
	SalaryCurrency string     `json:"salary_currency"`
	Description    string     `json:"description"`
	Requirements   string     `json:"requirements"`
	URL            string     `json:"url"`
	CompanyURL     string     `json:"company_url"`
	PostedAt       *time.Time `json:"posted_at,omitempty"`
}

// JobResponse is the public API representation of a job.
type JobResponse struct {
	ID              uuid.UUID       `json:"id"`
	SourceID        uuid.UUID       `json:"source_id"`
	SourceName      string          `json:"source_name"`
	ExternalID      string          `json:"external_id"`
	Title           string          `json:"title"`
	Company         string          `json:"company"`
	Location        string          `json:"location"`
	RemoteType      string          `json:"remote_type"`
	SalaryMin       int             `json:"salary_min"`
	SalaryMax       int             `json:"salary_max"`
	SalaryCurrency  string          `json:"salary_currency"`
	Description     string          `json:"description"`
	Requirements    string          `json:"requirements"`
	URL             string          `json:"url"`
	CompanyURL      string          `json:"company_url"`
	PostedAt        *time.Time      `json:"posted_at,omitempty"`
	ScrapedAt       time.Time       `json:"scraped_at"`
	MatchScore      float64         `json:"match_score"`
	MatchDetails    json.RawMessage `json:"match_details,omitempty"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// JobListResponse wraps paginated job results.
type JobListResponse struct {
	Jobs  []JobResponse `json:"jobs"`
	Total int           `json:"total"`
	Limit int           `json:"limit"`
	Offset int          `json:"offset"`
}

// UpdateJobRequest is the payload for PATCH /jobs/:id.
type UpdateJobRequest struct {
	Status string `json:"status" binding:"omitempty,oneof=discovered matched applied archived"`
}

// BulkImportResult tracks the outcome of a bulk import operation.
type BulkImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

// ToResponse converts a Job model to a JobResponse DTO.
func ToResponse(j *Job) JobResponse {
	return JobResponse{
		ID:             j.ID,
		SourceID:       j.SourceID,
		SourceName:     j.SourceName,
		ExternalID:     j.ExternalID,
		Title:          j.Title,
		Company:        j.Company,
		Location:       j.Location,
		RemoteType:     j.RemoteType,
		SalaryMin:      j.SalaryMin,
		SalaryMax:      j.SalaryMax,
		SalaryCurrency: j.SalaryCurrency,
		Description:    j.Description,
		Requirements:   j.Requirements,
		URL:            j.URL,
		CompanyURL:     j.CompanyURL,
		PostedAt:       j.PostedAt,
		ScrapedAt:      j.ScrapedAt,
		MatchScore:     j.MatchScore,
		MatchDetails:   j.MatchDetails,
		Status:         j.Status,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
}
