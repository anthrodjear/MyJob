// Handler holds the interview session HTTP handlers.
//
// Responsibilities:
//   - Bind request input (JSON, path params, query params)
//   - Call service methods
//   - Map domain errors to HTTP status codes
//   - Return structured responses via httpresp helpers
//
// This file contains NO business logic. It is a thin translation layer
// between HTTP and the service layer.
//
// Routes:
//   POST   /api/v1/interviews              → CreateInterview
//   GET    /api/v1/interviews              → ListInterviews
//   GET    /api/v1/interviews/:id          → GetInterview
//   POST   /api/v1/interviews/:id/start    → StartInterview
//   POST   /api/v1/interviews/:id/stop     → StopInterview
//   POST   /internal/interviews/:id/events → HandleEvent (voice service only)
package interviews

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the interview session HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new interviews handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("interviews"),
	}
}

// RegisterRoutes registers interview routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	interviews := rg.Group("/interviews")
	{
		interviews.POST("", h.CreateInterview)
		interviews.GET("", h.ListInterviews)
		interviews.GET("/:id", h.GetInterview)
		interviews.POST("/:id/start", h.StartInterview)
		interviews.POST("/:id/stop", h.StopInterview)
	}

	// Internal endpoint for voice service callbacks (no auth — internal network only)
	internal := rg.Group("/internal/interviews")
	{
		internal.POST("/:id/events", h.HandleEvent)
	}
}

// ---------------------------------------------------------------------------
// Handlers: Mutations
// ---------------------------------------------------------------------------

// CreateInterview handles POST /interviews.
//
// Creates a new interview session in "pending" status.
// The session is not started until StartInterview is called.
func (h *Handler) CreateInterview(c *gin.Context) {
	var req CreateInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	session, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_MODE", err.Error())
			return
		}
		h.logger.Error("create interview", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToResponse(session))
}

// StartInterview handles POST /interviews/:id/start.
//
// Starts the interview session. The backend enqueues a voice_session task
// for the browser-agent, which joins the LiveKit room and begins the interview.
func (h *Handler) StartInterview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid interview id")
		return
	}

	var req StartInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	session, err := h.svc.Start(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "INTERVIEW_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		h.logger.Error("start interview", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(session))
}

// StopInterview handles POST /interviews/:id/stop.
//
// Stops an active interview session. The voice service is notified
// and the session transitions to "cancelled".
func (h *Handler) StopInterview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid interview id")
		return
	}

	var req StopInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.Stop(c.Request.Context(), id, req); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "INTERVIEW_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		h.logger.Error("stop interview", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "interview stopped"})
}

// HandleEvent handles POST /internal/interviews/:id/events.
//
// This is an INTERNAL endpoint used by the voice service (browser-agent)
// to report transcript entries, status changes, scores, and feedback.
// It is NOT exposed to the frontend.
func (h *Handler) HandleEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid interview id")
		return
	}

	var req InterviewEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.HandleEvent(c.Request.Context(), id, req); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "INTERVIEW_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpresp.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		h.logger.Error("handle interview event",
			zap.String("id", id.String()),
			zap.String("type", req.Type),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "event processed"})
}

// ---------------------------------------------------------------------------
// Handlers: Queries
// ---------------------------------------------------------------------------

// GetInterview handles GET /interviews/:id.
func (h *Handler) GetInterview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid interview id")
		return
	}

	session, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "INTERVIEW_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get interview", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(session))
}

// listInterviewsQuery holds query parameters for listing interviews.
type listInterviewsQuery struct {
	ApplicationID string `form:"application_id"`
	Status        string `form:"status"`
	Mode          string `form:"mode"`
	Limit         int    `form:"limit"`
	Offset        int    `form:"offset"`
}

// ListInterviews handles GET /interviews.
func (h *Handler) ListInterviews(c *gin.Context) {
	var q listInterviewsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	// Validate status filter
	if q.Status != "" && !IsValidStatus(q.Status) {
		httpresp.BadRequest(c, "INVALID_STATUS", "invalid status filter")
		return
	}

	// Validate mode filter
	if q.Mode != "" && !IsValidMode(q.Mode) {
		httpresp.BadRequest(c, "INVALID_MODE", "invalid mode filter")
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
		Status: q.Status,
		Mode:   q.Mode,
		Limit:  q.Limit,
		Offset: q.Offset,
	}

	if q.ApplicationID != "" {
		aid, err := uuid.Parse(q.ApplicationID)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_APPLICATION_ID", "invalid application_id")
			return
		}
		filter.ApplicationID = aid
	}

	sessions, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list interviews", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := InterviewListResponse{
		Interviews: make([]InterviewResponse, len(sessions)),
		Total:      total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}
	for i := range sessions {
		resp.Interviews[i] = ToResponse(&sessions[i])
	}

	httpresp.OK(c, resp)
}
