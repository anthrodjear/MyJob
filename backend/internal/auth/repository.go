package auth

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"backend/internal/config"
	"github.com/jmoiron/sqlx"
)

// Repository provides access to the single user's credentials.
// Uses PostgreSQL for persistence with in-memory mutex for thread safety.
type Repository struct {
	mu   sync.RWMutex
	db   *sqlx.DB
	user *User // cached user for fast reads
}

// NewRepository creates a repository from the auth config and database.
// Seeds the users table with the initial password hash from config if empty.
func NewRepository(db *sqlx.DB, cfg config.AuthConfig) (*Repository, error) {
	repo := &Repository{
		db: db,
	}

	// Ensure users table has the local-user record
	if err := repo.seedIfNeeded(context.Background(), cfg.PasswordHash); err != nil {
		return nil, fmt.Errorf("auth: seed user: %w", err)
	}

	// Load initial user
	user, err := repo.loadUser(context.Background())
	if err != nil {
		return nil, fmt.Errorf("auth: load user: %w", err)
	}
	repo.user = user

	return repo, nil
}

// seedIfNeeded inserts the local-user with initial password hash if table is empty.
func (r *Repository) seedIfNeeded(ctx context.Context, initialHash string) error {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE id = 'local-user'`)
	if err != nil {
		return fmt.Errorf("auth: count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	// Insert initial user with hash from config
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO users (id, password_hash, session_version, password_changed_at)
		VALUES ('local-user', $1, 1, NOW())
	`, initialHash)
	if err != nil {
		return fmt.Errorf("auth: seed user: %w", err)
	}
	return nil
}

// loadUser fetches the user from database.
func (r *Repository) loadUser(ctx context.Context) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, password_hash, session_version, last_login_at, password_changed_at, created_at, updated_at
		FROM users
		WHERE id = 'local-user'
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("auth: user not found")
		}
		return nil, fmt.Errorf("auth: get user: %w", err)
	}
	return &user, nil
}

// GetUser returns the single user.
func (r *Repository) GetUser(_ context.Context) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.user == nil {
		return nil, fmt.Errorf("auth: user not loaded")
	}
	return r.user, nil
}

// GetPasswordHash returns the bcrypt hash for login verification.
func (r *Repository) GetPasswordHash(_ context.Context) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.user == nil {
		return "", fmt.Errorf("auth: user not loaded")
	}
	return r.user.PasswordHash, nil
}

// GetSessionVersion returns the current session version.
func (r *Repository) GetSessionVersion(_ context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.user == nil {
		return 0, fmt.Errorf("auth: user not loaded")
	}
	return r.user.SessionVersion, nil
}

// UpdatePasswordHash updates the password hash and increments session version.
// This invalidates all existing JWT tokens for the user.
func (r *Repository) UpdatePasswordHash(ctx context.Context, newHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $1,
		    session_version = session_version + 1,
		    password_changed_at = $2,
		    updated_at = $3
		WHERE id = 'local-user'
	`, newHash, now, now)
	if err != nil {
		return fmt.Errorf("auth: update password: %w", err)
	}

	// Reload user to update cache
	user, err := r.loadUser(ctx)
	if err != nil {
		return fmt.Errorf("auth: reload user: %w", err)
	}
	r.user = user

	return nil
}

// UpdateLastLogin updates the last login timestamp.
func (r *Repository) UpdateLastLogin(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET last_login_at = $1,
		    updated_at = $2
		WHERE id = 'local-user'
	`, now, now)
	if err != nil {
		return fmt.Errorf("auth: update last login: %w", err)
	}

	if r.user != nil {
		r.user.LastLoginAt = &now
		r.user.UpdatedAt = now
	}

	return nil
}

// IncrementSessionVersion manually increments the session version (for logout everywhere).
func (r *Repository) IncrementSessionVersion(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET session_version = session_version + 1,
		    updated_at = NOW()
		WHERE id = 'local-user'
	`)
	if err != nil {
		return fmt.Errorf("auth: increment session version: %w", err)
	}

	if r.user != nil {
		r.user.SessionVersion++
		r.user.UpdatedAt = time.Now()
	}

	return nil
}
