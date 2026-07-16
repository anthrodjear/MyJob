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
		resumes.GET("/:id/content", h.GetContent)
		resumes.PUT("/:id/content", h.UpdateContent)
		resumes.POST("/:id/generate", h.GenerateContent)
		resumes.GET("/:id/versions", h.ListVersions)
		resumes.GET("/:id/versions/:version", h.GetVersion)
	}

	coverLetters := rg.Group("/cover-letters")
	{
		coverLetters.GET("", h.ListCoverLetters)
		coverLetters.GET("/:id", h.GetCoverLetter)
		coverLetters.POST("", h.CreateCoverLetter)
		coverLetters.POST("/:id/generate", h.GenerateCoverLetter)
		coverLetters.PUT("/:id/content", h.UpdateCoverLetterContent)
		coverLetters.DELETE("/:id", h.DeleteCoverLetter)
		coverLetters.GET("/:id/versions", h.ListCoverLetterVersions)
		coverLetters.GET("/:id/versions/:version", h.GetCoverLetterVersion)
	}
}

// --- Resume handlers ---

// ListResumes handles GET /resumes.
// @Summary List resumes
// @Description Get paginated list of resumes (content omitted)
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Results per page (max 100)" default(20) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} ResumeListResponse "Paginated resume list"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes [get]
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
// @Summary Get resume by ID
// @Description Get detailed resume information including structured content
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Success 200 {object} ResumeDetailResponse "Resume with content"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid resume ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id} [get]
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

	httpresp.OK(c, ToDetailResponse(resume))
}

// CreateResume handles POST /resumes.
// @Summary Create resume
// @Description Create a new resume profile
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateResumeRequest true "Resume creation request"
// @Success 201 {object} ResumeDetailResponse "Created resume"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes [post]
func (h *Handler) CreateResume(c *gin.Context) {
	var req CreateResumeRequest
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

	httpresp.Created(c, ToDetailResponse(resume))
}

// UpdateResume handles PUT /resumes/:id.
// @Summary Update resume
// @Description Update resume profile (optimistic locking via version)
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Param request body UpdateResumeRequest true "Resume update request"
// @Success 200 {object} ResumeDetailResponse "Updated resume"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 409 {object} httpresp.ErrorResponse "Version conflict - resume modified by another process"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id} [put]
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

	// Bind to dedicated DTO — never bind directly onto domain model (field injection risk)
	var req UpdateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	// Apply DTO fields to existing resume (preserves ID, Version, timestamps)
	existing.Name = req.Name
	existing.Specialization = req.Specialization
	existing.TemplatePath = req.TemplatePath
	existing.FocusSkills = req.FocusSkills
	existing.HighlightExperience = req.HighlightExperience

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "resume was modified by another process — please refresh")
			return
		}
		h.logger.Error("update resume", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToDetailResponse(existing))
}

// DeleteResume handles DELETE /resumes/:id.
// @Summary Delete resume
// @Description Delete a resume permanently
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Success 200 {object} map[string]string "Resume deleted"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid resume ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id} [delete]
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

// --- Content handlers ---

// GetContent handles GET /resumes/:id/content.
// @Summary Get resume content
// @Description Get the structured content of a resume
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Success 200 {object} ResumeContentResponse "Resume content with version"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid resume ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found or no content"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id}/content [get]
func (h *Handler) GetContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	content, version, err := h.svc.GetContent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrNoContent) {
			httpresp.NotFound(c, "CONTENT_NOT_FOUND", "resume has no generated content")
			return
		}
		h.logger.Error("get content", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ResumeContentResponse{
		ResumeID: id,
		Version:  version,
		Content:  *content,
	})
}

// UpdateContent handles PUT /resumes/:id/content.
// @Summary Update resume content
// @Description Manually override structured resume content (optimistic locking)
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Param request body UpdateResumeContentRequest true "Content update"
// @Success 200 {object} ResumeContentResponse "Updated content with version"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 409 {object} httpresp.ErrorResponse "Version conflict"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id}/content [put]
func (h *Handler) UpdateContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	var req UpdateResumeContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	content, version, err := h.svc.UpdateContent(c.Request.Context(), id, req.Content)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "resume was modified by another process — please refresh")
			return
		}
		h.logger.Error("update content", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ResumeContentResponse{
		ResumeID: id,
		Version:  version,
		Content:  *content,
	})
}

// GenerateContent handles POST /resumes/:id/generate.
// @Summary Generate resume content
// @Description Generate structured resume content using LLM (synchronous)
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Param request body GenerateResumeContentRequest true "Generation options"
// @Success 200 {object} ResumeContentResponse "Generated content with version"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 409 {object} httpresp.ErrorResponse "Version conflict"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id}/generate [post]
func (h *Handler) GenerateContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	var req GenerateResumeContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	content, version, err := h.svc.GenerateContent(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "resume was modified by another process — please refresh")
			return
		}
		h.logger.Error("generate content", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ResumeContentResponse{
		ResumeID: id,
		Version:  version,
		Content:  *content,
	})
}

// ListVersions handles GET /resumes/:id/versions.
// @Summary List resume versions
// @Description Get all historical versions of a resume
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Success 200 {object} ResumeVersionListResponse "List of resume versions"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid resume ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Resume not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id}/versions [get]
func (h *Handler) ListVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	versions, err := h.svc.GetVersions(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "RESUME_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("list versions", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ResumeVersionListResponse{
		Versions: ToVersionResponses(versions),
	})
}

// GetVersion handles GET /resumes/:id/versions/:version.
// @Summary Get resume version
// @Description Get a specific historical version of a resume
// @Tags Resumes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resume UUID" format(uuid)
// @Param version path int true "Version number"
// @Success 200 {object} ResumeVersionResponse "Resume version with content"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid resume ID or version"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Version not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /resumes/{id}/versions/{version} [get]
func (h *Handler) GetVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid resume id")
		return
	}

	version, err := strconv.ParseInt(c.Param("version"), 10, 32)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_VERSION", "invalid version number")
		return
	}

	v, err := h.svc.GetVersion(c.Request.Context(), id, int32(version))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "VERSION_NOT_FOUND", "version not found")
			return
		}
		h.logger.Error("get version", zap.String("id", id.String()), zap.Int64("version", version), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToVersionResponse(v))
}

// --- Cover Letter handlers ---

// ListCoverLetters handles GET /cover-letters.
// @Summary List cover letters
// @Description Get paginated list of cover letters
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Results per page (max 100)" default(20) minimum(1) maximum(100)
// @Param offset query int false "Pagination offset" default(0) minimum(0)
// @Success 200 {object} CoverLetterListResponse "Paginated cover letter list"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters [get]
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
// @Summary Get cover letter by ID
// @Description Get detailed cover letter information
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Success 200 {object} CoverLetterResponse "Cover letter details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid cover letter ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Cover letter not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id} [get]
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
// @Summary Create cover letter
// @Description Create a new cover letter placeholder
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCoverLetterRequest true "Cover letter creation request"
// @Success 201 {object} CoverLetterResponse "Created cover letter"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters [post]
func (h *Handler) CreateCoverLetter(c *gin.Context) {
	var req CreateCoverLetterRequest
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

// GenerateCoverLetter handles POST /cover-letters/:id/generate.
// @Summary Generate cover letter
// @Description Generate cover letter content using LLM
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Param request body GenerateCoverLetterRequest true "Generation options"
// @Success 200 {object} CoverLetterResponse "Generated cover letter"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Cover letter not found"
// @Failure 409 {object} httpresp.ErrorResponse "Version conflict"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id}/generate [post]
func (h *Handler) GenerateCoverLetter(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	var req GenerateCoverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	cl, err := h.svc.GenerateCoverLetter(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "COVER_LETTER_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "cover letter was modified by another process — please refresh")
			return
		}
		h.logger.Error("generate cover letter", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToCoverLetterResponse(cl))
}

// UpdateCoverLetterContent handles PUT /cover-letters/:id/content.
// @Summary Update cover letter content
// @Description Manually override cover letter content (optimistic locking)
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Param request body UpdateCoverLetterContentRequest true "Content update"
// @Success 200 {object} CoverLetterResponse "Updated cover letter"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Cover letter not found"
// @Failure 409 {object} httpresp.ErrorResponse "Version conflict"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id}/content [put]
func (h *Handler) UpdateCoverLetterContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	var req UpdateCoverLetterContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	cl, err := h.svc.UpdateCoverLetterContent(c.Request.Context(), id, req.Content)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "COVER_LETTER_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "cover letter was modified by another process — please refresh")
			return
		}
		h.logger.Error("update cover letter content", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToCoverLetterResponse(cl))
}

// DeleteCoverLetter handles DELETE /cover-letters/:id.
// @Summary Delete cover letter
// @Description Delete a cover letter permanently
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Success 200 {object} map[string]string "Cover letter deleted"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid cover letter ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Cover letter not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id} [delete]
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

// --- Cover Letter Version handlers ---

// ListCoverLetterVersions handles GET /cover-letters/:id/versions.
// @Summary List cover letter versions
// @Description Get all historical versions of a cover letter
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Success 200 {object} CoverLetterVersionListResponse "List of cover letter versions"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid cover letter ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Cover letter not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id}/versions [get]
func (h *Handler) ListCoverLetterVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	versions, err := h.svc.ListCoverLetterVersions(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "COVER_LETTER_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("list cover letter versions", zap.String("id", id.String()), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, CoverLetterVersionListResponse{
		Versions: ToCoverLetterVersionResponses(versions),
	})
}

// GetCoverLetterVersion handles GET /cover-letters/:id/versions/:version.
// @Summary Get cover letter version
// @Description Get a specific historical version of a cover letter
// @Tags CoverLetters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Cover letter UUID" format(uuid)
// @Param version path int true "Version number"
// @Success 200 {object} CoverLetterVersionResponse "Cover letter version with content"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid cover letter ID or version"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Version not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /cover-letters/{id}/versions/{version} [get]
func (h *Handler) GetCoverLetterVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid cover letter id")
		return
	}

	version, err := strconv.ParseInt(c.Param("version"), 10, 32)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_VERSION", "invalid version number")
		return
	}

	v, err := h.svc.GetCoverLetterVersion(c.Request.Context(), id, int32(version))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "VERSION_NOT_FOUND", "version not found")
			return
		}
		h.logger.Error("get cover letter version", zap.String("id", id.String()), zap.Int64("version", version), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToCoverLetterVersionResponse(v))
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
