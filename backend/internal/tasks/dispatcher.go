package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// taskConfig defines retry and timeout settings per task type.
var taskConfig = map[string]struct {
	Retries int
	Timeout time.Duration
}{
	TypeJobDiscovery:      {Retries: 3, Timeout: 5 * time.Minute},
	TypeJobScoring:        {Retries: 3, Timeout: 2 * time.Minute},
	TypeApplicationSubmit: {Retries: 3, Timeout: 10 * time.Minute},
	TypeEmbeddingGenerate: {Retries: 5, Timeout: 1 * time.Minute},
	TypeCoverLetterGen:    {Retries: 3, Timeout: 3 * time.Minute},
	TypeResumeGenerate:    {Retries: 3, Timeout: 3 * time.Minute},
	TypeResumeTailor:      {Retries: 3, Timeout: 3 * time.Minute},
	TypeEmailCheck:        {Retries: 5, Timeout: 1 * time.Minute},
	TypeInterviewPrep:     {Retries: 3, Timeout: 5 * time.Minute},
}

// Dispatcher enqueues tasks to the asynq queue.
// It is a pure dispatch layer — no DB calls, no business logic.
type Dispatcher struct {
	client *asynq.Client
	logger *zap.Logger
}

// NewDispatcher creates a new task dispatcher.
func NewDispatcher(client *asynq.Client, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{client: client, logger: logger}
}

// dispatch is the internal helper that all public methods delegate to.
// Returns the Asynq task ID for polling.
func (d *Dispatcher) dispatch(
	ctx context.Context,
	taskType string,
	payload interface{},
) (string, error) {
	cfg := taskConfig[taskType]

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("tasks: marshal %s: %w", taskType, err)
	}

	task := asynq.NewTask(taskType, data)
	info, err := d.client.EnqueueContext(ctx, task, asynq.MaxRetry(cfg.Retries), asynq.Timeout(cfg.Timeout))
	if err != nil {
		return "", fmt.Errorf("tasks: enqueue %s: %w", taskType, err)
	}

	d.logger.Debug("task dispatched",
		zap.String("task_type", taskType),
		zap.String("asynq_id", info.ID),
		zap.Int("retries", cfg.Retries),
		zap.Duration("timeout", cfg.Timeout),
	)

	return info.ID, nil
}

// --- Public dispatch methods (thin wrappers for type-safe API) ---
// Each returns the Asynq task ID for polling.

func (d *Dispatcher) DispatchJobDiscovery(ctx context.Context, payload JobDiscoveryPayload) (string, error) {
	return d.dispatch(ctx, TypeJobDiscovery, payload)
}

func (d *Dispatcher) DispatchJobScoring(ctx context.Context, payload JobScoringPayload) (string, error) {
	return d.dispatch(ctx, TypeJobScoring, payload)
}

func (d *Dispatcher) DispatchApplicationSubmit(ctx context.Context, payload ApplicationSubmitPayload) (string, error) {
	return d.dispatch(ctx, TypeApplicationSubmit, payload)
}

func (d *Dispatcher) DispatchEmbeddingGenerate(ctx context.Context, payload EmbeddingPayload) (string, error) {
	return d.dispatch(ctx, TypeEmbeddingGenerate, payload)
}

func (d *Dispatcher) DispatchCoverLetterGen(ctx context.Context, payload CoverLetterGenPayload) (string, error) {
	return d.dispatch(ctx, TypeCoverLetterGen, payload)
}

func (d *Dispatcher) DispatchResumeGenerate(ctx context.Context, payload ResumeGeneratePayload) (string, error) {
	return d.dispatch(ctx, TypeResumeGenerate, payload)
}

func (d *Dispatcher) DispatchResumeTailor(ctx context.Context, payload ResumeTailorPayload) (string, error) {
	return d.dispatch(ctx, TypeResumeTailor, payload)
}

func (d *Dispatcher) DispatchEmailCheck(ctx context.Context, payload EmailCheckPayload) (string, error) {
	return d.dispatch(ctx, TypeEmailCheck, payload)
}

func (d *Dispatcher) DispatchInterviewPrep(ctx context.Context, payload InterviewPrepPayload) (string, error) {
	return d.dispatch(ctx, TypeInterviewPrep, payload)
}
