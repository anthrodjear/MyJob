package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobInput(t *testing.T) {
	postedAt := time.Now().Add(-48 * time.Hour)
	sourceID := uuid.New()

	input := CreateJobInput{
		SourceID:       sourceID,
		ExternalID:     "ext-456",
		Title:          "Backend Engineer",
		Company:        "Tech Corp",
		Location:       "New York",
		RemoteType:     "hybrid",
		SalaryMin:      120000,
		SalaryMax:      180000,
		SalaryCurrency: "USD",
		Description:    "Build APIs",
		Requirements:   "Go, PostgreSQL",
		URL:            "https://example.com/job/456",
		ApplicationURL: "https://example.com/apply/456",
		CompanyURL:     "https://techcorp.com",
		Source:         "lever",
		PostedAt:       &postedAt,
	}

	assert.Equal(t, sourceID, input.SourceID)
	assert.Equal(t, "ext-456", input.ExternalID)
	assert.Equal(t, "Backend Engineer", input.Title)
	assert.Equal(t, "Tech Corp", input.Company)
	assert.Equal(t, "New York", input.Location)
	assert.Equal(t, "hybrid", input.RemoteType)
	assert.Equal(t, 120000, input.SalaryMin)
	assert.Equal(t, 180000, input.SalaryMax)
	assert.Equal(t, "USD", input.SalaryCurrency)
	assert.Equal(t, "Build APIs", input.Description)
	assert.Equal(t, "Go, PostgreSQL", input.Requirements)
	assert.Equal(t, "https://example.com/job/456", input.URL)
	assert.Equal(t, "https://example.com/apply/456", input.ApplicationURL)
	assert.Equal(t, "https://techcorp.com", input.CompanyURL)
	assert.Equal(t, "lever", input.Source)
	assert.Equal(t, &postedAt, input.PostedAt)
}

func TestCreateJobInput_NilPostedAt(t *testing.T) {
	input := CreateJobInput{}
	assert.Nil(t, input.PostedAt)
}

func TestJobResponse(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	details := json.RawMessage(`{"score":90}`)

	resp := JobResponse{
		ID:           id,
		SourceID:     uuid.New(),
		Title:        "Engineer",
		Company:      "Acme",
		Location:     "Remote",
		SalaryMin:    100000,
		SalaryMax:    150000,
		MatchScore:   90.5,
		MatchDetails: details,
		Status:       StatusDiscovered,
		ScrapedAt:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "Engineer", resp.Title)
	assert.Equal(t, 90.5, resp.MatchScore)
	assert.NotNil(t, resp.MatchDetails)
	assert.Equal(t, StatusDiscovered, resp.Status)
}

func TestJobListResponse(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		resp := JobListResponse{
			Jobs:   []JobResponse{},
			Total:  0,
			Limit:  20,
			Offset: 0,
		}
		assert.Empty(t, resp.Jobs)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("with items", func(t *testing.T) {
		resp := JobListResponse{
			Jobs: []JobResponse{
				{ID: uuid.New(), Title: "Job 1"},
				{ID: uuid.New(), Title: "Job 2"},
			},
			Total:  2,
			Limit:  10,
			Offset: 0,
		}
		assert.Len(t, resp.Jobs, 2)
		assert.Equal(t, 2, resp.Total)
	})
}

func TestUpdateJobRequest(t *testing.T) {
	t.Run("with status", func(t *testing.T) {
		req := UpdateJobRequest{Status: StatusMatched}
		assert.Equal(t, StatusMatched, req.Status)
	})

	t.Run("empty status", func(t *testing.T) {
		req := UpdateJobRequest{}
		assert.Empty(t, req.Status)
	})
}

func TestBulkImportResult(t *testing.T) {
	res := BulkImportResult{
		Imported: 10,
		Skipped:  2,
	}
	assert.Equal(t, 10, res.Imported)
	assert.Equal(t, 2, res.Skipped)

	zero := BulkImportResult{}
	assert.Equal(t, 0, zero.Imported)
	assert.Equal(t, 0, zero.Skipped)
}

func TestJobToResponse(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	details := json.RawMessage(`{"score":85}`)

	job := &Job{
		ID:           id,
		Title:        "Engineer",
		Company:      "Acme",
		MatchScore:   85.0,
		MatchDetails: details,
		Status:       StatusDiscovered,
		ScrapedAt:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := ToResponse(job)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "Engineer", resp.Title)
	assert.Equal(t, "Acme", resp.Company)
	assert.Equal(t, 85.0, resp.MatchScore)
	assert.Equal(t, StatusDiscovered, resp.Status)
	assert.Equal(t, now, resp.ScrapedAt)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestJobToResponse_NilPointers(t *testing.T) {
	job := &Job{ID: uuid.New(), Title: "Test"}
	resp := ToResponse(job)
	assert.Nil(t, resp.PostedAt)
	assert.Nil(t, resp.MatchDetails)
}
