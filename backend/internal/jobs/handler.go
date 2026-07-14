package jobs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the jobs HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new jobs handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("jobs"),
	}
}

// RegisterRoutes registers job routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	jobs := rg.Group("/jobs")
	{
		jobs.GET("", h.ListJobs)
		jobs.GET("/:id", h.GetJob)
		jobs.PATCH("/:id", h.UpdateJob)
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
