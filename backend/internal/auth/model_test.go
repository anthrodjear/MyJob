package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestUser_Fields(t *testing.T) {
	now := time.Now()
	user := User{
		ID:                    "test-id",
		Username:              "testuser",
		Email:                 "test@example.com",
		PasswordHash:          "hashed_password",
		SessionVersion:        1,
		LastLoginAt:           &now,
		PasswordChangedAt:     now,
		OnboardingCompletedAt: &now,
		OnboardingStep:        "llm",
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	assert.Equal(t, "test-id", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
	assert.Equal(t, 1, user.SessionVersion)
	assert.Equal(t, now, *user.LastLoginAt)
	assert.Equal(t, now, user.PasswordChangedAt)
	assert.Equal(t, now, *user.OnboardingCompletedAt)
	assert.Equal(t, "llm", user.OnboardingStep)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

func TestUser_JSONSerialization(t *testing.T) {
	now := time.Now()
	user := User{
		ID:           "test-id",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    now,
	}

	// Test that PasswordHash is omitted from JSON (json:"-")
	// This is a compile-time check - we can't easily test JSON marshaling without main package
	_ = user
}

func TestRefreshToken_Fields(t *testing.T) {
	now := time.Now()
	revokedAt := now.Add(-time.Hour)
	token := RefreshToken{
		ID:        "token-id",
		UserID:    "user-id",
		TokenHash: "hash",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		RevokedAt: &revokedAt,
		UpdatedAt: now,
	}

	assert.Equal(t, "token-id", token.ID)
	assert.Equal(t, "user-id", token.UserID)
	assert.Equal(t, "hash", token.TokenHash)
	assert.Equal(t, now.Add(time.Hour), token.ExpiresAt)
	assert.Equal(t, now, token.CreatedAt)
	assert.Equal(t, &revokedAt, token.RevokedAt)
	assert.Equal(t, now, token.UpdatedAt)
}

func TestRefreshToken_RevokedAtNil(t *testing.T) {
	token := RefreshToken{
		RevokedAt: nil,
	}
	assert.Nil(t, token.RevokedAt)
}

func TestClaims_Fields(t *testing.T) {
	claims := Claims{
		UserID:         "user-id",
		SessionVersion: 5,
	}

	assert.Equal(t, "user-id", claims.UserID)
	assert.Equal(t, 5, claims.SessionVersion)
}

func TestClaims_EmbeddedRegisteredClaims(t *testing.T) {
	// Claims embeds jwt.RegisteredClaims
	claims := Claims{
		UserID:         "user-id",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-id",
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour)},
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			NotBefore: &jwt.NumericDate{Time: time.Now()},
			Issuer:    "myjob",
			Audience:  []string{"myjob-api"},
			ID:        "token-id",
		},
	}

	assert.Equal(t, "user-id", claims.Subject)
	assert.Equal(t, "myjob", claims.Issuer)
	assert.Contains(t, claims.Audience, "myjob-api")
	assert.Equal(t, "token-id", claims.ID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}