package applications

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles application business logic.
type Service struct {
	repo   *Repository
	logger *zap.Logger
}

// NewService creates a new applications service.
func NewService(repo *Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("applications"),
	}
}

// Create creates a new application with default status "draft".
func (s *Service) Create(ctx context.Context, req CreateApplicationRequest) (*Application, error) {
	app := &Application{
		ID:            uuid.New(),
		JobID:         req.JobID,
		ResumeID:      req.ResumeID,
		CoverLetterID: req.CoverLetterID,
		Status:        StatusDraft,
		ApprovalTier:  TierReview, // default; caller can override
		PortalType:    req.PortalType,
		PortalURL:     req.PortalURL,
	}

	if err := s.repo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	s.logger.Info("application created",
		zap.String("id", app.ID.String()),
		zap.String("job_id", app.JobID.String()),
	)
	return app, nil
}

// GetByID returns an application by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get application: %w", err)
	}
	return app, nil
}

// List returns applications matching the filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]Application, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

// UpdateStatus transitions an application to a new status.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string, notes string) error {
	if !IsValidStatus(status) {
		return ErrInvalidStatus
	}

	// Fetch current status for transition check
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("update status get application: %w", err)
	}

	if !CanTransition(app.Status, status) {
		s.logger.Warn("invalid status transition",
			zap.String("from", app.Status),
			zap.String("to", status),
		)
		return ErrInvalidStatus
	}

	if err := s.repo.UpdateStatus(ctx, id, status, notes); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	s.logger.Info("application status updated",
		zap.String("id", id.String()),
		zap.String("from", app.Status),
		zap.String("to", status),
	)
	return nil
}

// GetStats returns aggregate application statistics.
func (s *Service) GetStats(ctx context.Context) (*ApplicationStatsResponse, error) {
	return s.repo.GetStats(ctx)
}

// UpdateNotes updates permanent notes on an application.
func (s *Service) UpdateNotes(ctx context.Context, id uuid.UUID, notes string) error {
	if err := s.repo.UpdateNotes(ctx, id, notes); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("update notes: %w", err)
	}
	s.logger.Info("notes updated", zap.String("id", id.String()))
	return nil
}

// GetTimeline returns the audit trail for an application.
func (s *Service) GetTimeline(ctx context.Context, applicationID uuid.UUID) ([]ApplicationEvent, error) {
	// Verify application exists
	if _, err := s.repo.GetByID(ctx, applicationID); err != nil {
		return nil, err
	}
	return s.repo.GetEvents(ctx, applicationID)
}
