// Package systemconfig provides runtime configuration override management for the
// job search agent. This file defines the API request/response DTOs for the
// System Config endpoints.
//
// DTOs handle all JSON serialization — domain models (Override, EffectiveConfig)
// do NOT carry JSON tags. This separation ensures the database layer and API layer
// can evolve independently.
package systemconfig

// ---------------------------------------------------------------------------
// SetOverrideRequest — PATCH /api/v1/system/config
// ---------------------------------------------------------------------------

// SetOverrideRequest is the request body for creating or updating a runtime
// configuration override via PATCH /api/v1/system/config.
//
// The Key uses dotted notation (e.g., "scoring.auto_threshold", "llm.primary.model").
// The Value accepts any valid JSON type (int, float, bool, string, array, object)
// and will be marshaled to JSONB for storage in the database.
//
// Example:
//
//	// Set auto-apply threshold to 95
//	SetOverrideRequest{
//	  Key:   "scoring.auto_threshold",
//	  Value: 95,
//	}
//
//	// Set LLM model to a specific version
//	SetOverrideRequest{
//	  Key:   "llm.primary.model",
//	  Value: "gpt-4o-mini",
//	}
//
//	// Set queue concurrency
//	SetOverrideRequest{
//	  Key:   "queue.concurrency",
//	  Value: 8,
//	}
type SetOverrideRequest struct {
	// Key is the dotted-notation configuration key.
	// Must match the pattern: lowercase letters, digits, dots, minimum 2 segments.
	// Example: "scoring.auto_threshold"
	Key string `json:"key" binding:"required" example:"scoring.auto_threshold"`

	// Value is the JSON value to store. Accepts int, float, bool, string, array, or object.
	// Will be marshaled to JSONB for database storage.
	// Example: 95, "hybrid", true, ["folder1", "folder2"]
	Value any `json:"value" binding:"required" example:"95"`
}

// ---------------------------------------------------------------------------
// SetOverrideResponse — 200 response for PATCH /api/v1/system/config
// ---------------------------------------------------------------------------

// SetOverrideResponse is the success response for creating or updating an override.
//
// Example:
//
//	SetOverrideResponse{
//	  Message: "override saved",
//	  Key:     "scoring.auto_threshold",
//	}
type SetOverrideResponse struct {
	// Message is a human-readable status message.
	Message string `json:"message" example:"override saved"`

	// Key is the dotted-notation key that was set.
	Key string `json:"key" example:"scoring.auto_threshold"`
}

// ---------------------------------------------------------------------------
// DeleteOverrideResponse — 200 response for DELETE /api/v1/system/config/:key
// ---------------------------------------------------------------------------

// DeleteOverrideResponse is the success response for deleting an override.
//
// Example:
//
//	DeleteOverrideResponse{
//	  Message: "override deleted",
//	  Key:     "scoring.auto_threshold",
//	}
type DeleteOverrideResponse struct {
	// Message is a human-readable status message.
	Message string `json:"message" example:"override deleted"`

	// Key is the dotted-notation key that was deleted.
	Key string `json:"key" example:"scoring.auto_threshold"`
}

// ---------------------------------------------------------------------------
// EffectiveConfigResponse — GET /api/v1/system/config
// ---------------------------------------------------------------------------

// EffectiveConfigResponse wraps the fully resolved EffectiveConfig for the
// GET /api/v1/system/config endpoint. The wrapper allows future extensibility
// such as adding a version/ETag for optimistic concurrency control (OCC),
// last-modified timestamps, or pagination for large configs.
//
// Currently, the EffectiveConfig is embedded directly. In the future, this
// response can include:
//   - Version string (e.g., hash of the config for OCC)
//   - LastModified time.Time
//   - Sources map (already embedded in EffectiveConfig)
//
// Example:
//
//	EffectiveConfigResponse{
//	  EffectiveConfig: effect,
//	  Version:         "sha256:abc123...",
//	}
type EffectiveConfigResponse struct {
	// EffectiveConfig is the fully resolved configuration tree merging
	// defaults, YAML, env vars, and DB overrides.
	EffectiveConfig EffectiveConfig `json:"config"`

	// Version is an opaque version identifier for the config snapshot.
	// Currently empty; reserved for future OCC support (e.g., SHA256 hash
	// of the merged config). Clients can use this for cache validation.
	Version string `json:"version,omitempty" example:"sha256:abc123def456"`
}
