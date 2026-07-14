// Handler handles HTTP requests for the approvals domain.
//
// Approvals are list-based resources with ID parameters.
// API surface:
//   - GET    /approvals              → List all approval requests (with filters)
//   - GET    /approvals/:id          → Get single approval request
//   - POST   /approvals/:id/approve  → Approve (triggers submission workflow)
//   - POST   /approvals/:id/reject   → Reject with reason
//
// This file contains NO business logic. It binds HTTP requests to
// workflow/service calls and maps domain errors to HTTP responses.
//
// The handler depends on Workflow (not Service directly) because
// approve→submit is a business invariant, not HTTP logic.
package approvals

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler holds the approvals HTTP handlers.
type Handler struct {
	workflow *Workflow
	svc      *Service
	logger   *zap.Logger
}

// NewHandler creates a new approvals handler.
func NewHandler(workflow *Workflow, svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		workflow: workflow,
		svc:      svc,
		logger:   logger.Named("approvals.handler"),
	}
}

// RegisterRoutes registers approvals routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	approvals := rg.Group("/approvals")
	{
		approvals.GET("", h.ListApprovals)
		approvals.GET("/:id", h.GetApproval)
		approvals.POST("/:id/approve", h.ApproveApproval)
		approvals.POST("/:id/reject", h.RejectApproval)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseApprovalID extracts and validates the UUID from the :id path parameter.
func parseApprovalID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid approval request ID")
		return uuid.Nil, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// ListApprovals handles GET /approvals.
// @Summary List approval requests
// @Description Get paginated list of approval requests with optional filters
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(pending,approved,rejected)
// @Param application_id query string false "Filter by application UUID"
// @Param limit query int false "Results per page (max 100)" default(50) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} ApprovalListResponse "Paginated approval requests"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid query parameters"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /approvals [get]
func (h *Handler) ListApprovals(c *gin.Context) {
	var filter ListFilter

	if status := c.Query("status"); status != "" {
		if !IsValidStatus(status) {
			httpresp.BadRequest(c, "INVALID_STATUS", "invalid status filter")
			return
		}
		filter.Status = status
	}
	if appIDStr := c.Query("application_id"); appIDStr != "" {
		appID, err := uuid.Parse(appIDStr)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_APPLICATION_ID", "invalid application_id filter")
			return
		}
		filter.ApplicationID = appID
	}

	// Pagination defaults
	filter.Limit = 50
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			httpresp.BadRequest(c, "INVALID_LIMIT", "invalid limit parameter")
			return
		}
		if limit > 100 {
			limit = 100
		}
		filter.Limit = limit
	}
	filter.Offset = 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			httpresp.BadRequest(c, "INVALID_OFFSET", "invalid offset parameter")
			return
		}
		filter.Offset = offset
	}

	approvals, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list approvals", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Convert to response DTOs — index-explicit to avoid loop variable capture
	resp := make([]ApprovalResponse, len(approvals))
	for i := range approvals {
		resp[i] = ToResponse(&approvals[i])
	}

	httpresp.OK(c, ApprovalListResponse{
		Approvals: resp,
		Total:     total,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	})
}

// GetApproval handles GET /approvals/:id.
// @Summary Get approval request by ID
// @Description Get detailed approval request with job snapshot and previews
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Approval UUID" format(uuid)
// @Success 200 {object} ApprovalResponse "Approval request details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid approval ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Approval request not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /approvals/{id} [get]
func (h *Handler) GetApproval(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}

	a, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "NOT_FOUND", "approval request not found")
			return
		}
		h.logger.Error("get approval", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(a))
}

// ApproveApproval handles POST /approvals/:id/approve.
// @Summary Approve application
// @Description Approve the application and trigger automatic submission workflow
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Approval UUID" format(uuid)
// @Param request body ApproveRequest true "Empty body - approval decision is in endpoint"
// @Success 200 {object} ApprovalResponse "Application approved and submission queued"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid approval ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Approval not found"
// @Failure 409 {object} httpresp.ErrorResponse "Invalid status transition"
// @Failure 207 {object} ApprovePartialResponse "Approved but submission dispatch failed (queued for retry)"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /approvals/{id}/approve [post]
func (h *Handler) ApproveApproval(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}

	approval, err := h.workflow.Approve(c.Request.Context(), id)

	// Check for partial failure FIRST: approval succeeded but dispatch failed.
	// DispatchError means the state change persisted — don't treat as generic error.
	var dispatchErr *DispatchError
	if errors.As(err, &dispatchErr) {
		httpresp.MultiStatus(c, ApprovePartialResponse{
			Status:   "approved",
			Warning:  "application submission queued for retry",
			Approval: ToResponse(approval),
		})
		return
	}

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "NOT_FOUND", "approval request not found")
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", "invalid status transition")
			return
		}
		h.logger.Error("approve approval", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(approval))
}

// RejectApproval handles POST /approvals/:id/reject.
// @Summary Reject application
// @Description Reject the application with a required reason
// @Tags Approvals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Approval UUID" format(uuid)
// @Param request body RejectRequest true "Rejection reason"
// @Success 200 {object} ApprovalResponse "Application rejected"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid approval ID or missing reason"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Approval not found"
// @Failure 409 {object} httpresp.ErrorResponse "Invalid status transition"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /approvals/{id}/reject [post]
func (h *Handler) RejectApproval(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.Reject(c.Request.Context(), id, req.Reason); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "NOT_FOUND", "approval request not found")
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", "invalid status transition")
			return
		}
		if errors.Is(err, ErrReasonRequired) {
			httpresp.BadRequest(c, "REASON_REQUIRED", err.Error())
			return
		}
		h.logger.Error("reject approval", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Re-fetch to return the updated entity
	approval, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("re-fetch after reject", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(approval))
}
