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
		auth.POST("/setup/test-llm", h.TestLLMKey)
		auth.POST("/setup/test-voice", h.TestVoiceConfig)
		auth.POST("/setup/test-email", h.TestEmailConfig)
		auth.POST("/setup/config", h.SaveOnboardingConfig)
		auth.POST("/setup/onboarding-step", h.UpdateOnboardingStep)
		auth.POST("/setup/complete-onboarding", h.CompleteOnboarding)
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

	httpresp.OK(c, OnboardingConfigResponse{Message: "password changed"})
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

// TestLLMKey handles POST /auth/setup/test-llm.
func (h *Handler) TestLLMKey(c *gin.Context) {
	var req TestLLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	valid, err := h.service.TestLLMKey(c.Request.Context(), req.Provider, req.APIKey)
	if err != nil {
		h.logger.Error("test llm key error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, TestLLMResponse{Valid: valid})
}

// TestVoiceConfig handles POST /auth/setup/test-voice.
func (h *Handler) TestVoiceConfig(c *gin.Context) {
	var req TestVoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	valid, err := h.service.TestVoiceConfig(c.Request.Context(), req.URL, req.APIKey, req.APISecret)
	if err != nil {
		h.logger.Error("test voice config error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, TestLLMResponse{Valid: valid})
}

// TestEmailConfig handles POST /auth/setup/test-email.
func (h *Handler) TestEmailConfig(c *gin.Context) {
	var req TestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	valid, err := h.service.TestEmailConfig(c.Request.Context(), req.TenantID, req.ClientID, req.ClientSecret)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpresp.BadRequest(c, "INVALID_TENANT_ID", "invalid tenant ID format")
			return
		}
		h.logger.Error("test email config error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, TestLLMResponse{Valid: valid})
}

// SaveOnboardingConfig handles POST /auth/setup/config.
func (h *Handler) SaveOnboardingConfig(c *gin.Context) {
	var req OnboardingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.service.SaveOnboardingConfig(c.Request.Context(), &req); err != nil {
		h.logger.Error("save onboarding config", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, OnboardingConfigResponse{Message: "configuration saved"})
}

// UpdateOnboardingStep handles POST /auth/setup/onboarding-step.
func (h *Handler) UpdateOnboardingStep(c *gin.Context) {
	var req OnboardingStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.service.UpdateOnboardingStep(c.Request.Context(), req.Step); err != nil {
		h.logger.Error("update onboarding step", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, OnboardingConfigResponse{Message: "step updated"})
}

// CompleteOnboarding handles POST /auth/setup/complete-onboarding.
func (h *Handler) CompleteOnboarding(c *gin.Context) {
	if err := h.service.CompleteOnboarding(c.Request.Context()); err != nil {
		h.logger.Error("complete onboarding", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, OnboardingConfigResponse{Message: "onboarding completed"})
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	resp, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenInvalid) || errors.Is(err, ErrRefreshTokenExpired) || errors.Is(err, ErrRefreshTokenRevoked) {
			httpresp.Unauthorized(c, "INVALID_REFRESH_TOKEN", "refresh token is invalid or expired")
			return
		}
		h.logger.Error("refresh token error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, resp)
}
