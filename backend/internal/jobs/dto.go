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
	ApplicationURL string     `json:"application_url"`
	CompanyURL     string     `json:"company_url"`
	Source         string     `json:"source"`
	PostedAt       *time.Time `json:"posted_at,omitempty"`
}

// JobResponse is the public API representation of a job.
type JobResponse struct {
	ID               uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceID         uuid.UUID       `json:"source_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	SourceName       string          `json:"source_name" example:"Greenhouse - OpenAI"`
	ExternalID       string          `json:"external_id" example:"gh-12345"`
	Title            string          `json:"title" example:"Senior Go Engineer"`
	Company          string          `json:"company" example:"OpenAI"`
	Location         string          `json:"location" example:"San Francisco, CA (Remote)"`
	RemoteType       string          `json:"remote_type" example:"remote" enums:"remote,hybrid,onsite"`
	SalaryMin        int             `json:"salary_min" example:"150000"`
	SalaryMax        int             `json:"salary_max" example:"200000"`
	SalaryCurrency   string          `json:"salary_currency" example:"USD"`
	Description      string          `json:"description" example:"We are looking for a Senior Go Engineer..."`
	Requirements     string          `json:"requirements" example:"5+ years Go experience, Kubernetes, PostgreSQL"`
	URL              string          `json:"url" example:"https://boards.greenhouse.io/openai/jobs/12345"`
	ApplicationURL   string          `json:"application_url" example:"https://boards.greenhouse.io/openai/jobs/12345"`
	CompanyURL       string          `json:"company_url" example:"https://openai.com"`
	Source           string          `json:"source" example:"greenhouse"`
	PostedAt         *time.Time      `json:"posted_at,omitempty" example:"2026-01-10T10:00:00Z"`
	ScrapedAt        time.Time       `json:"scraped_at" example:"2026-01-15T08:00:00Z"`
	MatchScore       float64         `json:"match_score" example:"92.5"`
	MatchDetails     json.RawMessage `json:"match_details,omitempty" swaggertype:"object"`
	ScoreTier        *string         `json:"score_tier,omitempty" example:"AUTO"`
	ScoredAt         *time.Time      `json:"scored_at,omitempty" example:"2026-01-15T09:00:00Z"`
	ScoringReasoning *string         `json:"scoring_reasoning,omitempty" example:"Strong match for Go and Kubernetes skills"`
	ScoringModel     *string         `json:"scoring_model,omitempty" example:"gpt-4o"`
	ScoringSource    *string         `json:"scoring_source,omitempty" example:"llm"`
	Status           string          `json:"status" example:"matched" enums:"discovered,matched,applied,archived"`
	CreatedAt        time.Time       `json:"created_at" example:"2026-01-15T08:00:00Z"`
	UpdatedAt        time.Time       `json:"updated_at" example:"2026-01-15T08:00:00Z"`
}

// JobListResponse wraps paginated job results.
type JobListResponse struct {
	Jobs   []JobResponse `json:"jobs"`
	Total  int           `json:"total" example:"42"`
	Limit  int           `json:"limit" example:"20"`
	Offset int           `json:"offset" example:"0"`
}

// UpdateJobRequest is the payload for PATCH /jobs/:id.
type UpdateJobRequest struct {
	Status string `json:"status" binding:"omitempty,oneof=discovered matched applied archived" example:"applied" enums:"discovered,matched,applied,archived"`
}

// BulkImportResult tracks the outcome of a bulk import operation.
type BulkImportResult struct {
	Imported int `json:"imported" example:"10"`
	Skipped  int `json:"skipped" example:"2"`
}

// ToResponse converts a Job model to a JobResponse DTO.
func ToResponse(j *Job) JobResponse {
	return JobResponse{
		ID:               j.ID,
		SourceID:         j.SourceID,
		SourceName:       j.SourceName,
		ExternalID:       j.ExternalID,
		Title:            j.Title,
		Company:          j.Company,
		Location:         j.Location,
		RemoteType:       j.RemoteType,
		SalaryMin:        j.SalaryMin,
		SalaryMax:        j.SalaryMax,
		SalaryCurrency:   j.SalaryCurrency,
		Description:      j.Description,
		Requirements:     j.Requirements,
		URL:              j.URL,
		ApplicationURL:   j.ApplicationURL,
		CompanyURL:       j.CompanyURL,
		Source:           j.Source,
		PostedAt:         j.PostedAt,
		ScrapedAt:        j.ScrapedAt,
		MatchScore:       j.MatchScore,
		MatchDetails:     j.MatchDetails,
		ScoreTier:        j.ScoreTier,
		ScoredAt:         j.ScoredAt,
		ScoringReasoning: j.ScoringReasoning,
		ScoringModel:     j.ScoringModel,
		ScoringSource:    j.ScoringSource,
		Status:           j.Status,
		CreatedAt:        j.CreatedAt,
		UpdatedAt:        j.UpdatedAt,
	}
}
