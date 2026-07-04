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

	// Load initial user — tolerate missing user for first-time setup.
	// The setup flow will create the user via CompleteSetup.
	user, err := repo.loadUser(context.Background())
	if err != nil {
		// User not found is expected on first run — not a fatal error
		repo.user = nil
	} else {
		repo.user = user
	}

	return repo, nil
}

// seedIfNeeded inserts the local-user with initial password hash if table is empty.
// If initialHash is empty, it skips seeding — the user will complete setup via the web UI.
func (r *Repository) seedIfNeeded(ctx context.Context, initialHash string) error {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE id = 'local-user'`)
	if err != nil {
		return fmt.Errorf("auth: count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	// Skip seeding if no password hash configured — setup flow will create the user
	if initialHash == "" {
		return nil
	}

	// Insert initial user with hash from config (backward compatibility)
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
		SELECT id, username, email, password_hash, session_version, last_login_at,
		       password_changed_at, onboarding_completed_at, onboarding_step,
		       created_at, updated_at
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

// IsSetupRequired returns true if no users exist in the database.
// Used by the setup middleware to determine if setup mode should be active.
func (r *Repository) IsSetupRequired(ctx context.Context) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM users`)
	if err != nil {
		return false, fmt.Errorf("auth: count users: %w", err)
	}
	return count == 0, nil
}

// CreateAdminUser inserts the first user with username, email, and password hash.
// Only succeeds if no users exist (enforced by setup middleware + endpoint guard).
func (r *Repository) CreateAdminUser(ctx context.Context, username, email, passwordHash string) error {
	// Double-check: only allow if table is empty (defense in depth)
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM users`)
	if err != nil {
		return fmt.Errorf("auth: count users for setup: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("auth: setup blocked — users already exist")
	}

	now := time.Now()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, password_hash, session_version, password_changed_at, created_at, updated_at)
		VALUES ('local-user', $1, $2, $3, 1, $4, $4, $4)
	`, username, email, passwordHash, now)
	if err != nil {
		return fmt.Errorf("auth: create admin user: %w", err)
	}

	// Update cached user
	r.mu.Lock()
	defer r.mu.Unlock()
	r.user = &User{
		ID:                "local-user",
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		SessionVersion:    1,
		PasswordChangedAt: now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	return nil
}

// IsOnboardingCompleted returns true if onboarding_completed_at is set.
func (r *Repository) IsOnboardingCompleted(ctx context.Context) (bool, error) {
	var completed bool
	err := r.db.GetContext(ctx, &completed,
		`SELECT onboarding_completed_at IS NOT NULL FROM users WHERE id = 'local-user'`)
	if err != nil {
		return false, fmt.Errorf("auth: check onboarding completed: %w", err)
	}
	return completed, nil
}

// GetOnboardingStep returns the current onboarding step for resume capability.
func (r *Repository) GetOnboardingStep(ctx context.Context) (string, error) {
	var step sql.NullString
	err := r.db.GetContext(ctx, &step,
		`SELECT onboarding_step FROM users WHERE id = 'local-user'`)
	if err != nil {
		return "", fmt.Errorf("auth: get onboarding step: %w", err)
	}
	if !step.Valid {
		return "account", nil
	}
	return step.String, nil
}

// SetOnboardingCompleted marks onboarding as finished with timestamp.
func (r *Repository) SetOnboardingCompleted(ctx context.Context, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET onboarding_completed_at = $1, updated_at = $2 WHERE id = 'local-user'`, t, t)
	if err != nil {
		return fmt.Errorf("auth: set onboarding completed: %w", err)
	}
	return nil
}

// UpdateOnboardingStep tracks progress for resume capability.
func (r *Repository) UpdateOnboardingStep(ctx context.Context, step string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET onboarding_step = $1, updated_at = $2 WHERE id = 'local-user'`, step, time.Now())
	if err != nil {
		return fmt.Errorf("auth: update onboarding step: %w", err)
	}
	return nil
}
