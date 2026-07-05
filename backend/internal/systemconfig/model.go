// Package systemconfig provides runtime configuration override management for the
// job search agent. It implements a YAML → env vars → DB overrides merge strategy
// where each layer can override the previous, and changes take effect without restart.
//
// The package defines the domain model for configuration overrides (stored in DB),
// the fully resolved EffectiveConfig tree (returned by the GET API), and constants
// for configuration categories, scoring modes, and integration statuses.
//
// # Design Principles
//
//   - Domain models (Override) do NOT carry JSON tags — API serialization is handled
//     by DTOs in the handler layer. DB column mapping uses sqlx struct tags only.
//   - The scoring.Weights type is imported from the scoring package, not redefined here.
//     This avoids duplication and ensures weight validation stays in one place.
//   - EffectiveConfig mirrors the YAML structure but is the fully merged result.
//     Each leaf value's origin is tracked in the Sources map.
//   - Validation helpers enforce key format (dotted notation) and value constraints
//     at the domain boundary before persistence.
//
// # Usage
//
//	resolver := systemconfig.NewResolver(db, yamlConfig)
//	effect, err := resolver.Resolve(ctx)
//
//	override := &systemconfig.Override{
//	  Key:       "scoring.auto_threshold",
//	  Value:     json.RawMessage(`90`),
//	  Category:  systemconfig.CategoryRuntime,
//	}
//	if err := systemconfig.ValidateOverrideKey(override.Key); err != nil {
//	  return err
//	}
//
// # What This Package Does NOT Do
//
//   - Does not parse YAML or read env vars — that is config.Load()'s responsibility.
//   - Does not handle HTTP routing or request parsing — that lives in the handler layer.
//   - Does not define API request/response DTOs — those are in dto.go.
//   - Does not enforce scoring weight sum-to-1.0 validation — use scoring.Weights.Validate().
package systemconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"backend/internal/scoring"
)

// Sentinel errors for domain-specific failures.
// Use errors.Is() to check these in the handler layer.
var (
	// ErrKeyNotAllowed indicates the key is not in the allowlist of configurable keys.
	ErrKeyNotAllowed = errors.New("systemconfig: key not in allowlist")

	// ErrInvalidValue indicates the value failed key-specific validation.
	ErrInvalidValue = errors.New("systemconfig: invalid override value")

	// ErrInvalidKeyFormat indicates the key doesn't match the dotted-notation format.
	ErrInvalidKeyFormat = errors.New("systemconfig: invalid key format")
)

// ---------------------------------------------------------------------------
// ConfigSource — origin of a configuration value
// ---------------------------------------------------------------------------

// ConfigSource represents the origin of a configuration value in the merge chain.
// Values are resolved in order: default → yaml → env → db, where each layer
// overrides the previous. The Sources map on EffectiveConfig records which layer
// produced each leaf value.
type ConfigSource string

const (
	// SourceDefault is the hardcoded fallback when no other source provides a value.
	SourceDefault ConfigSource = "default"

	// SourceYAML indicates the value was loaded from config/application.yaml.
	SourceYAML ConfigSource = "yaml"

	// SourceEnv indicates the value was set via an environment variable.
	SourceEnv ConfigSource = "env"

	// SourceDB indicates the value came from a database override row.
	// DB overrides take highest precedence in the merge chain.
	SourceDB ConfigSource = "db"
)

// ---------------------------------------------------------------------------
// ConfigCategory — functional grouping for overrides
// ---------------------------------------------------------------------------

// ConfigCategory classifies overrides into functional groups for the admin UI.
// Categories are stored in the category column of system_config_overrides and
// determine which section of the config dashboard an override appears in.
type ConfigCategory string

const (
	// CategoryRuntime covers values that affect scoring behavior and thresholds.
	// Examples: scoring.auto_threshold, scoring.mode, scoring.weights.skill.
	CategoryRuntime ConfigCategory = "runtime"

	// CategoryOperational covers values that affect queue processing and email polling.
	// Examples: queue.concurrency, email.check_interval, rate_limits.rpm.
	CategoryOperational ConfigCategory = "operational"

	// CategoryInfrastructure covers values that affect LLM providers and integrations.
	// Examples: llm.primary.model, voice.provider, interview.memory.max_recent_segments.
	CategoryInfrastructure ConfigCategory = "infrastructure"
)

// ---------------------------------------------------------------------------
// ScoringMode — string enum for scoring strategy
// ---------------------------------------------------------------------------

// ScoringMode defines which scoring strategy the system uses to evaluate job matches.
type ScoringMode string

const (
	// ModeHeuristic uses keyword-based matching only. Fast, deterministic, no LLM cost.
	ModeHeuristic ScoringMode = "heuristic"

	// ModeLLM uses semantic evaluation via a language model. Higher quality, slower, costs tokens.
	ModeLLM ScoringMode = "llm"

	// ModeHybrid pre-filters with heuristic scoring, then sends ambiguous cases to the LLM.
	// This is the default — it balances speed and quality.
	ModeHybrid ScoringMode = "hybrid"
)

// IsValid returns true if m is a recognized scoring mode.
func (m ScoringMode) IsValid() bool {
	switch m {
	case ModeHeuristic, ModeLLM, ModeHybrid:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// IntegrationStatusType — connection health for external services
// ---------------------------------------------------------------------------

// IntegrationStatusType represents the connection health of an external service
// integration (LiveKit, email provider, LLM provider, etc.).
type IntegrationStatusType string

const (
	// StatusConnected means the integration is reachable and healthy.
	StatusConnected IntegrationStatusType = "connected"

	// StatusDisconnected means the integration is configured but not currently reachable.
	StatusDisconnected IntegrationStatusType = "disconnected"

	// StatusError means the integration attempted connection but returned an error.
	StatusError IntegrationStatusType = "error"
)

// ---------------------------------------------------------------------------
// Override — DB-persisted runtime configuration override
// ---------------------------------------------------------------------------

// Override represents a single runtime configuration override stored in the
// database. Overrides use dotted-notation keys (e.g., "scoring.auto_threshold")
// and store values as raw JSON to support int, float, bool, string, and array types.
//
// Domain models intentionally omit JSON tags — API serialization is handled by
// DTOs in the handler layer. Only sqlx struct tags are present for DB mapping.
//
// Example:
//
//	&Override{
//	  Key:       "scoring.auto_threshold",
//	  Value:     json.RawMessage(`90`),
//	  Category:  CategoryRuntime,
//	  UpdatedBy: &userID,
//	}
type Override struct {
	ID          uuid.UUID       `db:"id"`
	Key         string          `db:"key"`
	Value       json.RawMessage `db:"value"`
	Category    ConfigCategory  `db:"category"`
	Description string          `db:"description"`
	UpdatedBy   *uuid.UUID      `db:"updated_by"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

// TableName returns the database table name for Override.
func (Override) TableName() string {
	return "system_config_overrides"
}

// ---------------------------------------------------------------------------
// EffectiveConfig — fully resolved configuration tree
// ---------------------------------------------------------------------------

// EffectiveConfig is the fully resolved configuration tree returned by the GET
// /api/system-config endpoint. It merges defaults, YAML, env vars, and DB
// overrides into a single struct that mirrors the YAML hierarchy.
//
// The Sources map tracks which layer (default/yaml/env/db) produced each leaf
// value, enabling the frontend to display which values are user-overridden vs
// system defaults.
//
// Fields intentionally mirror config/application.yaml sections, NOT the
// config.Config struct — this is a read-only view for the API, not a
// processing struct.
type EffectiveConfig struct {
	// Scoring holds scoring thresholds, weights, and mode.
	Scoring ScoringSection `json:"scoring"`

	// LLM holds primary, local, and embedding provider settings.
	LLM LLMSection `json:"llm"`

	// Voice holds voice provider and LiveKit configuration.
	Voice VoiceSection `json:"voice"`

	// ApprovalTiers holds auto/review/reject tier definitions with score ranges.
	ApprovalTiers ApprovalTiersSection `json:"approval_tiers"`

	// ResumeConfig holds resume generation engine and template settings.
	ResumeConfig ResumeConfigSection `json:"resume"`

	// CoverLetterConfig holds cover letter generation engine and template settings.
	CoverLetterConfig CoverLetterConfigSection `json:"cover_letter"`

	// Automation holds queue concurrency and auto-generation toggles.
	Automation AutomationSection `json:"automation"`

	// Interview holds interview agent runtime settings (memory, responder, planner).
	Interview InterviewSection `json:"interview"`

	// Email holds email polling interval and folder configuration.
	Email EmailSection `json:"email"`

	// RateLimits holds API rate limit settings.
	RateLimits RateLimitsSection `json:"rate_limits"`

	// Integrations holds connection health for external services.
	Integrations IntegrationsSection `json:"integrations"`

	// Sources maps dotted-notation config keys to their origin layer.
	// Example: {"scoring.auto_threshold": "db", "llm.primary.model": "yaml"}
	Sources map[string]ConfigSource `json:"sources"`
}

// ScoringSection holds scoring thresholds, weights, and mode.
// Weights uses scoring.Weights from the scoring package — not a local copy.
type ScoringSection struct {
	AutoThreshold      int             `json:"auto_threshold"`
	ReviewThreshold    int             `json:"review_threshold"`
	Mode               ScoringMode     `json:"mode"`
	HybridRejectMargin int             `json:"hybrid_reject_margin"`
	Weights            scoring.Weights `json:"weights"`
}

// LLMSection holds LLM provider configurations.
type LLMSection struct {
	Primary    LLMProviderSection `json:"primary"`
	Local      LLMProviderSection `json:"local"`
	Embeddings LLMProviderSection `json:"embeddings"`
}

// LLMProviderSection holds a single LLM provider's model and connection info.
type LLMProviderSection struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// VoiceSection holds voice provider and LiveKit configuration.
type VoiceSection struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	LiveKit  LiveKitSection    `json:"livekit"`
	Settings map[string]string `json:"settings"`
}

// LiveKitSection holds LiveKit connection parameters.
// The API secret is intentionally omitted — secrets are never exposed via API responses.
type LiveKitSection struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// ApprovalTierDef defines a single approval tier with its score range and action.
type ApprovalTierDef struct {
	MinScore int    `json:"min_score"`
	MaxScore int    `json:"max_score,omitempty"`
	Action   string `json:"action"`
	Notify   bool   `json:"notify,omitempty"`
	Log      bool   `json:"log,omitempty"`
}

// ApprovalTiersSection holds auto/review/reject tier definitions.
type ApprovalTiersSection struct {
	AutoApply ApprovalTierDef `json:"auto_apply"`
	Review    ApprovalTierDef `json:"review"`
	Reject    ApprovalTierDef `json:"reject"`
}

// ResumeConfigSection holds resume generation engine and template settings.
type ResumeConfigSection struct {
	Engine      string `json:"engine"`
	TemplateDir string `json:"template_dir"`
}

// CoverLetterConfigSection holds cover letter generation engine and template settings.
type CoverLetterConfigSection struct {
	Engine      string `json:"engine"`
	TemplateDir string `json:"template_dir"`
	MaxLength   int    `json:"max_length"`
}

// AutomationSection holds queue and auto-generation settings.
type AutomationSection struct {
	Queue        QueueSection        `json:"queue"`
	AutoGenerate AutoGenerateSection `json:"auto_generate"`
}

// QueueSection holds async queue processing settings.
type QueueSection struct {
	Concurrency   int `json:"concurrency"`
	RetryAttempts int `json:"retry_attempts"`
}

// AutoGenerateSection controls automatic resume and cover letter generation.
type AutoGenerateSection struct {
	Resume      bool `json:"resume"`
	CoverLetter bool `json:"cover_letter"`
}

// InterviewSection holds interview agent runtime settings.
type InterviewSection struct {
	Memory    InterviewMemory    `json:"memory"`
	Responder InterviewResponder `json:"responder"`
	Planner   InterviewPlanner   `json:"planner"`
}

// InterviewMemory holds transcript window and eviction settings.
type InterviewMemory struct {
	MaxRecentSegments  int `json:"max_recent_segments"`
	KeepAfterSummarize int `json:"keep_after_summarize"`
}

// InterviewResponder holds LLM timeout and retry settings for the responder.
type InterviewResponder struct {
	LLM LLMTimeout `json:"llm"`
}

// LLMTimeout holds timeout and retry configuration for LLM calls.
type LLMTimeout struct {
	TimeoutMs int `json:"timeout_ms"`
	Retries   int `json:"retries"`
}

// InterviewPlanner holds decision thresholds for the interview planner.
type InterviewPlanner struct {
	DuplicateThreshold   float64 `json:"duplicate_threshold"`
	MinSubstantiveLength int     `json:"min_substantive_length"`
}

// EmailSection holds email polling interval and folder configuration.
type EmailSection struct {
	Provider      string   `json:"provider"`
	CheckInterval string   `json:"check_interval"`
	Folders       []string `json:"folders"`
}

// RateLimitsSection holds API rate limit settings.
type RateLimitsSection struct {
	RPM   int `json:"rpm"`
	Burst int `json:"burst"`
}

// IntegrationsSection holds connection health for external services.
type IntegrationsSection struct {
	LiveKit     IntegrationStatus         `json:"livekit"`
	Email       IntegrationStatus         `json:"email"`
	AIProviders map[string]AIProviderInfo `json:"ai_providers"`
}

// IntegrationStatus holds connection health and optional URL for a service.
type IntegrationStatus struct {
	Status IntegrationStatusType `json:"status"`
	URL    *string               `json:"url,omitempty"`
}

// AIProviderInfo holds a provider's model and connection status.
type AIProviderInfo struct {
	Model  string                `json:"model"`
	Status IntegrationStatusType `json:"status"`
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

// ValidateOverrideKey checks that a dotted-notation config key is well-formed.
// Keys must be non-empty, contain only lowercase letters, digits, and dots,
// and have at least two segments (e.g., "scoring.mode").
//
// Returns nil on success, or a descriptive error on failure.
//
// Example:
//
//	systemconfig.ValidateOverrideKey("scoring.auto_threshold") // nil
//	systemconfig.ValidateOverrideKey("mode")                   // error: must have at least 2 segments
//	systemconfig.ValidateOverrideKey("Scoring.Mode")           // error: must be lowercase
func ValidateOverrideKey(key string) error {
	if key == "" {
		return fmt.Errorf("systemconfig: override key must not be empty")
	}
	if strings.Contains(key, " ") {
		return fmt.Errorf("systemconfig: override key must not contain spaces: %q", key)
	}
	if key != strings.ToLower(key) {
		return fmt.Errorf("systemconfig: override key must be lowercase: %q", key)
	}
	segments := strings.Split(key, ".")
	if len(segments) < 2 {
		return fmt.Errorf("systemconfig: override key must have at least 2 segments (e.g., 'scoring.mode'): %q", key)
	}
	for i, seg := range segments {
		if seg == "" {
			return fmt.Errorf("systemconfig: override key has empty segment at position %d: %q", i, key)
		}
		for _, c := range seg {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				return fmt.Errorf("systemconfig: override key segment %q contains invalid character %q in key %q", seg, string(c), key)
			}
		}
	}
	return nil
}

// ValidateOverrideValue checks that a JSON raw message is a valid, non-empty
// override value. It rejects null, empty objects {}, empty arrays [], and
// whitespace-only strings. It does NOT validate semantic correctness (e.g., that
// an int field actually contains an int) — that is the service layer's responsibility.
//
// Returns nil on success, or a descriptive error on failure.
//
// Example:
//
//	systemconfig.ValidateOverrideValue(json.RawMessage(`90`))                          // nil
//	systemconfig.ValidateOverrideValue(json.RawMessage(`"hybrid"`))                    // nil
//	systemconfig.ValidateOverrideValue(json.RawMessage(`null`))                        // error: null not allowed
//	systemconfig.ValidateOverrideValue(json.RawMessage(`{}`))                          // error: empty object not allowed
func ValidateOverrideValue(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("systemconfig: override value must not be empty")
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return fmt.Errorf("systemconfig: override value must not be null")
	}
	if trimmed == "{}" {
		return fmt.Errorf("systemconfig: override value must not be an empty object")
	}
	if trimmed == "[]" {
		return fmt.Errorf("systemconfig: override value must not be an empty array")
	}

	// For string values, reject empty strings
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("systemconfig: override value is not valid JSON: %w", err)
		}
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("systemconfig: override value must not be an empty string")
		}
	}

	return nil
}

// KeyPrefix returns the first segment of a dotted key, useful for category
// assignment. For example, "scoring.auto_threshold" returns "scoring".
//
// Returns empty string if key has no segments.
func KeyPrefix(key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// CategoryForKey infers the ConfigCategory for a given dotted key based on
// its prefix. This is used as a default when the caller doesn't specify a category.
//
// Prefix mapping:
//   - "scoring" → CategoryRuntime
//   - "queue", "email", "rate_limits" → CategoryOperational
//   - "llm", "voice", "interview", "livekit" → CategoryInfrastructure
//
// Returns CategoryRuntime for unrecognized prefixes as a safe default.
func CategoryForKey(key string) ConfigCategory {
	prefix := KeyPrefix(key)
	switch prefix {
	case "scoring":
		return CategoryRuntime
	case "queue", "email", "rate_limits", "auto_generate":
		return CategoryOperational
	case "llm", "voice", "interview", "livekit", "embeddings":
		return CategoryInfrastructure
	default:
		return CategoryRuntime
	}
}
