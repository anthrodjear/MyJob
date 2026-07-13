package applications

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateApplicationRequest(t *testing.T) {
	jobID := uuid.New()
	resumeID := uuid.New()
	coverID := uuid.New()
	portalType := "greenhouse"
	portalURL := "https://boards.greenhouse.io/jobs/123"

	t.Run("full request", func(t *testing.T) {
		req := CreateApplicationRequest{
			JobID:         jobID,
			ResumeID:      &resumeID,
			CoverLetterID: &coverID,
			PortalType:    &portalType,
			PortalURL:     &portalURL,
		}
		assert.Equal(t, jobID, req.JobID)
		assert.Equal(t, &resumeID, req.ResumeID)
		assert.Equal(t, &coverID, req.CoverLetterID)
		assert.Equal(t, &portalType, req.PortalType)
		assert.Equal(t, &portalURL, req.PortalURL)
	})

	t.Run("minimal request", func(t *testing.T) {
		req := CreateApplicationRequest{
			JobID: jobID,
		}
		assert.Equal(t, jobID, req.JobID)
		assert.Nil(t, req.ResumeID)
		assert.Nil(t, req.CoverLetterID)
		assert.Nil(t, req.PortalType)
		assert.Nil(t, req.PortalURL)
	})

	t.Run("zero UUID is valid", func(t *testing.T) {
		req := CreateApplicationRequest{}
		assert.Equal(t, uuid.Nil, req.JobID)
	})
}

func TestUpdateStatusRequest(t *testing.T) {
	t.Run("with status and notes", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: StatusApplied,
			Notes:  "submitted successfully",
		}
		assert.Equal(t, StatusApplied, req.Status)
		assert.Equal(t, "submitted successfully", req.Notes)
	})

	t.Run("status only", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: StatusRejected,
		}
		assert.Equal(t, StatusRejected, req.Status)
		assert.Empty(t, req.Notes)
	})

	t.Run("empty status", func(t *testing.T) {
		req := UpdateStatusRequest{}
		assert.Empty(t, req.Status)
		assert.Empty(t, req.Notes)
	})
}

func TestUpdateApplicationNotesRequest(t *testing.T) {
	t.Run("with notes", func(t *testing.T) {
		req := UpdateApplicationNotesRequest{
			Notes: "updated notes",
		}
		assert.Equal(t, "updated notes", req.Notes)
	})

	t.Run("empty notes", func(t *testing.T) {
		req := UpdateApplicationNotesRequest{}
		assert.Empty(t, req.Notes)
	})
}

func TestApplicationResponse(t *testing.T) {
	id := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	t.Run("full response", func(t *testing.T) {
		resp := ApplicationResponse{
			ID:           id,
			JobID:        jobID,
			Status:       StatusApplied,
			ApprovalTier: TierAuto,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		assert.Equal(t, id, resp.ID)
		assert.Equal(t, jobID, resp.JobID)
		assert.Equal(t, StatusApplied, resp.Status)
		assert.Equal(t, TierAuto, resp.ApprovalTier)
		assert.Equal(t, now, resp.CreatedAt)
		assert.Equal(t, now, resp.UpdatedAt)
	})

	t.Run("response with optional fields", func(t *testing.T) {
		notes := "some notes"
		resp := ApplicationResponse{
			ID:    id,
			JobID: jobID,
			Notes: &notes,
		}
		assert.Equal(t, &notes, resp.Notes)
	})
}

func TestApplicationListResponse(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		resp := ApplicationListResponse{
			Applications: []ApplicationResponse{},
			Total:        0,
			Limit:        20,
			Offset:       0,
		}
		assert.Empty(t, resp.Applications)
		assert.Equal(t, int64(0), resp.Total)
		assert.Equal(t, 20, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
	})

	t.Run("with items", func(t *testing.T) {
		resp := ApplicationListResponse{
			Applications: []ApplicationResponse{
				{ID: uuid.New(), Status: StatusDraft},
				{ID: uuid.New(), Status: StatusApplied},
			},
			Total:  2,
			Limit:  10,
			Offset: 0,
		}
		assert.Len(t, resp.Applications, 2)
		assert.Equal(t, int64(2), resp.Total)
	})
}

func TestApplicationStatsResponse(t *testing.T) {
	t.Run("empty stats", func(t *testing.T) {
		stats := ApplicationStatsResponse{
			Total:    0,
			ByStatus: make(map[string]int64),
			ByTier:   make(map[string]int64),
		}
		assert.Equal(t, int64(0), stats.Total)
		assert.Empty(t, stats.ByStatus)
		assert.Empty(t, stats.ByTier)
	})

	t.Run("with stats data", func(t *testing.T) {
		stats := ApplicationStatsResponse{
			Total: 5,
			ByStatus: map[string]int64{
				StatusDraft:   2,
				StatusApplied: 3,
			},
			ByTier: map[string]int64{
				TierAuto:   4,
				TierReview: 1,
			},
		}
		assert.Equal(t, int64(5), stats.Total)
		assert.Equal(t, int64(2), stats.ByStatus[StatusDraft])
		assert.Equal(t, int64(3), stats.ByStatus[StatusApplied])
		assert.Equal(t, int64(4), stats.ByTier[TierAuto])
		assert.Equal(t, int64(1), stats.ByTier[TierReview])
	})
}

func TestApplicationEventResponse(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	resp := ApplicationEventResponse{
		ID:        id,
		OldStatus: StatusDraft,
		NewStatus: StatusQueued,
		Notes:     "transition",
		CreatedAt: now,
	}

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, StatusDraft, resp.OldStatus)
	assert.Equal(t, StatusQueued, resp.NewStatus)
	assert.Equal(t, "transition", resp.Notes)
	assert.Equal(t, now, resp.CreatedAt)
}

func TestApplicationTimelineResponse(t *testing.T) {
	appID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	timeline := ApplicationTimelineResponse{
		ApplicationID: appID,
		Events: []ApplicationEventResponse{
			{
				ID:        eventID,
				OldStatus: StatusDraft,
				NewStatus: StatusQueued,
				CreatedAt: now,
			},
		},
	}

	assert.Equal(t, appID, timeline.ApplicationID)
	assert.Len(t, timeline.Events, 1)
}

func TestToResponse(t *testing.T) {
	id := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	app := &Application{
		ID:           id,
		JobID:        jobID,
		Status:       StatusApplied,
		ApprovalTier: TierAuto,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := ToResponse(app)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, jobID, resp.JobID)
	assert.Equal(t, StatusApplied, resp.Status)
	assert.Equal(t, TierAuto, resp.ApprovalTier)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestToResponse_NilPointers(t *testing.T) {
	app := &Application{
		ID:    uuid.New(),
		JobID: uuid.New(),
	}

	resp := ToResponse(app)
	assert.Nil(t, resp.ResumeID)
	assert.Nil(t, resp.CoverLetterID)
	assert.Nil(t, resp.AppliedAt)
	assert.Nil(t, resp.ResponseAt)
	assert.Nil(t, resp.InterviewAt)
	assert.Nil(t, resp.Notes)
	assert.Nil(t, resp.PortalType)
	assert.Nil(t, resp.PortalURL)
}

func TestToEventResponse(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()

	event := &ApplicationEvent{
		ID:            id,
		ApplicationID: appID,
		OldStatus:     StatusDraft,
		NewStatus:     StatusQueued,
		Notes:         "proceeding",
		CreatedAt:     now,
	}

	resp := ToEventResponse(event)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, StatusDraft, resp.OldStatus)
	assert.Equal(t, StatusQueued, resp.NewStatus)
	assert.Equal(t, "proceeding", resp.Notes)
	assert.Equal(t, now, resp.CreatedAt)
}
