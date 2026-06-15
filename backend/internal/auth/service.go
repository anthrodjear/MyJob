package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrTokenInvalid       = errors.New("auth: token invalid")
	ErrTokenExpired       = errors.New("auth: token expired")
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrSessionInvalidated = errors.New("auth: session invalidated")
	ErrPasswordSame       = errors.New("auth: new password must differ from current password")
)

// Service handles authentication business logic.
type Service struct {
	repo *Repository
	cfg  config.AuthConfig
}

// NewService creates a new auth service.
func NewService(repo *Repository, cfg config.AuthConfig) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
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

// UpdateLastLogin updates the user's last login timestamp implimente in handler.
func (s *Service) UpdateLastLogin(ctx context.Context) error {
	return s.repo.UpdateLastLogin(ctx)
}

// IncrementSessionVersion manually increments the session version (for logout everywhere).
func (s *Service) IncrementSessionVersion(ctx context.Context) error {
	return s.repo.IncrementSessionVersion(ctx)
}
