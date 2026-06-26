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
	UserID        string `json:"user_id"`
	SessionVersion int   `json:"session_version"`
	jwt.RegisteredClaims
}

// --- Setup DTOs ---

// SetupStatusResponse is returned by GET /auth/setup/status.
type SetupStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
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
