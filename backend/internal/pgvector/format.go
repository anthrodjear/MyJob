// Package pgvector provides utilities for working with pgvector in Go.
//
// The primary function FormatVector converts a []float32 slice to the
// pgvector literal format "[1.0,2.0,3.0,...]" used in SQL queries.
//
// Usage:
//
//	vecStr := pgvector.FormatVector([]float32{0.1, 0.2, 0.3})
//	// vecStr == "[0.100000,0.200000,0.300000]"
package pgvector

import (
	"fmt"
	"strings"
)

// FormatVector converts a float32 slice to the pgvector literal format.
// Returns "[]" for empty slices.
//
// The output format is: "[1.000000,2.000000,3.000000]"
// with 6 decimal places (matching PostgreSQL float precision).
func FormatVector(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.Grow(2 + len(vec)*10) // pre-allocate: [ + ] + each number ~10 chars
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%f", v)
	}
	b.WriteByte(']')
	return b.String()
}
