// Service handles profile business logic.
//
// Responsibilities:
//   - Get or create the singleton profile on first-run
//   - Validate and persist profile updates (PUT)
//   - Apply partial patches with domain merge rules (PATCH)
//
// This file contains NO HTTP handlers, NO database queries.
// It orchestrates repository calls and enforces business rules.
//
// Error handling:
//   - Returns domain errors (ErrNotFound, ErrVersionConflict)
//   - Wraps unexpected errors with context
//   - Never logs and returns the same error (handler decides to log)
package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Repository interface — enables unit testing without a real database
// ---------------------------------------------------------------------------

// RepositoryInterface defines the contract for profile data access.
// The service depends on this interface, not the concrete implementation.
type RepositoryInterface interface {
	Get(ctx context.Context) (*Profile, error)
	Create(ctx context.Context, p *Profile) error
	Update(ctx context.Context, id uuid.UUID, data ProfileData, expectedVersion int) (*Profile, error)
	UpdatePartial(ctx context.Context, id uuid.UUID, patch PatchProfileRequest, expectedVersion int) (*Profile, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service handles profile business logic.
type Service struct {
	repo   RepositoryInterface
	logger *zap.Logger
}

// NewService creates a new profile service.
func NewService(repo RepositoryInterface, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("profile"),
	}
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// GetOrCreate returns the singleton profile.
// On first-run (no profile exists), creates a default profile and returns it.
// The caller never sees ErrNotFound — first access always succeeds.
//
// Named GetOrCreate (not Get) because it has a write side effect.
func (s *Service) GetOrCreate(ctx context.Context) (*Profile, error) {
	p, err := s.repo.Get(ctx)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	// First-run: create default profile
	s.logger.Debug("no profile found, creating default")
	p = &Profile{
		Data: ProfileData{},
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create default profile: %w", err)
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

// Update replaces the entire profile data (PUT).
//
// The clientVersion comes from the If-Match header (ETag from last GET).
// If the DB version does not match, returns ErrVersionConflict.
func (s *Service) Update(ctx context.Context, req UpdateProfileRequest, clientVersion int) (*Profile, error) {
	current, err := s.GetOrCreate(ctx)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	newData := ProfileData{
		Preferences: req.Preferences,
		Skills:      req.Skills,
		Education:   req.Education,
		Links:       req.Links,
	}

	if err := newData.Validate(); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	// Use clientVersion for OCC — if DB version differs, update affects 0 rows
	updated, err := s.repo.Update(ctx, current.ID, newData, clientVersion)
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return nil, fmt.Errorf("update profile: %w", ErrVersionConflict)
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}

	s.logger.Info("profile updated",
		zap.Int("version", updated.Version),
	)
	return updated, nil
}

// UpdatePartial merges a patch into the existing profile data (PATCH).
//
// The clientVersion comes from the If-Match header (ETag from last GET).
// The domain method ProfileData.ApplyPatch owns the merge rules.
// The repository validates the merged result inside a transaction
// before committing — invalid data never persists.
func (s *Service) UpdatePartial(ctx context.Context, req PatchProfileRequest, clientVersion int) (*Profile, error) {
	current, err := s.GetOrCreate(ctx)
	if err != nil {
		return nil, fmt.Errorf("partial update profile: %w", err)
	}

	// Use clientVersion for OCC — if DB version differs, update affects 0 rows
	updated, err := s.repo.UpdatePartial(ctx, current.ID, req, clientVersion)
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return nil, fmt.Errorf("partial update profile: %w", ErrVersionConflict)
		}
		return nil, fmt.Errorf("partial update profile: %w", err)
	}

	s.logger.Info("profile partially updated",
		zap.Int("version", updated.Version),
	)
	return updated, nil
}
