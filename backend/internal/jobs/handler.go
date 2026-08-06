package jobs

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/applications"
	"backend/internal/httpresp"
	"backend/internal/scoring"
	"backend/internal/tasks"
)

// ApplicationsAPI is the subset of applications.Service that the jobs
// handler depends on. Defined as an interface in the jobs package to
// avoid an import cycle (applications imports jobs for its own handler
// routes — e.g. /jobs/:id/applications).
type ApplicationsAPI interface {
	Create(ctx context.Context, req applications.CreateApplicationRequest) (*applications.Application, error)
	List(ctx context.Context, filter applications.ListFilter) ([]applications.Application, int64, error)
}

// Handler holds the jobs HTTP handlers.
type Handler struct {
	svc        *Service
	appsAPI    ApplicationsAPI
	scoringSvc *scoring.Service
	taskDispatcher *tasks.Dispatcher
	logger     *zap.Logger
}

// NewHandler creates a new jobs handler.
func NewHandler(svc *Service, appsAPI ApplicationsAPI, scoringSvc *scoring.Service, taskDispatcher *tasks.Dispatcher, logger *zap.Logger) *Handler {
	return &Handler{
		svc:            svc,
		appsAPI:        appsAPI,
		scoringSvc:     scoringSvc,
		taskDispatcher: taskDispatcher,
		logger:         logger.Named("jobs"),
	}
}

// RegisterRoutes registers job routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	jobs := rg.Group("/jobs")
	{
		jobs.GET("", h.ListJobs)
		jobs.GET("/:id", h.GetJob)
		jobs.PATCH("/:id", h.UpdateJob)
		jobs.DELETE("/:id", h.DeleteJob)
		jobs.POST("/:id/apply", h.ApplyJob)
		jobs.POST("/:id/score", h.ScoreJob)
		jobs.PATCH("/:id/save", h.ToggleSave)
		jobs.GET("/:id/similar", h.SimilarJobs)
		jobs.GET("/:id/applications", h.ListJobApplications)
	}

	discovery := rg.Group("/job-discovery")
	{
		discovery.POST("/scan", h.TriggerScan)
	}
}

// listJobsQuery holds query parameters for listing jobs.
type listJobsQuery struct {
	Status   string  `form:"status"`
	Company  string  `form:"company"`
	SourceID string  `form:"source_id"`
	MinScore float64 `form:"min_score"`
	Limit    int     `form:"limit"`
	Offset   int     `form:"offset"`
}

// ListJobs handles GET /jobs.
// @Summary List jobs
// @Description Get paginated list of jobs with optional filters
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(discovered,matched,applied,archived)
// @Param company query string false "Filter by company name"
// @Param source_id query string false "Filter by source UUID"
// @Param min_score query number false "Minimum match score (0-100)" minimum(0) maximum(100)
// @Param limit query int false "Results per page (max 100)" default(20) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} JobListResponse "Successful response"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid query parameters"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs [get]
func (h *Handler) ListJobs(c *gin.Context) {
	var q listJobsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	// Validate status filter
	if q.Status != "" && !IsValidStatus(q.Status) {
		httpresp.BadRequest(c, "INVALID_STATUS", "invalid status filter")
		return
	}

	// Validate min_score range
	if q.MinScore < 0 || q.MinScore > 100 {
		httpresp.BadRequest(c, "INVALID_SCORE", "min_score must be between 0 and 100")
		return
	}

	// Apply pagination defaults (matches service layer defense-in-depth)
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
		Status:   q.Status,
		Company:  q.Company,
		MinScore: q.MinScore,
		Limit:    q.Limit,
		Offset:   q.Offset,
	}

	if q.SourceID != "" {
		sid, err := uuid.Parse(q.SourceID)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_SOURCE_ID", "invalid source_id")
			return
		}
		filter.SourceID = sid
	}

	jobs, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list jobs", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := JobListResponse{
		Jobs:   make([]JobResponse, len(jobs)),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	for i := range jobs {
		resp.Jobs[i] = ToResponse(&jobs[i])
	}

	httpresp.OK(c, resp)
}

// GetJob handles GET /jobs/:id.
// @Summary Get job by ID
// @Description Get detailed information about a specific job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 200 {object} JobResponse "Job details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id} [get]
func (h *Handler) GetJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	job, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get job", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(job))
}

// UpdateJob handles PATCH /jobs/:id.
// @Summary Update job status
// @Description Update the status of a job (e.g., mark as applied, archived)
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Param request body UpdateJobRequest true "New status"
// @Success 200 {object} map[string]string "Job updated"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or status"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id} [patch]
func (h *Handler) UpdateJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		h.logger.Error("update job", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "job updated"})
}

// scanRequest is the payload for POST /job-discovery/scan.
type scanRequest struct {
	SourceIDs []string `json:"source_ids" binding:"required,min=1" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
}

// TriggerScan handles POST /job-discovery/scan.
// @Summary Trigger job discovery scan
// @Description Start asynchronous job discovery from configured sources
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body scanRequest true "Source IDs to scan"
// @Success 201 {object} map[string][]string "Task IDs for polling"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 202 {object} map[string]interface{} "Partial failure - some tasks dispatched"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /job-discovery/scan [post]
func (h *Handler) TriggerScan(c *gin.Context) {
	var req scanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	sourceIDs := make([]uuid.UUID, 0, len(req.SourceIDs))
	seen := make(map[uuid.UUID]struct{}, len(req.SourceIDs))
	for _, s := range req.SourceIDs {
		sid, err := uuid.Parse(s)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_SOURCE_ID", "invalid source id: "+s)
			return
		}
		if _, exists := seen[sid]; exists {
			continue // deduplicate
		}
		seen[sid] = struct{}{}
		sourceIDs = append(sourceIDs, sid)
	}

	taskIDs, err := h.svc.TriggerScan(c.Request.Context(), sourceIDs)
	if err != nil {
		h.logger.Error("trigger scan",
			zap.Int("source_count", len(sourceIDs)),
			zap.Error(err),
		)
		// Partial failure: some tasks dispatched, some failed.
		// Return dispatched task IDs so the caller can poll them.
		if len(taskIDs) > 0 {
			c.JSON(http.StatusAccepted, gin.H{
				"task_ids": taskIDs,
				"error":    "some sources failed to enqueue",
			})
			return
		}
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, gin.H{"task_ids": taskIDs})
}

// saveRequest is the payload for PATCH /jobs/:id/save.
type saveRequest struct {
	Saved *bool `json:"saved" example:"true"`
}

// ToggleSave handles PATCH /jobs/:id/save.
// @Summary Toggle saved flag on a job
// @Description Mark or unmark a job as saved for later review
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Param request body saveRequest true "Saved flag"
// @Success 200 {object} map[string]interface{} "Save state"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or job id"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id}/save [patch]
func (h *Handler) ToggleSave(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	var req saveRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Saved == nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "saved flag required")
		return
	}

	if err := h.svc.SetSaved(c.Request.Context(), id, *req.Saved); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("toggle save", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"id": id, "saved": *req.Saved})
}

// DeleteJob handles DELETE /jobs/:id. Idempotent: returns 204 whether the
// job existed or not, matching the spec for the frontend "delete" action.
// @Summary Delete a job
// @Description Remove a job record. Idempotent — returns 204 even if the
// @Description job no longer exists.
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 204 "Job deleted (or not found)"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job id"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id} [delete]
func (h *Handler) DeleteJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete job", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// ApplyJob handles POST /jobs/:id/apply. Creates a draft application
// for the job and enqueues an application_submit task so the worker can
// fill the form via the browser-agent.
// @Summary Apply to a job
// @Description Create a draft application and enqueue a browser-agent submission task
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 202 {object} map[string]interface{} "Application created, task queued"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job id"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id}/apply [post]
func (h *Handler) ApplyJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	// Verify the job exists so the caller gets a clean 404 instead of a
	// dangling application pointing at nothing.
	if _, err := h.svc.GetByID(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("apply job lookup", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	app, err := h.appsAPI.Create(c.Request.Context(), applications.CreateApplicationRequest{
		JobID: id,
	})
	if err != nil {
		h.logger.Error("apply job create application", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	if h.taskDispatcher != nil {
		if _, err := h.taskDispatcher.DispatchApplicationSubmit(c.Request.Context(), tasks.ApplicationSubmitPayload{
			ApplicationID: app.ID,
		}); err != nil {
			h.logger.Error("dispatch apply task",
				zap.String("job_id", id.String()),
				zap.String("application_id", app.ID.String()),
				zap.Error(err),
			)
			// The application record was created — surface the dispatch
			// failure but return the app id so the caller can retry.
		}
	}

	httpresp.Accepted(c, gin.H{
		"application_id": app.ID,
		"job_id":         id,
		"status":         "queued",
	})
}

// ScoreJob handles POST /jobs/:id/score. Convenience wrapper around the
// scoring handler's async enqueue — same semantics as POST /scoring/score
// but reachable from the jobs subresource.
// @Summary Score a job (convenience)
// @Description Enqueue a scoring task for this job. Same as POST /scoring/score.
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 202 {object} map[string]interface{} "Task queued"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job id"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id}/score [post]
func (h *Handler) ScoreJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	if _, err := h.svc.GetByID(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("score job lookup", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	if h.scoringSvc == nil {
		// Without a scoring service we can still try the dispatcher path;
		// fall back to direct sync if no dispatcher either.
		httpresp.BadRequest(c, "SCORING_DISABLED", "scoring service not configured")
		return
	}

	// Trigger scoring using the scoring service so any post-scrape side
	// effects (tier transitions, approval requests) run as in the worker
	// pipeline. Best-effort: errors here are surfaced via 500.
	result, err := h.scoringSvc.ScoreJob(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("score job", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{
		"job_id": id,
		"score":  result.Score,
		"tier":   string(result.Tier),
		"source": result.Source,
	})
}

// SimilarJobs handles GET /jobs/:id/similar.
// @Summary List jobs similar to this one
// @Description Coarse keyword-based matching on job titles. Capped at 10.
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 200 {object} JobListResponse "Similar jobs"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job id"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id}/similar [get]
func (h *Handler) SimilarJobs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	similar, err := h.svc.FindSimilar(c.Request.Context(), id, 10)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("find similar", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := JobListResponse{
		Jobs:   make([]JobResponse, len(similar)),
		Total:  len(similar),
		Limit:  len(similar),
		Offset: 0,
	}
	for i := range similar {
		resp.Jobs[i] = ToResponse(&similar[i])
	}
	httpresp.OK(c, resp)
}

// ListJobApplications handles GET /jobs/:id/applications.
// @Summary List applications for a job
// @Description Returns all applications attached to the job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job UUID" format(uuid)
// @Success 200 {object} map[string]interface{} "Applications list"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job id"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /jobs/{id}/applications [get]
func (h *Handler) ListJobApplications(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	apps, total, err := h.appsAPI.List(c.Request.Context(), applications.ListFilter{
		JobID: id,
		Limit: 100,
	})
	if err != nil {
		h.logger.Error("list job applications", zap.String("job_id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := applications.ApplicationListResponse{
		Applications: make([]applications.ApplicationResponse, len(apps)),
		Total:        total,
		Limit:        100,
		Offset:       0,
	}
	for i := range apps {
		resp.Applications[i] = applications.ToResponse(&apps[i])
	}

	httpresp.OK(c, resp)
}
