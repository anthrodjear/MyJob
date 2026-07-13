package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJobStatusConstants(t *testing.T) {
	assert.Equal(t, "discovered", StatusDiscovered)
	assert.Equal(t, "matched", StatusMatched)
	assert.Equal(t, "applied", StatusApplied)
	assert.Equal(t, "archived", StatusArchived)
}

func TestValidStatuses(t *testing.T) {
	statuses := ValidStatuses()
	assert.Len(t, statuses, 4)
	assert.Contains(t, statuses, StatusDiscovered)
	assert.Contains(t, statuses, StatusMatched)
	assert.Contains(t, statuses, StatusApplied)
	assert.Contains(t, statuses, StatusArchived)
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{StatusDiscovered, true},
		{StatusMatched, true},
		{StatusApplied, true},
		{StatusArchived, true},
		{"unknown", false},
		{"", false},
		{"DISCOVERED", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidStatus(tt.status))
		})
	}
}

func TestJob_TableName(t *testing.T) {
	j := Job{}
	assert.Equal(t, "jobs", j.TableName())
}

func TestJob_Fields(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	postedAt := time.Now().Add(-24 * time.Hour)
	scrapedAt := time.Now()
	details := json.RawMessage(`{"skills":["Go","React"]}`)

	job := Job{
		ID:             id,
		SourceID:       sourceID,
		ExternalID:     "ext-123",
		Title:          "Software Engineer",
		Company:        "Acme Corp",
		Location:       "Remote",
		RemoteType:     "remote",
		SalaryMin:      100000,
		SalaryMax:      150000,
		SalaryCurrency: "USD",
		Description:    "Great job",
		Requirements:   "Go, React",
		URL:            "https://example.com/job/123",
		ApplicationURL: "https://example.com/apply/123",
		CompanyURL:     "https://example.com",
		Source:         "greenhouse",
		PostedAt:       &postedAt,
		ScrapedAt:      scrapedAt,
		MatchScore:     92.5,
		MatchDetails:   details,
		Status:         StatusDiscovered,
		SourceName:     "Greenhouse",
		CreatedAt:      scrapedAt,
		UpdatedAt:      scrapedAt,
	}

	assert.Equal(t, id, job.ID)
	assert.Equal(t, sourceID, job.SourceID)
	assert.Equal(t, "ext-123", job.ExternalID)
	assert.Equal(t, "Software Engineer", job.Title)
	assert.Equal(t, "Acme Corp", job.Company)
	assert.Equal(t, "Remote", job.Location)
	assert.Equal(t, "remote", job.RemoteType)
	assert.Equal(t, 100000, job.SalaryMin)
	assert.Equal(t, 150000, job.SalaryMax)
	assert.Equal(t, "USD", job.SalaryCurrency)
	assert.Equal(t, "Great job", job.Description)
	assert.Equal(t, "Go, React", job.Requirements)
	assert.Equal(t, "https://example.com/job/123", job.URL)
	assert.Equal(t, "https://example.com/apply/123", job.ApplicationURL)
	assert.Equal(t, "https://example.com", job.CompanyURL)
	assert.Equal(t, "greenhouse", job.Source)
	assert.Equal(t, &postedAt, job.PostedAt)
	assert.Equal(t, scrapedAt, job.ScrapedAt)
	assert.Equal(t, 92.5, job.MatchScore)
	assert.NotNil(t, job.MatchDetails)
	assert.Equal(t, StatusDiscovered, job.Status)
	assert.Equal(t, "Greenhouse", job.SourceName)
	assert.Equal(t, scrapedAt, job.CreatedAt)
	assert.Equal(t, scrapedAt, job.UpdatedAt)
}

func TestJob_NilPostedAt(t *testing.T) {
	job := Job{}
	assert.Nil(t, job.PostedAt)
	assert.Nil(t, job.MatchDetails)
}

func TestJob_MatchDetailsJSON(t *testing.T) {
	details := json.RawMessage(`{"skill_match":85,"location_match":100}`)
	job := Job{MatchDetails: details}

	var parsed map[string]float64
	err := json.Unmarshal(job.MatchDetails, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, 85.0, parsed["skill_match"])
	assert.Equal(t, 100.0, parsed["location_match"])
}
