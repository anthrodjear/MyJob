// Repository handles database operations for the singleton profile.
//
// The profiles table holds one row — there is no user_id because this
// is a local-first, single-user system.
//
// Responsibilities:
//   - Read the single profile row
//   - Create the first profile on first-run
//   - Replace (Update) or merge (UpdatePartial) profile data
//   - Optimistic concurrency via version column
//
// This file contains NO business logic. It translates SQL errors to
// domain errors for the service layer. The domain owns mutation rules
// (e.g., PATCH merge logic lives in ProfileData.ApplyPatch, not here).
//
// Rules followed:
//   - No SELECT * — columns listed explicitly
//   - Parameterized queries only — no string interpolation
//   - Errors wrapped with context for debugging
package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Domain errors
// ---------------------------------------------------------------------------

var (
	// ErrNotFound indicates no profile row exists yet (first-run scenario).
	ErrNotFound = errors.New("profile not found")

	// ErrVersionConflict indicates the profile was modified since the client
	// last read it. The client should re-fetch and retry.
	ErrVersionConflict = errors.New("profile version conflict — re-fetch and retry")
)

// ---------------------------------------------------------------------------
// Column list — single source of truth for SELECT queries
// ---------------------------------------------------------------------------

const profileColumns = `id, data, version, created_at, updated_at`

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository handles database operations for the singleton profile.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new profile repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Queries: Read
// ---------------------------------------------------------------------------

// Get fetches the single profile row.
// Returns ErrNotFound if no profile exists yet (first-run scenario).
func (r *Repository) Get(ctx context.Context) (*Profile, error) {
	var p Profile
	err := r.db.GetContext(ctx, &p,
		`SELECT `+profileColumns+`
		 FROM profiles
		 LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return &p, nil
}

// ---------------------------------------------------------------------------
// Queries: Write
// ---------------------------------------------------------------------------

// Create inserts the first profile row.
// Called once on first-run when no profile exists.
// Guarantees a non-nil UUID — caller does not need to set p.ID.
func (r *Repository) Create(ctx context.Context, p *Profile) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO profiles (id, data, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.ID, p.Data, p.Version, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

// Update replaces the entire profile data with optimistic concurrency.
// Returns the updated profile on success.
//
// The caller must provide the expectedVersion — typically the version
// read from Get(). If the row was modified since then, Update returns
// ErrVersionConflict and the caller should re-fetch.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, data ProfileData, expectedVersion int) (*Profile, error) {
	now := time.Now()
	newVersion := expectedVersion + 1

	result, err := r.db.ExecContext(ctx,
		`UPDATE profiles
		 SET data = $1, version = $2, updated_at = $3
		 WHERE id = $4 AND version = $5`,
		data, newVersion, now, id, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ErrVersionConflict
	}

	// Re-read to return updated version/timestamps
	var p Profile
	err = r.db.GetContext(ctx, &p,
		`SELECT `+profileColumns+`
		 FROM profiles WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("re-read profile after update: %w", err)
	}
	return &p, nil
}

// UpdatePartial applies a merge via ProfileData.ApplyPatch and writes back.
//
// The domain method ApplyPatch owns the merge rules. This method only
// reads, delegates to the domain, and writes — no business logic here.
//
// Uses SELECT FOR UPDATE + RETURNING to avoid an extra round trip.
func (r *Repository) UpdatePartial(ctx context.Context, id uuid.UUID, patch PatchProfileRequest, expectedVersion int) (*Profile, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the row to prevent concurrent patches from stomping each other
	var current Profile
	err = tx.GetContext(ctx, &current,
		`SELECT `+profileColumns+`
		 FROM profiles WHERE id = $1
		 FOR UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock profile for update: %w", err)
	}

	// Domain owns the merge logic
	current.Data.ApplyPatch(patch)

	// Validate merged result inside the transaction — invalid data never persists
	if err := current.Data.Validate(); err != nil {
		return nil, fmt.Errorf("validate merged profile: %w", err)
	}

	// Write back with concurrency check
	newVersion := expectedVersion + 1
	now := time.Now()
	result, err := tx.ExecContext(ctx,
		`UPDATE profiles
		 SET data = $1, version = $2, updated_at = $3
		 WHERE id = $4 AND version = $5`,
		current.Data, newVersion, now, id, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ErrVersionConflict
	}

	// Re-read to return updated version/timestamps
	err = tx.GetContext(ctx, &current,
		`SELECT `+profileColumns+`
		 FROM profiles WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("re-read profile after update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &current, nil
}
