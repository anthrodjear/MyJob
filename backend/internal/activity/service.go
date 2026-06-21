package activity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service provides business logic for activity logging.
// Other domains use LogEvent to record events; the frontend uses
// List and GetByID to display the activity feed.
//
// Pagination is validated by the handler before calling List.
// The service delegates all persistence to the Repository.
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService creates a new activity service.
// The logger is named "activity" for structured log grouping.
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("activity"),
	}
}

// GetByID returns an activity log entry by ID.
// Returns ErrNotFound if the entry does not exist.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get activity by id: %w", err)
	}
	return a, nil
}

// List returns activity logs matching the filter with total count.
// Pagination is validated by the handler before calling this method.
// Results are ordered by created_at DESC (newest first).
func (s *Service) List(ctx context.Context, filter ListFilter) ([]ActivityLog, int64, error) {
	return s.repo.List(ctx, filter)
}

// LogEvent creates a new activity log entry.
// This is the primary write method — other domains call this to record events.
//
// The eventType must be one of the defined Event* constants.
// The entityType identifies the affected domain (e.g. "jobs", "applications").
// The entityID identifies the specific entity affected.
// The details map carries event-specific context (stored as JSONB).
//
// Example:
//
//	svc.LogEvent(ctx, activity.EventJobDiscovered, "jobs", jobID, activity.Details{
//	    "title": "Backend Engineer",
//	    "company": "Acme Corp",
//	    "source": "indeed",
//	})
func (s *Service) LogEvent(ctx context.Context, eventType, entityType string, entityID uuid.UUID, details Details) error {
	a := &ActivityLog{
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("log event: %w", err)
	}

	s.logger.Debug("activity logged",
		zap.String("event_type", eventType),
		zap.String("entity_type", entityType),
		zap.String("entity_id", entityID.String()),
	)
	return nil
}
