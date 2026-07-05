package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/systemconfig"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials   = errors.New("auth: invalid credentials")
	ErrTokenInvalid         = errors.New("auth: token invalid")
	ErrTokenExpired         = errors.New("auth: token expired")
	ErrUserNotFound         = errors.New("auth: user not found")
	ErrSessionInvalidated   = errors.New("auth: session invalidated")
	ErrPasswordSame         = errors.New("auth: new password must differ from current password")
	ErrSetupAlreadyComplete = errors.New("auth: setup already complete — users exist")
)

// Service handles authentication business logic.
type Service struct {
	repo      *Repository
	cfg       config.AuthConfig
	configSvc *systemconfig.Service
}

// NewService creates a new auth service.
func NewService(repo *Repository, cfg config.AuthConfig, configSvc *systemconfig.Service) *Service {
	return &Service{
		repo:      repo,
		cfg:       cfg,
		configSvc: configSvc,
	}
}

// Login verifies the password and returns a JWT on success.
// Updates last login timestamp on success.
func (s *Service) Login(ctx context.Context, password string) (*LoginResponse, error) {
	hash, err := s.getPasswordHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: login: %w", err)
	}

	if hash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: login: generate token: %w", err)
	}

	return &LoginResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt,
	}, nil
}

// ValidateToken parses and validates a JWT, returning the claims.
// Uses strict validation: issuer must match "myjob".
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithIssuer("myjob"))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// ValidateTokenWithSession validates JWT and checks session version matches.
// Returns ErrSessionInvalidated if session version has changed (password changed, logout everywhere).
func (s *Service) ValidateTokenWithSession(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Check session version
	currentVersion, err := s.repo.GetSessionVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: get session version: %w", err)
	}

	if claims.SessionVersion != currentVersion {
		return nil, ErrSessionInvalidated
	}

	return claims, nil
}

// ChangePassword updates the user's password hash.
// Validates that new password differs from current.
func (s *Service) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	if currentPassword == newPassword {
		return ErrPasswordSame
	}

	hash, err := s.getPasswordHash(ctx)
	if err != nil {
		return fmt.Errorf("auth: change password: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: change password: hash: %w", err)
	}

	if err := s.repo.UpdatePasswordHash(ctx, string(newHash)); err != nil {
		return fmt.Errorf("auth: change password: update: %w", err)
	}

	return nil
}

// Logout increments session version to invalidate all existing tokens (logout everywhere).
func (s *Service) Logout(ctx context.Context) error {
	return s.repo.IncrementSessionVersion(ctx)
}

// Me returns the current user.
func (s *Service) Me(ctx context.Context) (*User, error) {
	return s.getUser(ctx)
}

// getUser fetches the single user.
func (s *Service) getUser(ctx context.Context) (*User, error) {
	user, err := s.repo.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// getPasswordHash fetches the stored password hash.
func (s *Service) getPasswordHash(ctx context.Context) (string, error) {
	hash, err := s.repo.GetPasswordHash(ctx)
	if err != nil {
		return "", fmt.Errorf("auth: get password hash: %w", err)
	}
	return hash, nil
}

// generateToken creates a new JWT with standard claims including session version.
// Fails closed if session version cannot be retrieved.
func (s *Service) generateToken(ctx context.Context) (string, int64, error) {
	expiresAt := time.Now().Add(s.cfg.JWTExpiry)

	// Get current session version (fail closed)
	sessionVersion, err := s.repo.GetSessionVersion(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("auth: get session version: %w", err)
	}

	claims := &Claims{
		UserID:         "local-user",
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "local-user",
			Issuer:    "myjob",
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt.Unix(), nil
}

// UpdateLastLogin updates the user's last login timestamp implemented in handler.
func (s *Service) UpdateLastLogin(ctx context.Context) error {
	return s.repo.UpdateLastLogin(ctx)
}

// IncrementSessionVersion manually increments the session version (for logout everywhere).
func (s *Service) IncrementSessionVersion(ctx context.Context) error {
	return s.repo.IncrementSessionVersion(ctx)
}

// GetSetupStatus returns whether setup is required (no users exist).
func (s *Service) GetSetupStatus(ctx context.Context) (*SetupStatusResponse, error) {
	required, err := s.repo.IsSetupRequired(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: get setup status: %w", err)
	}
	return &SetupStatusResponse{SetupRequired: required}, nil
}

// CompleteSetup creates the first admin user.
// Validates input, hashes password, and inserts the user.
// Returns ErrSetupAlreadyComplete if users already exist.
func (s *Service) CompleteSetup(ctx context.Context, username, email, password string) error {
	// Check if setup is still needed
	required, err := s.repo.IsSetupRequired(ctx)
	if err != nil {
		return fmt.Errorf("auth: complete setup: %w", err)
	}
	if !required {
		return ErrSetupAlreadyComplete
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: complete setup: hash password: %w", err)
	}

	// Create the user
	if err := s.repo.CreateAdminUser(ctx, username, email, string(hash)); err != nil {
		return fmt.Errorf("auth: complete setup: %w", err)
	}

	return nil
}

// CompleteOnboarding marks onboarding as finished.
func (s *Service) CompleteOnboarding(ctx context.Context) error {
	return s.repo.SetOnboardingCompleted(ctx, time.Now())
}

// UpdateOnboardingStep tracks progress for resume capability.
func (s *Service) UpdateOnboardingStep(ctx context.Context, step string) error {
	return s.repo.UpdateOnboardingStep(ctx, step)
}

// SaveOnboardingConfig persists all onboarding config overrides.
func (s *Service) SaveOnboardingConfig(ctx context.Context, req *OnboardingConfigRequest) error {
	configs := map[string]string{}
	if req.OpenAIKey != "" {
		configs["llm.primary.api_key"] = req.OpenAIKey
	}
	if req.AnthropicKey != "" {
		configs["llm.fallback.api_key"] = req.AnthropicKey
	}
	if req.LivekitURL != "" {
		configs["voice.livekit.url"] = req.LivekitURL
	}
	if req.LivekitKey != "" {
		configs["voice.livekit.api_key"] = req.LivekitKey
	}
	if req.LivekitSecret != "" {
		configs["voice.livekit.api_secret"] = req.LivekitSecret
	}
	if req.MSTenantID != "" {
		configs["email.ms_365.tenant_id"] = req.MSTenantID
	}
	if req.MSClientID != "" {
		configs["email.ms_365.client_id"] = req.MSClientID
	}
	if req.MSClientSecret != "" {
		configs["email.ms_365.client_secret"] = req.MSClientSecret
	}

	for key, value := range configs {
		if err := s.configSvc.SetOverride(ctx, key, []byte(value)); err != nil {
			return fmt.Errorf("auth: save config %s: %w", key, err)
		}
	}
	return nil
}

// TestLLMKey validates an LLM API key by calling the provider's API.
func (s *Service) TestLLMKey(ctx context.Context, provider, apiKey string) (bool, error) {
	switch provider {
	case "openai":
		return s.testOpenAIKey(ctx, apiKey)
	case "anthropic":
		return s.testAnthropicKey(ctx, apiKey)
	default:
		return false, fmt.Errorf("auth: unsupported provider: %s", provider)
	}
}

// testHTTPClient is a shared HTTP client with timeout for provider validation.
var testHTTPClient = &http.Client{Timeout: 10 * time.Second}

func (s *Service) testOpenAIKey(ctx context.Context, apiKey string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false, fmt.Errorf("auth: create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		return false, nil // network error, not invalid key
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

func (s *Service) testAnthropicKey(ctx context.Context, apiKey string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return false, fmt.Errorf("auth: create anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// TestVoiceConfig validates LiveKit credentials by listing rooms.
func (s *Service) TestVoiceConfig(ctx context.Context, livekitURL, apiKey, apiSecret string) (bool, error) {
	listURL := strings.TrimSuffix(livekitURL, "/") + "/rooms"
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return false, fmt.Errorf("auth: create livekit request: %w", err)
	}
	req.SetBasicAuth(apiKey, apiSecret)
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		return false, nil // connection failed
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// TestEmailConfig validates Microsoft Graph credentials via client_credentials flow.
func (s *Service) TestEmailConfig(ctx context.Context, tenantID, clientID, clientSecret string) (bool, error) {
	// Validate tenantID is a valid GUID to prevent URL injection
	if _, err := uuid.Parse(tenantID); err != nil {
		return false, fmt.Errorf("auth: invalid tenant ID format: %w", err)
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")

	body := strings.NewReader(data.Encode())
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, body)
	if err != nil {
		return false, fmt.Errorf("auth: create email test request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := testHTTPClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}
