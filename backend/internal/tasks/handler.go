package tasks

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/api"
)

// Handler holds dependencies for task HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new tasks handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// CreateTask handles POST /tasks.
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	task, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidType) {
			api.BadRequest(c, "INVALID_TYPE", err.Error())
			return
		}
		h.logger.Error(
			"create task",
			zap.String("task_type", req.Type),
			zap.Error(err),
		)
		api.InternalError(c)
		return
	}

	api.Created(c, ToResponse(task))
}

// GetTask handles GET /tasks/:id.
func (h *Handler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequest(c, "INVALID_ID", "invalid task id")
		return
	}

	task, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.NotFound(c, "TASK_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error(
			"get task",
			zap.String("task_id", id.String()),
			zap.Error(err))
		api.InternalError(c)
		return
	}

	api.OK(c, ToResponse(task))
}

// listTasksQuery holds the query parameters for listing tasks.
type listTasksQuery struct {
	Status string `form:"status"`
	Type   string `form:"type"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ListTasks handles GET /tasks.
func (h *Handler) ListTasks(c *gin.Context) {
	var q listTasksQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		api.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
		return
	}

	// Defaults
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	tasks, total, err := h.svc.List(c.Request.Context(), q.Status, q.Type, q.Limit, q.Offset)
	if err != nil {
		h.logger.Error(
			"list tasks",
			zap.Error(err),
		)
		api.InternalError(c)
		return
	}

	resp := TaskListResponse{
		Tasks: make([]TaskResponse, len(tasks)),
		Total: total,
	}
	for i := range tasks {
		resp.Tasks[i] = ToResponse(&tasks[i])
	}

	api.OK(c, resp)
}
