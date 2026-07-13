package rag

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// SearchRequest
// ============================================================================

func TestSearchRequest_RequiredQuery(t *testing.T) {
	req := SearchRequest{
		Query: "golang developer",
	}
	assert.Equal(t, "golang developer", req.Query)
}

func TestSearchRequest_EmptyQuery(t *testing.T) {
	req := SearchRequest{}
	assert.Empty(t, req.Query)
}

func TestSearchRequest_WithFilter(t *testing.T) {
	req := SearchRequest{
		Query: "python developer",
		Filter: SearchFilterRequest{
			SourceType: "job",
			Limit:      20,
			Similarity: 0.8,
		},
	}

	assert.Equal(t, "python developer", req.Query)
	assert.Equal(t, "job", req.Filter.SourceType)
	assert.Equal(t, 20, req.Filter.Limit)
	assert.Equal(t, 0.8, req.Filter.Similarity)
	assert.Nil(t, req.Filter.ExcludeSource)
}

func TestSearchRequest_WithExcludeSource(t *testing.T) {
	req := SearchRequest{
		Query: "similar resumes",
		Filter: SearchFilterRequest{
			SourceType: "resume",
			ExcludeSource: &SourceFilterRequest{
				SourceType: "resume",
				SourceID:   uuid.New().String(),
			},
		},
	}

	require.NotNil(t, req.Filter.ExcludeSource)
	assert.Equal(t, "resume", req.Filter.ExcludeSource.SourceType)
	assert.NotEmpty(t, req.Filter.ExcludeSource.SourceID)
}

func TestSearchRequest_JSONBinding(t *testing.T) {
	req := SearchRequest{
		Query: "full stack engineer",
		Filter: SearchFilterRequest{
			SourceType: "job",
			Limit:      10,
			Similarity: 0.7,
			ExcludeSource: &SourceFilterRequest{
				SourceType: "job",
				SourceID:   "550e8400-e29b-41d4-a716-446655440000",
			},
		},
	}

	assert.Equal(t, "full stack engineer", req.Query)
	require.NotNil(t, req.Filter.ExcludeSource)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", req.Filter.ExcludeSource.SourceID)
}

// ============================================================================
// SearchFilterRequest
// ============================================================================

func TestSearchFilterRequest_Defaults(t *testing.T) {
	f := SearchFilterRequest{}

	assert.Empty(t, f.SourceType)
	assert.Zero(t, f.Limit)
	assert.Zero(t, f.Similarity)
	assert.Nil(t, f.ExcludeSource)
}

func TestSearchFilterRequest_AllFields(t *testing.T) {
	f := SearchFilterRequest{
		SourceType: "application",
		Limit:      5,
		Similarity: 0.9,
	}

	assert.Equal(t, "application", f.SourceType)
	assert.Equal(t, 5, f.Limit)
	assert.Equal(t, 0.9, f.Similarity)
}

// ============================================================================
// SourceFilterRequest
// ============================================================================

func TestSourceFilterRequest_Fields(t *testing.T) {
	sf := SourceFilterRequest{
		SourceType: "job",
		SourceID:   uuid.New().String(),
	}

	assert.Equal(t, "job", sf.SourceType)
	assert.NotEmpty(t, sf.SourceID)
}

func TestSourceFilterRequest_ZeroValues(t *testing.T) {
	sf := SourceFilterRequest{}

	assert.Empty(t, sf.SourceType)
	assert.Empty(t, sf.SourceID)
}

// ============================================================================
// ListFilterRequest
// ============================================================================

func TestListFilterRequest_Defaults(t *testing.T) {
	lf := ListFilterRequest{}

	assert.Empty(t, lf.SourceType)
	assert.Zero(t, lf.Limit)
	assert.Zero(t, lf.Offset)
}

func TestListFilterRequest_AllFields(t *testing.T) {
	lf := ListFilterRequest{
		SourceType: "resume",
		Limit:      25,
		Offset:     10,
	}

	assert.Equal(t, "resume", lf.SourceType)
	assert.Equal(t, 25, lf.Limit)
	assert.Equal(t, 10, lf.Offset)
}

// ============================================================================
// SearchResponse
// ============================================================================

func TestSearchResponse_Fields(t *testing.T) {
	resp := SearchResponse{
		Results: []SearchResultResponse{
			{
				ID:         uuid.New(),
				SourceType: "job",
				SourceID:   uuid.New(),
				Content:    "Match 1",
				Metadata:   &Metadata{Title: "Job 1"},
				Similarity: 0.95,
				CreatedAt:  "2026-07-13T12:00:00Z",
			},
			{
				ID:         uuid.New(),
				SourceType: "job",
				SourceID:   uuid.New(),
				Content:    "Match 2",
				Similarity: 0.85,
				CreatedAt:  "2026-07-13T12:01:00Z",
			},
		},
		Total: 2,
		Query: "golang developer",
		Model: "text-embedding-ada-002",
	}

	assert.Equal(t, 2, len(resp.Results))
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "golang developer", resp.Query)
	assert.Equal(t, "text-embedding-ada-002", resp.Model)
	assert.Equal(t, 0.95, resp.Results[0].Similarity)
	assert.Equal(t, 0.85, resp.Results[1].Similarity)
	assert.Equal(t, "Match 1", resp.Results[0].Content)
	assert.Equal(t, "Match 2", resp.Results[1].Content)
}

func TestSearchResponse_EmptyResults(t *testing.T) {
	resp := SearchResponse{
		Results: []SearchResultResponse{},
		Total:   0,
		Query:   "no matches",
		Model:   "text-embedding-3-small",
	}

	assert.Empty(t, resp.Results)
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, "no matches", resp.Query)
	assert.NotEmpty(t, resp.Model)
}

func TestSearchResponse_ZeroValues(t *testing.T) {
	resp := SearchResponse{}

	assert.Nil(t, resp.Results)
	assert.Zero(t, resp.Total)
	assert.Empty(t, resp.Query)
	assert.Empty(t, resp.Model)
}

func TestSearchResponse_JSONRoundTrip(t *testing.T) {
	resp := SearchResponse{
		Results: []SearchResultResponse{
			{
				ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				SourceType: "job",
				SourceID:   uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
				Content:    "Results content",
				Metadata:   &Metadata{Title: "Result Title"},
				Similarity: 0.92,
				CreatedAt:  "2026-07-13T12:00:00Z",
			},
		},
		Total: 1,
		Query: "test query",
		Model: "test-model",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SearchResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Total, decoded.Total)
	assert.Equal(t, resp.Query, decoded.Query)
	assert.Equal(t, resp.Model, decoded.Model)
	require.Equal(t, len(resp.Results), len(decoded.Results))
	assert.Equal(t, resp.Results[0].ID, decoded.Results[0].ID)
	assert.Equal(t, resp.Results[0].SourceType, decoded.Results[0].SourceType)
	assert.Equal(t, resp.Results[0].Content, decoded.Results[0].Content)
	assert.Equal(t, resp.Results[0].Similarity, decoded.Results[0].Similarity)
	require.NotNil(t, decoded.Results[0].Metadata)
	assert.Equal(t, resp.Results[0].Metadata.Title, decoded.Results[0].Metadata.Title)
}

func TestSearchResponse_JSONEmptyResults(t *testing.T) {
	resp := SearchResponse{
		Results: []SearchResultResponse{},
		Total:   0,
		Query:   "empty",
		Model:   "model-v2",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SearchResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded.Results)
	assert.Equal(t, 0, decoded.Total)
}

// ============================================================================
// SearchResultResponse
// ============================================================================

func TestSearchResultResponse_Fields(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	meta := &Metadata{Title: "Result", ChunkIndex: 0, TotalChunks: 2}

	r := SearchResultResponse{
		ID:         id,
		SourceType: "job",
		SourceID:   sourceID,
		Content:    "Matched content",
		Metadata:   meta,
		Similarity: 0.88,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Equal(t, id, r.ID)
	assert.Equal(t, "job", r.SourceType)
	assert.Equal(t, sourceID, r.SourceID)
	assert.Equal(t, "Matched content", r.Content)
	assert.Equal(t, meta, r.Metadata)
	assert.Equal(t, 0.88, r.Similarity)
	assert.Equal(t, "2026-07-13T12:00:00Z", r.CreatedAt)
}

func TestSearchResultResponse_NilMetadata(t *testing.T) {
	r := SearchResultResponse{
		ID:         uuid.New(),
		SourceType: "resume",
		SourceID:   uuid.New(),
		Content:    "No metadata",
		Similarity: 0.75,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Nil(t, r.Metadata)
}

func TestSearchResultResponse_ZeroSimilarity(t *testing.T) {
	r := SearchResultResponse{
		ID:         uuid.New(),
		SourceType: "cover_letter",
		SourceID:   uuid.New(),
		Content:    "Zero similarity",
		Similarity: 0,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}
	assert.Equal(t, 0.0, r.Similarity)
}

func TestSearchResultResponse_MaxSimilarity(t *testing.T) {
	r := SearchResultResponse{
		ID:         uuid.New(),
		SourceType: "application",
		SourceID:   uuid.New(),
		Content:    "Perfect match",
		Similarity: 1.0,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}
	assert.Equal(t, 1.0, r.Similarity)
}

func TestSearchResultResponse_JSONOmitEmptyMetadata(t *testing.T) {
	r := SearchResultResponse{
		ID:         uuid.New(),
		SourceType: "job",
		SourceID:   uuid.New(),
		Content:    "Content",
		Similarity: 0.9,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "job", raw["source_type"])
	assert.Equal(t, 0.9, raw["similarity"])
	assert.NotContains(t, raw, "metadata")
}

// ============================================================================
// EmbeddingResponse
// ============================================================================

func TestEmbeddingResponse_Fields(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()

	e := EmbeddingResponse{
		ID:         id,
		SourceType: "job",
		SourceID:   sourceID,
		Content:    "Embedded content",
		Metadata:   &Metadata{Title: "Test", URL: "https://example.com"},
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Equal(t, id, e.ID)
	assert.Equal(t, "job", e.SourceType)
	assert.Equal(t, sourceID, e.SourceID)
	assert.Equal(t, "Embedded content", e.Content)
	require.NotNil(t, e.Metadata)
	assert.Equal(t, "Test", e.Metadata.Title)
	assert.Equal(t, "2026-07-13T12:00:00Z", e.CreatedAt)
}

func TestEmbeddingResponse_ZeroValues(t *testing.T) {
	e := EmbeddingResponse{}

	assert.Equal(t, uuid.Nil, e.ID)
	assert.Empty(t, e.SourceType)
	assert.Equal(t, uuid.Nil, e.SourceID)
	assert.Empty(t, e.Content)
	assert.Nil(t, e.Metadata)
	assert.Empty(t, e.CreatedAt)
}

func TestEmbeddingResponse_NilMetadata(t *testing.T) {
	e := EmbeddingResponse{
		ID:         uuid.New(),
		SourceType: "resume",
		SourceID:   uuid.New(),
		Content:    "Resume content",
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	assert.Nil(t, e.Metadata)
}

func TestEmbeddingResponse_JSONRoundTrip(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()

	orig := EmbeddingResponse{
		ID:         id,
		SourceType: "application",
		SourceID:   sourceID,
		Content:    "Application content",
		Metadata:   &Metadata{Title: "App", ChunkIndex: 1, TotalChunks: 3},
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded EmbeddingResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, orig.ID, decoded.ID)
	assert.Equal(t, orig.SourceType, decoded.SourceType)
	assert.Equal(t, orig.SourceID, decoded.SourceID)
	assert.Equal(t, orig.Content, decoded.Content)
	require.NotNil(t, decoded.Metadata)
	assert.Equal(t, orig.Metadata.Title, decoded.Metadata.Title)
	assert.Equal(t, orig.Metadata.ChunkIndex, decoded.Metadata.ChunkIndex)
	assert.Equal(t, orig.Metadata.TotalChunks, decoded.Metadata.TotalChunks)
	assert.Equal(t, orig.CreatedAt, decoded.CreatedAt)
}

func TestEmbeddingResponse_JSONNilMetadata(t *testing.T) {
	orig := EmbeddingResponse{
		ID:         uuid.New(),
		SourceType: "cover_letter",
		SourceID:   uuid.New(),
		Content:    "Cover letter content",
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "metadata")
}

// ============================================================================
// EmbeddingListResponse
// ============================================================================

func TestEmbeddingListResponse_Fields(t *testing.T) {
	e1 := EmbeddingResponse{
		ID:         uuid.New(),
		SourceType: "job",
		SourceID:   uuid.New(),
		Content:    "Embedding 1",
		CreatedAt:  "2026-07-13T12:00:00Z",
	}
	e2 := EmbeddingResponse{
		ID:         uuid.New(),
		SourceType: "resume",
		SourceID:   uuid.New(),
		Content:    "Embedding 2",
		CreatedAt:  "2026-07-13T12:01:00Z",
	}

	list := EmbeddingListResponse{
		Embeddings: []EmbeddingResponse{e1, e2},
		Total:      2,
		Limit:      50,
		Offset:     0,
	}

	assert.Equal(t, 2, len(list.Embeddings))
	assert.Equal(t, int64(2), list.Total)
	assert.Equal(t, 50, list.Limit)
	assert.Equal(t, 0, list.Offset)
}

func TestEmbeddingListResponse_EmptyEmbeddings(t *testing.T) {
	list := EmbeddingListResponse{
		Embeddings: []EmbeddingResponse{},
		Total:      0,
		Limit:      10,
		Offset:     0,
	}

	assert.Empty(t, list.Embeddings)
	assert.Equal(t, int64(0), list.Total)
}

func TestEmbeddingListResponse_ZeroValues(t *testing.T) {
	list := EmbeddingListResponse{}

	assert.Nil(t, list.Embeddings)
	assert.Zero(t, list.Total)
	assert.Zero(t, list.Limit)
	assert.Zero(t, list.Offset)
}

func TestEmbeddingListResponse_JSONRoundTrip(t *testing.T) {
	orig := EmbeddingListResponse{
		Embeddings: []EmbeddingResponse{
			{
				ID:         uuid.New(),
				SourceType: "job",
				SourceID:   uuid.New(),
				Content:    "Content",
				CreatedAt:  "2026-07-13T12:00:00Z",
			},
		},
		Total:  1,
		Limit:  25,
		Offset: 0,
	}

	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded EmbeddingListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, len(orig.Embeddings), len(decoded.Embeddings))
	assert.Equal(t, orig.Total, decoded.Total)
	assert.Equal(t, orig.Limit, decoded.Limit)
	assert.Equal(t, orig.Offset, decoded.Offset)
	assert.Equal(t, orig.Embeddings[0].ID, decoded.Embeddings[0].ID)
}

func TestEmbeddingListResponse_Pagination(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "first page", limit: 10, offset: 0},
		{name: "second page", limit: 10, offset: 10},
		{name: "large offset", limit: 50, offset: 500},
		{name: "single result", limit: 1, offset: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := EmbeddingListResponse{
				Embeddings: []EmbeddingResponse{},
				Total:      100,
				Limit:      tt.limit,
				Offset:     tt.offset,
			}
			assert.Equal(t, tt.limit, list.Limit)
			assert.Equal(t, tt.offset, list.Offset)
		})
	}
}

// ============================================================================
// ToSearchResultResponse — Mapper
// ============================================================================

func TestToSearchResultResponse(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	meta := &Metadata{Title: "Mapped Result", ChunkIndex: 0, TotalChunks: 1}

	sr := SearchResult{
		Embedding: Embedding{
			ID:         id,
			SourceType: SourceTypeJob,
			SourceID:   sourceID,
			Content:    "Mapped content",
			Metadata:   meta,
			CreatedAt:  "2026-07-13T12:00:00Z",
		},
		Similarity: 0.93,
	}

	resp := ToSearchResultResponse(&sr)

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "job", resp.SourceType)
	assert.Equal(t, sourceID, resp.SourceID)
	assert.Equal(t, "Mapped content", resp.Content)
	assert.Equal(t, meta, resp.Metadata)
	assert.Equal(t, 0.93, resp.Similarity)
	assert.Equal(t, "2026-07-13T12:00:00Z", resp.CreatedAt)
}

func TestToSearchResultResponse_AllSourceTypes(t *testing.T) {
	sourceTypes := []struct {
		name string
		st   SourceType
	}{
		{name: "job", st: SourceTypeJob},
		{name: "resume", st: SourceTypeResume},
		{name: "application", st: SourceTypeApplication},
		{name: "cover_letter", st: SourceTypeCoverLetter},
	}

	for _, tt := range sourceTypes {
		t.Run(tt.name, func(t *testing.T) {
			sr := SearchResult{
				Embedding: Embedding{
					ID:         uuid.New(),
					SourceType: tt.st,
					SourceID:   uuid.New(),
					Content:    "Content for " + tt.name,
					CreatedAt:  "2026-07-13T12:00:00Z",
				},
				Similarity: 0.8,
			}

			resp := ToSearchResultResponse(&sr)
			assert.Equal(t, tt.name, resp.SourceType)
		})
	}
}

func TestToSearchResultResponse_NilMetadata(t *testing.T) {
	sr := SearchResult{
		Embedding: Embedding{
			ID:         uuid.New(),
			SourceType: SourceTypeJob,
			SourceID:   uuid.New(),
			Content:    "No metadata",
			CreatedAt:  "2026-07-13T12:00:00Z",
		},
		Similarity: 0.5,
	}

	resp := ToSearchResultResponse(&sr)
	assert.Nil(t, resp.Metadata)
}

func TestToSearchResultResponse_ZeroSimilarity(t *testing.T) {
	sr := SearchResult{
		Embedding: Embedding{
			ID:         uuid.New(),
			SourceType: SourceTypeResume,
			SourceID:   uuid.New(),
			Content:    "Zero similarity",
			CreatedAt:  "2026-07-13T12:00:00Z",
		},
		Similarity: 0,
	}

	resp := ToSearchResultResponse(&sr)
	assert.Equal(t, 0.0, resp.Similarity)
}

func TestToSearchResultResponse_MaxSimilarity(t *testing.T) {
	sr := SearchResult{
		Embedding: Embedding{
			ID:         uuid.New(),
			SourceType: SourceTypeJob,
			SourceID:   uuid.New(),
			Content:    "Perfect match",
			CreatedAt:  "2026-07-13T12:00:00Z",
		},
		Similarity: 1.0,
	}

	resp := ToSearchResultResponse(&sr)
	assert.Equal(t, 1.0, resp.Similarity)
}

func TestToSearchResultResponse_ChunkMetadata(t *testing.T) {
	sr := SearchResult{
		Embedding: Embedding{
			ID:         uuid.New(),
			SourceType: SourceTypeResume,
			SourceID:   uuid.New(),
			Content:    "Chunk 2 of 5",
			Metadata: &Metadata{
				Title:       "Multi-chunk Resume",
				ChunkIndex:  2,
				TotalChunks: 5,
			},
			CreatedAt: "2026-07-13T12:00:00Z",
		},
		Similarity: 0.88,
	}

	resp := ToSearchResultResponse(&sr)
	require.NotNil(t, resp.Metadata)
	assert.Equal(t, 2, resp.Metadata.ChunkIndex)
	assert.Equal(t, 5, resp.Metadata.TotalChunks)
}

// ============================================================================
// ToEmbeddingResponse — Mapper
// ============================================================================

func TestToEmbeddingResponse(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	meta := &Metadata{Title: "Embedding Title", URL: "https://example.com"}

	e := Embedding{
		ID:         id,
		SourceType: SourceTypeJob,
		SourceID:   sourceID,
		Content:    "Original content",
		Metadata:   meta,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	resp := ToEmbeddingResponse(&e)

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "job", resp.SourceType)
	assert.Equal(t, sourceID, resp.SourceID)
	assert.Equal(t, "Original content", resp.Content)
	assert.Equal(t, meta, resp.Metadata)
	assert.Equal(t, "2026-07-13T12:00:00Z", resp.CreatedAt)
}

func TestToEmbeddingResponse_AllSourceTypes(t *testing.T) {
	sourceTypes := []struct {
		name string
		st   SourceType
	}{
		{name: "job", st: SourceTypeJob},
		{name: "resume", st: SourceTypeResume},
		{name: "application", st: SourceTypeApplication},
		{name: "cover_letter", st: SourceTypeCoverLetter},
	}

	for _, tt := range sourceTypes {
		t.Run(tt.name, func(t *testing.T) {
			e := Embedding{
				ID:         uuid.New(),
				SourceType: tt.st,
				SourceID:   uuid.New(),
				Content:    "Content",
				CreatedAt:  "2026-07-13T12:00:00Z",
			}

			resp := ToEmbeddingResponse(&e)
			assert.Equal(t, tt.name, resp.SourceType)
		})
	}
}

func TestToEmbeddingResponse_NilMetadata(t *testing.T) {
	e := Embedding{
		ID:         uuid.New(),
		SourceType: SourceTypeResume,
		SourceID:   uuid.New(),
		Content:    "No metadata",
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	resp := ToEmbeddingResponse(&e)
	assert.Nil(t, resp.Metadata)
}

func TestToEmbeddingResponse_ZeroValues(t *testing.T) {
	e := Embedding{}

	resp := ToEmbeddingResponse(&e)

	assert.Equal(t, uuid.Nil, resp.ID)
	assert.Empty(t, resp.SourceType)
	assert.Equal(t, uuid.Nil, resp.SourceID)
	assert.Empty(t, resp.Content)
	assert.Nil(t, resp.Metadata)
	assert.Empty(t, resp.CreatedAt)
}

// ============================================================================
// Cross-Domain: Embedding → SearchResult → Response mapping
// ============================================================================

func TestEmbeddingToSearchResultToResponse(t *testing.T) {
	// Simulate the full path: Embedding → SearchResult → SearchResultResponse.
	e := Embedding{
		ID:         uuid.New(),
		SourceType: SourceTypeJob,
		SourceID:   uuid.New(),
		Content:    "Full pipeline test",
		Metadata:   &Metadata{Title: "Pipeline", ChunkIndex: 0, TotalChunks: 1},
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	sr := SearchResult{
		Embedding:  e,
		Similarity: 0.91,
	}

	resp := ToSearchResultResponse(&sr)

	assert.Equal(t, e.ID, resp.ID)
	assert.Equal(t, string(e.SourceType), resp.SourceType)
	assert.Equal(t, e.SourceID, resp.SourceID)
	assert.Equal(t, e.Content, resp.Content)
	assert.Equal(t, e.Metadata, resp.Metadata)
	assert.Equal(t, sr.Similarity, resp.Similarity)
	assert.Equal(t, e.CreatedAt, resp.CreatedAt)
}

func TestEmbeddingToListResponse(t *testing.T) {
	embedding := Embedding{
		ID:         uuid.New(),
		SourceType: SourceTypeJob,
		SourceID:   uuid.New(),
		Content:    "Listing test",
		Metadata:   &Metadata{Title: "List Entry"},
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	resp := ToEmbeddingResponse(&embedding)

	listResp := EmbeddingListResponse{
		Embeddings: []EmbeddingResponse{resp},
		Total:      1,
		Limit:      10,
		Offset:     0,
	}

	assert.Equal(t, 1, len(listResp.Embeddings))
	assert.Equal(t, resp.ID, listResp.Embeddings[0].ID)
	assert.Equal(t, resp.Content, listResp.Embeddings[0].Content)
}

// ============================================================================
// JSON Tags Verification
// ============================================================================

func TestEmbeddingResponse_JSONTags(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()

	resp := EmbeddingResponse{
		ID:         id,
		SourceType: "job",
		SourceID:   sourceID,
		Content:    "Content",
		Metadata:   &Metadata{Title: "Test"},
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, id.String(), raw["id"])
	assert.Equal(t, "job", raw["source_type"])
	assert.Equal(t, sourceID.String(), raw["source_id"])
	assert.Equal(t, "Content", raw["content"])
	assert.Equal(t, "2026-07-13T12:00:00Z", raw["created_at"])
	assert.Contains(t, raw, "metadata")
}

func TestSearchResultResponse_JSONTags(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()

	resp := SearchResultResponse{
		ID:         id,
		SourceType: "job",
		SourceID:   sourceID,
		Content:    "Result",
		Similarity: 0.85,
		CreatedAt:  "2026-07-13T12:00:00Z",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, id.String(), raw["id"])
	assert.Equal(t, 0.85, raw["similarity"])
	assert.Equal(t, "job", raw["source_type"])
}
