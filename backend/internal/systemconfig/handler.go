// Package systemconfig provides HTTP handlers for the system configuration API.
// This file implements the Gin handlers for GET/PATCH/DELETE /api/v1/system/config.
//
// # Endpoints
//
//   - GET /api/v1/system/config — returns the fully resolved EffectiveConfig
//   - PATCH /api/v1/system/config — creates or updates a runtime override
//   - DELETE /api/v1/system/config/:key — removes a runtime override
//
// # Design Constraints
//
//   - Handlers parse input, call service, return response. No business logic here.
//   - All responses use the httpresp pattern for consistent error/success formatting.
//   - The handler depends on the Service (not Repository or Resolver) — single responsibility.
package systemconfig

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler implements the HTTP handlers for system configuration endpoints.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new systemconfig handler.
// The service handles business logic; logger is for request-level logging.
//
// Example:
//
//	handler := systemconfig.NewHandler(service, logger)
//	handler.RegisterRoutes(protected)
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("systemconfig"),
	}
}

// RegisterRoutes registers the system config routes on the given router group.
// Expects to be called with a protected (auth-required) route group.
//
// Routes:
//
//	GET    /config        → GetEffectiveConfig
//	PATCH  /config        → SetOverride
//	DELETE /config/:key   → DeleteOverride
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	system := rg.Group("/system")
	{
		system.GET("/config", h.GetEffectiveConfig)
		system.PATCH("/config", h.SetOverride)
		system.DELETE("/config/:key", h.DeleteOverride)
	}
}

// GetEffectiveConfig handles GET /api/v1/system/config.
// Returns the fully resolved configuration tree merging YAML, env, and DB layers.
//
// Response: EffectiveConfigResponse with config and optional version.
func (h *Handler) GetEffectiveConfig(c *gin.Context) {
	effect, err := h.service.GetEffectiveConfig(c.Request.Context())
	if err != nil {
		h.logger.Error("get config", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, EffectiveConfigResponse{
		EffectiveConfig: *effect,
	})
}

// SetOverride handles PATCH /api/v1/system/config.
// Creates or updates a runtime configuration override.
//
// Request body: { "key": "scoring.auto_threshold", "value": 90 }
// Response: { "message": "override saved", "key": "scoring.auto_threshold" }
func (h *Handler) SetOverride(c *gin.Context) {
	var req SetOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	// Convert value to json.RawMessage for storage
	rawValue, err := json.Marshal(req.Value)
	if err != nil {
		httpresp.BadRequest(c, "INVALID_VALUE", "value must be valid JSON")
		return
	}

	if err := h.service.SetOverride(c.Request.Context(), req.Key, rawValue); err != nil {
		// Map domain errors to HTTP codes
		if errors.Is(err, ErrInvalidKeyFormat) {
			httpresp.BadRequest(c, "INVALID_KEY_FORMAT", err.Error())
			return
		}
		if errors.Is(err, ErrKeyNotAllowed) {
			httpresp.BadRequest(c, "KEY_NOT_ALLOWED", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidValue) {
			httpresp.BadRequest(c, "INVALID_VALUE", err.Error())
			return
		}
		h.logger.Error("set override",
			zap.String("key", req.Key),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, SetOverrideResponse{
		Message: "override saved",
		Key:     req.Key,
	})
}

// DeleteOverride handles DELETE /api/v1/system/config/:key.
// Removes a runtime configuration override by key.
//
// Response: { "message": "override deleted", "key": "scoring.auto_threshold" }
func (h *Handler) DeleteOverride(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		httpresp.BadRequest(c, "INVALID_KEY", "key parameter is required")
		return
	}

	if err := h.service.DeleteOverride(c.Request.Context(), key); err != nil {
		// Map domain errors to HTTP codes
		if errors.Is(err, ErrInvalidKeyFormat) {
			httpresp.BadRequest(c, "INVALID_KEY_FORMAT", err.Error())
			return
		}
		h.logger.Error("delete override",
			zap.String("key", key),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, DeleteOverrideResponse{
		Message: "override deleted",
		Key:     key,
	})
}
