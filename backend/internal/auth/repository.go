package auth

import (
	"context"
	"time"

	"backend/internal/config"
)

// Repository provides access to the single user's credentials.
// For a local-first app, credentials are loaded from config at startup.
// Password changes update the in-memory hash (not persisted to disk).
type Repository struct {
	passwordHash string
}

// NewRepository creates a repository from the auth config.
func NewRepository(cfg config.AuthConfig) *Repository {
	return &Repository{
		passwordHash: cfg.PasswordHash,
	}
}

// GetUser returns the single user.
func (r *Repository) GetUser(_ context.Context) (*User, error) {
	return &User{
		ID:           "local-user",
		PasswordHash: r.passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// GetPasswordHash returns the bcrypt hash for login verification.
func (r *Repository) GetPasswordHash(_ context.Context) (string, error) {
	return r.passwordHash, nil
}

// UpdatePasswordHash updates the in-memory password hash.
func (r *Repository) UpdatePasswordHash(_ context.Context, newHash string) error {
	r.passwordHash = newHash
	return nil
}
