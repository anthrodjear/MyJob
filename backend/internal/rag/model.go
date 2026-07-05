// Package rag provides semantic search and embedding storage (RAG).
//
// The RAG domain exposes the embeddings table for:
//   - Semantic search (cosine similarity on embedding vectors)
//   - Embedding CRUD (list, get, delete)
//   - Embedding generation trigger (via task queue)
//
// The embeddings table schema (from 001_initial.up.sql):
//
//	CREATE TABLE embeddings (
//	  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
//	  source_type VARCHAR(50) NOT NULL,
//	  source_id UUID NOT NULL,
//	  content TEXT NOT NULL,
//	  metadata JSONB,
//	  embedding vector(1024),
//	  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	);
//	CREATE INDEX ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
//	CREATE INDEX idx_embeddings_source ON embeddings(source_type, source_id);
//
// Source types (from tasks.EmbeddingPayload):
//   - "job"          → jobs table
//   - "resume"       → resumes table
//   - "application"  → applications table
//   - "cover_letter" → cover_letters table
//
// The RAG domain does NOT generate embeddings — that happens in the
// worker via embedding_generate tasks. This domain only provides
// read/search access to the persisted embeddings.
package rag

import (
	"github.com/google/uuid"
)

// SourceType identifies the originating entity type for an embedding.
type SourceType string

const (
	SourceTypeJob         SourceType = "job"
	SourceTypeResume      SourceType = "resume"
	SourceTypeApplication SourceType = "application"
	SourceTypeCoverLetter SourceType = "cover_letter"
)

// IsValidSourceType returns true if the source type is known.
func IsValidSourceType(st SourceType) bool {
	switch st {
	case SourceTypeJob, SourceTypeResume, SourceTypeApplication, SourceTypeCoverLetter:
		return true
	default:
		return false
	}
}

// ============================================================================
// Database Row Model
// ============================================================================

// Embedding represents a stored embedding vector with metadata.
// Schema: embeddings(id, source_type, source_id, content, metadata, embedding, created_at)
type Embedding struct {
	ID         uuid.UUID  `db:"id"`
	SourceType SourceType `db:"source_type"`
	SourceID   uuid.UUID  `db:"source_id"`
	Content    string     `db:"content"`
	Metadata   *Metadata  `db:"metadata"`
	Embedding  []float32  `db:"embedding"`
	CreatedAt  string     `db:"created_at"`
}

// Metadata is the JSONB metadata stored with embeddings.
// Extensible for future use (e.g., chunk_index, document_title, etc.)
type Metadata struct {
	// Title of the source document (job title, resume name, etc.)
	Title string `json:"title,omitempty"`
	// ChunkIndex for documents split into multiple embeddings
	ChunkIndex int `json:"chunk_index,omitempty"`
	// TotalChunks for multi-chunk documents
	TotalChunks int `json:"total_chunks,omitempty"`
	// URL of the source (for jobs)
	URL string `json:"url,omitempty"`
}

// SearchResult is a single result from semantic search.
// Includes the embedding data plus the similarity score (0-1, higher = more similar).
type SearchResult struct {
	Embedding
	Similarity float64 `json:"similarity"` // cosine similarity: 1 = identical, 0 = orthogonal
}

// SearchFilter defines filters for semantic search.
type SearchFilter struct {
	SourceType    SourceType
	Limit         int           // default 10, max 50
	Similarity    float64       // minimum similarity threshold (0-1), default 0.0
	ExcludeSource *SourceFilter // exclude specific source
}

// SourceFilter identifies a specific embedding to exclude from search results.
// Useful when searching "similar to X" but excluding X itself.
type SourceFilter struct {
	SourceType SourceType
	SourceID   uuid.UUID
}

// ============================================================================
// Column List
// ============================================================================

const embeddingColumns = `
	id, source_type, source_id, content, metadata, embedding, created_at
`
