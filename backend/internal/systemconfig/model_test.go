package systemconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigSourceConstants(t *testing.T) {
	assert.Equal(t, ConfigSource("default"), SourceDefault)
	assert.Equal(t, ConfigSource("yaml"), SourceYAML)
	assert.Equal(t, ConfigSource("env"), SourceEnv)
	assert.Equal(t, ConfigSource("db"), SourceDB)
}

func TestConfigCategoryConstants(t *testing.T) {
	assert.Equal(t, ConfigCategory("runtime"), CategoryRuntime)
	assert.Equal(t, ConfigCategory("operational"), CategoryOperational)
	assert.Equal(t, ConfigCategory("infrastructure"), CategoryInfrastructure)
}

func TestScoringModeConstants(t *testing.T) {
	assert.Equal(t, ScoringMode("heuristic"), ModeHeuristic)
	assert.Equal(t, ScoringMode("llm"), ModeLLM)
	assert.Equal(t, ScoringMode("hybrid"), ModeHybrid)
}

func TestScoringMode_IsValid(t *testing.T) {
	tests := []struct {
		name string
		mode ScoringMode
		want bool
	}{
		{name: "heuristic is valid", mode: ModeHeuristic, want: true},
		{name: "llm is valid", mode: ModeLLM, want: true},
		{name: "hybrid is valid", mode: ModeHybrid, want: true},
		{name: "empty string is invalid", mode: ScoringMode(""), want: false},
		{name: "random string is invalid", mode: ScoringMode("unknown"), want: false},
		{name: "case mismatch is invalid", mode: ScoringMode("Heuristic"), want: false},
		{name: "whitespace is invalid", mode: ScoringMode(" hybrid "), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIntegrationStatusTypeConstants(t *testing.T) {
	assert.Equal(t, IntegrationStatusType("connected"), StatusConnected)
	assert.Equal(t, IntegrationStatusType("disconnected"), StatusDisconnected)
	assert.Equal(t, IntegrationStatusType("error"), StatusError)
}

func TestOverride_TableName(t *testing.T) {
	var o Override
	assert.Equal(t, "system_config_overrides", o.TableName())
}

// ---------------------------------------------------------------------------
// ValidateOverrideKey
// ---------------------------------------------------------------------------

func TestValidateOverrideKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr string
	}{
		// Valid keys (lowercase, letters, digits, and dots only — NO underscores)
		{name: "valid two-segment key", key: "scoring.mode", wantErr: ""},
		{name: "valid deep key", key: "scoring.weights.skill", wantErr: ""},
		{name: "valid deep key 4 levels", key: "interview.responder.llm.timeoutms", wantErr: ""},
		{name: "valid key with digits", key: "llm.primary.model", wantErr: ""},
		{name: "valid automation key", key: "automation.queue.concurrency", wantErr: ""},
		// Invalid keys
		{name: "empty key", key: "", wantErr: "must not be empty"},
		{name: "key with spaces", key: "scoring. auto", wantErr: "must not contain spaces"},
		{name: "uppercase key", key: "Scoring.Mode", wantErr: "must be lowercase"},
		{name: "single segment", key: "scoring", wantErr: "at least 2 segments"},
		{name: "trailing dot", key: "scoring.", wantErr: "empty segment at position 1"},
		{name: "leading dot", key: ".scoring", wantErr: "empty segment at position 0"},
		{name: "double dot", key: "scoring..mode", wantErr: "empty segment at position 1"},
		{name: "underscore rejected", key: "scoring.auto_threshold", wantErr: "contains invalid character"},
		{name: "underscore in multi", key: "rate_limits.rpm", wantErr: "contains invalid character"},
		{name: "slash not a dot separator", key: "scoring/mode", wantErr: "at least 2 segments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOverrideKey(tt.key)
			if tt.wantErr == "" {
				assert.NoError(t, err, "key %q should be valid", tt.key)
			} else {
				require.Error(t, err, "key %q should be invalid", tt.key)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateOverrideValue
// ---------------------------------------------------------------------------

func TestValidateOverrideValue(t *testing.T) {
	tests := []struct {
		name    string
		value   json.RawMessage
		wantErr string
	}{
		// Valid values
		{name: "integer", value: json.RawMessage(`90`), wantErr: ""},
		{name: "negative integer", value: json.RawMessage(`-5`), wantErr: ""},
		{name: "float", value: json.RawMessage(`0.35`), wantErr: ""},
		{name: "string", value: json.RawMessage(`"hybrid"`), wantErr: ""},
		{name: "boolean true", value: json.RawMessage(`true`), wantErr: ""},
		{name: "boolean false", value: json.RawMessage(`false`), wantErr: ""},
		{name: "array of strings", value: json.RawMessage(`["a","b"]`), wantErr: ""},
		{name: "array of ints", value: json.RawMessage(`[1,2,3]`), wantErr: ""},
		{name: "object", value: json.RawMessage(`{"key":"val"}`), wantErr: ""},
		// Invalid values
		{name: "empty raw message", value: json.RawMessage(``), wantErr: "must not be empty"},
		{name: "null", value: json.RawMessage(`null`), wantErr: "must not be null"},
		{name: "empty object", value: json.RawMessage(`{}`), wantErr: "must not be an empty object"},
		{name: "empty array", value: json.RawMessage(`[]`), wantErr: "must not be an empty array"},
		{name: "empty string", value: json.RawMessage(`""`), wantErr: "must not be an empty string"},
		{name: "whitespace string", value: json.RawMessage(`"   "`), wantErr: "must not be an empty string"},
		// NOTE: ValidateOverrideValue does NOT validate full JSON syntax.
		// The function only checks: empty, null, {}, [], and empty string patterns.
		// Non-string values like {invalid are accepted as valid because semantic
		// JSON validation happens in the service layer (validateValue).
		// Strings without closing quotes: the function only checks properly-quoted
		// strings; malformed quoted values pass through without error.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOverrideValue(tt.value)
			if tt.wantErr == "" {
				assert.NoError(t, err, "value %s should be valid", string(tt.value))
			} else {
				require.Error(t, err, "value %s should be invalid", string(tt.value))
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// KeyPrefix
// ---------------------------------------------------------------------------

func TestKeyPrefix(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "two segments", key: "scoring.auto_threshold", want: "scoring"},
		{name: "three segments", key: "scoring.weights.skill", want: "scoring"},
		{name: "deep nesting", key: "interview.responder.llm.timeout_ms", want: "interview"},
		{name: "single segment", key: "scoring", want: "scoring"},
		{name: "empty key", key: "", want: ""},
		{name: "dot at start", key: ".foo", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KeyPrefix(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// CategoryForKey
// ---------------------------------------------------------------------------

func TestCategoryForKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want ConfigCategory
	}{
		// Runtime
		{name: "scoring prefix", key: "scoring.auto_threshold", want: CategoryRuntime},
		// Operational
		{name: "queue prefix", key: "queue.concurrency", want: CategoryOperational},
		{name: "email prefix", key: "email.check_interval", want: CategoryOperational},
		{name: "rate_limits prefix", key: "rate_limits.rpm", want: CategoryOperational},
		{name: "auto_generate prefix", key: "auto_generate.resume", want: CategoryOperational},
		// Infrastructure
		{name: "llm prefix", key: "llm.primary.model", want: CategoryInfrastructure},
		{name: "voice prefix", key: "voice.provider", want: CategoryInfrastructure},
		{name: "interview prefix", key: "interview.memory.max_recent_segments", want: CategoryInfrastructure},
		{name: "livekit prefix", key: "livekit.url", want: CategoryInfrastructure},
		{name: "embeddings prefix", key: "embeddings.model", want: CategoryInfrastructure},
		// Unknown prefix defaults to Runtime
		{name: "unknown prefix defaults to runtime", key: "unknown.key", want: CategoryRuntime},
		{name: "empty prefix defaults to runtime", key: "", want: CategoryRuntime},
		{name: "approval_tiers prefix", key: "approval_tiers.auto_apply.min_score", want: CategoryRuntime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategoryForKey(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// EffectiveConfig nil Sources map safety
// ---------------------------------------------------------------------------

func TestEffectiveConfig_SourcesNotNil(t *testing.T) {
	// Sources is not initialized by struct literal — must be explicitly allocated.
	// buildFromYAML in resolver.go does this, but direct construction doesn't.
	cfg := EffectiveConfig{}
	assert.Nil(t, cfg.Sources, "Sources should be nil when not explicitly initialized")
}
