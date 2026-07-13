package tasks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CreateTaskRequest ---

func TestCreateTaskRequest_JSONDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantType string
		wantErr  bool
	}{
		{
			name:     "valid type",
			json:     `{"type":"job_discovery"}`,
			wantType: "job_discovery",
		},
		{
			name:     "full valid",
			json:     `{"type":"job_scoring","params":{"job_id":"abc"},"priority":5,"scheduled_at":"2026-07-13T12:00:00Z"}`,
			wantType: "job_scoring",
		},
		{
			name:     "missing type — json.Unmarshal does not enforce required tag",
			json:     `{}`,
			wantType: "",
		},
		{
			name:     "empty type — valid JSON, empty string",
			json:     `{"type":""}`,
			wantType: "",
		},
		{
			name:     "null params",
			json:     `{"type":"embedding_generate","params":null}`,
			wantType: "embedding_generate",
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:     "zero priority",
			json:     `{"type":"cover_letter_gen","priority":0}`,
			wantType: "cover_letter_gen",
		},
		{
			name:     "negative priority",
			json:     `{"type":"interview_prep","priority":-1}`,
			wantType: "interview_prep",
		},
		{
			name:     "future scheduled_at",
			json:     `{"type":"voice_session","scheduled_at":"2026-12-31T23:59:59Z"}`,
			wantType: "voice_session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CreateTaskRequest
			err := json.Unmarshal([]byte(tt.json), &req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, req.Type)
		})
	}
}

func TestCreateTaskRequest_JSONRoundTrip(t *testing.T) {
	scheduledAt := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		request CreateTaskRequest
	}{
		{
			name: "minimal",
			request: CreateTaskRequest{
				Type: TypeJobDiscovery,
			},
		},
		{
			name: "with params",
			request: CreateTaskRequest{
				Type:   TypeEmbeddingGenerate,
				Params: json.RawMessage(`{"content":"hello world"}`),
			},
		},
		{
			name: "with priority",
			request: CreateTaskRequest{
				Type:     TypeJobScoring,
				Priority: 10,
			},
		},
		{
			name: "with scheduled_at",
			request: CreateTaskRequest{
				Type:        TypeResumeGenerate,
				ScheduledAt: &scheduledAt,
			},
		},
		{
			name: "all fields",
			request: CreateTaskRequest{
				Type:        TypeApplicationSubmit,
				Params:      json.RawMessage(`{"application_id":"abc"}`),
				Priority:    5,
				ScheduledAt: &scheduledAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)

			var decoded CreateTaskRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.request.Type, decoded.Type)
			assert.Equal(t, tt.request.Priority, decoded.Priority)

			// json.RawMessage(nil) marshals to "null", which unmarshals back to json.RawMessage("null").
			if tt.request.Params == nil {
				// Expect the round-tripped value to be JSON "null"
				assert.JSONEq(t, `null`, string(decoded.Params))
			} else {
				assert.JSONEq(t, string(tt.request.Params), string(decoded.Params))
			}

			if tt.request.ScheduledAt != nil {
				require.NotNil(t, decoded.ScheduledAt)
				assert.Equal(t, tt.request.ScheduledAt.Unix(), decoded.ScheduledAt.Unix())
			} else {
				assert.Nil(t, decoded.ScheduledAt)
			}
		})
	}
}

func TestCreateTaskRequest_SerializesZeroValues(t *testing.T) {
	// CreateTaskRequest does NOT have omitempty on its fields, so even
	// zero-values are serialised.
	req := CreateTaskRequest{Type: TypeEmailCheck}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.Contains(t, string(data), `"params":null`)
	require.Contains(t, string(data), `"priority":0`)
	require.Contains(t, string(data), `"scheduled_at":null`)
}

// --- ToResponse ---

func TestToResponse(t *testing.T) {
	now := time.Now().Truncate(time.Second).UTC()
	errMsg := "task failed"

	tests := []struct {
		name string
		task *Task
	}{
		{
			name: "pending task",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeJobDiscovery,
				Status:      StatusPending,
				Params:      json.RawMessage(`{"keywords":["golang"]}`),
				MaxAttempts: 3,
				Priority:    5,
				ScheduledAt: now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "running task",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeJobScoring,
				Status:      StatusRunning,
				Attempts:    1,
				MaxAttempts: 3,
				ScheduledAt: now,
				StartedAt:   &now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "completed task",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeEmbeddingGenerate,
				Status:      StatusCompleted,
				Result:      json.RawMessage(`{"chunks":10}`),
				Attempts:    1,
				MaxAttempts: 3,
				ScheduledAt: now,
				StartedAt:   &now,
				CompletedAt: &now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "failed task",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeApplicationSubmit,
				Status:      StatusFailed,
				Error:       &errMsg,
				Attempts:    3,
				MaxAttempts: 3,
				ScheduledAt: now,
				StartedAt:   &now,
				CompletedAt: &now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "cancelled task",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeVoiceSession,
				Status:      StatusCancelled,
				Attempts:    0,
				MaxAttempts: 1,
				ScheduledAt: now,
				CompletedAt: &now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "task with nil pointers",
			task: &Task{
				ID:          uuid.New(),
				Type:        TypeFillForm,
				Status:      StatusPending,
				MaxAttempts: 3,
				ScheduledAt: now,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ToResponse(tt.task)

			assert.Equal(t, tt.task.ID, resp.ID)
			assert.Equal(t, tt.task.Type, resp.Type)
			assert.Equal(t, tt.task.Status, resp.Status)
			assert.Equal(t, tt.task.Attempts, resp.Attempts)
			assert.Equal(t, tt.task.MaxAttempts, resp.MaxAttempts)
			assert.Equal(t, tt.task.Priority, resp.Priority)

			if tt.task.Params != nil {
				assert.JSONEq(t, string(tt.task.Params), string(resp.Params))
			} else {
				assert.Nil(t, resp.Params)
			}
			if tt.task.Result != nil {
				assert.JSONEq(t, string(tt.task.Result), string(resp.Result))
			} else {
				assert.Nil(t, resp.Result)
			}
			if tt.task.Error != nil {
				require.NotNil(t, resp.Error)
				assert.Equal(t, *tt.task.Error, *resp.Error)
			} else {
				assert.Nil(t, resp.Error)
			}
			if tt.task.StartedAt != nil {
				require.NotNil(t, resp.StartedAt)
				assert.Equal(t, tt.task.StartedAt.Unix(), resp.StartedAt.Unix())
			} else {
				assert.Nil(t, resp.StartedAt)
			}
			if tt.task.CompletedAt != nil {
				require.NotNil(t, resp.CompletedAt)
				assert.Equal(t, tt.task.CompletedAt.Unix(), resp.CompletedAt.Unix())
			} else {
				assert.Nil(t, resp.CompletedAt)
			}

			assert.Equal(t, tt.task.ScheduledAt.Unix(), resp.ScheduledAt.Unix())
			assert.Equal(t, tt.task.CreatedAt.Unix(), resp.CreatedAt.Unix())
			assert.Equal(t, tt.task.UpdatedAt.Unix(), resp.UpdatedAt.Unix())
		})
	}
}

func TestToResponse_NilTask(t *testing.T) {
	// ToResponse does not guard against nil — it dereferences the pointer,
	// so passing nil will panic.  This test documents that contract; callers
	// must ensure non-nil.
	assert.Panics(t, func() { ToResponse(nil) }, "ToResponse(nil) should panic")
}

// --- TaskListResponse ---

func TestTaskListResponse_JSONRoundTrip(t *testing.T) {
	resp := TaskListResponse{
		Tasks: []TaskResponse{
			{ID: uuid.New(), Type: TypeJobDiscovery, Status: StatusPending},
			{ID: uuid.New(), Type: TypeJobScoring, Status: StatusRunning},
		},
		Total: 2,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded TaskListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Total, decoded.Total)
	assert.Len(t, decoded.Tasks, 2)
	assert.Equal(t, resp.Tasks[0].ID, decoded.Tasks[0].ID)
	assert.Equal(t, resp.Tasks[0].Type, decoded.Tasks[0].Type)
	assert.Equal(t, resp.Tasks[1].Status, decoded.Tasks[1].Status)
}

func TestTaskListResponse_Empty(t *testing.T) {
	resp := TaskListResponse{
		Tasks: []TaskResponse{},
		Total: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded TaskListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded.Tasks)
	assert.Equal(t, 0, decoded.Total)
}

func TestTaskListResponse_NilTasks(t *testing.T) {
	resp := TaskListResponse{
		Tasks: nil,
		Total: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded TaskListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Nil(t, decoded.Tasks)
	assert.Equal(t, 0, decoded.Total)
}

// --- TaskResponse ---

func TestTaskResponse_JSONTags(t *testing.T) {
	id := uuid.New()
	now := time.Now().Truncate(time.Second).UTC()
	errMsg := "error"

	resp := TaskResponse{
		ID:          id,
		Type:        TypeCoverLetterGen,
		Status:      StatusFailed,
		Params:      json.RawMessage(`{"id":"abc"}`),
		Result:      json.RawMessage(`{"ok":true}`),
		Error:       &errMsg,
		Attempts:    2,
		MaxAttempts: 3,
		Priority:    1,
		ScheduledAt: now,
		StartedAt:   &now,
		CompletedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded TaskResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.ID, decoded.ID)
	assert.Equal(t, resp.Type, decoded.Type)
	assert.Equal(t, resp.Status, decoded.Status)
	assert.JSONEq(t, string(resp.Params), string(decoded.Params))
	assert.JSONEq(t, string(resp.Result), string(decoded.Result))
	require.NotNil(t, decoded.Error)
	assert.Equal(t, *resp.Error, *decoded.Error)
	assert.Equal(t, resp.Attempts, decoded.Attempts)
	assert.Equal(t, resp.MaxAttempts, decoded.MaxAttempts)
	assert.Equal(t, resp.Priority, decoded.Priority)
}

// --- Internal Payload DTOs (queue serialization) ---

func TestJobDiscoveryPayload_JSONRoundTrip(t *testing.T) {
	payload := JobDiscoveryPayload{
		SourceID:      uuid.New(),
		Keywords:      []string{"golang", "rust", "python"},
		Location:      "remote",
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded JobDiscoveryPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.SourceID, decoded.SourceID)
	assert.Equal(t, payload.Keywords, decoded.Keywords)
	assert.Equal(t, payload.Location, decoded.Location)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestJobDiscoveryPayload_EmptyKeywords(t *testing.T) {
	payload := JobDiscoveryPayload{
		SourceID:      uuid.New(),
		Keywords:      []string{},
		Location:      "",
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded JobDiscoveryPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.Keywords)
	assert.Empty(t, decoded.Location)
}

func TestJobScoringPayload_JSONRoundTrip(t *testing.T) {
	payload := JobScoringPayload{
		JobID:         uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded JobScoringPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.JobID, decoded.JobID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestApplicationSubmitPayload_JSONRoundTrip(t *testing.T) {
	payload := ApplicationSubmitPayload{
		ApplicationID: uuid.New(),
		FormData:      json.RawMessage(`{"field1":"value1","field2":42}`),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded ApplicationSubmitPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.ApplicationID, decoded.ApplicationID)
	assert.JSONEq(t, string(payload.FormData), string(decoded.FormData))
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestApplicationSubmitPayload_NilFormData(t *testing.T) {
	payload := ApplicationSubmitPayload{
		ApplicationID: uuid.New(),
		FormData:      nil,
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded ApplicationSubmitPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// json.RawMessage(nil) marshals to "null", which unmarshals back to
	// json.RawMessage("null") — not nil.  This is standard Go encoding/json
	// behaviour for RawMessage: null → []byte("null").
	require.NotNil(t, decoded.FormData)
	assert.JSONEq(t, `null`, string(decoded.FormData))
}

func TestEmbeddingPayload_JSONRoundTrip(t *testing.T) {
	payload := EmbeddingPayload{
		SourceType:    "job_description",
		SourceID:      uuid.New(),
		Content:       "This is a long job description...",
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded EmbeddingPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.SourceType, decoded.SourceType)
	assert.Equal(t, payload.SourceID, decoded.SourceID)
	assert.Equal(t, payload.Content, decoded.Content)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestEmbeddingPayload_EmptyContent(t *testing.T) {
	payload := EmbeddingPayload{
		SourceType:    "",
		SourceID:      uuid.New(),
		Content:       "",
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded EmbeddingPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.SourceType)
	assert.Empty(t, decoded.Content)
}

func TestCoverLetterGenPayload_JSONRoundTrip(t *testing.T) {
	payload := CoverLetterGenPayload{
		CoverLetterID: uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded CoverLetterGenPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.CoverLetterID, decoded.CoverLetterID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestResumeGeneratePayload_JSONRoundTrip(t *testing.T) {
	payload := ResumeGeneratePayload{
		JobID:         uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded ResumeGeneratePayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.JobID, decoded.JobID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestResumeTailorPayload_JSONRoundTrip(t *testing.T) {
	payload := ResumeTailorPayload{
		JobID:         uuid.New(),
		ResumeID:      uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded ResumeTailorPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.JobID, decoded.JobID)
	assert.Equal(t, payload.ResumeID, decoded.ResumeID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestEmailCheckPayload_JSONRoundTrip(t *testing.T) {
	payload := EmailCheckPayload{
		ApplicationID: uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded EmailCheckPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.ApplicationID, decoded.ApplicationID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestInterviewPrepPayload_JSONRoundTrip(t *testing.T) {
	payload := InterviewPrepPayload{
		ApplicationID: uuid.New(),
		CorrelationID: uuid.New(),
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded InterviewPrepPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.ApplicationID, decoded.ApplicationID)
	assert.Equal(t, payload.CorrelationID, decoded.CorrelationID)
}

func TestVoiceSessionPayload_JSONRoundTrip(t *testing.T) {
	payload := VoiceSessionPayload{
		InterviewID:     uuid.New(),
		ApplicationID:   uuid.New(),
		Mode:            "interview",
		ExternalSession: "session_abc123",
		Provider:        "livekit",
		Model:           "gpt-4o",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded VoiceSessionPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.InterviewID, decoded.InterviewID)
	assert.Equal(t, payload.ApplicationID, decoded.ApplicationID)
	assert.Equal(t, payload.Mode, decoded.Mode)
	assert.Equal(t, payload.ExternalSession, decoded.ExternalSession)
	assert.Equal(t, payload.Provider, decoded.Provider)
	assert.Equal(t, payload.Model, decoded.Model)
}

func TestVoiceSessionPayload_EmptyOptionalFields(t *testing.T) {
	// All string fields can be empty
	payload := VoiceSessionPayload{
		InterviewID:     uuid.New(),
		ApplicationID:   uuid.New(),
		Mode:            "",
		ExternalSession: "",
		Provider:        "",
		Model:           "",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded VoiceSessionPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.Mode)
	assert.Empty(t, decoded.ExternalSession)
	assert.Empty(t, decoded.Provider)
	assert.Empty(t, decoded.Model)
}

// --- UUID zero-value edge cases ---

func TestPayloads_NilUUID(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
	}{
		{name: "JobDiscoveryPayload", payload: JobDiscoveryPayload{
			SourceID: uuid.Nil, Keywords: nil, Location: "", CorrelationID: uuid.Nil,
		}},
		{name: "JobScoringPayload", payload: JobScoringPayload{
			JobID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
		{name: "ApplicationSubmitPayload", payload: ApplicationSubmitPayload{
			ApplicationID: uuid.Nil, FormData: nil, CorrelationID: uuid.Nil,
		}},
		{name: "CoverLetterGenPayload", payload: CoverLetterGenPayload{
			CoverLetterID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
		{name: "ResumeGeneratePayload", payload: ResumeGeneratePayload{
			JobID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
		{name: "ResumeTailorPayload", payload: ResumeTailorPayload{
			JobID: uuid.Nil, ResumeID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
		{name: "EmailCheckPayload", payload: EmailCheckPayload{
			ApplicationID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
		{name: "InterviewPrepPayload", payload: InterviewPrepPayload{
			ApplicationID: uuid.Nil, CorrelationID: uuid.Nil,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.payload)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}
