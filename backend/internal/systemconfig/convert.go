// Package systemconfig provides configuration resolution logic for the job search agent.
// This file contains type conversion helpers for parsing JSON raw messages into
// specific Go types. These are used by the resolver when applying DB overrides.
//
// # Design Constraints
//
//   - All functions return descriptive errors on type mismatch or parse failure.
//   - These are internal helpers — not exported, not part of the public API.
package systemconfig

import (
	"encoding/json"
	"fmt"
)

// toInt converts a JSON raw message to an integer.
// Returns an error if the value is not a valid JSON number.
func toInt(raw json.RawMessage) (int, error) {
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, fmt.Errorf("not a valid integer: %w", err)
	}
	return v, nil
}

// toFloat converts a JSON raw message to a float64.
// Returns an error if the value is not a valid JSON number.
func toFloat(raw json.RawMessage) (float64, error) {
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, fmt.Errorf("not a valid float: %w", err)
	}
	return v, nil
}

// toBool converts a JSON raw message to a boolean.
// Returns an error if the value is not a valid JSON boolean.
func toBool(raw json.RawMessage) (bool, error) {
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return false, fmt.Errorf("not a valid boolean: %w", err)
	}
	return v, nil
}

// toString converts a JSON raw message to a string.
// Returns an error if the value is not a valid JSON string.
func toString(raw json.RawMessage) (string, error) {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("not a valid string: %w", err)
	}
	return v, nil
}

// toStringSlice converts a JSON raw message to a string slice.
// Returns an error if the value is not a valid JSON array of strings.
func toStringSlice(raw json.RawMessage) ([]string, error) {
	var v []string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("not a valid string array: %w", err)
	}
	return v, nil
}
