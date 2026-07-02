// Repository handles database operations for embeddings.
//
// Responsibilities:
//   - Semantic search via cosine similarity on embedding vectors
//   - CRUD operations on the embeddings table
//
// This file contains NO business logic. It translates SQL errors to
// domain errors (ErrNotFound) for the service layer.
//
// Rules followed:
//   - No SELECT * — columns listed explicitly
//   - Parameterized queries only — no string interpolation
//   - Errors wrapped with context for debugging
package rag

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Domain Errors
// ============================================================================

var (
	// ErrNotFound indicates the embedding does not exist.
	ErrNotFound = errors.New("embedding not found")
)

// ============================================================================
// Repository
// ============================================================================

// Repository handles database operations for embeddings.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new RAG repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Queries: Semantic Search
// ---------------------------------------------------------------------------

// SemanticSearch performs cosine similarity search on embedding vectors.
// Returns results ordered by similarity (highest first).
//
// Uses pgvector's <=> operator (cosine distance). The similarity is
// computed as 1 - cosine_distance.
//
// When filter.Similarity > 0, the threshold is applied in SQL (WHERE clause)
// rather than post-query in Go, so pgvector can use the index efficiently
// and the LIMIT applies to the filtered set.
func (r *Repository) SemanticSearch(ctx context.Context, queryVec string, filter SearchFilter) ([]SearchResult, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.SourceType != "" {
		conditions = append(conditions, fmt.Sprintf("source_type = $%d", argIdx))
		args = append(args, filter.SourceType)
		argIdx++
	}
	if filter.ExcludeSource != nil {
		conditions = append(conditions, fmt.Sprintf("NOT (source_type = $%d AND source_id = $%d)", argIdx, argIdx+1))
		args = append(args, filter.ExcludeSource.SourceType, filter.ExcludeSource.SourceID)
		argIdx += 2
	}

	// Apply similarity threshold in SQL when specified.
	// This lets pgvector filter before LIMIT, returning accurate result counts.
	if filter.Similarity > 0 {
		conditions = append(conditions, fmt.Sprintf("(1 - (embedding <=> $%d::vector)) >= $%d", argIdx, argIdx+1))
		args = append(args, queryVec, filter.Similarity)
		argIdx += 2
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := 10
	if filter.Limit > 0 && filter.Limit <= 50 {
		limit = filter.Limit
	}

	// Cosine similarity: 1 - (embedding <=> query)
	// <=> returns cosine distance (0 = identical, 2 = opposite)
	query := fmt.Sprintf(`
		SELECT %s, 1 - (embedding <=> $%d::vector) AS similarity
		FROM embeddings%s
		ORDER BY embedding <=> $%d::vector
		LIMIT %d
	`, embeddingColumns, argIdx, where, argIdx, limit)

	// When similarity threshold is set, the query vector is already bound at a lower
	// arg index (for the WHERE clause). We need to use the same binding for the
	// SELECT/ORDER BY. Adjust: if similarity filter is active, argIdx was incremented
	// past the vector binding, so we need to reference the earlier binding.
	// Rebuild the query to use the correct arg index for the ORDER BY.
	if filter.Similarity > 0 {
		// The vector is bound at argIdx-2 (first occurrence in the WHERE clause)
		vecArgIdx := argIdx - 2
		query = fmt.Sprintf(`
			SELECT %s, 1 - (embedding <=> $%d::vector) AS similarity
			FROM embeddings%s
			ORDER BY embedding <=> $%d::vector
			LIMIT %d
		`, embeddingColumns, vecArgIdx, where, vecArgIdx, limit)
	} else {
		args = append(args, queryVec)
	}

	var results []SearchResult
	if err := r.db.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// Queries: Read
// ---------------------------------------------------------------------------

// GetByID fetches an embedding by ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Embedding, error) {
	var e Embedding
	err := r.db.GetContext(ctx, &e,
		`SELECT `+embeddingColumns+`
		 FROM embeddings WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get embedding by id: %w", err)
	}
	return &e, nil
}

// ListFilter defines filters for listing embeddings.
type ListFilter struct {
	SourceType SourceType
	Limit      int
	Offset     int
}

// buildWhere builds the WHERE clause and args from a filter.
func (f ListFilter) buildWhere() (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if f.SourceType != "" {
		conditions = append(conditions, fmt.Sprintf("source_type = $%d", argIdx))
		args = append(args, f.SourceType)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// List returns embeddings matching the filter with total count.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Embedding, int64, error) {
	where, args := filter.buildWhere()

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM embeddings" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count embeddings: %w", err)
	}

	// Fetch page
	query := `SELECT ` + embeddingColumns + ` FROM embeddings` + where + " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var embeddings []Embedding
	if err := r.db.SelectContext(ctx, &embeddings, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list embeddings: %w", err)
	}
	return embeddings, total, nil
}

// ---------------------------------------------------------------------------
// Queries: Write
// ---------------------------------------------------------------------------

// Delete removes an embedding by ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM embeddings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
