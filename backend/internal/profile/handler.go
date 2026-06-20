// Handler handles HTTP requests for the profile domain.
//
// Profile is a singleton resource — no ID parameter in routes.
// Optimistic concurrency uses ETag / If-Match headers (RFC 7232).
//
// Routes:
//   - GET    /profile  → GetOrCreate (returns ETag header)
//   - PUT    /profile  → Update (requires If-Match header)
//   - PATCH  /profile  → UpdatePartial (requires If-Match header)
//
// This file contains NO business logic. It binds HTTP requests to
// service calls and maps domain errors to HTTP responses.
package profile

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler holds the profile HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new profile handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("profile.handler"),
	}
}

// RegisterRoutes registers profile routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	profile := rg.Group("/profile")
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.PATCH("", h.PatchProfile)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseETag extracts the version integer from an ETag or If-Match header.
// Header format: "123" (quoted integer per RFC 7232).
// Returns 0 and an error if the header is missing or malformed.
func parseETag(raw string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("header required")
	}
	// Strip surrounding quotes
	v := strings.Trim(raw, `"`)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid ETag format: %q", raw)
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// GetProfile handles GET /profile.
// Returns the singleton profile with embedded stats.
// Sets ETag header with the current version for use with If-Match.
func (h *Handler) GetProfile(c *gin.Context) {
	p, err := h.svc.GetOrCreate(c.Request.Context())
	if err != nil {
		h.logger.Error("get profile", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// ETag for optimistic concurrency — client sends back via If-Match
	c.Header("ETag", fmt.Sprintf(`"%d"`, p.Version))
	httpresp.OK(c, ToResponse(p))
}

// UpdateProfile handles PUT /profile.
// Replaces the entire profile data.
// Requires If-Match header with the version from the last GET.
func (h *Handler) UpdateProfile(c *gin.Context) {
	version, err := parseETag(c.GetHeader("If-Match"))
	if err != nil {
		httpresp.BadRequest(c, "MISSING_ETAG", "If-Match header with current version is required")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	updated, err := h.svc.Update(c.Request.Context(), req, version)
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "profile was modified — re-fetch and retry")
			return
		}
		h.logger.Error("update profile", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	c.Header("ETag", fmt.Sprintf(`"%d"`, updated.Version))
	httpresp.OK(c, ToResponse(updated))
}

// PatchProfile handles PATCH /profile.
// Partially merges fields into the existing profile data.
// Requires If-Match header with the version from the last GET.
func (h *Handler) PatchProfile(c *gin.Context) {
	version, err := parseETag(c.GetHeader("If-Match"))
	if err != nil {
		httpresp.BadRequest(c, "MISSING_ETAG", "If-Match header with current version is required")
		return
	}

	var req PatchProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	updated, err := h.svc.UpdatePartial(c.Request.Context(), req, version)
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			httpresp.Conflict(c, "VERSION_CONFLICT", "profile was modified — re-fetch and retry")
			return
		}
		h.logger.Error("patch profile", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	c.Header("ETag", fmt.Sprintf(`"%d"`, updated.Version))
	httpresp.OK(c, ToResponse(updated))
}
