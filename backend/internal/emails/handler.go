// Handler contains HTTP handlers for the emails domain.
//
// Endpoints:
//
//	POST   /emails                  → Store incoming email
//	GET    /emails                  → List emails with filters
//	GET    /emails/:id              → Get single email
//	PATCH  /emails/:id              → Update read status or reply draft
//	POST   /emails/:id/classify     → Re-classify email via LLM
//
// All endpoints require authentication (handled by middleware).
package emails

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// RegisterRoutes registers email routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	emails := rg.Group("/emails")
	{
		emails.POST("", h.Store)
		emails.GET("", h.List)
		emails.GET("/:id", h.GetByID)
		emails.PATCH("/:id", h.Update)
		emails.POST("/:id/classify", h.Reclassify)
	}
}

// Handler contains dependencies for email HTTP handlers.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new emails handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("emails.handler"),
	}
}

// ============================================================================
// Handlers
// ============================================================================

// Store handles POST /emails
// Stores an incoming email (called by worker after browser-agent fetch).
// @Summary Store email
// @Tags emails
// @Accept json
// @Produce json
// @Param request body StoreEmailRequest true "Email to store"
// @Success 201 {object} EmailResponse
// @Failure 400 {object} httpresp.ErrorResponse
// @Failure 500 {object} httpresp.ErrorResponse
// @Router /emails [post]
func (h *Handler) Store(c *gin.Context) {
	var req StoreEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid store email request", zap.Error(err))
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	params := StoreEmailParams(req)

	id, _, err := h.service.Store(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("failed to store email", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	email, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to retrieve stored email", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToEmailResponse(email))
}

// List handles GET /emails
// Returns paginated list of emails with optional filters.
// @Summary List emails
// @Tags emails
// @Produce json
// @Param application_id query string false "Filter by application ID"
// @Param classification query string false "Filter by classification"
// @Param limit query int false "Limit (default 50, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} EmailListResponse
// @Failure 400 {object} httpresp.ErrorResponse
// @Failure 500 {object} httpresp.ErrorResponse
// @Router /emails [get]
func (h *Handler) List(c *gin.Context) {
	var req ListFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn("invalid list emails request", zap.Error(err))
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	filter, err := req.ToListFilter()
	if err != nil {
		if errors.Is(err, ErrInvalidApplicationID) {
			httpresp.BadRequest(c, "INVALID_APPLICATION_ID", "invalid application_id")
			return
		}
		h.logger.Error("failed to convert list filter", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	emails, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list emails", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := EmailListResponse{
		Emails: make([]EmailResponse, len(emails)),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	for i, e := range emails {
		resp.Emails[i] = ToEmailResponse(&e)
	}

	httpresp.OK(c, resp)
}

// GetByID handles GET /emails/:id
// Returns a single email by ID.
// @Summary Get email by ID
// @Tags emails
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} EmailResponse
// @Failure 400 {object} httpresp.ErrorResponse
// @Failure 404 {object} httpresp.ErrorResponse
// @Failure 500 {object} httpresp.ErrorResponse
// @Router /emails/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid email id")
		return
	}

	email, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "EMAIL_NOT_FOUND", "email not found")
			return
		}
		h.logger.Error("failed to get email", zap.Error(err), zap.String("id", idStr))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToEmailResponse(email))
}

// Update handles PATCH /emails/:id
// Updates read status or reply draft for an email.
// @Summary Update email
// @Tags emails
// @Accept json
// @Produce json
// @Param id path string true "Email ID"
// @Param request body UpdateEmailRequest true "Fields to update"
// @Success 200 {object} EmailResponse
// @Failure 400 {object} httpresp.ErrorResponse
// @Failure 404 {object} httpresp.ErrorResponse
// @Failure 500 {object} httpresp.ErrorResponse
// @Router /emails/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid email id")
		return
	}

	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid update email request", zap.Error(err))
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.IsRead != nil {
		if err := h.service.MarkRead(c.Request.Context(), id, *req.IsRead); err != nil {
			if errors.Is(err, ErrNotFound) {
				httpresp.NotFound(c, "EMAIL_NOT_FOUND", "email not found")
				return
			}
			h.logger.Error("failed to update read status", zap.Error(err), zap.String("id", idStr))
			httpresp.InternalError(c)
			return
		}
	}

	if req.ReplyDraft != nil {
		if err := h.service.UpdateDraft(c.Request.Context(), id, req.ReplyDraft); err != nil {
			if errors.Is(err, ErrNotFound) {
				httpresp.NotFound(c, "EMAIL_NOT_FOUND", "email not found")
				return
			}
			h.logger.Error("failed to update draft", zap.Error(err), zap.String("id", idStr))
			httpresp.InternalError(c)
			return
		}
	}

	email, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to retrieve updated email", zap.Error(err), zap.String("id", idStr))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToEmailResponse(email))
}

// Reclassify handles POST /emails/:id/classify
// Re-classifies an email using the LLM and updates its classification.
// @Summary Re-classify email
// @Tags emails
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} ClassifyResponse
// @Failure 400 {object} httpresp.ErrorResponse
// @Failure 404 {object} httpresp.ErrorResponse
// @Failure 500 {object} httpresp.ErrorResponse
// @Router /emails/{id}/classify [post]
func (h *Handler) Reclassify(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid email id")
		return
	}

	result, err := h.service.Reclassify(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "EMAIL_NOT_FOUND", "email not found")
			return
		}
		h.logger.Error("failed to reclassify email", zap.Error(err), zap.String("id", idStr))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ClassifyResponse{
		EmailID:        id,
		Classification: result.Category,
		Confidence:     result.Confidence,
		Reasoning:      result.Reasoning,
	})
}
