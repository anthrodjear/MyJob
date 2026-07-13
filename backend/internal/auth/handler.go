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

// Login handles POST /auth/login.
// @Summary User login
// @Description Authenticate user with password and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Successful login with tokens"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Invalid credentials"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/login [post]
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
// @Summary Change password
// @Description Change the current user's password (requires authentication)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} OnboardingConfigResponse "Password changed successfully"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Current password incorrect or unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/change-password [post]
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
// @Summary Get setup status
// @Description Check if initial setup is required and get current onboarding step
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} SetupStatusResponse "Setup status"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/status [get]
func (h *Handler) SetupStatus(c *gin.Context) {
	resp, err := h.service.GetSetupStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("get setup status error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, resp)
}

// Logout handles POST /auth/logout.
// @Summary User logout
// @Description Revoke all refresh tokens and increment session version (requires authentication)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutResponse "Logged out successfully"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	if err := h.service.Logout(c.Request.Context()); err != nil {
		h.logger.Error("logout error", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, LogoutResponse{Message: "logged out"})
}

// CompleteSetup handles POST /auth/setup.
// @Summary Complete initial setup
// @Description Create the first user account and complete initial setup
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body SetupRequest true "Setup credentials"
// @Success 200 {object} SetupResponse "Setup completed successfully"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 409 {object} httpresp.ErrorResponse "Setup already completed"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup [post]
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
// @Summary Test LLM API key
// @Description Validate an LLM provider API key during onboarding
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body TestLLMRequest true "LLM provider and API key"
// @Success 200 {object} TestLLMResponse "Validation result"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/test-llm [post]
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
// @Summary Test voice configuration
// @Description Validate LiveKit voice configuration during onboarding
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body TestVoiceRequest true "Voice configuration"
// @Success 200 {object} TestLLMResponse "Validation result"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/test-voice [post]
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
// @Summary Test email configuration
// @Description Validate Microsoft Graph email configuration during onboarding
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body TestEmailRequest true "Email configuration"
// @Success 200 {object} TestLLMResponse "Validation result"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body or tenant ID"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/test-email [post]
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
// @Summary Save onboarding configuration
// @Description Save LLM keys, voice config, email config, and preferences during onboarding
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body OnboardingConfigRequest true "Onboarding configuration"
// @Success 200 {object} OnboardingConfigResponse "Configuration saved"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/config [post]
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
// @Summary Update onboarding step
// @Description Update the current onboarding step to track progress
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body OnboardingStepRequest true "Step to set"
// @Success 200 {object} OnboardingConfigResponse "Step updated"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/onboarding-step [post]
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
// @Summary Complete onboarding
// @Description Mark onboarding as complete and enable full API access
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} OnboardingConfigResponse "Onboarding completed"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/setup/complete-onboarding [post]
func (h *Handler) CompleteOnboarding(c *gin.Context) {
	if err := h.service.CompleteOnboarding(c.Request.Context()); err != nil {
		h.logger.Error("complete onboarding", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, OnboardingConfigResponse{Message: "onboarding completed"})
}

// Refresh handles POST /auth/refresh.
// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshResponse "New access and refresh tokens"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
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
