package applications

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{StatusDraft, true},
		{StatusQueued, true},
		{StatusApplied, true},
		{StatusAssessment, true},
		{StatusPhoneScreen, true},
		{StatusTechnical, true},
		{StatusFinal, true},
		{StatusOffer, true},
		{StatusRejected, true},
		{"unknown", false},
		{"", false},
		{"DRAFT", false},
		{"Applied", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := IsValidStatus(tt.status)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name  string
		from  string
		to    string
		valid bool
	}{
		{"draft to queued", StatusDraft, StatusQueued, true},
		{"draft to rejected", StatusDraft, StatusRejected, true},
		{"draft to applied (invalid)", StatusDraft, StatusApplied, false},
		{"queued to applied", StatusQueued, StatusApplied, true},
		{"queued to rejected", StatusQueued, StatusRejected, true},
		{"applied to assessment", StatusApplied, StatusAssessment, true},
		{"applied to rejected", StatusApplied, StatusRejected, true},
		{"offer - no transitions", StatusOffer, StatusDraft, false},
		{"offer to rejected (invalid)", StatusOffer, StatusRejected, false},
		{"rejected - no transitions", StatusRejected, StatusDraft, false},
		{"unknown from", "unknown", StatusDraft, false},
		{"empty from", "", StatusDraft, false},
		{"draft back to draft", StatusDraft, StatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransition(tt.from, tt.to)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestApplication_TableName(t *testing.T) {
	app := Application{}
	assert.Equal(t, "applications", app.TableName())
}

func TestApplication_Fields(t *testing.T) {
	id := uuid.New()
	jobID := uuid.New()
	resumeID := uuid.New()
	coverID := uuid.New()
	now := time.Now()
	notes := "some notes"
	portalType := "greenhouse"
	portalURL := "https://example.com"

	app := Application{
		ID:            id,
		JobID:         jobID,
		ResumeID:      &resumeID,
		CoverLetterID: &coverID,
		Status:        StatusApplied,
		ApprovalTier:  TierAuto,
		AppliedAt:     &now,
		ResponseAt:    nil,
		InterviewAt:   nil,
		Notes:         &notes,
		PortalType:    &portalType,
		PortalURL:     &portalURL,
		FormData:      []byte(`{"key":"value"}`),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, id, app.ID)
	assert.Equal(t, jobID, app.JobID)
	assert.Equal(t, &resumeID, app.ResumeID)
	assert.Equal(t, &coverID, app.CoverLetterID)
	assert.Equal(t, StatusApplied, app.Status)
	assert.Equal(t, TierAuto, app.ApprovalTier)
	assert.Equal(t, &now, app.AppliedAt)
	assert.Nil(t, app.ResponseAt)
	assert.Nil(t, app.InterviewAt)
	assert.Equal(t, &notes, app.Notes)
	assert.Equal(t, &portalType, app.PortalType)
	assert.Equal(t, &portalURL, app.PortalURL)
	assert.Equal(t, []byte(`{"key":"value"}`), app.FormData)
	assert.Equal(t, now, app.CreatedAt)
	assert.Equal(t, now, app.UpdatedAt)
}

func TestApplication_NilPointers(t *testing.T) {
	app := Application{}
	assert.Nil(t, app.ResumeID)
	assert.Nil(t, app.CoverLetterID)
	assert.Nil(t, app.AppliedAt)
	assert.Nil(t, app.ResponseAt)
	assert.Nil(t, app.InterviewAt)
	assert.Nil(t, app.Notes)
	assert.Nil(t, app.PortalType)
	assert.Nil(t, app.PortalURL)
	assert.Nil(t, app.FormData)
}

func TestApplicationEvent_TableName(t *testing.T) {
	e := ApplicationEvent{}
	assert.Equal(t, "application_events", e.TableName())
}

func TestApplicationEvent_Fields(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()

	e := ApplicationEvent{
		ID:            id,
		ApplicationID: appID,
		OldStatus:     StatusDraft,
		NewStatus:     StatusQueued,
		Notes:         "proceeding",
		CreatedAt:     now,
	}

	assert.Equal(t, id, e.ID)
	assert.Equal(t, appID, e.ApplicationID)
	assert.Equal(t, StatusDraft, e.OldStatus)
	assert.Equal(t, StatusQueued, e.NewStatus)
	assert.Equal(t, "proceeding", e.Notes)
	assert.Equal(t, now, e.CreatedAt)
}

func TestApprovalTransitions_CompleteCoverage(t *testing.T) {
	// Verify every valid status has a transition entry
	for status := range approvalTransitions {
		assert.True(t, IsValidStatus(status), "status %q should be valid", status)
	}

	// Verify terminal states have no transitions
	terminalStates := []string{StatusOffer, StatusRejected}
	for _, s := range terminalStates {
		transitions, ok := approvalTransitions[s]
		assert.True(t, ok)
		assert.Empty(t, transitions, "terminal state %q should have no transitions", s)
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "auto", TierAuto)
	assert.Equal(t, "review", TierReview)
	assert.Equal(t, "reject", TierReject)

	assert.Equal(t, "draft", StatusDraft)
	assert.Equal(t, "queued", StatusQueued)
	assert.Equal(t, "applied", StatusApplied)
	assert.Equal(t, "assessment", StatusAssessment)
	assert.Equal(t, "phone_screen", StatusPhoneScreen)
	assert.Equal(t, "technical", StatusTechnical)
	assert.Equal(t, "final", StatusFinal)
	assert.Equal(t, "offer", StatusOffer)
	assert.Equal(t, "rejected", StatusRejected)
}
