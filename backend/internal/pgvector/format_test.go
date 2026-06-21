package pgvector

import (
	"strings"
	"testing"
)

func TestFormatVector_NormalVector(t *testing.T) {
	result := FormatVector([]float32{0.1, 0.2, 0.3})
	expected := "[0.100000,0.200000,0.300000]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatVector_EmptySlice(t *testing.T) {
	result := FormatVector([]float32{})
	if result != "[]" {
		t.Errorf("expected '[]', got %q", result)
	}
}

func TestFormatVector_NilSlice(t *testing.T) {
	result := FormatVector(nil)
	if result != "[]" {
		t.Errorf("expected '[]' for nil, got %q", result)
	}
}

func TestFormatVector_SingleElement(t *testing.T) {
	result := FormatVector([]float32{1.0})
	expected := "[1.000000]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatVector_StartsWithBracket(t *testing.T) {
	result := FormatVector([]float32{1.0, 2.0})
	if !strings.HasPrefix(result, "[") {
		t.Errorf("expected string to start with '[', got %q", result)
	}
}

func TestFormatVector_EndsWithBracket(t *testing.T) {
	result := FormatVector([]float32{1.0, 2.0})
	if !strings.HasSuffix(result, "]") {
		t.Errorf("expected string to end with ']', got %q", result)
	}
}

func TestFormatVector_CommaSeparated(t *testing.T) {
	result := FormatVector([]float32{1.0, 2.0, 3.0, 4.0})
	parts := strings.Split(strings.Trim(result, "[]"), ",")
	if len(parts) != 4 {
		t.Errorf("expected 4 comma-separated values, got %d in %q", len(parts), result)
	}
}

func TestFormatVector_SixDecimalPlaces(t *testing.T) {
	result := FormatVector([]float32{1.0})
	// Trim brackets and check the number has exactly 6 decimal places
	num := strings.Trim(result, "[]")
	parts := strings.Split(num, ".")
	if len(parts) != 2 {
		t.Fatalf("expected number with decimal point, got %q", num)
	}
	decimals := parts[1]
	if len(decimals) != 6 {
		t.Errorf("expected 6 decimal places, got %d (%q)", len(decimals), decimals)
	}
}

func TestFormatVector_LargeDimension(t *testing.T) {
	// Simulate an embedding vector (1536 dimensions like OpenAI text-embedding-ada-002)
	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = float32(i) * 0.001
	}
	result := FormatVector(vec)

	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Errorf("expected brackets, got %q...", result[:50])
	}

	// Count commas: should be 1535 (N-1 for N elements)
	inner := strings.Trim(result, "[]")
	commaCount := strings.Count(inner, ",")
	if commaCount != 1535 {
		t.Errorf("expected 1535 commas for 1536 elements, got %d", commaCount)
	}
}

func TestFormatVector_NegativeValues(t *testing.T) {
	result := FormatVector([]float32{-1.5, 0.0, 2.5})
	expected := "[-1.500000,0.000000,2.500000]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatVector_VerySmallValues(t *testing.T) {
	result := FormatVector([]float32{0.000001})
	// float32 has ~7 decimal digits of precision; 0.000001 may round
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Errorf("expected brackets, got %q", result)
	}
}
