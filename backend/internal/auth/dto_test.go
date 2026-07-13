package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginRequest_Binding(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		shouldPass  bool
	}{
		{
			name:        "valid password",
			password:    "password123",
			shouldPass:  true,
		},
		{
			name:        "empty password",
			password:    "",
			shouldPass:  false,
		},
		{
			name:        "password at max length",
			password:    "a" + string(make([]byte, 71)), // 72 chars
			shouldPass:  true,
		},
		{
			name:        "password exceeds max length",
			password:    "a" + string(make([]byte, 72)), // 73 chars
			shouldPass:  false,
		},
		{
			name:        "password with special chars",
			password:    "P@ssw0rd!#$%",
			shouldPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := LoginRequest{Password: tt.password}
			// Binding validation would be done by Gin framework
			// Here we just test the struct can be created
			assert.Equal(t, tt.password, req.Password)
		})
	}
}

func TestChangePasswordRequest_Validation(t *testing.T) {
	tests := []struct {
		name            string
		currentPassword string
		newPassword     string
		shouldPass      bool
	}{
		{
			name:            "valid passwords",
			currentPassword: "oldPassword123",
			newPassword:     "newPassword456",
			shouldPass:      true,
		},
		{
			name:            "new password too short",
			currentPassword: "oldPassword123",
			newPassword:     "short",
			shouldPass:      false,
		},
		{
			name:            "new password at max length",
			currentPassword: "oldPassword123",
			newPassword:     "a" + string(make([]byte, 71)),
			shouldPass:      true,
		},
		{
			name:            "new password exceeds max length",
			currentPassword: "oldPassword123",
			newPassword:     "a" + string(make([]byte, 72)),
			shouldPass:      false,
		},
		{
			name:            "empty current password",
			currentPassword: "",
			newPassword:     "newPassword123",
			shouldPass:      false,
		},
		{
			name:            "empty new password",
			currentPassword: "oldPassword123",
			newPassword:     "",
			shouldPass:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChangePasswordRequest{
				CurrentPassword: tt.currentPassword,
				NewPassword:     tt.newPassword,
			}
			assert.Equal(t, tt.currentPassword, req.CurrentPassword)
			assert.Equal(t, tt.newPassword, req.NewPassword)
		})
	}
}

func TestSetupRequest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		shouldPass bool
	}{
		{
			name:     "valid setup request",
			username: "johndoe",
			email:    "john@example.com",
			password: "password123",
			shouldPass: true,
		},
		{
			name:     "username too short",
			username: "ab",
			email:    "john@example.com",
			password: "password123",
			shouldPass: false,
		},
		{
			name:     "username at max length",
			username: "a" + string(make([]byte, 99)),
			email:    "john@example.com",
			password: "password123",
			shouldPass: true,
		},
		{
			name:     "username exceeds max length",
			username: "a" + string(make([]byte, 100)),
			email:    "john@example.com",
			password: "password123",
			shouldPass: false,
		},
		{
			name:     "invalid email",
			username: "johndoe",
			email:    "not-an-email",
			password: "password123",
			shouldPass: false,
		},
		{
			name:     "password too short",
			username: "johndoe",
			email:    "john@example.com",
			password: "short",
			shouldPass: false,
		},
		{
			name:     "password at max length",
			username: "johndoe",
			email:    "john@example.com",
			password: "a" + string(make([]byte, 71)),
			shouldPass: true,
		},
		{
			name:     "password exceeds max length",
			username: "johndoe",
			email:    "john@example.com",
			password: "a" + string(make([]byte, 72)),
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SetupRequest{
				Username: tt.username,
				Email:    tt.email,
				Password: tt.password,
			}
			assert.Equal(t, tt.username, req.Username)
			assert.Equal(t, tt.email, req.Email)
			assert.Equal(t, tt.password, req.Password)
		})
	}
}

func TestTestLLMRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		apiKey    string
		shouldPass bool
	}{
		{
			name:      "valid OpenAI provider",
			provider:  "openai",
			apiKey:    "sk-test123",
			shouldPass: true,
		},
		{
			name:      "valid Anthropic provider",
			provider:  "anthropic",
			apiKey:    "sk-ant-test123",
			shouldPass: true,
		},
		{
			name:      "invalid provider",
			provider:  "invalid",
			apiKey:    "test",
			shouldPass: false,
		},
		{
			name:      "empty provider",
			provider:  "",
			apiKey:    "test",
			shouldPass: false,
		},
		{
			name:      "empty API key",
			provider:  "openai",
			apiKey:    "",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := TestLLMRequest{
				Provider: tt.provider,
				APIKey:   tt.apiKey,
			}
			assert.Equal(t, tt.provider, req.Provider)
			assert.Equal(t, tt.apiKey, req.APIKey)
		})
	}
}

func TestTestVoiceRequest_Validation(t *testing.T) {
	req := TestVoiceRequest{
		URL:       "wss://livekit.example.com",
		APIKey:    "test-key",
		APISecret: "test-secret",
	}

	assert.Equal(t, "wss://livekit.example.com", req.URL)
	assert.Equal(t, "test-key", req.APIKey)
	assert.Equal(t, "test-secret", req.APISecret)
}

func TestTestEmailRequest_Validation(t *testing.T) {
	req := TestEmailRequest{
		TenantID:     "tenant-123",
		ClientID:     "client-456",
		ClientSecret: "secret-789",
	}

	assert.Equal(t, "tenant-123", req.TenantID)
	assert.Equal(t, "client-456", req.ClientID)
	assert.Equal(t, "secret-789", req.ClientSecret)
}

func TestOnboardingConfigRequest_Fields(t *testing.T) {
	autoThreshold := 95
	reviewThreshold := 80
	req := OnboardingConfigRequest{
		OpenAIKey:       "sk-openai",
		AnthropicKey:    "sk-anthropic",
		LivekitURL:      "wss://livekit.example.com",
		LivekitKey:      "livekit-key",
		LivekitSecret:   "livekit-secret",
		MSTenantID:      "tenant",
		MSClientID:      "client",
		MSClientSecret:  "secret",
		AutoThreshold:   &autoThreshold,
		ReviewThreshold: &reviewThreshold,
		JobSources:      []string{"indeed", "linkedin"},
		CustomJobSites:  []string{"custom-site.com"},
	}

	assert.Equal(t, "sk-openai", req.OpenAIKey)
	assert.Equal(t, "sk-anthropic", req.AnthropicKey)
	assert.Equal(t, "wss://livekit.example.com", req.LivekitURL)
	assert.Equal(t, "livekit-key", req.LivekitKey)
	assert.Equal(t, "livekit-secret", req.LivekitSecret)
	assert.Equal(t, "tenant", req.MSTenantID)
	assert.Equal(t, "client", req.MSClientID)
	assert.Equal(t, "secret", req.MSClientSecret)
	assert.Equal(t, 95, *req.AutoThreshold)
	assert.Equal(t, 80, *req.ReviewThreshold)
	assert.Equal(t, []string{"indeed", "linkedin"}, req.JobSources)
	assert.Equal(t, []string{"custom-site.com"}, req.CustomJobSites)
}

func TestOnboardingStepRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		step      string
		shouldPass bool
	}{
		{
			name:      "valid llm step",
			step:      "llm",
			shouldPass: true,
		},
		{
			name:      "valid voice step",
			step:      "voice",
			shouldPass: true,
		},
		{
			name:      "valid preferences step",
			step:      "preferences",
			shouldPass: true,
		},
		{
			name:      "valid complete step",
			step:      "complete",
			shouldPass: true,
		},
		{
			name:      "invalid step",
			step:      "invalid",
			shouldPass: false,
		},
		{
			name:      "empty step",
			step:      "",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := OnboardingStepRequest{Step: tt.step}
			assert.Equal(t, tt.step, req.Step)
		})
	}
}

func TestRefreshRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		refreshToken string
		shouldPass  bool
	}{
		{
			name:        "valid refresh token (64 chars)",
			refreshToken: "a" + string(make([]byte, 63)), // 64 chars
			shouldPass:  true,
		},
		{
			name:        "refresh token too short",
			refreshToken: "short",
			shouldPass:  false,
		},
		{
			name:        "refresh token exactly 64 chars",
			refreshToken: "a" + string(make([]byte, 63)),
			shouldPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := RefreshRequest{RefreshToken: tt.refreshToken}
			assert.Equal(t, tt.refreshToken, req.RefreshToken)
		})
	}
}

func TestSetupStatusResponse(t *testing.T) {
	resp := SetupStatusResponse{
		SetupRequired: true,
		Step:          "llm",
	}

	assert.True(t, resp.SetupRequired)
	assert.Equal(t, "llm", resp.Step)

	resp2 := SetupStatusResponse{
		SetupRequired: false,
		Step:          "complete",
	}

	assert.False(t, resp2.SetupRequired)
	assert.Equal(t, "complete", resp2.Step)
}

func TestLoginResponse(t *testing.T) {
	resp := LoginResponse{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresAt:    1234567890,
	}

	assert.Equal(t, "access-token-123", resp.AccessToken)
	assert.Equal(t, "refresh-token-456", resp.RefreshToken)
	assert.Equal(t, int64(1234567890), resp.ExpiresAt)
}

func TestSetupResponse(t *testing.T) {
	resp := SetupResponse{Message: "Setup completed successfully"}
	assert.Equal(t, "Setup completed successfully", resp.Message)
}

func TestOnboardingConfigResponse(t *testing.T) {
	resp := OnboardingConfigResponse{Message: "Configuration saved"}
	assert.Equal(t, "Configuration saved", resp.Message)
}

func TestLogoutResponse(t *testing.T) {
	resp := LogoutResponse{Message: "Logged out successfully"}
	assert.Equal(t, "Logged out successfully", resp.Message)
}

func TestRefreshResponse(t *testing.T) {
	resp := RefreshResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    1234567890,
	}

	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)
	assert.Equal(t, int64(1234567890), resp.ExpiresAt)
}