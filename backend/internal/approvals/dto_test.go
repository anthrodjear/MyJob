package approvals

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestApproveRequest(t *testing.T) {
	req := ApproveRequest{}
	assert.NotNil(t, req)
	// ApproveRequest is an empty struct with a no-op field
	_ = req
}

func TestRejectRequest(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		req := RejectRequest{Reason: "Salary too low"}
		assert.Equal(t, "Salary too low", req.Reason)
	})

	t.Run("empty reason", func(t *testing.T) {
		req := RejectRequest{}
		assert.Empty(t, req.Reason)
	})
}

func TestApprovalResponse(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()
	reason := "Salary too low"

	resp := ApprovalResponse{
		ID:            id,
		ApplicationID: appID,
		JobSnapshot: JobSnapshot{
			Title:   "Software Engineer",
			Company: "Acme Corp",
			Score:   85.5,
			Tier:    "review",
		},
		Status:          ApprovalStatusRejected,
		RejectionReason: &reason,
		CreatedAt:       now,
		ReviewedAt:      &now,
	}

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, appID, resp.ApplicationID)
	assert.Equal(t, "Software Engineer", resp.JobSnapshot.Title)
	assert.Equal(t, "Acme Corp", resp.JobSnapshot.Company)
	assert.Equal(t, 85.5, resp.JobSnapshot.Score)
	assert.Equal(t, ApprovalStatusRejected, resp.Status)
	assert.Equal(t, &reason, resp.RejectionReason)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, &now, resp.ReviewedAt)
}

func TestApprovalResponse_NilPointers(t *testing.T) {
	resp := ApprovalResponse{}
	assert.Nil(t, resp.ResumePreviewPath)
	assert.Nil(t, resp.CoverLetterPreview)
	assert.Nil(t, resp.RejectionReason)
	assert.Nil(t, resp.ReviewedAt)
}

func TestApprovalListResponse(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		resp := ApprovalListResponse{
			Approvals: []ApprovalResponse{},
			Total:     0,
			Limit:     20,
			Offset:    0,
		}
		assert.Empty(t, resp.Approvals)
		assert.Equal(t, int64(0), resp.Total)
		assert.Equal(t, 20, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
	})

	t.Run("with items", func(t *testing.T) {
		resp := ApprovalListResponse{
			Approvals: []ApprovalResponse{
				{ID: uuid.New(), Status: ApprovalStatusPending},
				{ID: uuid.New(), Status: ApprovalStatusApproved},
			},
			Total:  2,
			Limit:  10,
			Offset: 0,
		}
		assert.Len(t, resp.Approvals, 2)
		assert.Equal(t, int64(2), resp.Total)
	})
}

func TestApprovePartialResponse(t *testing.T) {
	id := uuid.New()
	resp := ApprovePartialResponse{
		Status:  "approved",
		Warning: "task dispatch failed, needs retry",
		Approval: ApprovalResponse{
			ID:     id,
			Status: ApprovalStatusApproved,
		},
	}

	assert.Equal(t, "approved", resp.Status)
	assert.Equal(t, "task dispatch failed, needs retry", resp.Warning)
	assert.Equal(t, id, resp.Approval.ID)
	assert.Equal(t, ApprovalStatusApproved, resp.Approval.Status)
}

func TestToResponse(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()

	a := &ApprovalRequest{
		ID:            id,
		ApplicationID: appID,
		JobSnapshot: JobSnapshot{
			Title:   "Engineer",
			Company: "Acme",
		},
		Status:    ApprovalStatusPending,
		CreatedAt: now,
	}

	resp := ToResponse(a)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, appID, resp.ApplicationID)
	assert.Equal(t, "Engineer", resp.JobSnapshot.Title)
	assert.Equal(t, "Acme", resp.JobSnapshot.Company)
	assert.Equal(t, ApprovalStatusPending, resp.Status)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Nil(t, resp.ResumePreviewPath)
	assert.Nil(t, resp.CoverLetterPreview)
	assert.Nil(t, resp.RejectionReason)
	assert.Nil(t, resp.ReviewedAt)
}

func TestToResponse_WithOptionalFields(t *testing.T) {
	now := time.Now()
	resumePath := "resumes/test.pdf"
	reason := "bad fit"

	a := &ApprovalRequest{
		ID:                uuid.New(),
		ApplicationID:     uuid.New(),
		JobSnapshot:       JobSnapshot{Title: "Engineer", Company: "Acme"},
		ResumePreviewPath: &resumePath,
		RejectionReason:   &reason,
		ReviewedAt:        &now,
	}

	resp := ToResponse(a)
	assert.Equal(t, &resumePath, resp.ResumePreviewPath)
	assert.Equal(t, &reason, resp.RejectionReason)
	assert.Equal(t, &now, resp.ReviewedAt)
}
