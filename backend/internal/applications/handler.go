package applications

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the applications HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new applications handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("applications"),
	}
}

// RegisterRoutes registers application routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	apps := rg.Group("/applications")
	{
		apps.GET("", h.ListApplications)
		apps.GET("/stats", h.GetStats)
		apps.GET("/:id", h.GetApplication)
		apps.POST("", h.CreateApplication)
		apps.PUT("/:id/status", h.UpdateStatus)
		apps.PATCH("/:id/notes", h.UpdateNotes)
		apps.GET("/:id/events", h.GetTimeline)
	}
}

// listApplicationsQuery holds query parameters for listing applications.
type listApplicationsQuery struct {
	Status     string `form:"status"`
	JobID      string `form:"job_id"`
	PortalType string `form:"portal_type"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

// ListApplications handles GET /applications.
// @Summary List applications
// @Description Get paginated list of applications with optional filters
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(pending,applied,rejected,responded,interviewed,archived)
// @Param job_id query string false "Filter by job UUID"
// @Param portal_type query string false "Filter by portal type" Enums(greenhouse,lever,remoteok,indeed,manual,email)
// @Param limit query int false "Results per page (max 100)" default(20) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} ApplicationListResponse "Paginated applications"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid query parameters"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications [get]
func (h *Handler) ListApplications(c *gin.Context) {
	var q listApplicationsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	// Validate status filter
	if q.Status != "" && !IsValidStatus(q.Status) {
		httpresp.BadRequest(c, "INVALID_STATUS", "invalid status filter")
		return
	}

	// Apply pagination defaults
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	filter := ListFilter{
		Status:     q.Status,
		PortalType: q.PortalType,
		Limit:      q.Limit,
		Offset:     q.Offset,
	}

	if q.JobID != "" {
		jid, err := uuid.Parse(q.JobID)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_JOB_ID", "invalid job_id")
			return
		}
		filter.JobID = jid
	}

	apps, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list applications", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := ApplicationListResponse{
		Applications: make([]ApplicationResponse, len(apps)),
		Total:        total,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	}
	for i := range apps {
		resp.Applications[i] = ToResponse(&apps[i])
	}

	httpresp.OK(c, resp)
}

// GetApplication handles GET /applications/:id.
// @Summary Get application by ID
// @Description Get detailed information about a specific application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Success 200 {object} ApplicationResponse "Application details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid application ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Application not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id} [get]
func (h *Handler) GetApplication(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid application id")
		return
	}

	app, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "APPLICATION_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get application", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(app))
}

// CreateApplication handles POST /applications.
// @Summary Create application
// @Description Create a new application for a job
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateApplicationRequest true "Application creation request"
// @Success 201 {object} ApplicationResponse "Created application"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications [post]
func (h *Handler) CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	app, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("create application", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToResponse(app))
}

// UpdateStatus handles PUT /applications/:id/status.
// @Summary Update application status
// @Description Update the status of an application with optional notes (audit trail)
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Param request body UpdateStatusRequest true "Status update"
// @Success 200 {object} map[string]string "Status updated"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or status"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Application not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/status [put]
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid application id")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status, req.Notes); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "APPLICATION_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		h.logger.Error("update status", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "status updated"})
}

// UpdateNotes handles PATCH /applications/:id/notes.
// @Summary Update application notes
// @Description Update permanent notes on an application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Param request body UpdateApplicationNotesRequest true "Notes update"
// @Success 200 {object} map[string]string "Notes updated"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Application not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/notes [patch]
func (h *Handler) UpdateNotes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid application id")
		return
	}

	var req UpdateApplicationNotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.UpdateNotes(c.Request.Context(), id, req.Notes); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "APPLICATION_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("update notes", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "notes updated"})
}

// GetTimeline handles GET /applications/:id/events.
// @Summary Get application timeline
// @Description Get the audit trail of status changes for an application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Success 200 {object} ApplicationTimelineResponse "Timeline of events"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid application ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Application not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/events [get]
func (h *Handler) GetTimeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid application id")
		return
	}

	events, err := h.svc.GetTimeline(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "APPLICATION_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get timeline", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := ApplicationTimelineResponse{
		ApplicationID: id,
		Events:        make([]ApplicationEventResponse, len(events)),
	}
	for i := range events {
		resp.Events[i] = ToEventResponse(&events[i])
	}

	httpresp.OK(c, resp)
}

// GetStats handles GET /applications/stats.
// @Summary Get application statistics
// @Description Get dashboard statistics for applications
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ApplicationStatsResponse "Application statistics"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		h.logger.Error("get stats", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, stats)
}
