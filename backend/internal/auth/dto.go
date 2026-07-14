package auth

import "github.com/golang-jwt/jwt/v5"

// --- Request DTOs ---

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Password string `json:"password" binding:"required,max=72" example:"securepassword123"`
}

// ChangePasswordRequest is the payload for POST /auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=72" example:"oldpassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72" example:"newpassword456"`
}

// --- Response DTOs ---

// LoginResponse is returned on successful login.
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt    int64  `json:"expires_at" example:"1705276800"`
}

// --- JWT Claims ---

// Claims holds the JWT claims for the single local user.
type Claims struct {
	UserID         string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SessionVersion int    `json:"session_version" example:"1"`
	jwt.RegisteredClaims
}

// --- Setup DTOs ---

// SetupStatusResponse is returned by GET /auth/setup/status.
// Extended with onboarding step for resume capability.
type SetupStatusResponse struct {
	SetupRequired bool   `json:"setup_required" example:"true"`
	Step          string `json:"step,omitempty" example:"llm"`
}

// SetupRequest is the payload for POST /auth/setup.
type SetupRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100" example:"john_doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"securepassword123"`
}

// SetupResponse is returned on successful setup.
type SetupResponse struct {
	Message string `json:"message" example:"setup completed successfully"`
}

// --- Onboarding DTOs ---

// TestLLMRequest is the payload for POST /auth/setup/test-llm.
type TestLLMRequest struct {
	Provider string `json:"provider" binding:"required,oneof=openai anthropic" example:"openai"`
	APIKey   string `json:"api_key" binding:"required" example:"sk-..."`
}

// TestLLMResponse is returned by POST /auth/setup/test-llm.
type TestLLMResponse struct {
	Valid bool `json:"valid" example:"true"`
}

// TestVoiceRequest is the payload for POST /auth/setup/test-voice.
type TestVoiceRequest struct {
	URL       string `json:"url" binding:"required" example:"wss://livekit.example.com"`
	APIKey    string `json:"api_key" binding:"required" example:"APIxxxxxxxxxxxxxxx"`
	APISecret string `json:"api_secret" binding:"required" example:"secretxxxxxxxxxxxxxxxxxxxxxxxxxx"`
}

// TestEmailRequest is the payload for POST /auth/setup/test-email.
type TestEmailRequest struct {
	TenantID     string `json:"tenant_id" binding:"required" example:"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"`
	ClientID     string `json:"client_id" binding:"required" example:"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"`
	ClientSecret string `json:"client_secret" binding:"required" example:"secretxxxxxxxxxxxxxxxxxxxxxxxxxx"`
}

// OnboardingConfigRequest is the payload for POST /auth/setup/config.
type OnboardingConfigRequest struct {
	OpenAIKey       string   `json:"openai_key,omitempty" example:"sk-..."`
	AnthropicKey    string   `json:"anthropic_key,omitempty" example:"sk-ant-..."`
	LivekitURL      string   `json:"livekit_url,omitempty" example:"wss://livekit.example.com"`
	LivekitKey      string   `json:"livekit_key,omitempty" example:"APIxxxxxxxxxxxxxxx"`
	LivekitSecret   string   `json:"livekit_secret,omitempty" example:"secretxxxxxxxxxxxxxxxxxxxxxxxxxx"`
	MSTenantID      string   `json:"ms_tenant_id,omitempty" example:"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"`
	MSClientID      string   `json:"ms_client_id,omitempty" example:"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"`
	MSClientSecret  string   `json:"ms_client_secret,omitempty" example:"secretxxxxxxxxxxxxxxxxxxxxxxxxxx"`
	AutoThreshold   *int     `json:"auto_threshold,omitempty" example:"95"`
	ReviewThreshold *int     `json:"review_threshold,omitempty" example:"80"`
	JobSources      []string `json:"job_sources,omitempty" example:"[\"greenhouse:openai\",\"lever:figma\"]"`
	CustomJobSites  []string `json:"custom_job_sites,omitempty" example:"[\"https://careers.example.com\"]"`
}

// OnboardingStepRequest is the payload for POST /auth/setup/onboarding-step.
type OnboardingStepRequest struct {
	Step string `json:"step" binding:"required,oneof=llm voice preferences complete" example:"llm"`
}

// OnboardingConfigResponse is returned on successful config save.
type OnboardingConfigResponse struct {
	Message string `json:"message" example:"configuration saved"`
}

// LogoutResponse is returned on successful logout.
type LogoutResponse struct {
	Message string `json:"message" example:"logged out"`
}

// --- Refresh Token DTOs ---

// RefreshRequest is the payload for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required,min=64" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// RefreshResponse is returned on successful token refresh.
type RefreshResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt    int64  `json:"expires_at" example:"1705276800"`
}
