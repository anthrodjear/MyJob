// Workflow orchestrates approval-to-submission business flows.
//
// The workflow owns the complete lifecycle:
//  1. Approve the approval request (state change)
//  2. Dispatch application submission task (async work)
//
// This separation exists because:
//   - The approve→submit flow is a business invariant, not HTTP logic
//   - Multiple callers need this flow (HTTP handler, CLI, admin worker, AI agent)
//   - Task dispatch needs a detached context (HTTP context dies with the request)
//   - CorrelationID belongs to the workflow, not the transport layer
//
// Error handling:
//   - If approve succeeds but dispatch fails, returns DispatchError
//   - The approval is persisted; dispatch can be retried
//   - Callers decide whether to surface dispatch failure to the user
package approvals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/activity"
	"backend/internal/applications"
)

// ---------------------------------------------------------------------------
// Task dispatcher — decoupled from asynq
// ---------------------------------------------------------------------------

// SubmitDispatcher dispatches application submission tasks.
// The concrete implementation is injected by the handler/DI layer.
// Returns an error if the enqueue fails. The caller uses correlationID
// for end-to-end tracing (the asynq task ID is internal to the dispatcher).
type SubmitDispatcher interface {
	DispatchApplicationSubmit(ctx context.Context, applicationID uuid.UUID, correlationID uuid.UUID) error
}

// ---------------------------------------------------------------------------
// DispatchError — approval succeeded but dispatch failed
// ---------------------------------------------------------------------------

// DispatchError indicates the approval was persisted but the submission
// task could not be enqueued. The system is in a partially-consistent state:
// the approval is "approved" but the application was not submitted.
//
// Callers should log this and consider retrying the dispatch.
type DispatchError struct {
	ApprovalID    uuid.UUID
	ApplicationID uuid.UUID
	Err           error
}

func (e *DispatchError) Error() string {
	return fmt.Sprintf("approval %s approved but dispatch for application %s failed: %v",
		e.ApprovalID, e.ApplicationID, e.Err)
}

func (e *DispatchError) Unwrap() error { return e.Err }

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// dispatchTimeout is the maximum time to wait for task queue write.
// The HTTP request context may cancel before Redis enqueue completes,
// so we use a detached background context with this timeout.
const dispatchTimeout = 10 * time.Second

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

// Workflow orchestrates approval-to-submission business flows.
type Workflow struct {
	svc            *Service
	applicationsSvc *applications.Service
	dispatcher     SubmitDispatcher
	activitySvc    *activity.Service
	logger         *zap.Logger
}

// NewWorkflow creates a new approval workflow.
func NewWorkflow(svc *Service, applicationsSvc *applications.Service, dispatcher SubmitDispatcher, activitySvc *activity.Service, logger *zap.Logger) *Workflow {
	return &Workflow{
		svc:            svc,
		applicationsSvc: applicationsSvc,
		dispatcher:     dispatcher,
		activitySvc:    activitySvc,
		logger:         logger.Named("approvals.workflow"),
	}
}

// Approve approves the request and dispatches application submission.
//
// The flow:
//  1. Service.Approve transitions status pending → approved, returns updated entity
//  2. Workflow dispatches ApplicationSubmit task using the correct ApplicationID
//
// The dispatch uses a detached background context with a timeout so the
// HTTP request lifecycle doesn't kill the Redis enqueue.
//
// Returns:
//   - (*ApprovalRequest, nil) — approve + dispatch succeeded
//   - (*ApprovalRequest, *DispatchError) — approve succeeded, dispatch failed
//   - (nil, error) — approve failed (ErrNotFound, ErrInvalidStatus)
func (w *Workflow) Approve(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	// Step 1: Approve (state change)
	approval, err := w.svc.Approve(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("workflow approve: %w", err)
	}

	// Log the approval event
	if err := w.activitySvc.LogEvent(ctx, activity.EventApprovalApproved, "approval", id, nil); err != nil {
		w.logger.Error("failed to log approval event",
			zap.String("approval_id", id.String()),
			zap.Error(err),
		)
	}

	// Transition application from draft → queued (required by FSM before queued → applied)
	if err := w.applicationsSvc.UpdateStatus(ctx, approval.ApplicationID, applications.StatusQueued, "Approved — queued for submission"); err != nil {
		w.logger.Error("failed to queue application — approval succeeded but submission cannot proceed",
			zap.String("application_id", approval.ApplicationID.String()),
			zap.Error(err),
		)
		return nil, &DispatchError{
			ApprovalID: id,
			Err:        fmt.Errorf("queue application: %w", err),
		}
	}

	// Step 2: Dispatch submission task with detached context
	// The HTTP request context may cancel before the Redis enqueue completes.
	// Use background context with a reasonable timeout for queue write.
	dispatchCtx, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
	defer cancel()

	correlationID := uuid.New()
	if err := w.dispatcher.DispatchApplicationSubmit(dispatchCtx, approval.ApplicationID, correlationID); err != nil {
		w.logger.Error("dispatch failed after approval",
			zap.String("approval_id", id.String()),
			zap.String("application_id", approval.ApplicationID.String()),
			zap.String("correlation_id", correlationID.String()),
			zap.Error(err),
		)
		return approval, &DispatchError{
			ApprovalID:    id,
			ApplicationID: approval.ApplicationID,
			Err:           err,
		}
	}

	w.logger.Info("approval workflow complete",
		zap.String("approval_id", id.String()),
		zap.String("application_id", approval.ApplicationID.String()),
		zap.String("correlation_id", correlationID.String()),
	)
	return approval, nil
}

// Reject rejects the approval request and logs the event.
//
// The flow:
//  1. Service.Reject transitions status pending → rejected
//  2. Workflow logs the rejection event for the activity feed
//
// Returns:
//   - nil — reject succeeded
//   - error — reject failed (ErrNotFound, ErrInvalidStatus, ErrReasonRequired)
func (w *Workflow) Reject(ctx context.Context, id uuid.UUID, reason string) error {
	if err := w.svc.Reject(ctx, id, reason); err != nil {
		return fmt.Errorf("workflow reject: %w", err)
	}

	// Log the rejection event
	if err := w.activitySvc.LogEvent(ctx, activity.EventApprovalRejected, "approval", id, nil); err != nil {
		w.logger.Error("failed to log rejection event",
			zap.String("approval_id", id.String()),
			zap.Error(err),
		)
	}

	w.logger.Info("approval rejected",
		zap.String("approval_id", id.String()),
	)
	return nil
}
