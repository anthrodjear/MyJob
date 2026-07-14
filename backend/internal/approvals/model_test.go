package approvals

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", ApprovalStatusPending)
	assert.Equal(t, "approved", ApprovalStatusApproved)
	assert.Equal(t, "rejected", ApprovalStatusRejected)
}

func TestApprovalIsValidStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{ApprovalStatusPending, true},
		{ApprovalStatusApproved, true},
		{ApprovalStatusRejected, true},
		{"unknown", false},
		{"", false},
		{"PENDING", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidStatus(tt.status))
		})
	}
}

func TestApprovalCanTransition(t *testing.T) {
	tests := []struct {
		name  string
		from  string
		to    string
		valid bool
	}{
		{"pending to approved", ApprovalStatusPending, ApprovalStatusApproved, true},
		{"pending to rejected", ApprovalStatusPending, ApprovalStatusRejected, true},
		{"approved to pending", ApprovalStatusApproved, ApprovalStatusPending, false},
		{"approved to rejected", ApprovalStatusApproved, ApprovalStatusRejected, false},
		{"rejected to pending", ApprovalStatusRejected, ApprovalStatusPending, false},
		{"rejected to approved", ApprovalStatusRejected, ApprovalStatusApproved, false},
		{"unknown status", "unknown", ApprovalStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, CanTransition(tt.from, tt.to))
		})
	}
}

func TestDomainErrors(t *testing.T) {
	assert.Error(t, ErrNotFound)
	assert.Error(t, ErrInvalidStatus)
	assert.Error(t, ErrReasonRequired)
	assert.Contains(t, ErrNotFound.Error(), "approval request not found")
	assert.Contains(t, ErrInvalidStatus.Error(), "invalid approval status transition")
	assert.Contains(t, ErrReasonRequired.Error(), "approval rejection reason is required")
}

func TestApprovalRequest_Fields(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()
	resumePath := "resumes/test.pdf"
	coverPreview := "Cover letter preview text"
	rejectionReason := "Salary too low"

	req := ApprovalRequest{
		ID:            id,
		ApplicationID: appID,
		JobSnapshot: JobSnapshot{
			Title:       "Software Engineer",
			Company:     "Acme Corp",
			Location:    "Remote",
			URL:         "https://example.com/job/123",
			Description: "Job description",
			Score:       85.5,
			Tier:        "review",
			ScoredAt:    now.Format(time.RFC3339),
		},
		ResumePreviewPath:  &resumePath,
		CoverLetterPreview: &coverPreview,
		Status:             ApprovalStatusPending,
		RejectionReason:    &rejectionReason,
		CreatedAt:          now,
		ReviewedAt:         &now,
	}

	assert.Equal(t, id, req.ID)
	assert.Equal(t, appID, req.ApplicationID)
	assert.Equal(t, "Software Engineer", req.JobSnapshot.Title)
	assert.Equal(t, "Acme Corp", req.JobSnapshot.Company)
	assert.Equal(t, &resumePath, req.ResumePreviewPath)
	assert.Equal(t, &coverPreview, req.CoverLetterPreview)
	assert.Equal(t, ApprovalStatusPending, req.Status)
	assert.Equal(t, &rejectionReason, req.RejectionReason)
	assert.Equal(t, now, req.CreatedAt)
	assert.Equal(t, &now, req.ReviewedAt)
}

func TestApprovalRequest_NilPointers(t *testing.T) {
	req := ApprovalRequest{}
	assert.Nil(t, req.ResumePreviewPath)
	assert.Nil(t, req.CoverLetterPreview)
	assert.Nil(t, req.RejectionReason)
	assert.Nil(t, req.ReviewedAt)
}

func TestApprovalRequest_TransitionTo(t *testing.T) {
	t.Run("pending to approved", func(t *testing.T) {
		req := ApprovalRequest{Status: ApprovalStatusPending}
		err := req.TransitionTo(ApprovalStatusApproved)
		assert.NoError(t, err)
		assert.Equal(t, ApprovalStatusApproved, req.Status)
		assert.NotNil(t, req.ReviewedAt)
	})

	t.Run("pending to rejected", func(t *testing.T) {
		req := ApprovalRequest{Status: ApprovalStatusPending}
		before := time.Now()
		err := req.TransitionTo(ApprovalStatusRejected)
		assert.NoError(t, err)
		assert.Equal(t, ApprovalStatusRejected, req.Status)
		assert.NotNil(t, req.ReviewedAt)
		assert.False(t, req.ReviewedAt.Before(before))
	})

	t.Run("invalid transition", func(t *testing.T) {
		req := ApprovalRequest{Status: ApprovalStatusApproved}
		err := req.TransitionTo(ApprovalStatusPending)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidStatus)
		assert.Contains(t, err.Error(), "approved → pending")
		assert.Equal(t, ApprovalStatusApproved, req.Status)
	})

	t.Run("terminal state cannot transition", func(t *testing.T) {
		req := ApprovalRequest{Status: ApprovalStatusRejected}
		err := req.TransitionTo(ApprovalStatusApproved)
		assert.Error(t, err)
	})
}

func TestApprovalRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Title:   "Software Engineer",
				Company: "Acme Corp",
				Score:   85.5,
				Tier:    "review",
			},
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("missing application_id", func(t *testing.T) {
		req := ApprovalRequest{
			JobSnapshot: JobSnapshot{
				Title:   "Software Engineer",
				Company: "Acme Corp",
				Score:   85.5,
				Tier:    "review",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "application_id is required")
	})

	t.Run("missing title", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Company: "Acme Corp",
				Score:   85.5,
				Tier:    "review",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("missing company", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Title: "Software Engineer",
				Score: 85.5,
				Tier:  "review",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "company is required")
	})

	t.Run("score too low", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Title:   "Software Engineer",
				Company: "Acme Corp",
				Score:   -1,
				Tier:    "review",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be 0-100")
	})

	t.Run("score too high", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Title:   "Software Engineer",
				Company: "Acme Corp",
				Score:   101,
				Tier:    "review",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be 0-100")
	})

	t.Run("not review tier", func(t *testing.T) {
		req := ApprovalRequest{
			ApplicationID: uuid.New(),
			JobSnapshot: JobSnapshot{
				Title:   "Software Engineer",
				Company: "Acme Corp",
				Score:   85.5,
				Tier:    "auto",
			},
		}
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only review-tier")
	})
}

func TestJobSnapshot_Value(t *testing.T) {
	js := JobSnapshot{
		Title:   "Software Engineer",
		Company: "Acme Corp",
		Score:   85.5,
		Tier:    "review",
	}

	value, err := js.Value()
	require.NoError(t, err)

	data, ok := value.([]byte)
	require.True(t, ok)

	var decoded JobSnapshot
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, js.Title, decoded.Title)
	assert.Equal(t, js.Company, decoded.Company)
	assert.Equal(t, js.Score, decoded.Score)
}

func TestJobSnapshot_ValueError(t *testing.T) {
	// Create a circular reference to force JSON marshal error
	// Actually, JobSnapshot is a simple struct, so Value() will never error.
	// Test the normal path instead to confirm it works.
	js := JobSnapshot{}
	_, err := js.Value()
	assert.NoError(t, err)
}

func TestJobSnapshot_Scan(t *testing.T) {
	t.Run("scan from []byte", func(t *testing.T) {
		js := JobSnapshot{}
		data := []byte(`{"title":"Engineer","company":"Acme","score":90,"tier":"review"}`)
		err := js.Scan(data)
		require.NoError(t, err)
		assert.Equal(t, "Engineer", js.Title)
		assert.Equal(t, "Acme", js.Company)
		assert.Equal(t, 90.0, js.Score)
	})

	t.Run("scan from string", func(t *testing.T) {
		js := JobSnapshot{}
		err := js.Scan(`{"title":"Engineer","company":"Acme","score":90,"tier":"review"}`)
		require.NoError(t, err)
		assert.Equal(t, "Engineer", js.Title)
	})

	t.Run("scan from nil", func(t *testing.T) {
		js := JobSnapshot{}
		err := js.Scan(nil)
		require.NoError(t, err)
		assert.Empty(t, js.Title)
	})

	t.Run("scan from unsupported type", func(t *testing.T) {
		js := JobSnapshot{}
		err := js.Scan(42)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})

	t.Run("scan from invalid JSON", func(t *testing.T) {
		js := JobSnapshot{}
		err := js.Scan([]byte(`{invalid`))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")
	})
}

func TestApprovalRequestColumns(t *testing.T) {
	assert.Contains(t, approvalRequestColumns, "id")
	assert.Contains(t, approvalRequestColumns, "application_id")
	assert.Contains(t, approvalRequestColumns, "job_snapshot")
	assert.Contains(t, approvalRequestColumns, "status")
	assert.Contains(t, approvalRequestColumns, "created_at")
	assert.Contains(t, approvalRequestColumns, "reviewed_at")
}
