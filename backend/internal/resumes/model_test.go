package resumes

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// --- StringSliceDB.Value ---

func TestStringSliceDB_Value_NormalSlice(t *testing.T) {
	s := StringSliceDB{"go", "python", "rust"}
	val, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	// Verify it's valid JSON
	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}
	if result[0] != "go" || result[1] != "python" || result[2] != "rust" {
		t.Errorf("unexpected elements: %v", result)
	}
}

func TestStringSliceDB_Value_NilSlice(t *testing.T) {
	var s StringSliceDB
	val, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestStringSliceDB_Value_EmptySlice(t *testing.T) {
	s := StringSliceDB{}
	val, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	// Empty JSON array
	if string(bytes) != "[]" {
		t.Errorf("expected '[]', got %q", string(bytes))
	}
}

func TestStringSliceDB_Value_SingleElement(t *testing.T) {
	s := StringSliceDB{"only"}
	val, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val.([]byte)) != `["only"]` {
		t.Errorf("expected [\"only\"], got %s", string(val.([]byte)))
	}
}

// --- StringSliceDB.Scan ---

func TestStringSliceDB_Scan_NormalJSON(t *testing.T) {
	var s StringSliceDB
	input := []byte(`["java","c++"]`)
	if err := s.Scan(input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(s))
	}
	if s[0] != "java" || s[1] != "c++" {
		t.Errorf("unexpected elements: %v", []string(s))
	}
}

func TestStringSliceDB_Scan_NilValue(t *testing.T) {
	var s StringSliceDB
	s = append(s, "existing")
	if err := s.Scan(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Errorf("expected nil after scanning nil, got %v", []string(s))
	}
}

func TestStringSliceDB_Scan_EmptyArray(t *testing.T) {
	var s StringSliceDB
	if err := s.Scan([]byte(`[]`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", []string(s))
	}
}

func TestStringSliceDB_Scan_InvalidType(t *testing.T) {
	var s StringSliceDB
	err := s.Scan(12345) // int, not []byte
	if err == nil {
		t.Error("expected error for non-[]byte type, got nil")
	}
}

func TestStringSliceDB_Scan_InvalidJSON(t *testing.T) {
	var s StringSliceDB
	err := s.Scan([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// --- StringSliceDB round-trip ---

func TestStringSliceDB_RoundTrip(t *testing.T) {
	original := StringSliceDB{"leadership", "communication", "problem-solving"}

	// Value → JSON bytes
	val, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	// Scan ← JSON bytes
	var restored StringSliceDB
	if err := restored.Scan(val); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Compare
	if len(original) != len(restored) {
		t.Fatalf("length mismatch: %d vs %d", len(original), len(restored))
	}
	for i := range original {
		if original[i] != restored[i] {
			t.Errorf("index %d: %q vs %q", i, original[i], restored[i])
		}
	}
}

func TestStringSliceDB_RoundTrip_Empty(t *testing.T) {
	original := StringSliceDB{}
	val, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	var restored StringSliceDB
	if err := restored.Scan(val); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(restored) != 0 {
		t.Errorf("expected empty slice, got %v", []string(restored))
	}
}

// --- Verify driver.Valuer interface ---

func TestStringSliceDB_ImplementsValuer(t *testing.T) {
	var _ driver.Valuer = StringSliceDB{}
	var _ driver.Valuer = (*StringSliceDB)(nil)
}
