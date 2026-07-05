// Handler contains HTTP handlers for the activity domain.
//
// Endpoints:
//
//	GET /activity-logs         → List activity logs with filters
//	GET /activity-logs/:id     → Get single activity log
//
// Activity logs are read-only via HTTP — other domains write events
// through Service.LogEvent(). The handler translates domain errors
// to HTTP status codes and uses shared response helpers.
//
// All endpoints require authentication (handled by middleware).
package activity

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler contains dependencies for activity HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new activity handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("activity.handler"),
	}
}

// RegisterRoutes registers activity routes on the router group.
// Routes are mounted under /activity-logs as a sub-resource.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	activity := rg.Group("/activity-logs")
	{
		activity.GET("", h.List)
		activity.GET("/:id", h.GetByID)
	}
}

// List handles GET /activity-logs
// Returns paginated list of activity logs with optional filters.
// Query params: entity_type, entity_id, event_type, start_time, end_time, limit, offset.
func (h *Handler) List(c *gin.Context) {
	var req ListFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn("invalid list activity request", zap.Error(err))
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	filter, err := req.ToListFilter()
	if err != nil {
		if errors.Is(err, ErrInvalidEntityID) {
			httpresp.BadRequest(c, "INVALID_ENTITY_ID", "invalid entity_id")
			return
		}
		if errors.Is(err, ErrInvalidTimeRange) {
			httpresp.BadRequest(c, "INVALID_TIME_RANGE", "start_time and end_time must be RFC3339")
			return
		}
		h.logger.Error("failed to convert list filter", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Validate pagination at handler boundary (service receives already-validated input)
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	activities, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list activities", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := ActivityListResponse{
		Activities: make([]ActivityResponse, len(activities)),
		Total:      total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}
	for i, a := range activities {
		resp.Activities[i] = ToActivityResponse(&a)
	}

	httpresp.OK(c, resp)
}

// GetByID handles GET /activity-logs/:id
// Returns a single activity log entry by ID.
func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid activity log id")
		return
	}

	a, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "ACTIVITY_NOT_FOUND", "activity log not found")
			return
		}
		h.logger.Error("failed to get activity", zap.Error(err), zap.String("id", idStr))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToActivityResponse(a))
}
