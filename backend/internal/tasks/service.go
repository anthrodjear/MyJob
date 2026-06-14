package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors for the tasks domain.
var (
	ErrNotFound      = fmt.Errorf("tasks: not found")
	ErrInvalidType   = fmt.Errorf("tasks: invalid task type")
	ErrInvalidStatus = fmt.Errorf("tasks: invalid status transition")
)

// validTypes is the set of allowed task types.
var validTypes = map[string]bool{
	TypeJobDiscovery:      true,
	TypeResumeScoring:     true,
	TypeApplicationSubmit: true,
	TypeEmbeddingGenerate: true,
	TypeCoverLetterGen:    true,
	TypeResumeTailor:      true,
	TypeEmailCheck:        true,
	TypeInterviewPrep:     true,
}

// validTransitions defines which status transitions are allowed.
var validTransitions = map[string][]string{
	StatusPending:   {StatusRunning, StatusCancelled},
	StatusRunning:   {StatusCompleted, StatusFailed, StatusCancelled},
	StatusCompleted: {},
	StatusFailed:    {},
	StatusCancelled: {},
}

// Service handles business logic for async tasks.
type Service struct {
	repo *Repository
}

// NewService creates a new tasks service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// --- Internal helpers ---

// getTask fetches a task by ID, translating sql.ErrNoRows to ErrNotFound.
func (s *Service) getTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return task, nil
}

// canTransition checks whether a status transition is allowed.
func canTransition(from, to string) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// --- Public API ---

// Create creates a new task with pending status.
func (s *Service) Create(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	if !validTypes[req.Type] {
		return nil, ErrInvalidType
	}

	task := &Task{
		ID:          uuid.New(),
		Type:        req.Type,
		Status:      StatusPending,
		Params:      req.Params,
		MaxAttempts: 3,
		Priority:    req.Priority,
	}

	if req.ScheduledAt != nil {
		task.ScheduledAt = *req.ScheduledAt
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks: create: %w", err)
	}

	return task, nil
}

// GetByID retrieves a task by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.getTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tasks: get by id: %w", err)
	}
	return task, nil
}

// maxListLimit is the upper bound for list queries to prevent memory exhaustion.
const maxListLimit = 100

// List returns paginated tasks filtered by status and type.
func (s *Service) List(ctx context.Context, status, taskType string, limit, offset int) ([]Task, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	tasks, total, err := s.repo.List(ctx, status, taskType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("tasks: list: %w", err)
	}
	return tasks, total, nil
}

// Start transitions a task from pending to running.
func (s *Service) Start(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.getTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tasks: start: %w", err)
	}
	if !canTransition(task.Status, StatusRunning) {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	task.Status = StatusRunning
	task.Attempts++
	task.StartedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks: start update: %w", err)
	}
	return task, nil
}

// Complete marks a running task as completed with a result payload.
func (s *Service) Complete(ctx context.Context, id uuid.UUID, result json.RawMessage) (*Task, error) {
	task, err := s.getTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tasks: complete: %w", err)
	}
	if !canTransition(task.Status, StatusCompleted) {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	task.Status = StatusCompleted
	task.Result = result
	task.CompletedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks: complete update: %w", err)
	}
	return task, nil
}

// Fail marks a running task as failed. Retries if attempts remain.
func (s *Service) Fail(ctx context.Context, id uuid.UUID, errMsg string) (*Task, error) {
	task, err := s.getTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tasks: fail: %w", err)
	}
	if !canTransition(task.Status, StatusFailed) {
		return nil, ErrInvalidStatus
	}

	task.Error = &errMsg

	// Retry: re-queue with backoff if attempts remain
	if task.Attempts < task.MaxAttempts {
		task.Status = StatusPending
		task.StartedAt = nil
		backoff := time.Duration(task.Attempts) * 30 * time.Second
		task.ScheduledAt = time.Now().Add(backoff)
	} else {
		task.Status = StatusFailed
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks: fail update: %w", err)
	}
	return task, nil
}

// Cancel transitions a task from pending or running to cancelled.
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.getTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tasks: cancel: %w", err)
	}
	if !canTransition(task.Status, StatusCancelled) {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	task.Status = StatusCancelled
	task.CompletedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks: cancel update: %w", err)
	}
	return task, nil
}
