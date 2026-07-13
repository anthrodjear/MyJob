// DTOs (Data Transfer Types) for the RAG domain.
//
// API surface:
//   - POST   /rag/search           → Semantic search (vector similarity)
//   - GET    /rag/embeddings       → List embeddings with filters
//   - GET    /rag/embeddings/:id   → Get single embedding
//   - DELETE /rag/embeddings/:id   → Delete embedding
//
// The RAG domain provides READ access to embeddings. Generation
// happens via the worker (embedding_generate task).
package rag

import (
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// SearchRequest is the payload for POST /rag/search.
// Performs semantic search using cosine similarity on the embedding vector.
type SearchRequest struct {
	// Query is the text to search for. Will be embedded and compared against stored vectors.
	Query string `json:"query" binding:"required" example:"senior go engineer kubernetes"`

	// Filter restricts the search scope.
	Filter SearchFilterRequest `json:"filter,omitempty"`
}

// SearchFilterRequest is the filter portion of SearchRequest.
// Separated for JSON binding clarity.
type SearchFilterRequest struct {
	// SourceType restricts results to a specific source type (job, resume, application, cover_letter).
	SourceType string `json:"source_type,omitempty" example:"job" enums:"job,resume,application,cover_letter"`

	// Limit is the maximum number of results to return. Default 10, max 50.
	Limit int `json:"limit,omitempty" example:"10" minimum:"1" maximum:"50"`

	// Similarity is the minimum cosine similarity threshold (0.0 to 1.0).
	// Higher = more similar. Default 0.0 (no threshold).
	Similarity float64 `json:"similarity,omitempty" example:"0.75" minimum:"0" maximum:"1"`

	// ExcludeSource excludes a specific embedding from results.
	// Useful when searching "similar to X" but excluding X itself.
	ExcludeSource *SourceFilterRequest `json:"exclude_source,omitempty"`
}

// SourceFilterRequest identifies a specific embedding to exclude.
type SourceFilterRequest struct {
	SourceType string `json:"source_type" binding:"required" example:"job" enums:"job,resume,application,cover_letter"`
	SourceID   string `json:"source_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ListFilterRequest is for GET /rag/embeddings query params.
type ListFilterRequest struct {
	SourceType string `form:"source_type" example:"job" enums:"job,resume,application,cover_letter"`
	Limit      int    `form:"limit" example:"20" minimum:"1" maximum:"50"`
	Offset     int    `form:"offset" example:"0" minimum:"0"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// SearchResponse is the response for POST /rag/search.
type SearchResponse struct {
	Results []SearchResultResponse `json:"results"`
	Total   int                    `json:"total" example:"5"`
	Query   string                 `json:"query" example:"senior go engineer kubernetes"`
	Model   string                 `json:"model" example:"mxbai-embed-large"`
}

// SearchResultResponse is a single search result in the API response.
type SearchResultResponse struct {
	ID         uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceType string    `json:"source_type" example:"job" enums:"job,resume,application,cover_letter"`
	SourceID   uuid.UUID `json:"source_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Content    string    `json:"content" example:"Senior Go Engineer needed for..."`
	Metadata   *Metadata `json:"metadata,omitempty"`
	Similarity float64   `json:"similarity" example:"0.92" minimum:"0" maximum:"1"`
	CreatedAt  string    `json:"created_at" example:"2026-01-15T10:00:00Z"`
}

// EmbeddingResponse is the API response for a single embedding.
type EmbeddingResponse struct {
	ID         uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceType string    `json:"source_type" example:"job" enums:"job,resume,application,cover_letter"`
	SourceID   uuid.UUID `json:"source_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Content    string    `json:"content" example:"Senior Go Engineer needed..."`
	Metadata   *Metadata `json:"metadata,omitempty"`
	CreatedAt  string    `json:"created_at" example:"2026-01-15T10:00:00Z"`
}

// EmbeddingListResponse is the response for GET /rag/embeddings.
type EmbeddingListResponse struct {
	Embeddings []EmbeddingResponse `json:"embeddings"`
	Total      int64               `json:"total" example:"100"`
	Limit      int                 `json:"limit" example:"20"`
	Offset     int                 `json:"offset" example:"0"`
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

// ToSearchResultResponse converts a domain SearchResult to API response.
func ToSearchResultResponse(r *SearchResult) SearchResultResponse {
	return SearchResultResponse{
		ID:         r.ID,
		SourceType: string(r.SourceType),
		SourceID:   r.SourceID,
		Content:    r.Content,
		Metadata:   r.Metadata,
		Similarity: r.Similarity,
		CreatedAt:  r.CreatedAt,
	}
}

// ToEmbeddingResponse converts a domain Embedding to API response.
func ToEmbeddingResponse(e *Embedding) EmbeddingResponse {
	return EmbeddingResponse{
		ID:         e.ID,
		SourceType: string(e.SourceType),
		SourceID:   e.SourceID,
		Content:    e.Content,
		Metadata:   e.Metadata,
		CreatedAt:  e.CreatedAt,
	}
}
