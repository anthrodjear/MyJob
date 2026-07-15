package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	TypeVoiceSession:      {Retries: 1, Timeout: 30 * time.Minute},
	TypeFillForm:          {Retries: 3, Timeout: 10 * time.Minute},
}

// Dispatcher enqueues tasks to the asynq queue and creates task records in the database.
// It provides a unified dispatch layer that keeps the DB and queue in sync.
type Dispatcher struct {
	client  *asynq.Client
	service *Service
	logger  *zap.Logger
}

// NewDispatcher creates a new task dispatcher.
// If service is non-nil, dispatch creates a DB record before enqueuing and returns the DB UUID.
func NewDispatcher(client *asynq.Client, service *Service, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{client: client, service: service, logger: logger}
}

// dispatch is the internal helper that all public methods delegate to.
// When a service is configured it creates the DB record first, then enqueues
// the asynq task with the DB UUID stored as metadata.  Returns the DB UUID
// for status tracking.
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

	// --- 1. Create task record in DB (before enqueue) ---
	var dbTaskID uuid.UUID
	if d.service != nil {
		task, err := d.service.Create(ctx, CreateTaskRequest{
			Type:   taskType,
			Params: data,
		})
		if err != nil {
			return "", fmt.Errorf("tasks: create db record %s: %w", taskType, err)
		}
		dbTaskID = task.ID
	}

	// --- 2. Enqueue to asynq ---
	task := asynq.NewTask(taskType, data)
	// Use the DB UUID as the asynq task ID so workers can look up the DB
	// record directly via the asynq task ID they receive.
	var enqueueOpts []asynq.Option
	enqueueOpts = append(enqueueOpts, asynq.MaxRetry(cfg.Retries), asynq.Timeout(cfg.Timeout))
	if d.service != nil {
		enqueueOpts = append(enqueueOpts, asynq.TaskID(dbTaskID.String()))
	}
	info, err := d.client.EnqueueContext(ctx, task, enqueueOpts...)
	if err != nil {
		// Enqueue failed — clean up the orphaned DB row
		if d.service != nil {
			if _, cancelErr := d.service.Cancel(ctx, dbTaskID); cancelErr != nil {
				d.logger.Warn("failed to clean up db record after enqueue error",
					zap.String("task_type", taskType),
					zap.String("db_task_id", dbTaskID.String()),
					zap.Error(cancelErr),
				)
			}
		}
		return "", fmt.Errorf("tasks: enqueue %s: %w", taskType, err)
	}

	d.logger.Debug("task dispatched",
		zap.String("task_type", taskType),
		zap.String("asynq_id", info.ID),
		zap.String("db_task_id", dbTaskID.String()),
		zap.Int("retries", cfg.Retries),
		zap.Duration("timeout", cfg.Timeout),
	)

	// Return the DB UUID for callers to track status; fall back to asynq ID when no service.
	if d.service != nil {
		return dbTaskID.String(), nil
	}
	return info.ID, nil
}

// --- Public dispatch methods (thin wrappers for type-safe API) ---
// Each returns the DB task UUID for status tracking.

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

func (d *Dispatcher) DispatchVoiceSession(ctx context.Context, payload VoiceSessionPayload) (string, error) {
	return d.dispatch(ctx, TypeVoiceSession, payload)
}
