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
// Constants
// ---------------------------------------------------------------------------

func TestModeConstants(t *testing.T) {
	assert.Equal(t, "assist", ModeAssist)
	assert.Equal(t, "autonomous", ModeAutonomous)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "starting", StatusStarting)
	assert.Equal(t, "active", StatusActive)
	assert.Equal(t, "completed", StatusCompleted)
	assert.Equal(t, "failed", StatusFailed)
	assert.Equal(t, "cancelled", StatusCancelled)
}

func TestSpeakerConstants(t *testing.T) {
	assert.Equal(t, "candidate", SpeakerCandidate)
	assert.Equal(t, "ai", SpeakerAI)
	assert.Equal(t, "system", SpeakerSystem)
}

// ---------------------------------------------------------------------------
// Domain errors
// ---------------------------------------------------------------------------

func TestDomainErrors(t *testing.T) {
	assert.Error(t, ErrNotFound)
	assert.Error(t, ErrInvalidStatus)
	assert.Contains(t, ErrNotFound.Error(), "interview session not found")
	assert.Contains(t, ErrInvalidStatus.Error(), "invalid status transition")
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		valid bool
	}{
		{"assist", ModeAssist, true},
		{"autonomous", ModeAutonomous, true},
		{"empty string", "", false},
		{"unknown mode", "hybrid", false},
		{"typo assist", "asist", false},
		{"typo auto", "autonomus", false},
		{"upper case ASSIST", "ASSIST", false},
		{"mixed case", "Assist", false},
		{"whitespace", " assist ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidMode(tt.mode))
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{"pending", StatusPending, true},
		{"starting", StatusStarting, true},
		{"active", StatusActive, true},
		{"completed", StatusCompleted, true},
		{"failed", StatusFailed, true},
		{"cancelled", StatusCancelled, true},
		{"empty string", "", false},
		{"unknown status", "unknown", false},
		{"typo pending", "pendng", false},
		{"upper case PENDING", "PENDING", false},
		{"mixed case", "Pending", false},
		{"whitespace", " pending ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidStatus(tt.status))
		})
	}
}

// ---------------------------------------------------------------------------
// State machine: sessionTransitions
// ---------------------------------------------------------------------------

func TestSessionTransitions_CompleteCoverage(t *testing.T) {
	transitions := sessionTransitions()

	// All six statuses must be present
	assert.Contains(t, transitions, StatusPending)
	assert.Contains(t, transitions, StatusStarting)
	assert.Contains(t, transitions, StatusActive)
	assert.Contains(t, transitions, StatusCompleted)
	assert.Contains(t, transitions, StatusFailed)
	assert.Contains(t, transitions, StatusCancelled)

	// pending transitions
	assert.ElementsMatch(t,
		[]string{StatusStarting, StatusFailed, StatusCancelled},
		transitions[StatusPending])

	// starting transitions
	assert.ElementsMatch(t,
		[]string{StatusActive, StatusFailed, StatusCancelled},
		transitions[StatusStarting])

	// active transitions
	assert.ElementsMatch(t,
		[]string{StatusCompleted, StatusFailed, StatusCancelled},
		transitions[StatusActive])

	// terminal states — empty slices
	assert.Empty(t, transitions[StatusCompleted])
	assert.Empty(t, transitions[StatusFailed])
	assert.Empty(t, transitions[StatusCancelled])
}

func TestSessionTransitions_Immutability(t *testing.T) {
	// Each call must return a fresh map — mutating one must not affect another
	first := sessionTransitions()
	second := sessionTransitions()

	// Mutate first
	first[StatusPending] = append(first[StatusPending], "bogus")

	// second should remain unchanged
	assert.ElementsMatch(t,
		[]string{StatusStarting, StatusFailed, StatusCancelled},
		second[StatusPending])
}

// ---------------------------------------------------------------------------
// State machine: CanTransition
// ---------------------------------------------------------------------------

func TestCanTransition_FromPending(t *testing.T) {
	tests := []struct {
		to    string
		valid bool
	}{
		{StatusStarting, true},
		{StatusFailed, true},
		{StatusCancelled, true},
		{StatusActive, false},
		{StatusCompleted, false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run("pending → "+tt.to, func(t *testing.T) {
			assert.Equal(t, tt.valid, CanTransition(StatusPending, tt.to))
		})
	}
}

func TestCanTransition_FromStarting(t *testing.T) {
	tests := []struct {
		to    string
		valid bool
	}{
		{StatusActive, true},
		{StatusFailed, true},
		{StatusCancelled, true},
		{StatusPending, false},
		{StatusCompleted, false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run("starting → "+tt.to, func(t *testing.T) {
			assert.Equal(t, tt.valid, CanTransition(StatusStarting, tt.to))
		})
	}
}

func TestCanTransition_FromActive(t *testing.T) {
	tests := []struct {
		to    string
		valid bool
	}{
		{StatusCompleted, true},
		{StatusFailed, true},
		{StatusCancelled, true},
		{StatusPending, false},
		{StatusStarting, false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run("active → "+tt.to, func(t *testing.T) {
			assert.Equal(t, tt.valid, CanTransition(StatusActive, tt.to))
		})
	}
}

func TestCanTransition_FromTerminalStates(t *testing.T) {
	terminalStates := []string{StatusCompleted, StatusFailed, StatusCancelled}
	targets := []string{
		StatusPending, StatusStarting, StatusActive,
		StatusCompleted, StatusFailed, StatusCancelled,
		"", "anything",
	}

	for _, from := range terminalStates {
		for _, to := range targets {
			t.Run(from+" → "+to, func(t *testing.T) {
				assert.False(t, CanTransition(from, to),
					"expected no transition from terminal state %q to %q", from, to)
			})
		}
	}
}

func TestCanTransition_UnknownFromState(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{"nonexistent", StatusPending},
		{"", StatusPending},
		{"PENDING", StatusStarting},
	}

	for _, tt := range tests {
		t.Run("unknown from "+tt.from, func(t *testing.T) {
			assert.False(t, CanTransition(tt.from, tt.to))
		})
	}
}

// ---------------------------------------------------------------------------
// TranscriptEntry
// ---------------------------------------------------------------------------

func TestTranscriptEntry_Fields(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()

	entry := TranscriptEntry{
		ID:        id,
		Speaker:   SpeakerCandidate,
		Content:   "I have 5 years of Go experience",
		Timestamp: now,
	}

	assert.Equal(t, id, entry.ID)
	assert.Equal(t, SpeakerCandidate, entry.Speaker)
	assert.Equal(t, "I have 5 years of Go experience", entry.Content)
	assert.Equal(t, now, entry.Timestamp)
}

func TestTranscriptEntry_ZeroValues(t *testing.T) {
	var entry TranscriptEntry

	assert.Equal(t, uuid.Nil, entry.ID)
	assert.Empty(t, entry.Speaker)
	assert.Empty(t, entry.Content)
	assert.True(t, entry.Timestamp.IsZero())
}

func TestTranscriptEntry_JSONTags(t *testing.T) {
	entry := TranscriptEntry{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Speaker:   SpeakerAI,
		Content:   "Tell me about your experience",
		Timestamp: time.Date(2026, 6, 19, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded TranscriptEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.Speaker, decoded.Speaker)
	assert.Equal(t, entry.Content, decoded.Content)
	assert.Equal(t, entry.Timestamp, decoded.Timestamp)
}

// ---------------------------------------------------------------------------
// InterviewSession
// ---------------------------------------------------------------------------

func TestInterviewSession_TableName(t *testing.T) {
	s := InterviewSession{}
	assert.Equal(t, "interview_sessions", s.TableName())
}

func TestInterviewSession_Fields(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now().UTC()
	extID := "RM12345"
	score := 87.5
	feedback := json.RawMessage(`{"categories":{"communication":90},"summary":"Good"}`)
	startedAt := now.Add(-30 * time.Minute)
	endedAt := now

	session := InterviewSession{
		ID:                id,
		ApplicationID:     appID,
		Mode:              ModeAssist,
		Status:            StatusActive,
		ExternalSessionID: &extID,
		Provider:          "openai_realtime",
		Model:             "gpt-4o-realtime-preview",
		Transcript: Transcript{
			{ID: uuid.New(), Speaker: SpeakerAI, Content: "Hello", Timestamp: now},
		},
		Score:     &score,
		Feedback:  feedback,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, id, session.ID)
	assert.Equal(t, appID, session.ApplicationID)
	assert.Equal(t, ModeAssist, session.Mode)
	assert.Equal(t, StatusActive, session.Status)
	assert.Equal(t, &extID, session.ExternalSessionID)
	assert.Equal(t, "openai_realtime", session.Provider)
	assert.Equal(t, "gpt-4o-realtime-preview", session.Model)
	assert.Len(t, session.Transcript, 1)
	assert.Equal(t, &score, session.Score)
	assert.Equal(t, feedback, session.Feedback)
	assert.Equal(t, &startedAt, session.StartedAt)
	assert.Equal(t, &endedAt, session.EndedAt)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, now, session.UpdatedAt)
}

func TestInterviewSession_NilPointers(t *testing.T) {
	// All pointer/optional fields should be nil at zero value
	s := InterviewSession{}
	assert.Nil(t, s.ExternalSessionID)
	assert.Nil(t, s.Score)
	assert.Nil(t, s.Feedback)
	assert.Nil(t, s.StartedAt)
	assert.Nil(t, s.EndedAt)
}

func TestInterviewSession_NilTranscript(t *testing.T) {
	// Transcript is a slice, nil by default
	s := InterviewSession{}
	assert.Nil(t, s.Transcript)
	assert.Len(t, s.Transcript, 0)
}

func TestInterviewSession_DBTags(t *testing.T) {
	// Verify expected db field tags via JSON round-trip and structural check
	s := InterviewSession{ID: uuid.New()}
	data, err := json.Marshal(s)
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

// ---------------------------------------------------------------------------
// InterviewSession.TransitionTo
// ---------------------------------------------------------------------------

func TestInterviewSession_TransitionTo_Valid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"pending → starting", StatusPending, StatusStarting},
		{"pending → failed", StatusPending, StatusFailed},
		{"pending → cancelled", StatusPending, StatusCancelled},
		{"starting → active", StatusStarting, StatusActive},
		{"starting → failed", StatusStarting, StatusFailed},
		{"starting → cancelled", StatusStarting, StatusCancelled},
		{"active → completed", StatusActive, StatusCompleted},
		{"active → failed", StatusActive, StatusFailed},
		{"active → cancelled", StatusActive, StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &InterviewSession{Status: tt.from}
			err := session.TransitionTo(tt.to)
			assert.NoError(t, err)
			assert.Equal(t, tt.to, session.Status)
		})
	}
}

func TestInterviewSession_TransitionTo_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"pending → active (skip starting)", StatusPending, StatusActive},
		{"pending → completed (skip all)", StatusPending, StatusCompleted},
		{"starting → pending (rollback)", StatusStarting, StatusPending},
		{"starting → completed (skip active)", StatusStarting, StatusCompleted},
		{"active → pending (rollback)", StatusActive, StatusPending},
		{"active → starting (rollback)", StatusActive, StatusStarting},
		{"completed → pending (terminal)", StatusCompleted, StatusPending},
		{"completed → active (terminal)", StatusCompleted, StatusActive},
		{"failed → pending (terminal)", StatusFailed, StatusPending},
		{"failed → starting (terminal)", StatusFailed, StatusStarting},
		{"cancelled → pending (terminal)", StatusCancelled, StatusPending},
		{"cancelled → active (terminal)", StatusCancelled, StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &InterviewSession{Status: tt.from}
			err := session.TransitionTo(tt.to)
			assert.Error(t, err)
			assert.Equal(t, tt.from, session.Status,
				"status should not change on failed transition")
		})
	}
}

func TestInterviewSession_TransitionTo_ErrorMessage(t *testing.T) {
	session := &InterviewSession{Status: StatusPending}
	err := session.TransitionTo(StatusActive) // invalid: must go through starting

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
	assert.Contains(t, err.Error(), "pending")
	assert.Contains(t, err.Error(), "active")
}

func TestInterviewSession_TransitionTo_TerminalStatesHaveNoOutgoing(t *testing.T) {
	terminalStates := []string{StatusCompleted, StatusFailed, StatusCancelled}
	allStatuses := []string{
		StatusPending, StatusStarting, StatusActive,
		StatusCompleted, StatusFailed, StatusCancelled,
	}

	for _, from := range terminalStates {
		for _, to := range allStatuses {
			t.Run(from+"→"+to, func(t *testing.T) {
				session := &InterviewSession{Status: from}
				err := session.TransitionTo(to)
				assert.Error(t, err, "terminal state %q should not transition to %q", from, to)
				assert.Equal(t, from, session.Status)
			})
		}
	}
}

func TestInterviewSession_TransitionTo_EmptyTo(t *testing.T) {
	session := &InterviewSession{Status: StatusPending}
	err := session.TransitionTo("")
	assert.Error(t, err)
	assert.Equal(t, StatusPending, session.Status)
}

func TestInterviewSession_TransitionTo_UnknownStatus(t *testing.T) {
	session := &InterviewSession{Status: StatusActive}
	err := session.TransitionTo("bogus_status")
	assert.Error(t, err)
	assert.Equal(t, StatusActive, session.Status)
}

// ---------------------------------------------------------------------------
// Edge cases: Immutable state machine
// ---------------------------------------------------------------------------

func TestCanTransition_EmptyAndUnknownCombinations(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"empty from, empty to", "", ""},
		{"empty from, valid to", "", StatusPending},
		{"valid from, empty to", StatusPending, ""},
		{"both unknown", "foo", "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, CanTransition(tt.from, tt.to))
		})
	}
}

func TestIsValidStatus_NotInTransitionsMap(t *testing.T) {
	// Any string not a key in the transitions map is invalid
	assert.False(t, IsValidStatus("random"))
	assert.False(t, IsValidStatus("init"))
	assert.False(t, IsValidStatus("deleted"))
	assert.False(t, IsValidStatus("archived"))
}

func TestConstants_AreRecognisedByValidators(t *testing.T) {
	// Sanity check: each exported mode constant must pass IsValidMode
	assert.True(t, IsValidMode(ModeAssist))
	assert.True(t, IsValidMode(ModeAutonomous))

	// Sanity check: each exported status constant must pass IsValidStatus
	assert.True(t, IsValidStatus(StatusPending))
	assert.True(t, IsValidStatus(StatusStarting))
	assert.True(t, IsValidStatus(StatusActive))
	assert.True(t, IsValidStatus(StatusCompleted))
	assert.True(t, IsValidStatus(StatusFailed))
	assert.True(t, IsValidStatus(StatusCancelled))
}

// ---------------------------------------------------------------------------
// Round-trip serialisation (JSON)
// ---------------------------------------------------------------------------

func TestInterviewSession_JSONRoundTrip(t *testing.T) {
	extID := "sess_livekit_abc123"
	score := 92.0
	now := time.Now().UTC().Truncate(time.Second) // truncate for JSON precision
	feedback := json.RawMessage(`{"strengths":["communication","technical"]}`)
	startedAt := now.Add(-45 * time.Minute)
	endedAt := now
	id := uuid.New()
	appID := uuid.New()

	session := InterviewSession{
		ID:                id,
		ApplicationID:     appID,
		Mode:              ModeAutonomous,
		Status:            StatusCompleted,
		ExternalSessionID: &extID,
		Provider:          "elevenlabs",
		Model:             "eleven_multilingual_v2",
		Transcript: Transcript{
			{
				ID:        uuid.New(),
				Speaker:   SpeakerCandidate,
				Content:   "My name is John",
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

	data, err := json.Marshal(session)
	require.NoError(t, err)

	var decoded InterviewSession
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, session.ID, decoded.ID)
	assert.Equal(t, session.ApplicationID, decoded.ApplicationID)
	assert.Equal(t, session.Mode, decoded.Mode)
	assert.Equal(t, session.Status, decoded.Status)
	require.NotNil(t, decoded.ExternalSessionID)
	assert.Equal(t, *session.ExternalSessionID, *decoded.ExternalSessionID)
	assert.Equal(t, session.Provider, decoded.Provider)
	assert.Equal(t, session.Model, decoded.Model)
	assert.Len(t, decoded.Transcript, 1)
	require.NotNil(t, decoded.Score)
	assert.Equal(t, *session.Score, *decoded.Score)
	assert.JSONEq(t, string(session.Feedback), string(decoded.Feedback))
	require.NotNil(t, decoded.StartedAt)
	assert.Equal(t, session.StartedAt.Unix(), decoded.StartedAt.Unix())
	require.NotNil(t, decoded.EndedAt)
	assert.Equal(t, session.EndedAt.Unix(), decoded.EndedAt.Unix())
}

func TestInterviewSession_JSONOmitEmpty(t *testing.T) {
	// Zero-value session should omit empty pointer fields
	session := InterviewSession{
		ID:     uuid.New(),
		Mode:   ModeAssist,
		Status: StatusPending,
	}

	data, err := json.Marshal(session)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "external_session_id")
	assert.NotContains(t, string(data), "score")
	assert.NotContains(t, string(data), "feedback")
	assert.NotContains(t, string(data), "started_at")
	assert.NotContains(t, string(data), "ended_at")
}

// ---------------------------------------------------------------------------
// Edge case: New session defaults
// ---------------------------------------------------------------------------

func TestNewSessionDefaults(t *testing.T) {
	// Simulates what Service.Create builds
	session := &InterviewSession{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Mode:          ModeAssist,
		Status:        StatusPending,
		Provider:      "",
		Model:         "",
		Transcript:    Transcript{},
	}

	assert.Equal(t, StatusPending, session.Status)
	assert.Empty(t, session.Provider)
	assert.Empty(t, session.Model)
	assert.Empty(t, session.Transcript)
	assert.NotNil(t, session.Transcript) // explicitly set to empty slice
	assert.Nil(t, session.ExternalSessionID)
	assert.Nil(t, session.Score)
	assert.Nil(t, session.Feedback)
	assert.Nil(t, session.StartedAt)
	assert.Nil(t, session.EndedAt)

	// Transition from pending to starting should succeed
	err := session.TransitionTo(StatusStarting)
	assert.NoError(t, err)
	assert.Equal(t, StatusStarting, session.Status)
}
