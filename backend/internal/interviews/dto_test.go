package interviews

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CreateInterviewRequest
// ---------------------------------------------------------------------------

func TestCreateInterviewRequest(t *testing.T) {
	appID := uuid.New()

	req := CreateInterviewRequest{
		ApplicationID: appID,
		Mode:          ModeAutonomous,
	}

	assert.Equal(t, appID, req.ApplicationID)
	assert.Equal(t, ModeAutonomous, req.Mode)
}

func TestCreateInterviewRequest_ZeroValues(t *testing.T) {
	var req CreateInterviewRequest

	assert.Equal(t, uuid.Nil, req.ApplicationID)
	assert.Empty(t, req.Mode)
}

func TestCreateInterviewRequest_JSONBinding(t *testing.T) {
	appID := uuid.New()

	// Simulate JSON deserialisation (as Gin would do)
	data := `{"application_id":"` + appID.String() + `","mode":"assist"}`
	var req CreateInterviewRequest
	err := json.Unmarshal([]byte(data), &req)
	require.NoError(t, err)
	assert.Equal(t, appID, req.ApplicationID)
	assert.Equal(t, "assist", req.Mode)
}

func TestCreateInterviewRequest_JSONMissingFields(t *testing.T) {
	// Empty JSON — zero values
	var req CreateInterviewRequest
	err := json.Unmarshal([]byte(`{}`), &req)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, req.ApplicationID)
	assert.Empty(t, req.Mode)
}

// ---------------------------------------------------------------------------
// StartInterviewRequest
// ---------------------------------------------------------------------------

func TestStartInterviewRequest(t *testing.T) {
	req := StartInterviewRequest{
		Provider: "openai_realtime",
		Model:    "gpt-4o-realtime-preview",
	}

	assert.Equal(t, "openai_realtime", req.Provider)
	assert.Equal(t, "gpt-4o-realtime-preview", req.Model)
}

func TestStartInterviewRequest_OptionalDefaults(t *testing.T) {
	// Provider and model are optional — empty string means "use config default"
	req := StartInterviewRequest{}
	assert.Empty(t, req.Provider)
	assert.Empty(t, req.Model)
}

func TestStartInterviewRequest_JSONRoundTrip(t *testing.T) {
	req := StartInterviewRequest{
		Provider: "elevenlabs",
		Model:    "eleven_multilingual_v2",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded StartInterviewRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, req.Provider, decoded.Provider)
	assert.Equal(t, req.Model, decoded.Model)
}

func TestStartInterviewRequest_JSONPartialBody(t *testing.T) {
	var req StartInterviewRequest
	err := json.Unmarshal([]byte(`{"provider":"test"}`), &req)
	require.NoError(t, err)
	assert.Equal(t, "test", req.Provider)
	assert.Empty(t, req.Model)
}

// ---------------------------------------------------------------------------
// StopInterviewRequest
// ---------------------------------------------------------------------------

func TestStopInterviewRequest(t *testing.T) {
	req := StopInterviewRequest{
		Reason: "user_cancelled",
	}

	assert.Equal(t, "user_cancelled", req.Reason)
}

func TestStopInterviewRequest_EmptyReason(t *testing.T) {
	// Reason is optional — empty string is acceptable
	req := StopInterviewRequest{}
	assert.Empty(t, req.Reason)
}

func TestStopInterviewRequest_JSONRoundTrip(t *testing.T) {
	req := StopInterviewRequest{Reason: "timeout"}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded StopInterviewRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "timeout", decoded.Reason)
}

// ---------------------------------------------------------------------------
// InterviewEventRequest
// ---------------------------------------------------------------------------

func TestInterviewEventRequest_TranscriptEvent(t *testing.T) {
	ts := time.Date(2026, 6, 19, 10, 30, 0, 0, time.UTC)

	req := InterviewEventRequest{
		Type:      "transcript",
		Speaker:   SpeakerCandidate,
		Content:   "I have 5 years of Go experience",
		Timestamp: &ts,
	}

	assert.Equal(t, "transcript", req.Type)
	assert.Equal(t, SpeakerCandidate, req.Speaker)
	assert.Equal(t, "I have 5 years of Go experience", req.Content)
	assert.Equal(t, &ts, req.Timestamp)

	// Unrelated fields should be zero
	assert.Empty(t, req.Status)
	assert.Nil(t, req.Score)
	assert.Nil(t, req.Feedback)
}

func TestInterviewEventRequest_StatusEvent(t *testing.T) {
	req := InterviewEventRequest{
		Type:   "status",
		Status: StatusActive,
	}

	assert.Equal(t, "status", req.Type)
	assert.Equal(t, StatusActive, req.Status)

	// Unrelated fields should be zero
	assert.Empty(t, req.Speaker)
	assert.Empty(t, req.Content)
	assert.Nil(t, req.Timestamp)
	assert.Nil(t, req.Score)
	assert.Nil(t, req.Feedback)
}

func TestInterviewEventRequest_ScoreEvent(t *testing.T) {
	score := 92.5

	req := InterviewEventRequest{
		Type:  "score",
		Score: &score,
	}

	assert.Equal(t, "score", req.Type)
	require.NotNil(t, req.Score)
	assert.Equal(t, 92.5, *req.Score)

	// Unrelated fields should be zero
	assert.Empty(t, req.Status)
	assert.Empty(t, req.Speaker)
	assert.Empty(t, req.Content)
	assert.Nil(t, req.Timestamp)
	assert.Nil(t, req.Feedback)
}

func TestInterviewEventRequest_FeedbackEvent(t *testing.T) {
	feedback := json.RawMessage(`{"strengths":["communication"],"score":85}`)

	req := InterviewEventRequest{
		Type:     "feedback",
		Feedback: feedback,
	}

	assert.Equal(t, "feedback", req.Type)
	assert.Equal(t, feedback, req.Feedback)

	// Unrelated fields should be zero
	assert.Empty(t, req.Status)
	assert.Empty(t, req.Speaker)
	assert.Empty(t, req.Content)
	assert.Nil(t, req.Timestamp)
	assert.Nil(t, req.Score)
}

func TestInterviewEventRequest_ZeroValues(t *testing.T) {
	var req InterviewEventRequest

	assert.Empty(t, req.Type)
	assert.Empty(t, req.Status)
	assert.Empty(t, req.Speaker)
	assert.Empty(t, req.Content)
	assert.Nil(t, req.Timestamp)
	assert.Nil(t, req.Score)
	assert.Nil(t, req.Feedback)
}

func TestInterviewEventRequest_JSONTranscriptEvent(t *testing.T) {
	raw := `{
		"type": "transcript",
		"speaker": "candidate",
		"content": "I am ready",
		"timestamp": "2026-06-19T10:30:00Z"
	}`

	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "transcript", req.Type)
	assert.Equal(t, "candidate", req.Speaker)
	assert.Equal(t, "I am ready", req.Content)
	require.NotNil(t, req.Timestamp)
	assert.Equal(t, 2026, req.Timestamp.Year())
	assert.True(t, req.Timestamp.Equal(time.Date(2026, 6, 19, 10, 30, 0, 0, time.UTC)))
}

func TestInterviewEventRequest_JSONStatusEvent(t *testing.T) {
	raw := `{"type":"status","status":"active"}`

	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "status", req.Type)
	assert.Equal(t, "active", req.Status)
}

func TestInterviewEventRequest_JSONScoreEvent(t *testing.T) {
	raw := `{"type":"score","score":87}`

	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "score", req.Type)
	require.NotNil(t, req.Score)
	assert.Equal(t, 87.0, *req.Score)
}

func TestInterviewEventRequest_JSONFeedbackEvent(t *testing.T) {
	raw := `{"type":"feedback","feedback":{"rating":"good","notes":["clear","concise"]}}`

	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "feedback", req.Type)
	require.NotNil(t, req.Feedback)
	assert.JSONEq(t, `{"rating":"good","notes":["clear","concise"]}`, string(req.Feedback))
}

func TestInterviewEventRequest_JSONMissingType(t *testing.T) {
	raw := `{"speaker":"candidate","content":"hello"}`

	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Empty(t, req.Type)
	assert.Equal(t, "candidate", req.Speaker)
	assert.Equal(t, "hello", req.Content)
}

func TestInterviewEventRequest_JSONOmitTimestamp(t *testing.T) {
	// Omitting timestamp should produce nil (not zero time)
	raw := `{"type":"transcript","speaker":"ai","content":"hello"}`
	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)
	assert.Nil(t, req.Timestamp)
}

func TestInterviewEventRequest_JSONOmitScore(t *testing.T) {
	raw := `{"type":"score"}`
	var req InterviewEventRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)
	assert.Equal(t, "score", req.Type)
	assert.Nil(t, req.Score)
}

// ---------------------------------------------------------------------------
// InterviewResponse
// ---------------------------------------------------------------------------

func TestInterviewResponse(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	extID := "RMabc123"
	score := 95.0
	feedback := json.RawMessage(`{"summary":"excellent"}`)
	startedAt := now.Add(-30 * time.Minute)
	endedAt := now

	resp := InterviewResponse{
		ID:                id,
		ApplicationID:     appID,
		Mode:              ModeAssist,
		Status:            StatusCompleted,
		ExternalSessionID: &extID,
		Provider:          "openai_realtime",
		Model:             "gpt-4o-realtime-preview",
		Transcript: []TranscriptEntry{
			{ID: uuid.New(), Speaker: SpeakerAI, Content: "Hello", Timestamp: now},
		},
		Score:     &score,
		Feedback:  feedback,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, appID, resp.ApplicationID)
	assert.Equal(t, ModeAssist, resp.Mode)
	assert.Equal(t, StatusCompleted, resp.Status)
	assert.Equal(t, &extID, resp.ExternalSessionID)
	assert.Equal(t, "openai_realtime", resp.Provider)
	assert.Equal(t, "gpt-4o-realtime-preview", resp.Model)
	assert.Len(t, resp.Transcript, 1)
	assert.Equal(t, &score, resp.Score)
	assert.Equal(t, feedback, resp.Feedback)
	assert.Equal(t, &startedAt, resp.StartedAt)
	assert.Equal(t, &endedAt, resp.EndedAt)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestInterviewResponse_NilPointers(t *testing.T) {
	resp := InterviewResponse{}
	assert.Nil(t, resp.ExternalSessionID)
	assert.Nil(t, resp.Score)
	assert.Nil(t, resp.Feedback)
	assert.Nil(t, resp.StartedAt)
	assert.Nil(t, resp.EndedAt)
}

func TestInterviewResponse_EmptyTranscript(t *testing.T) {
	resp := InterviewResponse{
		Transcript: []TranscriptEntry{},
	}
	assert.Empty(t, resp.Transcript)
	assert.NotNil(t, resp.Transcript)
}

func TestInterviewResponse_JSONOmitEmpty(t *testing.T) {
	resp := InterviewResponse{
		ID:    uuid.New(),
		Mode:  ModeAutonomous,
		Status: StatusPending,
		Transcript: []TranscriptEntry{},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "external_session_id")
	assert.NotContains(t, string(data), "score")
	assert.NotContains(t, string(data), "feedback")
	assert.NotContains(t, string(data), "started_at")
	assert.NotContains(t, string(data), "ended_at")
}

// ---------------------------------------------------------------------------
// InterviewListResponse
// ---------------------------------------------------------------------------

func TestInterviewListResponse_Empty(t *testing.T) {
	resp := InterviewListResponse{
		Interviews: []InterviewResponse{},
		Total:      0,
		Limit:      20,
		Offset:     0,
	}

	assert.Empty(t, resp.Interviews)
	assert.Equal(t, int64(0), resp.Total)
	assert.Equal(t, 20, resp.Limit)
	assert.Equal(t, 0, resp.Offset)
}

func TestInterviewListResponse_WithItems(t *testing.T) {
	resp := InterviewListResponse{
		Interviews: []InterviewResponse{
			{ID: uuid.New(), Mode: ModeAssist, Status: StatusPending},
			{ID: uuid.New(), Mode: ModeAutonomous, Status: StatusActive},
			{ID: uuid.New(), Mode: ModeAssist, Status: StatusCompleted},
		},
		Total:  3,
		Limit:  10,
		Offset: 0,
	}

	assert.Len(t, resp.Interviews, 3)
	assert.Equal(t, int64(3), resp.Total)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 0, resp.Offset)

	assert.Equal(t, ModeAssist, resp.Interviews[0].Mode)
	assert.Equal(t, StatusPending, resp.Interviews[0].Status)
	assert.Equal(t, StatusActive, resp.Interviews[1].Status)
	assert.Equal(t, StatusCompleted, resp.Interviews[2].Status)
}

func TestInterviewListResponse_JSONRoundTrip(t *testing.T) {
	resp := InterviewListResponse{
		Interviews: []InterviewResponse{
			{ID: uuid.New(), Status: StatusPending},
		},
		Total:  1,
		Limit:  20,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded InterviewListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Interviews, 1)
	assert.Equal(t, int64(1), decoded.Total)
	assert.Equal(t, 20, decoded.Limit)
	assert.Equal(t, 0, decoded.Offset)
	assert.Equal(t, resp.Interviews[0].ID, decoded.Interviews[0].ID)
	assert.Equal(t, resp.Interviews[0].Status, decoded.Interviews[0].Status)
}

// ---------------------------------------------------------------------------
// ToResponse mapper
// ---------------------------------------------------------------------------

func TestToResponse(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	session := &InterviewSession{
		ID:                id,
		ApplicationID:     appID,
		Mode:              ModeAssist,
		Status:            StatusPending,
		ExternalSessionID: nil,
		Provider:          "",
		Model:             "",
		Transcript:        []TranscriptEntry{},
		Score:             nil,
		Feedback:          nil,
		StartedAt:         nil,
		EndedAt:           nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	resp := ToResponse(session)

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, appID, resp.ApplicationID)
	assert.Equal(t, ModeAssist, resp.Mode)
	assert.Equal(t, StatusPending, resp.Status)
	assert.Nil(t, resp.ExternalSessionID)
	assert.Empty(t, resp.Provider)
	assert.Empty(t, resp.Model)
	assert.Empty(t, resp.Transcript)
	assert.Nil(t, resp.Score)
	assert.Nil(t, resp.Feedback)
	assert.Nil(t, resp.StartedAt)
	assert.Nil(t, resp.EndedAt)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestToResponse_WithAllFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	extID := "sess_abc"
	score := 88.0
	feedback := json.RawMessage(`{"categories":{"technical":90}}`)
	startedAt := now.Add(-20 * time.Minute)
	endedAt := now

	session := &InterviewSession{
		ID:                uuid.New(),
		ApplicationID:     uuid.New(),
		Mode:              ModeAutonomous,
		Status:            StatusCompleted,
		ExternalSessionID: &extID,
		Provider:          "elevenlabs",
		Model:             "eleven_turbo_v2",
		Transcript: []TranscriptEntry{
			{
				ID:        uuid.New(),
				Speaker:   SpeakerCandidate,
				Content:   "I have experience",
				Timestamp: now,
			},
		},
		Score:     &score,
		Feedback:  feedback,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := ToResponse(session)

	assert.Equal(t, session.ID, resp.ID)
	assert.Equal(t, session.ApplicationID, resp.ApplicationID)
	assert.Equal(t, session.Mode, resp.Mode)
	assert.Equal(t, session.Status, resp.Status)
	assert.Equal(t, session.ExternalSessionID, resp.ExternalSessionID)
	assert.Equal(t, session.Provider, resp.Provider)
	assert.Equal(t, session.Model, resp.Model)
	assert.Equal(t, session.Transcript, resp.Transcript)
	assert.Equal(t, session.Score, resp.Score)
	assert.Equal(t, session.Feedback, resp.Feedback)
	assert.Equal(t, session.StartedAt, resp.StartedAt)
	assert.Equal(t, session.EndedAt, resp.EndedAt)
	assert.Equal(t, session.CreatedAt, resp.CreatedAt)
	assert.Equal(t, session.UpdatedAt, resp.UpdatedAt)
}

func TestToResponse_NilPointers(t *testing.T) {
	// All optional pointer fields in the session should map to nil in the response
	session := &InterviewSession{
		ID:    uuid.New(),
		Mode:  ModeAssist,
		Status: StatusPending,
	}

	resp := ToResponse(session)
	assert.Nil(t, resp.ExternalSessionID)
	assert.Nil(t, resp.Score)
	assert.Nil(t, resp.Feedback)
	assert.Nil(t, resp.StartedAt)
	assert.Nil(t, resp.EndedAt)
}

func TestToResponse_EmptySession(t *testing.T) {
	// Empty session struct — verify no panics and zero values map correctly
	session := &InterviewSession{}
	resp := ToResponse(session)

	assert.Equal(t, uuid.Nil, resp.ID)
	assert.Equal(t, uuid.Nil, resp.ApplicationID)
	assert.Empty(t, resp.Mode)
	assert.Empty(t, resp.Status)
	assert.Nil(t, resp.ExternalSessionID)
	assert.Empty(t, resp.Provider)
	assert.Empty(t, resp.Model)
	assert.Nil(t, resp.Transcript)
	assert.Nil(t, resp.Score)
	assert.Nil(t, resp.Feedback)
	assert.Nil(t, resp.StartedAt)
	assert.Nil(t, resp.EndedAt)
	assert.True(t, resp.CreatedAt.IsZero())
	assert.True(t, resp.UpdatedAt.IsZero())
}

func TestToResponse_Immutability(t *testing.T) {
	// The response must be a copy — modifying the response must not affect the session
	extID := "test_ext"
	session := &InterviewSession{
		ID:                uuid.New(),
		ExternalSessionID: &extID,
	}

	resp := ToResponse(session)

	// Modify the response
	newExt := "modified"
	resp.ExternalSessionID = &newExt

	// Session must be unchanged
	require.NotNil(t, session.ExternalSessionID)
	assert.Equal(t, "test_ext", *session.ExternalSessionID)
}

// ---------------------------------------------------------------------------
// JSON tag consistency: DTOs
// ---------------------------------------------------------------------------

func TestInterviewResponse_JSONTags(t *testing.T) {
	resp := InterviewResponse{
		ID:    uuid.New(),
		Mode:  ModeAssist,
		Status: StatusActive,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "id")
	assert.Contains(t, raw, "application_id")
	assert.Contains(t, raw, "mode")
	assert.Contains(t, raw, "status")
	assert.Contains(t, raw, "provider")
	assert.Contains(t, raw, "model")
	assert.Contains(t, raw, "transcript")
	assert.Contains(t, raw, "created_at")
	assert.Contains(t, raw, "updated_at")
}

func TestInterviewListResponse_JSONTags(t *testing.T) {
	resp := InterviewListResponse{
		Interviews: []InterviewResponse{},
		Total:      0,
		Limit:      20,
		Offset:     0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "interviews")
	assert.Contains(t, raw, "total")
	assert.Contains(t, raw, "limit")
	assert.Contains(t, raw, "offset")
}

// ---------------------------------------------------------------------------
// Edge cases: Boundary values
// ---------------------------------------------------------------------------

func TestCreateInterviewRequest_ModeEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty JSON object", `{"application_id":"` + uuid.New().String() + `","mode":""}`},
		{"missing mode key", `{"application_id":"` + uuid.New().String() + `"}`},
		{"nil uuid", `{"application_id":"00000000-0000-0000-0000-000000000000","mode":"assist"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CreateInterviewRequest
			err := json.Unmarshal([]byte(tt.data), &req)
			require.NoError(t, err)
			// Validation is handled by Gin's binding tags, not JSON unmarshalling
			_ = req
		})
	}
}
