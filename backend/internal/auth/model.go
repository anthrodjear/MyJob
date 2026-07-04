package auth

import (
	"time"
)

// User represents the single local user.
// For a local-first app, this is minimal — just enough to support JWT auth.
type User struct {
	ID                    string     `db:"id" json:"id"`
	Username              string     `db:"username" json:"username"`
	Email                 string     `db:"email" json:"email"`
	PasswordHash          string     `db:"password_hash" json:"-"`
	SessionVersion        int        `db:"session_version" json:"session_version"`
	LastLoginAt           *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	PasswordChangedAt     time.Time  `db:"password_changed_at" json:"-"`
	OnboardingCompletedAt *time.Time `db:"onboarding_completed_at" json:"onboarding_completed_at,omitempty"`
	OnboardingStep        string     `db:"onboarding_step" json:"onboarding_step"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updated_at"`
}
