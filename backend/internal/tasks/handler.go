package tasks

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
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
// @Summary Create a task
// @Description Create a new async task (internal use - tasks are typically created by other endpoints)
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTaskRequest true "Task creation request"
// @Success 201 {object} TaskResponse "Task created"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or task type"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /tasks [post]
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	task, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidType) {
			httpresp.BadRequest(c, "INVALID_TYPE", err.Error())
			return
		}
		h.logger.Error(
			"create task",
			zap.String("task_type", req.Type),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToResponse(task))
}

// GetTask handles GET /tasks/:id.
// @Summary Get task status
// @Description Get the status and result of an async task by ID
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID (UUID)"
// @Success 200 {object} TaskResponse "Task details with status and result"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid task ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Task not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /tasks/{id} [get]
func (h *Handler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid task id")
		return
	}

	task, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "TASK_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error(
			"get task",
			zap.String("task_id", id.String()),
			zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(task))
}

// listTasksQuery holds the query parameters for listing tasks.
type listTasksQuery struct {
	Status string `form:"status"`
	Type   string `form:"type"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ListTasks handles GET /tasks.
// @Summary List tasks
// @Description Get paginated list of tasks with optional filters
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(pending,running,completed,failed,cancelled)
// @Param type query string false "Filter by task type" Enums(job_discovery,job_scoring,resume_generate,cover_letter_gen,application_submit,fill_form,email_check,interview_prep,embedding_generate,voice_session,resume_tailor)
// @Param limit query int false "Results per page (max 100)" default(20) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} TaskListResponse "Paginated task list"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid query parameters"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /tasks [get]
func (h *Handler) ListTasks(c *gin.Context) {
	var q listTasksQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpresp.BadRequest(c, "INVALID_QUERY", "invalid query parameters")
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
		httpresp.InternalError(c)
		return
	}

	resp := TaskListResponse{
		Tasks: make([]TaskResponse, len(tasks)),
		Total: total,
	}
	for i := range tasks {
		resp.Tasks[i] = ToResponse(&tasks[i])
	}

	httpresp.OK(c, resp)
}

// RegisterRoutes registers task routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	tasks := rg.Group("/tasks")
	{
		tasks.POST("", h.CreateTask)
		tasks.GET("", h.ListTasks)
		tasks.GET("/:id", h.GetTask)
	}
}
