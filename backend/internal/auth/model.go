package auth

import "time"

// User represents the single local user.
// For a local-first app, this is minimal — just enough to support JWT auth.
type User struct {
	ID           string    `db:"id" json:"id"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
