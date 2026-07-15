package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestDispatcher creates a Dispatcher with a real asynq.Client pointed at a
// non-existent Redis port.  This lets us test error paths (connection refused)
// and verify that error messages are correctly wrapped with the task type —
// without requiring a running Redis instance.
// The service is nil so dispatch skips the DB step and exercises the asynq-only path.
func newTestDispatcher(t *testing.T) *Dispatcher {
	t.Helper()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { client.Close() })
	return NewDispatcher(client, nil, zap.NewNop())
}

// ---------------------------------------------------------------------------
// taskConfig completeness
// ---------------------------------------------------------------------------

func TestTaskConfig_AllTypesPresent(t *testing.T) {
	expectedTypes := []string{
		TypeJobDiscovery,
		TypeJobScoring,
		TypeApplicationSubmit,
		TypeEmbeddingGenerate,
		TypeCoverLetterGen,
		TypeResumeGenerate,
		TypeResumeTailor,
		TypeEmailCheck,
		TypeInterviewPrep,
		TypeVoiceSession,
	}

	for _, tt := range expectedTypes {
		t.Run(tt, func(t *testing.T) {
			cfg, ok := taskConfig[tt]
			assert.True(t, ok, "taskConfig missing entry for %q", tt)
			assert.GreaterOrEqual(t, cfg.Retries, 0, "retries must be >= 0 for %q", tt)
			assert.Greater(t, cfg.Timeout, time.Duration(0), "timeout must be > 0 for %q", tt)
		})
	}
}

func TestTaskConfig_NoExtraneousTypes(t *testing.T) {
	for key := range taskConfig {
		switch key {
		case TypeJobDiscovery, TypeJobScoring, TypeApplicationSubmit,
			TypeEmbeddingGenerate, TypeCoverLetterGen, TypeResumeGenerate,
			TypeResumeTailor, TypeEmailCheck, TypeInterviewPrep,
			TypeVoiceSession:
			// known
		default:
			t.Errorf("unexpected key in taskConfig: %q", key)
		}
	}
}

func TestTaskConfig_Values(t *testing.T) {
	tests := []struct {
		typ         string
		wantRetries int
		wantTimeout time.Duration
	}{
		{typ: TypeJobDiscovery, wantRetries: 3, wantTimeout: 5 * time.Minute},
		{typ: TypeJobScoring, wantRetries: 3, wantTimeout: 2 * time.Minute},
		{typ: TypeApplicationSubmit, wantRetries: 3, wantTimeout: 10 * time.Minute},
		{typ: TypeEmbeddingGenerate, wantRetries: 5, wantTimeout: 1 * time.Minute},
		{typ: TypeCoverLetterGen, wantRetries: 3, wantTimeout: 3 * time.Minute},
		{typ: TypeResumeGenerate, wantRetries: 3, wantTimeout: 3 * time.Minute},
		{typ: TypeResumeTailor, wantRetries: 3, wantTimeout: 3 * time.Minute},
		{typ: TypeEmailCheck, wantRetries: 5, wantTimeout: 1 * time.Minute},
		{typ: TypeInterviewPrep, wantRetries: 3, wantTimeout: 5 * time.Minute},
		{typ: TypeVoiceSession, wantRetries: 1, wantTimeout: 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			cfg, ok := taskConfig[tt.typ]
			require.True(t, ok, "taskConfig missing entry for %q", tt.typ)
			assert.Equal(t, tt.wantRetries, cfg.Retries,
				"%q: expected %d retries, got %d", tt.typ, tt.wantRetries, cfg.Retries)
			assert.Equal(t, tt.wantTimeout, cfg.Timeout,
				"%q: expected %v timeout, got %v", tt.typ, tt.wantTimeout, cfg.Timeout)
		})
	}
}

// ---------------------------------------------------------------------------
// NewDispatcher
// ---------------------------------------------------------------------------

func TestNewDispatcher(t *testing.T) {
	logger := zap.NewNop()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	defer client.Close()

	d := NewDispatcher(client, nil, logger)
	require.NotNil(t, d)
	assert.Equal(t, client, d.client)
	assert.Equal(t, logger, d.logger)
}

func TestNewDispatcher_NilLogger(t *testing.T) {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	defer client.Close()

	// The constructor does not guard against nil logger — caller's
	// responsibility.  We just verify it doesn't panic at construction.
	d := NewDispatcher(client, nil, nil)
	require.NotNil(t, d)
	assert.Nil(t, d.logger)
}

// ---------------------------------------------------------------------------
// dispatch — marshal error path
// ---------------------------------------------------------------------------

// unencodable is a type that cannot be marshalled to JSON (contains a channel).
type unencodable struct {
	Ch chan int
}

func TestDispatch_MarshalError(t *testing.T) {
	logger := zap.NewNop()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	defer client.Close()

	d := NewDispatcher(client, nil, logger)

	// A channel cannot be serialised → marshal fails before enqueue.
	payload := unencodable{Ch: make(chan int)}
	taskID, err := d.dispatch(context.Background(), TypeJobDiscovery, payload)

	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tasks: marshal")
	assert.Contains(t, err.Error(), TypeJobDiscovery)
}

func TestDispatch_MarshalError_NilPayload(t *testing.T) {
	logger := zap.NewNop()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	defer client.Close()

	d := NewDispatcher(client, nil, logger)

	// nil is a valid JSON value ("null") — no marshal error expected here.
	taskID, err := d.dispatch(context.Background(), TypeInterviewPrep, nil)
	// We expect an enqueue error (no Redis), not a marshal error
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tasks: enqueue")
}

// ---------------------------------------------------------------------------
// dispatch — enqueue error path (no Redis)
// ---------------------------------------------------------------------------

func TestDispatch_EnqueueError(t *testing.T) {
	d := newTestDispatcher(t)

	taskID, err := d.dispatch(context.Background(), TypeJobDiscovery, JobDiscoveryPayload{
		SourceID: uuid.New(),
		Keywords: []string{"golang"},
		Location: "remote",
	})

	assert.Empty(t, taskID)
	require.Error(t, err)
	// The error should be wrapped by dispatch
	assert.Contains(t, err.Error(), "tasks: enqueue")
	assert.Contains(t, err.Error(), TypeJobDiscovery)
}

// ---------------------------------------------------------------------------
// Dispatch routing — every public method routes to the correct type
// ---------------------------------------------------------------------------

func TestDispatchJobDiscovery_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := JobDiscoveryPayload{
		SourceID:      uuid.New(),
		Keywords:      []string{"golang", "rust"},
		Location:      "remote",
		CorrelationID: uuid.New(),
	}
	taskID, err := d.DispatchJobDiscovery(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeJobDiscovery)
}

func TestDispatchJobDiscovery_ZeroValuePayload(t *testing.T) {
	d := newTestDispatcher(t)
	payload := JobDiscoveryPayload{}
	taskID, err := d.DispatchJobDiscovery(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeJobDiscovery)
}

func TestDispatchJobScoring_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := JobScoringPayload{JobID: uuid.New(), CorrelationID: uuid.New()}
	taskID, err := d.DispatchJobScoring(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeJobScoring)
}

func TestDispatchApplicationSubmit_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := ApplicationSubmitPayload{
		ApplicationID: uuid.New(),
		FormData:      nil,
		CorrelationID: uuid.New(),
	}
	taskID, err := d.DispatchApplicationSubmit(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeApplicationSubmit)
}

func TestDispatchEmbeddingGenerate_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := EmbeddingPayload{
		SourceType:    "job",
		SourceID:      uuid.New(),
		Content:       "some content",
		CorrelationID: uuid.New(),
	}
	taskID, err := d.DispatchEmbeddingGenerate(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeEmbeddingGenerate)
}

func TestDispatchCoverLetterGen_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := CoverLetterGenPayload{CoverLetterID: uuid.New(), CorrelationID: uuid.New()}
	taskID, err := d.DispatchCoverLetterGen(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeCoverLetterGen)
}

func TestDispatchResumeGenerate_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := ResumeGeneratePayload{JobID: uuid.New(), CorrelationID: uuid.New()}
	taskID, err := d.DispatchResumeGenerate(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeResumeGenerate)
}

func TestDispatchResumeTailor_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := ResumeTailorPayload{
		JobID:         uuid.New(),
		ResumeID:      uuid.New(),
		CorrelationID: uuid.New(),
	}
	taskID, err := d.DispatchResumeTailor(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeResumeTailor)
}

func TestDispatchEmailCheck_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := EmailCheckPayload{ApplicationID: uuid.New(), CorrelationID: uuid.New()}
	taskID, err := d.DispatchEmailCheck(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeEmailCheck)
}

func TestDispatchInterviewPrep_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := InterviewPrepPayload{ApplicationID: uuid.New(), CorrelationID: uuid.New()}
	taskID, err := d.DispatchInterviewPrep(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeInterviewPrep)
}

func TestDispatchVoiceSession_RoutesCorrectType(t *testing.T) {
	d := newTestDispatcher(t)
	payload := VoiceSessionPayload{
		InterviewID:     uuid.New(),
		ApplicationID:   uuid.New(),
		Mode:            "interview",
		ExternalSession: "ext_sess",
		Provider:        "livekit",
		Model:           "gpt-4o",
	}
	taskID, err := d.DispatchVoiceSession(context.Background(), payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), TypeVoiceSession)
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestDispatch_CancelledContext(t *testing.T) {
	d := newTestDispatcher(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	payload := JobDiscoveryPayload{
		SourceID:      uuid.New(),
		Keywords:      []string{"golang"},
		Location:      "remote",
		CorrelationID: uuid.New(),
	}

	taskID, err := d.dispatch(ctx, TypeJobDiscovery, payload)
	assert.Empty(t, taskID)
	require.Error(t, err)
	// The context cancellation error should be wrapped
	assert.Contains(t, err.Error(), TypeJobDiscovery)
}

func TestDispatch_CancelledContextOnEachMethod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		typ      string
		dispatch func(d *Dispatcher, ctx context.Context) (string, error)
	}{
		{
			name: TypeJobDiscovery,
			typ:  TypeJobDiscovery,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchJobDiscovery(ctx, JobDiscoveryPayload{SourceID: uuid.New()})
			},
		},
		{
			name: TypeJobScoring,
			typ:  TypeJobScoring,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchJobScoring(ctx, JobScoringPayload{JobID: uuid.New()})
			},
		},
		{
			name: TypeResumeGenerate,
			typ:  TypeResumeGenerate,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchResumeGenerate(ctx, ResumeGeneratePayload{JobID: uuid.New()})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDispatcher(t)
			taskID, err := tt.dispatch(d, ctx)
			assert.Empty(t, taskID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.typ)
		})
	}
}

// ---------------------------------------------------------------------------
// Zero-value / nil-safe payload dispatch
// ---------------------------------------------------------------------------

func TestDispatch_ZeroValuePayloads(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		typ      string
		payload  interface{}
		dispatch func(d *Dispatcher, ctx context.Context) (string, error)
	}{
		{
			name: "JobDiscoveryPayload",
			typ:  TypeJobDiscovery,
			payload: JobDiscoveryPayload{
				SourceID: uuid.Nil, Keywords: nil, Location: "", CorrelationID: uuid.Nil,
			},
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchJobDiscovery(ctx, JobDiscoveryPayload{})
			},
		},
		{
			name: "JobScoringPayload",
			typ:  TypeJobScoring,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchJobScoring(ctx, JobScoringPayload{})
			},
		},
		{
			name: "ApplicationSubmitPayload",
			typ:  TypeApplicationSubmit,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchApplicationSubmit(ctx, ApplicationSubmitPayload{})
			},
		},
		{
			name: "EmbeddingPayload",
			typ:  TypeEmbeddingGenerate,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchEmbeddingGenerate(ctx, EmbeddingPayload{})
			},
		},
		{
			name: "CoverLetterGenPayload",
			typ:  TypeCoverLetterGen,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchCoverLetterGen(ctx, CoverLetterGenPayload{})
			},
		},
		{
			name: "ResumeGeneratePayload",
			typ:  TypeResumeGenerate,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchResumeGenerate(ctx, ResumeGeneratePayload{})
			},
		},
		{
			name: "ResumeTailorPayload",
			typ:  TypeResumeTailor,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchResumeTailor(ctx, ResumeTailorPayload{})
			},
		},
		{
			name: "EmailCheckPayload",
			typ:  TypeEmailCheck,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchEmailCheck(ctx, EmailCheckPayload{})
			},
		},
		{
			name: "InterviewPrepPayload",
			typ:  TypeInterviewPrep,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchInterviewPrep(ctx, InterviewPrepPayload{})
			},
		},
		{
			name: "VoiceSessionPayload",
			typ:  TypeVoiceSession,
			dispatch: func(d *Dispatcher, ctx context.Context) (string, error) {
				return d.DispatchVoiceSession(ctx, VoiceSessionPayload{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDispatcher(t)
			taskID, err := tt.dispatch(d, ctx)
			// Zero-value payloads are valid JSON — they should reach the enqueue step
			assert.Empty(t, taskID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "tasks: enqueue")
			assert.Contains(t, err.Error(), tt.typ)
		})
	}
}

// ---------------------------------------------------------------------------
// Concurrency safety (basic)
// ---------------------------------------------------------------------------

func TestDispatch_ConcurrentCalls(t *testing.T) {
	d := newTestDispatcher(t)
	ctx := context.Background()

	payload := JobDiscoveryPayload{
		SourceID:      uuid.New(),
		Keywords:      []string{"golang"},
		Location:      "remote",
		CorrelationID: uuid.New(),
	}

	// fire N concurrent dispatch attempts
	const N = 10
	errs := make(chan error, N)
	for range N {
		go func() {
			_, err := d.DispatchJobDiscovery(ctx, payload)
			errs <- err
		}()
	}

	for range N {
		err := <-errs
		require.Error(t, err)
		// All errors should reference the same type
		assert.Contains(t, err.Error(), TypeJobDiscovery)
	}
}
