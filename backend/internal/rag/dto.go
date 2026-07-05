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
	Query string `json:"query" binding:"required"`

	// Filter restricts the search scope.
	Filter SearchFilterRequest `json:"filter,omitempty"`
}

// SearchFilterRequest is the filter portion of SearchRequest.
// Separated for JSON binding clarity.
type SearchFilterRequest struct {
	// SourceType restricts results to a specific source type (job, resume, application, cover_letter).
	SourceType string `json:"source_type,omitempty"`

	// Limit is the maximum number of results to return. Default 10, max 50.
	Limit int `json:"limit,omitempty"`

	// Similarity is the minimum cosine similarity threshold (0.0 to 1.0).
	// Higher = more similar. Default 0.0 (no threshold).
	Similarity float64 `json:"similarity,omitempty"`

	// ExcludeSource excludes a specific embedding from results.
	// Useful when searching "similar to X" but excluding X itself.
	ExcludeSource *SourceFilterRequest `json:"exclude_source,omitempty"`
}

// SourceFilterRequest identifies a specific embedding to exclude.
type SourceFilterRequest struct {
	SourceType string `json:"source_type" binding:"required"`
	SourceID   string `json:"source_id" binding:"required"`
}

// ListFilterRequest is for GET /rag/embeddings query params.
type ListFilterRequest struct {
	SourceType string `form:"source_type"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// SearchResponse is the response for POST /rag/search.
type SearchResponse struct {
	Results []SearchResultResponse `json:"results"`
	Total   int                    `json:"total"`
	Query   string                 `json:"query"`
	Model   string                 `json:"model"`
}

// SearchResultResponse is a single search result in the API response.
type SearchResultResponse struct {
	ID         uuid.UUID `json:"id"`
	SourceType string    `json:"source_type"`
	SourceID   uuid.UUID `json:"source_id"`
	Content    string    `json:"content"`
	Metadata   *Metadata `json:"metadata,omitempty"`
	Similarity float64   `json:"similarity"`
	CreatedAt  string    `json:"created_at"`
}

// EmbeddingResponse is the API response for a single embedding.
type EmbeddingResponse struct {
	ID         uuid.UUID `json:"id"`
	SourceType string    `json:"source_type"`
	SourceID   uuid.UUID `json:"source_id"`
	Content    string    `json:"content"`
	Metadata   *Metadata `json:"metadata,omitempty"`
	CreatedAt  string    `json:"created_at"`
}

// EmbeddingListResponse is the response for GET /rag/embeddings.
type EmbeddingListResponse struct {
	Embeddings []EmbeddingResponse `json:"embeddings"`
	Total      int64               `json:"total"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
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
