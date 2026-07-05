// Package systemconfig provides database access for runtime configuration overrides.
// It implements CRUD operations on the system_config_overrides table, which stores
// user-defined configuration values that take precedence over YAML and environment
// variable settings. These overrides enable dynamic configuration changes without
// application restart.
//
// # Design Principles
//
//   - All methods accept context.Context for cancellation and timeout propagation.
//   - Errors are wrapped with descriptive context using fmt.Errorf("systemconfig: ...: %w", err).
//   - No logging occurs in the repository layer — errors are returned to the caller.
//   - JSON values are stored and retrieved as json.RawMessage to preserve exact
//     input formatting and support all JSON types (int, float, bool, string, array, object).
//   - Upsert uses PostgreSQL ON CONFLICT for atomic insert-or-update.
//
// # Usage
//
//	repo := systemconfig.NewRepository(db)
//	overrides, err := repo.GetAllOverrides(ctx)
//	if err != nil {
//	    return fmt.Errorf("load overrides: %w", err)
//	}
//
//	err = repo.UpsertOverride(ctx, "scoring.auto_threshold", json.RawMessage(`90`),
//	    systemconfig.CategoryRuntime, "Auto-apply threshold", &userID)
//
// # What This Package Does NOT Do
//
//   - Does not validate override keys or values — validation happens in the service layer.
//   - Does not merge configuration layers — that is the resolver's responsibility.
//   - Does not handle HTTP requests or API serialization — that lives in the handler layer.
package systemconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides database access for system configuration overrides.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new systemconfig repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetAllOverrides retrieves all configuration overrides from the database.
// It returns a map of key -> raw JSON value for efficient merging in the resolver.
// Returns an empty map (not nil) if no overrides exist.
//
// Example:
//
//	overrides, err := repo.GetAllOverrides(ctx)
//	if err != nil {
//	    return fmt.Errorf("systemconfig: get all overrides: %w", err)
//	}
//	// overrides["scoring.auto_threshold"] = json.RawMessage(`90`)
func (r *Repository) GetAllOverrides(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT key, value
		FROM system_config_overrides
	`)
	if err != nil {
		return nil, fmt.Errorf("systemconfig: get all overrides query: %w", err)
	}
	defer rows.Close()

	overrides := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value json.RawMessage
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("systemconfig: get all overrides scan: %w", err)
		}
		overrides[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("systemconfig: get all overrides rows: %w", err)
	}

	return overrides, nil
}

// UpsertOverride inserts a new configuration override or updates an existing one.
// Uses INSERT ... ON CONFLICT (key) DO UPDATE to atomically upsert.
// The value parameter is already-validated JSON (json.RawMessage).
// category classifies the override for the admin UI (runtime, operational, infrastructure).
// description is an optional human-readable explanation.
// updatedBy is the user ID making the change, or nil for system-initiated changes.
//
// Example:
//
//	err := repo.UpsertOverride(ctx,
//	    "scoring.auto_threshold",
//	    json.RawMessage(`90`),
//	    systemconfig.CategoryRuntime,
//	    "Auto-apply threshold for scoring",
//	    &userID,
//	)
func (r *Repository) UpsertOverride(
	ctx context.Context,
	key string,
	value json.RawMessage,
	category ConfigCategory,
	description string,
	updatedBy *uuid.UUID,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO system_config_overrides (key, value, category, description, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			category = EXCLUDED.category,
			description = EXCLUDED.description,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
	`, key, value, category, description, updatedBy)
	if err != nil {
		return fmt.Errorf("systemconfig: upsert override: %w", err)
	}
	return nil
}

// DeleteOverride removes a configuration override by key.
// Returns nil if the key didn't exist (idempotent delete).
// Returns an error only for database failures.
//
// Example:
//
//	if err := repo.DeleteOverride(ctx, "scoring.auto_threshold"); err != nil {
//	    return fmt.Errorf("systemconfig: delete override: %w", err)
//	}
func (r *Repository) DeleteOverride(ctx context.Context, key string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM system_config_overrides
		WHERE key = $1
	`, key)
	if err != nil {
		return fmt.Errorf("systemconfig: delete override: %w", err)
	}

	// Idempotent: no error if key didn't exist
	_ = result // rows affected intentionally ignored

	return nil
}

// GetOverride retrieves a single override by key.
// Returns sql.ErrNoRows if the key doesn't exist (caller should check with errors.Is).
//
// Example:
//
//	override, err := repo.GetOverride(ctx, "scoring.auto_threshold")
//	if err != nil {
//	    if errors.Is(err, sql.ErrNoRows) {
//	        return nil // not found
//	    }
//	    return fmt.Errorf("systemconfig: get override: %w", err)
//	}
func (r *Repository) GetOverride(ctx context.Context, key string) (*Override, error) {
	var override Override
	err := r.db.GetContext(ctx, &override, `
		SELECT id, key, value, category, description, updated_by, created_at, updated_at
		FROM system_config_overrides
		WHERE key = $1
	`, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("systemconfig: get override: %w", err)
	}
	return &override, nil
}

// ListOverridesByCategory retrieves all overrides for a given category.
// Useful for the admin UI to display overrides grouped by functional area.
//
// Example:
//
//	overrides, err := repo.ListOverridesByCategory(ctx, systemconfig.CategoryRuntime)
//	if err != nil {
//	    return fmt.Errorf("systemconfig: list by category: %w", err)
//	}
func (r *Repository) ListOverridesByCategory(ctx context.Context, category ConfigCategory) ([]*Override, error) {
	var overrides []*Override
	err := r.db.SelectContext(ctx, &overrides, `
		SELECT id, key, value, category, description, updated_by, created_at, updated_at
		FROM system_config_overrides
		WHERE category = $1
		ORDER BY key
	`, category)
	if err != nil {
		return nil, fmt.Errorf("systemconfig: list by category: %w", err)
	}
	return overrides, nil
}
