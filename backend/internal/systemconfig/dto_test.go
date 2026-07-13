package systemconfig

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SetOverrideRequest
// ---------------------------------------------------------------------------

func TestSetOverrideRequest_JSONRoundTrip(t *testing.T) {
	req := SetOverrideRequest{
		Key:   "scoring.auto_threshold",
		Value: 95,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SetOverrideRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Key, decoded.Key)
	assert.Equal(t, float64(95), decoded.Value) // JSON numbers unmarshal to float64 by default
}

func TestSetOverrideRequest_StringValue(t *testing.T) {
	req := SetOverrideRequest{
		Key:   "scoring.mode",
		Value: "hybrid",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SetOverrideRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "scoring.mode", decoded.Key)
	assert.Equal(t, "hybrid", decoded.Value)
}

func TestSetOverrideRequest_BoolValue(t *testing.T) {
	req := SetOverrideRequest{
		Key:   "automation.auto_generate.resume",
		Value: true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SetOverrideRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, true, decoded.Value)
}

func TestSetOverrideRequest_ArrayValue(t *testing.T) {
	req := SetOverrideRequest{
		Key:   "email.folders",
		Value: []interface{}{"INBOX", "JOBS"},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SetOverrideRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "email.folders", decoded.Key)

	decodedArr, ok := decoded.Value.([]interface{})
	require.True(t, ok, "decoded value should be an array")
	assert.Equal(t, "INBOX", decodedArr[0])
	assert.Equal(t, "JOBS", decodedArr[1])
}

func TestSetOverrideRequest_BindingTags(t *testing.T) {
	// Verify the binding:"required" tag is present for both fields
	typ := reflect.TypeOf(SetOverrideRequest{})

	keyField, ok := typ.FieldByName("Key")
	require.True(t, ok, "Key field must exist")
	assert.Contains(t, keyField.Tag.Get("binding"), "required", "Key must have binding:required")
	assert.Contains(t, keyField.Tag.Get("json"), "key", "Key must have json:key")

	valueField, ok := typ.FieldByName("Value")
	require.True(t, ok, "Value field must exist")
	assert.Contains(t, valueField.Tag.Get("binding"), "required", "Value must have binding:required")
	assert.Contains(t, valueField.Tag.Get("json"), "value", "Value must have json:value")
}

// ---------------------------------------------------------------------------
// SetOverrideResponse
// ---------------------------------------------------------------------------

func TestSetOverrideResponse_JSONRoundTrip(t *testing.T) {
	resp := SetOverrideResponse{
		Message: "override saved",
		Key:     "scoring.auto_threshold",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SetOverrideResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Message, decoded.Message)
	assert.Equal(t, resp.Key, decoded.Key)
}

func TestSetOverrideResponse_JSONFieldNames(t *testing.T) {
	resp := SetOverrideResponse{
		Message: "override saved",
		Key:     "scoring.auto_threshold",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "override saved", raw["message"])
	assert.Equal(t, "scoring.auto_threshold", raw["key"])
}

// ---------------------------------------------------------------------------
// DeleteOverrideResponse
// ---------------------------------------------------------------------------

func TestDeleteOverrideResponse_JSONRoundTrip(t *testing.T) {
	resp := DeleteOverrideResponse{
		Message: "override deleted",
		Key:     "scoring.mode",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded DeleteOverrideResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Message, decoded.Message)
	assert.Equal(t, resp.Key, decoded.Key)
}

func TestDeleteOverrideResponse_JSONFieldNames(t *testing.T) {
	resp := DeleteOverrideResponse{
		Message: "override deleted",
		Key:     "scoring.mode",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "override deleted", raw["message"])
	assert.Equal(t, "scoring.mode", raw["key"])
}

// ---------------------------------------------------------------------------
// EffectiveConfigResponse
// ---------------------------------------------------------------------------

func TestEffectiveConfigResponse_JSON(t *testing.T) {
	resp := EffectiveConfigResponse{
		EffectiveConfig: EffectiveConfig{
			Scoring: ScoringSection{
				AutoThreshold: 95,
				Mode:          ModeHybrid,
			},
			Sources: map[string]ConfigSource{
				"scoring.auto_threshold": SourceYAML,
			},
		},
		Version: "sha256:abc123",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded EffectiveConfigResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 95, decoded.EffectiveConfig.Scoring.AutoThreshold)
	assert.Equal(t, ModeHybrid, decoded.EffectiveConfig.Scoring.Mode)
	assert.Equal(t, "sha256:abc123", decoded.Version)
	require.NotNil(t, decoded.EffectiveConfig.Sources)
	assert.Equal(t, SourceYAML, decoded.EffectiveConfig.Sources["scoring.auto_threshold"])
}

func TestEffectiveConfigResponse_JSONFieldNames(t *testing.T) {
	resp := EffectiveConfigResponse{
		EffectiveConfig: EffectiveConfig{
			Scoring: ScoringSection{AutoThreshold: 90},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	config, ok := raw["config"].(map[string]interface{})
	require.True(t, ok, "config should be nested under 'config' key")
	scoring, ok := config["scoring"].(map[string]interface{})
	require.True(t, ok, "scoring should be inside config")
	assert.Equal(t, float64(90), scoring["auto_threshold"])
}

func TestEffectiveConfigResponse_VersionOmitEmpty(t *testing.T) {
	resp := EffectiveConfigResponse{
		EffectiveConfig: EffectiveConfig{
			Scoring: ScoringSection{AutoThreshold: 85},
		},
		// Version is empty — should be omitted from JSON
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasVersion := raw["version"]
	assert.False(t, hasVersion, "version should be omitted when empty")
}

func TestEffectiveConfigResponse_WithVersion(t *testing.T) {
	// Verify version appears when set (omitempty behavior)
	cfg := EffectiveConfig{
		Scoring: ScoringSection{AutoThreshold: 95, ReviewThreshold: 80},
	}
	cfg.Sources = map[string]ConfigSource{"scoring.auto_threshold": SourceYAML}

	resp := EffectiveConfigResponse{
		EffectiveConfig: cfg,
		Version:         "sha256:abc123def456",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasVersion := raw["version"]
	assert.True(t, hasVersion, "version should be present when non-empty")
	assert.Equal(t, "sha256:abc123def456", raw["version"])
}


