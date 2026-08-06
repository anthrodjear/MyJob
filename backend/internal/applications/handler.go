package applications

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// JobView is the subset of jobs.Job that the applications handler
// returns in its voice-context responses. We don't import jobs.Job
// directly because that would create an import cycle (jobs imports
// applications for /jobs/:id/apply).
type JobView struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	CompanyURL  string    `json:"company_url,omitempty"`
}

// JobsAPI is the subset of jobs.Service that the applications handler
// depends on. Defined here to avoid an import cycle (jobs imports
// applications for its /jobs/:id/apply route).
type JobsAPI interface {
	GetByID(ctx context.Context, id uuid.UUID) (JobView, error)
}

// ErrJobNotFound is the sentinel returned by JobsAPI.GetByID when the
// bound job no longer exists. Mapped to HTTP 404 by the handler.
var ErrJobNotFound = errors.New("job not found")

// ResumeView is the subset of resumes.Resume that the applications
// handler returns in its voice-context responses.
type ResumeView struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Specialization string   `json:"specialization"`
	Version       int32     `json:"version"`
}

// ResumesAPI is the subset of resumes.Service that the applications
// handler depends on.
type ResumesAPI interface {
	GetByID(ctx context.Context, id uuid.UUID) (ResumeView, error)
}

// ErrResumeNotFound is the sentinel returned by ResumesAPI.GetByID when
// the resume no longer exists. Mapped to HTTP 404 by the handler.
var ErrResumeNotFound = errors.New("resume not found")

// Handler holds the applications HTTP handlers.
type Handler struct {
	svc        *Service
	jobsAPI    JobsAPI
	resumesAPI ResumesAPI
	logger     *zap.Logger
}

// NewHandler creates a new applications handler.
func NewHandler(svc *Service, jobsAPI JobsAPI, resumesAPI ResumesAPI, logger *zap.Logger) *Handler {
	return &Handler{
		svc:        svc,
		jobsAPI:    jobsAPI,
		resumesAPI: resumesAPI,
		logger:     logger.Named("applications"),
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
		apps.GET("/:id/resume", h.GetApplicationResume)
		apps.GET("/:id/job", h.GetApplicationJob)
		apps.GET("/:id/company", h.GetApplicationCompany)
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
		if errors.Is(err, ErrStatusConflict) {
			httpresp.Conflict(c, "STATUS_CONFLICT", err.Error())
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

// GetApplicationResume handles GET /applications/:id/resume.
// Returns the resume currently attached to the application (or 404 if
// the application has no resume_id or the resume has been deleted).
// @Summary Get the resume attached to an application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Success 200 {object} map[string]interface{} "Resume details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid application id"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not attached or not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/resume [get]
func (h *Handler) GetApplicationResume(c *gin.Context) {
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
		h.logger.Error("get application for resume", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	if app.ResumeID == nil {
		httpresp.NotFound(c, "RESUME_NOT_ATTACHED", "application has no resume attached")
		return
	}
	if h.resumesAPI == nil {
		httpresp.NotFound(c, "RESUME_NOT_ATTACHED", "resume service not configured")
		return
	}

	resume, err := h.resumesAPI.GetByID(c.Request.Context(), *app.ResumeID)
	if err != nil {
		if errors.Is(err, ErrResumeNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get application resume",
			zap.String("application_id", id.String()),
			zap.String("resume_id", app.ResumeID.String()),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, resume)
}

// GetApplicationJob handles GET /applications/:id/job.
// Returns the job (title, company, description) bound to this application.
// @Summary Get the job bound to an application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Success 200 {object} map[string]interface{} "Job details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid application id"
// @Failure 404 {object} httpresp.ErrorResponse "Application or job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/job [get]
func (h *Handler) GetApplicationJob(c *gin.Context) {
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
		h.logger.Error("get application for job", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	if h.jobsAPI == nil {
		httpresp.InternalError(c)
		return
	}

	job, err := h.jobsAPI.GetByID(c.Request.Context(), app.JobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get job for application",
			zap.String("application_id", id.String()),
			zap.String("job_id", app.JobID.String()),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{
		"id":          job.ID,
		"title":       job.Title,
		"company":     job.Company,
		"description": job.Description,
		"location":    job.Location,
		"url":         job.URL,
		"source":      job.Source,
		"status":      job.Status,
	})
}

// GetApplicationCompany handles GET /applications/:id/company.
// Returns company name + url from the bound job's CompanyURL/Company fields.
// @Summary Get the company for an application
// @Tags Applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Application UUID" format(uuid)
// @Success 200 {object} map[string]interface{} "Company info"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid application id"
// @Failure 404 {object} httpresp.ErrorResponse "Application or job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /applications/{id}/company [get]
func (h *Handler) GetApplicationCompany(c *gin.Context) {
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
		h.logger.Error("get application for company", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	if h.jobsAPI == nil {
		httpresp.InternalError(c)
		return
	}

	job, err := h.jobsAPI.GetByID(c.Request.Context(), app.JobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get company for application",
			zap.String("application_id", id.String()),
			zap.String("job_id", app.JobID.String()),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{
		"name": job.Company,
		"url":  job.CompanyURL,
	})
}
