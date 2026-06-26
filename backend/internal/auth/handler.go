package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the auth HTTP handlers.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("auth"),
	}
}

// RegisterRoutes registers auth routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/change-password", h.ChangePassword)
		auth.GET("/setup/status", h.SetupStatus)
		auth.POST("/setup", h.CompleteSetup)
	}
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.logger.Warn("login failed", zap.String("error", err.Error()))
			httpresp.Unauthorized(c, "INVALID_CREDENTIALS", "invalid credentials")
			return
		}
		h.logger.Error("login error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Update last login timestamp
	if err := h.service.UpdateLastLogin(c.Request.Context()); err != nil {
		h.logger.Error("update last login", zap.Error(err))
		// Non-fatal, don't fail the login
	}

	httpresp.OK(c, resp)
}

// ChangePassword handles POST /auth/change-password.
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpresp.Unauthorized(c, "INVALID_CREDENTIALS", "current password incorrect")
			return
		}
		h.logger.Error("change password error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"message": "password changed"})
}

// SetupStatus handles GET /auth/setup/status.
func (h *Handler) SetupStatus(c *gin.Context) {
	resp, err := h.service.GetSetupStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("get setup status error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}
	httpresp.OK(c, resp)
}

// CompleteSetup handles POST /auth/setup.
func (h *Handler) CompleteSetup(c *gin.Context) {
	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.service.CompleteSetup(c.Request.Context(), req.Username, req.Email, req.Password); err != nil {
		if errors.Is(err, ErrSetupAlreadyComplete) {
			httpresp.Conflict(c, "SETUP_COMPLETE", "setup already completed")
			return
		}
		h.logger.Error("complete setup error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, SetupResponse{Message: "setup completed successfully"})
}
