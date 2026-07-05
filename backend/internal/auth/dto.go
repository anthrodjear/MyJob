package auth

import "github.com/golang-jwt/jwt/v5"

// --- Request DTOs ---

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest is the payload for POST /auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// --- Response DTOs ---

// LoginResponse is returned on successful login.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

// --- JWT Claims ---

// Claims holds the JWT claims for the single local user.
type Claims struct {
	UserID         string `json:"user_id"`
	SessionVersion int    `json:"session_version"`
	jwt.RegisteredClaims
}

// --- Setup DTOs ---

// SetupStatusResponse is returned by GET /auth/setup/status.
// Extended with onboarding step for resume capability.
type SetupStatusResponse struct {
	SetupRequired bool   `json:"setup_required"`
	Step          string `json:"step,omitempty"`
}

// SetupRequest is the payload for POST /auth/setup.
type SetupRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// SetupResponse is returned on successful setup.
type SetupResponse struct {
	Message string `json:"message"`
}

// --- Onboarding DTOs ---

// TestLLMRequest is the payload for POST /auth/setup/test-llm.
type TestLLMRequest struct {
	Provider string `json:"provider" binding:"required,oneof=openai anthropic"`
	APIKey   string `json:"api_key" binding:"required"`
}

// TestLLMResponse is returned by POST /auth/setup/test-llm.
type TestLLMResponse struct {
	Valid bool `json:"valid"`
}

// TestVoiceRequest is the payload for POST /auth/setup/test-voice.
type TestVoiceRequest struct {
	URL       string `json:"url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	APISecret string `json:"api_secret" binding:"required"`
}

// TestEmailRequest is the payload for POST /auth/setup/test-email.
type TestEmailRequest struct {
	TenantID     string `json:"tenant_id" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
}

// OnboardingConfigRequest is the payload for POST /auth/setup/config.
type OnboardingConfigRequest struct {
	OpenAIKey       string   `json:"openai_key,omitempty"`
	AnthropicKey    string   `json:"anthropic_key,omitempty"`
	LivekitURL      string   `json:"livekit_url,omitempty"`
	LivekitKey      string   `json:"livekit_key,omitempty"`
	LivekitSecret   string   `json:"livekit_secret,omitempty"`
	MSTenantID      string   `json:"ms_tenant_id,omitempty"`
	MSClientID      string   `json:"ms_client_id,omitempty"`
	MSClientSecret  string   `json:"ms_client_secret,omitempty"`
	AutoThreshold   *int     `json:"auto_threshold,omitempty"`
	ReviewThreshold *int     `json:"review_threshold,omitempty"`
	JobSources      []string `json:"job_sources,omitempty"`
	CustomJobSites  []string `json:"custom_job_sites,omitempty"`
}

// OnboardingStepRequest is the payload for POST /auth/setup/onboarding-step.
type OnboardingStepRequest struct {
	Step string `json:"step" binding:"required,oneof=llm voice preferences complete"`
}

// OnboardingConfigResponse is returned on successful config save.
type OnboardingConfigResponse struct {
	Message string `json:"message"`
}
