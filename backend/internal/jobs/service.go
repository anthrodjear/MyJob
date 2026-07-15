package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/tasks"

	"github.com/google/uuid"
)

// Service handles job business logic.
type Service struct {
	repo       *Repository
	dispatcher *tasks.Dispatcher
	scoringCfg config.ScoringConfig
}

var (
	ErrNotFound      = errors.New("jobs: not found")
	ErrInvalidStatus = errors.New("jobs: invalid status")
	ErrInvalidScore  = errors.New("jobs: invalid match score")
	ErrInvalidInput  = errors.New("jobs: invalid input")
)

// NewService creates a new jobs service.
func NewService(repo *Repository, dispatcher *tasks.Dispatcher, scoringCfg config.ScoringConfig) *Service {
	return &Service{
		repo:       repo,
		dispatcher: dispatcher,
		scoringCfg: scoringCfg,
	}
}

// getJob retrieves a job by ID and translates sql.ErrNoRows to ErrNotFound.
func (s *Service) getJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("jobs: get by id: %w", err)
	}
	return job, nil
}

// GetByID retrieves a job by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	return s.getJob(ctx, id)
}

// List retrieves jobs with filtering and pagination.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]Job, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	jobs, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("jobs: list: %w", err)
	}
	return jobs, total, nil
}

// UpdateStatus updates a job's status.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if !IsValidStatus(status) {
		return ErrInvalidStatus
	}

	err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, ErrNoRowsAffected) {
			return ErrNotFound
		}
		return fmt.Errorf("jobs: update status: %w", err)
	}
	return nil
}

// validateImport checks that a CreateJobInput has required fields.
func validateImport(input CreateJobInput) error {
	if input.SourceID == uuid.Nil {
		return fmt.Errorf("%w: source_id required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		return fmt.Errorf("%w: external_id required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("%w: title required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.Company) == "" {
		return fmt.Errorf("%w: company required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.URL) == "" {
		return fmt.Errorf("%w: url required", ErrInvalidInput)
	}
	return nil
}

// BulkImport imports jobs from scrapers with deduplication.
// Returns count of imported vs skipped (duplicates).
func (s *Service) BulkImport(ctx context.Context, inputs []CreateJobInput) (*BulkImportResult, error) {
	if len(inputs) == 0 {
		return &BulkImportResult{Imported: 0, Skipped: 0}, nil
	}

	// Validate all inputs upfront
	for i, input := range inputs {
		if err := validateImport(input); err != nil {
			return nil, fmt.Errorf("jobs: bulk import input %d: %w", i, err)
		}
	}

	now := time.Now()

	// Convert inputs to Job models
	jobs := make([]*Job, len(inputs))
	for i, input := range inputs {
		job := &Job{
			ID:             uuid.New(),
			SourceID:       input.SourceID,
			ExternalID:     input.ExternalID,
			Title:          input.Title,
			Company:        input.Company,
			Location:       input.Location,
			RemoteType:     input.RemoteType,
			SalaryMin:      input.SalaryMin,
			SalaryMax:      input.SalaryMax,
			SalaryCurrency: input.SalaryCurrency,
			Description:    input.Description,
			Requirements:   input.Requirements,
			URL:            input.URL,
			ApplicationURL: input.ApplicationURL,
			CompanyURL:     input.CompanyURL,
			Source:         input.Source,
			PostedAt:       input.PostedAt,
			ScrapedAt:      now,
			MatchScore:     0,
			MatchDetails:   nil,
			Status:         StatusDiscovered,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		jobs[i] = job
	}

	imported, err := s.repo.BulkCreate(ctx, jobs)
	if err != nil {
		return nil, fmt.Errorf("jobs: bulk import: %w", err)
	}

	return &BulkImportResult{
		Imported: imported,
		Skipped:  len(inputs) - imported,
	}, nil
}

// TriggerScan enqueues a job discovery task for each source.
// Returns all Asynq task IDs for polling.
// On partial failure, returns all previously dispatched task IDs alongside the error.
func (s *Service) TriggerScan(ctx context.Context, sourceIDs []uuid.UUID) ([]string, error) {
	if len(sourceIDs) == 0 {
		return nil, fmt.Errorf("jobs: trigger scan: no source IDs provided")
	}

	var taskIDs []string
	for _, sourceID := range sourceIDs {
		payload := tasks.JobDiscoveryPayload{
			SourceID: sourceID,
		}

		taskID, err := s.dispatcher.DispatchJobDiscovery(ctx, payload)
		if err != nil {
			return taskIDs, fmt.Errorf("jobs: trigger scan (source %s): %w", sourceID, err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	return taskIDs, nil
}

// GetSourceNameByID returns the source name for a given job_sources UUID.
func (s *Service) GetSourceNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	return s.repo.GetSourceNameByID(ctx, id)
}

// ApplyMatchScore applies a scoring result to a job.
// Preserves user-owned statuses (applied, archived) — only transitions
// discovered/matched statuses based on config thresholds.
func (s *Service) ApplyMatchScore(ctx context.Context, jobID uuid.UUID, score float64, details json.RawMessage) error {
	if score < 0 || score > 100 {
		return ErrInvalidScore
	}

	job, err := s.getJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("jobs: apply match score: %w", err)
	}

	err = s.repo.UpdateMatchScore(ctx, jobID, score, details)
	if err != nil {
		return fmt.Errorf("jobs: apply match score update: %w", err)
	}

	// Preserve user-owned statuses — only transition discovered/matched
	var newStatus string
	switch job.Status {
	case StatusApplied, StatusArchived:
		// User has acted on this job — do not overwrite
		return nil
	case StatusDiscovered:
		if score >= float64(s.scoringCfg.AutoThreshold) {
			newStatus = StatusMatched
		} else {
			return nil // stays discovered
		}
	case StatusMatched:
		if score < float64(s.scoringCfg.ReviewThreshold) {
			newStatus = StatusDiscovered
		} else {
			return nil // stays matched
		}
	default:
		return nil
	}

	if err := s.repo.UpdateStatus(ctx, jobID, newStatus); err != nil {
		return fmt.Errorf("jobs: update status from score: %w", err)
	}

	return nil
}
