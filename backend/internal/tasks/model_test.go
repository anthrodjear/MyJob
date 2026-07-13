package tasks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Status Constants ---

func TestStatusConstants_Values(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "StatusPending", got: StatusPending, want: "pending"},
		{name: "StatusRunning", got: StatusRunning, want: "running"},
		{name: "StatusCompleted", got: StatusCompleted, want: "completed"},
		{name: "StatusFailed", got: StatusFailed, want: "failed"},
		{name: "StatusCancelled", got: StatusCancelled, want: "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestStatusConstants_Uniqueness(t *testing.T) {
	all := []string{StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled}
	seen := make(map[string]bool, len(all))
	for _, s := range all {
		assert.False(t, seen[s], "duplicate status constant: %s", s)
		seen[s] = true
	}
	// Sanity: we expect exactly 5 unique statuses
	assert.Len(t, seen, 5)
}

func TestStatusConstants_NonEmpty(t *testing.T) {
	all := []string{StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled}
	for _, s := range all {
		assert.NotEmpty(t, s, "status constant must not be empty")
	}
}

// --- Type Constants ---

func TestTypeConstants_Values(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "TypeJobDiscovery", got: TypeJobDiscovery, want: "job_discovery"},
		{name: "TypeJobScoring", got: TypeJobScoring, want: "job_scoring"},
		{name: "TypeApplicationSubmit", got: TypeApplicationSubmit, want: "application_submit"},
		{name: "TypeEmbeddingGenerate", got: TypeEmbeddingGenerate, want: "embedding_generate"},
		{name: "TypeCoverLetterGen", got: TypeCoverLetterGen, want: "cover_letter_gen"},
		{name: "TypeResumeGenerate", got: TypeResumeGenerate, want: "resume_generate"},
		{name: "TypeResumeTailor", got: TypeResumeTailor, want: "resume_tailor"},
		{name: "TypeEmailCheck", got: TypeEmailCheck, want: "email_check"},
		{name: "TypeInterviewPrep", got: TypeInterviewPrep, want: "interview_prep"},
		{name: "TypeVoiceSession", got: TypeVoiceSession, want: "voice_session"},
		{name: "TypeFillForm", got: TypeFillForm, want: "fill_form"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestTypeConstants_Uniqueness(t *testing.T) {
	all := []string{
		TypeJobDiscovery, TypeJobScoring, TypeApplicationSubmit,
		TypeEmbeddingGenerate, TypeCoverLetterGen, TypeResumeGenerate,
		TypeResumeTailor, TypeEmailCheck, TypeInterviewPrep,
		TypeVoiceSession, TypeFillForm,
	}
	seen := make(map[string]bool, len(all))
	for _, tt := range all {
		assert.False(t, seen[tt], "duplicate type constant: %s", tt)
		seen[tt] = true
	}
	assert.Len(t, seen, 11)
}

func TestTypeConstants_NonEmpty(t *testing.T) {
	all := []string{
		TypeJobDiscovery, TypeJobScoring, TypeApplicationSubmit,
		TypeEmbeddingGenerate, TypeCoverLetterGen, TypeResumeGenerate,
		TypeResumeTailor, TypeEmailCheck, TypeInterviewPrep,
		TypeVoiceSession, TypeFillForm,
	}
	for _, tt := range all {
		assert.NotEmpty(t, tt, "type constant must not be empty")
	}
}

// Status/type naming convention: all lower_snake_case
func TestStatusConstants_Format(t *testing.T) {
	all := []string{StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled}
	for _, s := range all {
		assert.Regexp(t, `^[a-z_]+$`, s, "status %q should be lower_snake_case", s)
	}
}

func TestTypeConstants_Format(t *testing.T) {
	all := []string{
		TypeJobDiscovery, TypeJobScoring, TypeApplicationSubmit,
		TypeEmbeddingGenerate, TypeCoverLetterGen, TypeResumeGenerate,
		TypeResumeTailor, TypeEmailCheck, TypeInterviewPrep,
		TypeVoiceSession, TypeFillForm,
	}
	for _, tt := range all {
		assert.Regexp(t, `^[a-z_]+$`, tt, "type %q should be lower_snake_case", tt)
	}
}

// --- TableName ---

func TestTask_TableName(t *testing.T) {
	task := Task{}
	assert.Equal(t, "tasks", task.TableName())
}

// TableName receiver must not mutate the instance
func TestTask_TableName_Receiver(t *testing.T) {
	task := Task{Type: TypeJobDiscovery}
	name := task.TableName()
	assert.Equal(t, "tasks", name)
	// Ensure no side effects
	assert.Equal(t, TypeJobDiscovery, task.Type)
}

// --- Default / Zero Value ---

func TestTask_DefaultValues(t *testing.T) {
	task := Task{}

	assert.Equal(t, uuid.Nil, task.ID, "ID should be zero-value UUID")
	assert.Empty(t, task.Type, "Type should be empty string")
	assert.Empty(t, task.Status, "Status should be empty string")
	assert.Nil(t, task.Params, "Params should be nil")
	assert.Nil(t, task.Result, "Result should be nil")
	assert.Nil(t, task.Error, "Error should be nil")
	assert.Equal(t, 0, task.Attempts, "Attempts should be 0")
	assert.Equal(t, 0, task.MaxAttempts, "MaxAttempts should be 0")
	assert.Equal(t, 0, task.Priority, "Priority should be 0")
	assert.True(t, task.ScheduledAt.IsZero(), "ScheduledAt should be zero time")
	assert.Nil(t, task.StartedAt, "StartedAt should be nil")
	assert.Nil(t, task.CompletedAt, "CompletedAt should be nil")
	assert.True(t, task.CreatedAt.IsZero(), "CreatedAt should be zero time")
	assert.True(t, task.UpdatedAt.IsZero(), "UpdatedAt should be zero time")
}

// --- JSON Serialization (full round-trip) ---

func TestTask_JSONRoundTrip(t *testing.T) {
	id := uuid.New()
	now := time.Now().Truncate(time.Second).UTC()
	errMsg := "something went wrong"

	task := Task{
		ID:          id,
		Type:        TypeJobDiscovery,
		Status:      StatusPending,
		Params:      json.RawMessage(`{"key":"value"}`),
		Result:      json.RawMessage(`{"result":"ok"}`),
		Error:       &errMsg,
		Attempts:    1,
		MaxAttempts: 3,
		Priority:    5,
		ScheduledAt: now,
		StartedAt:   &now,
		CompletedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.Type, decoded.Type)
	assert.Equal(t, task.Status, decoded.Status)
	assert.JSONEq(t, string(task.Params), string(decoded.Params))
	assert.JSONEq(t, string(task.Result), string(decoded.Result))
	require.NotNil(t, decoded.Error)
	assert.Equal(t, errMsg, *decoded.Error)
	assert.Equal(t, task.Attempts, decoded.Attempts)
	assert.Equal(t, task.MaxAttempts, decoded.MaxAttempts)
	assert.Equal(t, task.Priority, decoded.Priority)
	assert.Equal(t, task.ScheduledAt.Unix(), decoded.ScheduledAt.Unix())
	require.NotNil(t, decoded.StartedAt)
	assert.Equal(t, task.StartedAt.Unix(), decoded.StartedAt.Unix())
	require.NotNil(t, decoded.CompletedAt)
	assert.Equal(t, task.CompletedAt.Unix(), decoded.CompletedAt.Unix())
	assert.Equal(t, task.CreatedAt.Unix(), decoded.CreatedAt.Unix())
	assert.Equal(t, task.UpdatedAt.Unix(), decoded.UpdatedAt.Unix())
}

func TestTask_JSONRoundTrip_NilPointersAndSlices(t *testing.T) {
	task := Task{
		ID:     uuid.New(),
		Type:   TypeJobScoring,
		Status: StatusRunning,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	// Fields tagged omitempty should be absent when nil/zero
	assert.NotContains(t, string(data), "params")
	assert.NotContains(t, string(data), "result")
	assert.NotContains(t, string(data), "error")
	assert.NotContains(t, string(data), "started_at")
	assert.NotContains(t, string(data), "completed_at")

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Params)
	assert.Nil(t, decoded.Result)
	assert.Nil(t, decoded.Error)
	assert.Nil(t, decoded.StartedAt)
	assert.Nil(t, decoded.CompletedAt)
}

func TestTask_JSONRoundTrip_EmptyError(t *testing.T) {
	emptyErr := ""
	task := Task{
		ID:     uuid.New(),
		Type:   TypeEmailCheck,
		Status: StatusFailed,
		Error:  &emptyErr,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.NotNil(t, decoded.Error)
	assert.Equal(t, "", *decoded.Error)
}

func TestTask_JSONRoundTrip_ZeroIntFields(t *testing.T) {
	task := Task{
		ID:     uuid.New(),
		Type:   TypeFillForm,
		Status: StatusCancelled,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, 0, decoded.Attempts)
	assert.Equal(t, 0, decoded.MaxAttempts)
	assert.Equal(t, 0, decoded.Priority)
}

// --- DB Tags ---

func TestTask_DBTags(t *testing.T) {
	// Compile-time check: Task struct has JSON tags
	// We marshal to JSON as a proxy — db tags aren't directly testable
	// but all fields should be accessible
	task := Task{ID: uuid.New(), Type: TypeResumeGenerate, Status: StatusCompleted}
	data, err := json.Marshal(task)
	require.NoError(t, err)
	require.Contains(t, string(data), `"id"`)
	require.Contains(t, string(data), `"type"`)
	require.Contains(t, string(data), `"status"`)
}

// --- Task with all status/type combinations ---

func TestTask_AllStatusCombinations(t *testing.T) {
	types := []string{
		TypeJobDiscovery, TypeJobScoring, TypeApplicationSubmit,
		TypeEmbeddingGenerate, TypeCoverLetterGen, TypeResumeGenerate,
		TypeResumeTailor, TypeEmailCheck, TypeInterviewPrep,
		TypeVoiceSession, TypeFillForm,
	}
	statuses := []string{StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled}

	for _, tt := range types {
		for _, st := range statuses {
			t.Run(tt+"/"+st, func(t *testing.T) {
				task := Task{
					ID:     uuid.New(),
					Type:   tt,
					Status: st,
				}
				data, err := json.Marshal(task)
				assert.NoError(t, err)

				var decoded Task
				err = json.Unmarshal(data, &decoded)
				assert.NoError(t, err)
				assert.Equal(t, task.ID, decoded.ID)
				assert.Equal(t, task.Type, decoded.Type)
				assert.Equal(t, task.Status, decoded.Status)
			})
		}
	}
}
