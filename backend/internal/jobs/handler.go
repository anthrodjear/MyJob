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
	SourceIDs []string `json:"source_ids" binding:"required,min=1"`
}

// TriggerScan handles POST /job-discovery/scan.
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
