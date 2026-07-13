package systemconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// toInt
// ---------------------------------------------------------------------------

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    int
		wantErr string
	}{
		{name: "positive integer", raw: json.RawMessage(`95`), want: 95},
		{name: "zero", raw: json.RawMessage(`0`), want: 0},
		{name: "negative integer", raw: json.RawMessage(`-5`), want: -5},
		{name: "large int", raw: json.RawMessage(`2147483647`), want: 2147483647},
		{name: "null becomes zero", raw: json.RawMessage(`null`), want: 0},
		// Error cases
		{name: "float rejected", raw: json.RawMessage(`95.5`), wantErr: "not a valid integer"},
		{name: "string rejected", raw: json.RawMessage(`"95"`), wantErr: "not a valid integer"},
		{name: "bool rejected", raw: json.RawMessage(`true`), wantErr: "not a valid integer"},
		{name: "empty rejected", raw: json.RawMessage(``), wantErr: "not a valid integer"},
		{name: "invalid syntax", raw: json.RawMessage(`abc`), wantErr: "not a valid integer"},
		{name: "object rejected", raw: json.RawMessage(`{}`), wantErr: "not a valid integer"},
		{name: "array rejected", raw: json.RawMessage(`[1]`), wantErr: "not a valid integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toInt(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err, "should return error for %s", string(tt.raw))
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, 0, got, "should return zero on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toFloat
// ---------------------------------------------------------------------------

func TestToFloat(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    float64
		wantErr string
	}{
		{name: "integer as float", raw: json.RawMessage(`95`), want: 95.0},
		{name: "decimal float", raw: json.RawMessage(`0.35`), want: 0.35},
		{name: "zero", raw: json.RawMessage(`0`), want: 0.0},
		{name: "negative float", raw: json.RawMessage(`-5.5`), want: -5.5},
		{name: "scientific notation", raw: json.RawMessage(`1e-2`), want: 0.01},
		{name: "null becomes zero", raw: json.RawMessage(`null`), want: 0.0},
		// Error cases
		{name: "string rejected", raw: json.RawMessage(`"0.35"`), wantErr: "not a valid float"},
		{name: "bool rejected", raw: json.RawMessage(`false`), wantErr: "not a valid float"},
		{name: "object rejected", raw: json.RawMessage(`{}`), wantErr: "not a valid float"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, 0.0, got, "should return zero on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toBool
// ---------------------------------------------------------------------------

func TestToBool(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    bool
		wantErr string
	}{
		{name: "true", raw: json.RawMessage(`true`), want: true},
		{name: "false", raw: json.RawMessage(`false`), want: false},
		{name: "null becomes false", raw: json.RawMessage(`null`), want: false},
		// Error cases
		{name: "integer rejected", raw: json.RawMessage(`1`), wantErr: "not a valid boolean"},
		{name: "string rejected", raw: json.RawMessage(`"true"`), wantErr: "not a valid boolean"},
		{name: "float rejected", raw: json.RawMessage(`0.0`), wantErr: "not a valid boolean"},
		{name: "empty rejected", raw: json.RawMessage(``), wantErr: "not a valid boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toBool(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, false, got, "should return false on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toString
// ---------------------------------------------------------------------------

func TestToString(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    string
		wantErr string
	}{
		{name: "simple string", raw: json.RawMessage(`"hybrid"`), want: "hybrid"},
		{name: "string with spaces", raw: json.RawMessage(`"hello world"`), want: "hello world"},
		{name: "empty string value", raw: json.RawMessage(`""`), want: ""},
		{name: "numeric string", raw: json.RawMessage(`"95"`), want: "95"},
		{name: "null becomes empty", raw: json.RawMessage(`null`), want: ""},
		// Error cases
		{name: "integer rejected", raw: json.RawMessage(`95`), wantErr: "not a valid string"},
		{name: "bool rejected", raw: json.RawMessage(`true`), wantErr: "not a valid string"},
		{name: "array rejected", raw: json.RawMessage(`["a"]`), wantErr: "not a valid string"},
		{name: "empty rejected", raw: json.RawMessage(``), wantErr: "not a valid string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toString(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, "", got, "should return empty on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toStringSlice
// ---------------------------------------------------------------------------

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    []string
		wantErr string
	}{
		{name: "string array", raw: json.RawMessage(`["a","b","c"]`), want: []string{"a", "b", "c"}},
		{name: "single element", raw: json.RawMessage(`["only"]`), want: []string{"only"}},
		{name: "empty array", raw: json.RawMessage(`[]`), want: []string{}},
		{name: "null becomes nil", raw: json.RawMessage(`null`), want: nil},
		// Error cases
		{name: "int array rejected", raw: json.RawMessage(`[1,2,3]`), wantErr: "not a valid string array"},
		{name: "mixed array rejected", raw: json.RawMessage(`["a",1]`), wantErr: "not a valid string array"},
		{name: "string rejected", raw: json.RawMessage(`"not-an-array"`), wantErr: "not a valid string array"},
		{name: "object rejected", raw: json.RawMessage(`{}`), wantErr: "not a valid string array"},
		{name: "empty rejected", raw: json.RawMessage(``), wantErr: "not a valid string array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toStringSlice(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got, "should return nil on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
