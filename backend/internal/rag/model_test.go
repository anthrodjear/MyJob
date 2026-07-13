package rag

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// SourceType
// ============================================================================

func TestSourceTypeConstants(t *testing.T) {
	assert.Equal(t, SourceType("job"), SourceTypeJob)
	assert.Equal(t, SourceType("resume"), SourceTypeResume)
	assert.Equal(t, SourceType("application"), SourceTypeApplication)
	assert.Equal(t, SourceType("cover_letter"), SourceTypeCoverLetter)
}

func TestIsValidSourceType(t *testing.T) {
	tests := []struct {
		name  string
		st    SourceType
		valid bool
	}{
		{name: "job", st: SourceTypeJob, valid: true},
		{name: "resume", st: SourceTypeResume, valid: true},
		{name: "application", st: SourceTypeApplication, valid: true},
		{name: "cover_letter", st: SourceTypeCoverLetter, valid: true},
		{name: "empty string", st: SourceType(""), valid: false},
		{name: "random value", st: SourceType("random"), valid: false},
		{name: "uppercase JOB", st: SourceType("JOB"), valid: false},
		{name: "mixed case", st: SourceType("Job"), valid: false},
		{name: "trailing space", st: SourceType("job "), valid: false},
		{name: "numeric", st: SourceType("123"), valid: false},
		{name: "undefined source_type", st: SourceType("note"), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidSourceType(tt.st))
		})
	}
}

// ============================================================================
// Metadata — Struct
// ============================================================================

func TestMetadata_Fields(t *testing.T) {
	m := Metadata{
		Title:       "Software Engineer at Google",
		ChunkIndex:  0,
		TotalChunks: 3,
		URL:         "https://careers.google.com/job/123",
	}

	assert.Equal(t, "Software Engineer at Google", m.Title)
	assert.Equal(t, 0, m.ChunkIndex)
	assert.Equal(t, 3, m.TotalChunks)
	assert.Equal(t, "https://careers.google.com/job/123", m.URL)
}

func TestMetadata_ZeroValues(t *testing.T) {
	m := Metadata{}

	assert.Empty(t, m.Title)
	assert.Zero(t, m.ChunkIndex)
	assert.Zero(t, m.TotalChunks)
	assert.Empty(t, m.URL)
}

func TestMetadata_ChunkDocuments(t *testing.T) {
	// Simulate a document split into multiple chunks.
	chunks := []Metadata{
		{Title: "Software Engineer Resume", ChunkIndex: 0, TotalChunks: 3},
		{Title: "Software Engineer Resume", ChunkIndex: 1, TotalChunks: 3},
		{Title: "Software Engineer Resume", ChunkIndex: 2, TotalChunks: 3},
	}

	for i, c := range chunks {
		assert.Equal(t, "Software Engineer Resume", c.Title)
		assert.Equal(t, i, c.ChunkIndex)
		assert.Equal(t, 3, c.TotalChunks)
	}
}

func TestMetadata_ChunkIndexOnly(t *testing.T) {
	// ChunkIndex set but TotalChunks omitted (single-chunk with index).
	m := Metadata{
		Title:      "Single Chunk Document",
		ChunkIndex: 0,
	}
	assert.Equal(t, 0, m.TotalChunks)
}

func TestMetadata_TotalChunksOnly(t *testing.T) {
	// TotalChunks set but ChunkIndex omitted (legacy or future use).
	m := Metadata{
		Title:       "Pre-chunked Document",
		TotalChunks: 5,
	}
	assert.Zero(t, m.ChunkIndex)
}

// ============================================================================
// Metadata — JSON
// ============================================================================

func TestMetadata_JSONRoundTrip(t *testing.T) {
	m := Metadata{
		Title:       "Backend Developer",
		ChunkIndex:  1,
		TotalChunks: 5,
		URL:         "https://example.com/job/456",
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var decoded Metadata
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, m, decoded)
}

func TestMetadata_JSONOmitEmpty(t *testing.T) {
	m := Metadata{}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "title")
	assert.NotContains(t, raw, "chunk_index")
	assert.NotContains(t, raw, "total_chunks")
	assert.NotContains(t, raw, "url")
	assert.Empty(t, raw)
}

func TestMetadata_JSONPartialOmit(t *testing.T) {
	m := Metadata{
		Title: "Senior Engineer",
		URL:   "https://example.com",
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "Senior Engineer", raw["title"])
	assert.Equal(t, "https://example.com", raw["url"])
	assert.NotContains(t, raw, "chunk_index")
	assert.NotContains(t, raw, "total_chunks")
}

func TestMetadata_JSONChunkFields(t *testing.T) {
	m := Metadata{
		Title:       "Chunked Doc",
		ChunkIndex:  2,
		TotalChunks: 7,
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var decoded Metadata
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, 2, decoded.ChunkIndex)
	assert.Equal(t, 7, decoded.TotalChunks)
}

// ============================================================================
// Metadata — Value / Scan (driver.Valuer & sql.Scanner)
// ============================================================================

func TestMetadata_ImplementsValuer(t *testing.T) {
	var _ driver.Valuer = (*Metadata)(nil)
}

func TestMetadata_Value_Nil(t *testing.T) {
	var m *Metadata
	v, err := m.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestMetadata_Value_EmptyStruct(t *testing.T) {
	m := &Metadata{}
	v, err := m.Value()
	require.NoError(t, err)
	assert.NotNil(t, v)

	bytes, ok := v.([]byte)
	require.True(t, ok, "expected []byte from Value()")
	assert.Equal(t, "{}", string(bytes))
}

func TestMetadata_Value_Populated(t *testing.T) {
	m := &Metadata{
		Title:       "Test Document",
		ChunkIndex:  0,
		TotalChunks: 1,
		URL:         "https://example.com/doc",
	}

	v, err := m.Value()
	require.NoError(t, err)

	bytes, ok := v.([]byte)
	require.True(t, ok, "expected []byte from Value()")

	var decoded Metadata
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)
	assert.Equal(t, *m, decoded)
}

func TestMetadata_Value_OnlyTitle(t *testing.T) {
	m := &Metadata{Title: "Only Title"}
	v, err := m.Value()
	require.NoError(t, err)

	bytes, ok := v.([]byte)
	require.True(t, ok)

	var decoded Metadata
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "Only Title", decoded.Title)
	assert.Zero(t, decoded.ChunkIndex)
	assert.Zero(t, decoded.TotalChunks)
	assert.Empty(t, decoded.URL)
}

func TestMetadata_Scan_Nil(t *testing.T) {
	var m Metadata
	err := m.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, m.Title)
	assert.Zero(t, m.ChunkIndex)
}

func TestMetadata_Scan_FromBytes(t *testing.T) {
	src := []byte(`{"title":"Test","url":"https://example.com"}`)
	var m Metadata
	err := m.Scan(src)
	require.NoError(t, err)
	assert.Equal(t, "Test", m.Title)
	assert.Equal(t, "https://example.com", m.URL)
	assert.Zero(t, m.ChunkIndex)
	assert.Zero(t, m.TotalChunks)
}

func TestMetadata_Scan_FromString(t *testing.T) {
	src := `{"title":"From String","chunk_index":1,"total_chunks":4}`
	var m Metadata
	err := m.Scan(src)
	require.NoError(t, err)
	assert.Equal(t, "From String", m.Title)
	assert.Equal(t, 1, m.ChunkIndex)
	assert.Equal(t, 4, m.TotalChunks)
}

func TestMetadata_Scan_EmptyBytes(t *testing.T) {
	var m Metadata
	err := m.Scan([]byte{})
	require.NoError(t, err)
	assert.Empty(t, m.Title)
}

func TestMetadata_Scan_EmptyString(t *testing.T) {
	var m Metadata
	err := m.Scan("")
	require.NoError(t, err)
	assert.Empty(t, m.Title)
}

func TestMetadata_Scan_AllFields(t *testing.T) {
	src := []byte(`{"title":"Full","chunk_index":2,"total_chunks":10,"url":"https://full.com"}`)
	var m Metadata
	err := m.Scan(src)
	require.NoError(t, err)
	assert.Equal(t, "Full", m.Title)
	assert.Equal(t, 2, m.ChunkIndex)
	assert.Equal(t, 10, m.TotalChunks)
	assert.Equal(t, "https://full.com", m.URL)
}

func TestMetadata_Scan_UnsupportedType(t *testing.T) {
	var m Metadata
	err := m.Scan(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestMetadata_Scan_InvalidJSON(t *testing.T) {
	var m Metadata
	err := m.Scan([]byte(`{invalid json}`))
	assert.Error(t, err)
}

func TestMetadata_Scan_IntUnsupported(t *testing.T) {
	var m Metadata
	err := m.Scan(int64(100))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestMetadata_ValueScan_RoundTrip(t *testing.T) {
	original := &Metadata{
		Title:       "Round Trip Test",
		ChunkIndex:  3,
		TotalChunks: 8,
		URL:         "https://example.com/rt",
	}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded Metadata
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, *original, decoded)
}

func TestMetadata_ValueScan_RoundTrip_NilURL(t *testing.T) {
	original := &Metadata{
		Title:       "No URL",
		ChunkIndex:  0,
		TotalChunks: 1,
	}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded Metadata
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, "No URL", decoded.Title)
	assert.Empty(t, decoded.URL)
}

// ============================================================================
// EmbeddingVector — Value / Scan (driver.Valuer & sql.Scanner)
// ============================================================================

func TestEmbeddingVector_ImplementsValuer(t *testing.T) {
	var _ driver.Valuer = EmbeddingVector{}
	var _ driver.Valuer = (*EmbeddingVector)(nil)
}

func TestEmbeddingVector_Value_Nil(t *testing.T) {
	var v EmbeddingVector
	val, err := v.Value()
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestEmbeddingVector_Value_Empty(t *testing.T) {
	v := EmbeddingVector{}
	val, err := v.Value()
	require.NoError(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, "[]", val)
}

func TestEmbeddingVector_Value_Populated(t *testing.T) {
	v := EmbeddingVector{0.1, 0.2, 0.3}
	val, err := v.Value()
	require.NoError(t, err)
	assert.Equal(t, "[0.100000,0.200000,0.300000]", val)
}

func TestEmbeddingVector_Value_SingleElement(t *testing.T) {
	v := EmbeddingVector{1.0}
	val, err := v.Value()
	require.NoError(t, err)
	assert.Equal(t, "[1.000000]", val)
}

func TestEmbeddingVector_Value_NegativeValues(t *testing.T) {
	v := EmbeddingVector{-0.5, 0.0, 0.5}
	val, err := v.Value()
	require.NoError(t, err)
	assert.Equal(t, "[-0.500000,0.000000,0.500000]", val)
}

func TestEmbeddingVector_Value_LargeValues(t *testing.T) {
	v := EmbeddingVector{999.999, -999.999}
	val, err := v.Value()
	require.NoError(t, err)
	assert.Contains(t, val.(string), "999.999")
	assert.Contains(t, val.(string), "-999.999")
}

func TestEmbeddingVector_Scan_Nil(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestEmbeddingVector_Scan_FromString(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan("[0.100000,0.200000,0.300000]")
	require.NoError(t, err)
	assert.Equal(t, EmbeddingVector{0.1, 0.2, 0.3}, v)
}

func TestEmbeddingVector_Scan_FromBytes(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan([]byte("[0.500000,0.600000]"))
	require.NoError(t, err)
	assert.Equal(t, EmbeddingVector{0.5, 0.6}, v)
}

func TestEmbeddingVector_Scan_EmptyBrackets(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan("[]")
	require.NoError(t, err)
	assert.Equal(t, EmbeddingVector{}, v)
	assert.NotNil(t, v)
}

func TestEmbeddingVector_Scan_SingleElement(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan("[0.990000]")
	require.NoError(t, err)
	assert.Equal(t, EmbeddingVector{0.99}, v)
}

func TestEmbeddingVector_Scan_NegativeValues(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan("[-0.500000,0.000000,0.500000]")
	require.NoError(t, err)
	assert.Equal(t, EmbeddingVector{-0.5, 0, 0.5}, v)
}

func TestEmbeddingVector_Scan_LargeDimension(t *testing.T) {
	// Simulate a 1536-dimension embedding vector.
	src := "["
	for i := 0; i < 1535; i++ {
		src += "0.001000,"
	}
	src += "0.001000]"

	var v EmbeddingVector
	err := v.Scan(src)
	require.NoError(t, err)
	assert.Equal(t, 1536, len(v))
	assert.InDelta(t, 0.001, v[0], 0.0001)
	assert.InDelta(t, 0.001, v[1535], 0.0001)
}

func TestEmbeddingVector_Scan_UnsupportedType(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestEmbeddingVector_Scan_InvalidFormat(t *testing.T) {
	var v EmbeddingVector
	err := v.Scan("[not a float]")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse embedding element")
}

func TestEmbeddingVector_ValueScan_RoundTrip(t *testing.T) {
	original := EmbeddingVector{0.1, 0.2, 0.3, 0.4, 0.5}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded EmbeddingVector
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestEmbeddingVector_ValueScan_RoundTrip_Empty(t *testing.T) {
	original := EmbeddingVector{}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded EmbeddingVector
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestEmbeddingVector_ValueScan_RoundTrip_Single(t *testing.T) {
	original := EmbeddingVector{3.14159}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded EmbeddingVector
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.InDelta(t, 3.14159, decoded[0], 0.0001)
}

func TestEmbeddingVector_ValueScan_RoundTrip_Negatives(t *testing.T) {
	original := EmbeddingVector{-1.0, -0.5, 0, 0.5, 1.0}

	val, err := original.Value()
	require.NoError(t, err)

	var decoded EmbeddingVector
	err = decoded.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

// ============================================================================
// Embedding
// ============================================================================

func TestEmbedding_Fields(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	meta := &Metadata{
		Title:       "Test Embedding",
		ChunkIndex:  0,
		TotalChunks: 1,
	}
	vec := EmbeddingVector{0.1, 0.2, 0.3}

	e := Embedding{
		ID:         id,
		SourceType: SourceTypeJob,
		SourceID:   sourceID,
		Content:    "Some embedded content",
		Metadata:   meta,
		Embedding:  vec,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Equal(t, id, e.ID)
	assert.Equal(t, SourceTypeJob, e.SourceType)
	assert.Equal(t, sourceID, e.SourceID)
	assert.Equal(t, "Some embedded content", e.Content)
	assert.Equal(t, meta, e.Metadata)
	assert.Equal(t, vec, e.Embedding)
	assert.Equal(t, "2026-07-13T12:00:00Z", e.CreatedAt)
}

func TestEmbedding_ZeroValues(t *testing.T) {
	e := Embedding{}

	assert.Equal(t, uuid.Nil, e.ID)
	assert.Empty(t, e.SourceType)
	assert.Equal(t, uuid.Nil, e.SourceID)
	assert.Empty(t, e.Content)
	assert.Nil(t, e.Metadata)
	assert.Nil(t, e.Embedding)
	assert.Empty(t, e.CreatedAt)
}

func TestEmbedding_NilMetadataAndEmbedding(t *testing.T) {
	e := Embedding{
		ID:         uuid.New(),
		SourceType: SourceTypeResume,
		SourceID:   uuid.New(),
		Content:    "Resume content here",
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Nil(t, e.Metadata)
	assert.Nil(t, e.Embedding)
	assert.NotEmpty(t, e.Content)
}

func TestEmbedding_EmptyEmbedding(t *testing.T) {
	e := Embedding{
		ID:        uuid.New(),
		SourceID:  uuid.New(),
		Content:   "Content",
		Embedding: EmbeddingVector{},
	}

	assert.NotNil(t, e.Embedding)
	assert.Empty(t, e.Embedding)
}

// ============================================================================
// SearchResult
// ============================================================================

func TestSearchResult_Fields(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	meta := &Metadata{Title: "Search Result Test"}
	vec := EmbeddingVector{0.5, 0.6, 0.7}

	r := SearchResult{
		Embedding: Embedding{
			ID:         id,
			SourceType: SourceTypeApplication,
			SourceID:   sourceID,
			Content:    "Search content",
			Metadata:   meta,
			Embedding:  vec,
			CreatedAt:  "2026-07-13T12:00:00Z",
		},
		Similarity: 0.95,
	}

	assert.Equal(t, id, r.ID)
	assert.Equal(t, SourceTypeApplication, r.SourceType)
	assert.Equal(t, sourceID, r.SourceID)
	assert.Equal(t, "Search content", r.Content)
	assert.Equal(t, meta, r.Metadata)
	assert.Equal(t, vec, r.Embedding.Embedding)
	assert.Equal(t, "2026-07-13T12:00:00Z", r.CreatedAt)
	assert.Equal(t, 0.95, r.Similarity)
}

func TestSearchResult_ZeroSimilarity(t *testing.T) {
	r := SearchResult{
		Embedding: Embedding{
			ID: uuid.New(),
		},
		Similarity: 0,
	}
	assert.Equal(t, 0.0, r.Similarity)
}

func TestSearchResult_MaxSimilarity(t *testing.T) {
	r := SearchResult{
		Embedding:  Embedding{ID: uuid.New()},
		Similarity: 1.0,
	}
	assert.Equal(t, 1.0, r.Similarity)
}

func TestSearchResult_NilMetadata(t *testing.T) {
	r := SearchResult{
		Embedding: Embedding{
			ID:       uuid.New(),
			Metadata: nil,
		},
		Similarity: 0.85,
	}
	assert.Nil(t, r.Metadata)
	assert.Equal(t, 0.85, r.Similarity)
}

// ============================================================================
// SearchFilter
// ============================================================================

func TestSearchFilter_Defaults(t *testing.T) {
	f := SearchFilter{}

	assert.Empty(t, f.SourceType)
	assert.Zero(t, f.Limit)
	assert.Zero(t, f.Similarity)
	assert.Nil(t, f.ExcludeSource)
}

func TestSearchFilter_FullSpec(t *testing.T) {
	excludeID := uuid.New()
	f := SearchFilter{
		SourceType: SourceTypeResume,
		Limit:      20,
		Similarity: 0.75,
		ExcludeSource: &SourceFilter{
			SourceType: SourceTypeResume,
			SourceID:   excludeID,
		},
	}

	assert.Equal(t, SourceTypeResume, f.SourceType)
	assert.Equal(t, 20, f.Limit)
	assert.Equal(t, 0.75, f.Similarity)
	require.NotNil(t, f.ExcludeSource)
	assert.Equal(t, SourceTypeResume, f.ExcludeSource.SourceType)
	assert.Equal(t, excludeID, f.ExcludeSource.SourceID)
}

func TestSearchFilter_NilExcludeSource(t *testing.T) {
	f := SearchFilter{
		SourceType:    SourceTypeJob,
		Limit:         10,
		Similarity:    0.5,
		ExcludeSource: nil,
	}

	assert.Nil(t, f.ExcludeSource)
}

func TestSearchFilter_LimitBounds(t *testing.T) {
	tests := []struct {
		name  string
		limit int
	}{
		{name: "zero", limit: 0},
		{name: "one", limit: 1},
		{name: "fifty", limit: 50},
		{name: "max int", limit: 9999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := SearchFilter{Limit: tt.limit}
			assert.Equal(t, tt.limit, f.Limit)
		})
	}
}

func TestSearchFilter_SimilarityBounds(t *testing.T) {
	tests := []struct {
		name       string
		similarity float64
	}{
		{name: "zero", similarity: 0.0},
		{name: "half", similarity: 0.5},
		{name: "one", similarity: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := SearchFilter{Similarity: tt.similarity}
			assert.Equal(t, tt.similarity, f.Similarity)
		})
	}
}

// ============================================================================
// SourceFilter
// ============================================================================

func TestSourceFilter_Fields(t *testing.T) {
	id := uuid.New()
	sf := SourceFilter{
		SourceType: SourceTypeCoverLetter,
		SourceID:   id,
	}

	assert.Equal(t, SourceTypeCoverLetter, sf.SourceType)
	assert.Equal(t, id, sf.SourceID)
}

func TestSourceFilter_ZeroValues(t *testing.T) {
	sf := SourceFilter{}

	assert.Empty(t, sf.SourceType)
	assert.Equal(t, uuid.Nil, sf.SourceID)
}

// ============================================================================
// embeddingColumns
// ============================================================================

func TestEmbeddingColumns_ContainsExpectedFields(t *testing.T) {
	assert.Contains(t, embeddingColumns, "id")
	assert.Contains(t, embeddingColumns, "source_type")
	assert.Contains(t, embeddingColumns, "source_id")
	assert.Contains(t, embeddingColumns, "content")
	assert.Contains(t, embeddingColumns, "metadata")
	assert.Contains(t, embeddingColumns, "embedding")
	assert.Contains(t, embeddingColumns, "created_at")
}
