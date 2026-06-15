package resumes

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the resumes HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new resumes handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("resumes"),
	}
}

// RegisterRoutes registers resume and cover letter routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resumes := rg.Group("/resumes")
	{
		resumes.GET("", h.ListResumes)
		resumes.GET("/:id", h.GetResume)
		resumes.POST("", h.CreateResume)
		resumes.PUT("/:id", h.UpdateResume)
		resumes.DELETE("/:id", h.DeleteResume)
	}

	coverLetters := rg.Group("/cover-letters")
	{
		coverLetters.GET("", h.ListCoverLetters)
		coverLetters.GET("/:id", h.GetCoverLetter)
		coverLetters.POST("", h.CreateCoverLetter)
		coverLetters.DELETE("/:id", h.DeleteCoverLetter)
	}
}

// --- Resume handlers ---

// ListResumes handles GET /resumes.
func (h *Handler) ListResumes(c *gin.Context) {
	limit, offset := h.parsePagination(c)

	resumes, total, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("list resumes", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ResumeListResponse{
		Resumes: ToResponses(resumes),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// GetResume handles GET /resumes/:id.
func (h *Handler) GetResume(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	resume, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get resume", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(resume))
}

// CreateResume handles POST /resumes.
func (h *Handler) CreateResume(c *gin.Context) {
	var req GenerateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	resume, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			httpresp.BadRequest(c, "INVALID_INPUT", err.Error())
			return
		}
		h.logger.Error("create resume", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToResponse(resume))
}

// UpdateResume handles PUT /resumes/:id.
func (h *Handler) UpdateResume(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	// Fetch existing resume for version check
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get resume for update", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Bind update fields onto existing resume
	if err := c.ShouldBindJSON(existing); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("update resume", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToResponse(existing))
}

// DeleteResume handles DELETE /resumes/:id.
func (h *Handler) DeleteResume(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("delete resume", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "resume deleted"})
}

// --- Cover Letter handlers ---

// ListCoverLetters handles GET /cover-letters.
func (h *Handler) ListCoverLetters(c *gin.Context) {
	limit, offset := h.parsePagination(c)

	letters, total, err := h.svc.ListCoverLetters(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("list cover letters", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, CoverLetterListResponse{
		CoverLetters: ToCoverLetterResponses(letters),
		Total:        total,
		Limit:        limit,
		Offset:       offset,
	})
}

// GetCoverLetter handles GET /cover-letters/:id.
func (h *Handler) GetCoverLetter(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	cl, err := h.svc.GetCoverLetterByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "COVER_LETTER_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get cover letter", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToCoverLetterResponse(cl))
}

// CreateCoverLetter handles POST /cover-letters.
func (h *Handler) CreateCoverLetter(c *gin.Context) {
	var req GenerateCoverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	cl, err := h.svc.CreateCoverLetter(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("create cover letter", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.Created(c, ToCoverLetterResponse(cl))
}

// DeleteCoverLetter handles DELETE /cover-letters/:id.
func (h *Handler) DeleteCoverLetter(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	if err := h.svc.DeleteCoverLetter(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "COVER_LETTER_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("delete cover letter", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "cover letter deleted"})
}

// --- Helpers ---

// parsePagination extracts and validates limit/offset from query parameters.
func (h *Handler) parsePagination(c *gin.Context) (int, int) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
