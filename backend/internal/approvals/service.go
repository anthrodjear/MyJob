// Service handles approval request business logic.
//
// Responsibilities:
//   - Create approval requests when jobs score in the "review" tier
//   - List and retrieve approval requests with filters
//   - Approve or reject requests with status transition validation
//
// This file contains NO HTTP handlers, NO database queries, NO task dispatch.
// It orchestrates repository calls and enforces business rules.
//
// Task dispatch (application submission on approval) is the workflow
// layer's responsibility (ApprovalWorkflow). The service handles only
// the approval state change.
//
// Error handling:
//   - Returns domain errors (ErrNotFound, ErrInvalidStatus, ErrReasonRequired)
//   - Wraps unexpected errors with context
//   - Never logs and returns the same error (handler decides to log)
package approvals

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Repository interface — enables unit testing without a real database
// ---------------------------------------------------------------------------

// RepositoryInterface defines the contract for approval data access.
// The service depends on this interface, not the concrete implementation.
type RepositoryInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error)
	List(ctx context.Context, filter ListFilter) ([]ApprovalRequest, int64, error)
	Create(ctx context.Context, a *ApprovalRequest) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, rejectionReason *string) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service handles approval request business logic.
type Service struct {
	repo   RepositoryInterface
	logger *zap.Logger
}

// NewService creates a new approvals service.
func NewService(repo RepositoryInterface, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("approvals"),
	}
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// GetByID returns a single approval request by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get approval: %w", err)
	}
	return a, nil
}

// List returns approval requests matching the filter with total count.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]ApprovalRequest, int64, error) {
	approvals, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list approvals: %w", err)
	}
	return approvals, total, nil
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

// Create inserts a new approval request.
// Called by the scoring worker when a job lands in the "review" tier.
func (s *Service) Create(ctx context.Context, a *ApprovalRequest) error {
	if err := a.Validate(); err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	s.logger.Info("approval request created",
		zap.String("id", a.ID.String()),
		zap.String("application_id", a.ApplicationID.String()),
		zap.Float64("score", a.JobSnapshot.Score),
	)
	return nil
}

// Approve changes the approval status from "pending" to "approved".
// Returns the updated ApprovalRequest so the caller can access ApplicationID
// for downstream task dispatch.
// Returns ErrInvalidStatus if the current status is not "pending".
func (s *Service) Approve(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	if err := s.repo.UpdateStatus(ctx, id, ApprovalStatusApproved, nil); err != nil {
		return nil, fmt.Errorf("approve: %w", err)
	}
	// Re-fetch to return the updated entity (caller needs ApplicationID)
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("approve re-fetch: %w", err)
	}
	s.logger.Info("approval request approved",
		zap.String("id", id.String()),
	)
	return updated, nil
}

// Reject changes the approval status from "pending" to "rejected".
// A reason is required for audit trail.
// Returns ErrInvalidStatus if the current status is not "pending".
func (s *Service) Reject(ctx context.Context, id uuid.UUID, reason string) error {
	if reason == "" {
		return fmt.Errorf("reject: %w", ErrReasonRequired)
	}
	if err := s.repo.UpdateStatus(ctx, id, ApprovalStatusRejected, &reason); err != nil {
		return fmt.Errorf("reject: %w", err)
	}
	s.logger.Info("approval request rejected",
		zap.String("id", id.String()),
		zap.String("reason", reason),
	)
	return nil
}
