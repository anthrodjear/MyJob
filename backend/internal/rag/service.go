// Service handles RAG business logic.
//
// Responsibilities:
//   - Semantic search: embed query text → cosine similarity → filtered results
//   - Embedding CRUD: list, get, delete embeddings
//
// This file contains NO HTTP handlers, NO database queries.
// It orchestrates repository and embedding client calls.
//
// The embedding generation (write) path lives in the worker, not here.
// This service provides READ access and search for the API layer.
//
// Error handling:
//   - Returns domain errors (ErrNotFound)
//   - Wraps unexpected errors with context
//   - Never logs and returns the same error (handler decides to log)
package rag

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/pgvector"
)

// ============================================================================
// Interfaces
// ============================================================================

// RepositoryInterface defines the contract for embedding data access.
type RepositoryInterface interface {
	SemanticSearch(ctx context.Context, queryVec string, filter SearchFilter) ([]SearchResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Embedding, error)
	List(ctx context.Context, filter ListFilter) ([]Embedding, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EmbeddingClientInterface defines the contract for embedding generation.
// The service uses this to embed the search query text before comparison.
type EmbeddingClientInterface interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	ModelName() string
}

// ============================================================================
// Domain Errors
// ============================================================================

var (
	// ErrQueryRequired indicates the search query was empty.
	ErrQueryRequired = errors.New("search: query is required")
)

// ============================================================================

// Service handles RAG business logic.
type Service struct {
	repo            RepositoryInterface
	embeddingClient EmbeddingClientInterface
	logger          *zap.Logger
}

// NewService creates a new RAG service.
func NewService(repo RepositoryInterface, embeddingClient EmbeddingClientInterface, logger *zap.Logger) *Service {
	return &Service{
		repo:            repo,
		embeddingClient: embeddingClient,
		logger:          logger.Named("rag"),
	}
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// GetByID returns a single embedding by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Embedding, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get embedding: %w", err)
	}
	return e, nil
}

// List returns embeddings matching the filter with total count.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]Embedding, int64, error) {
	embeddings, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list embeddings: %w", err)
	}
	return embeddings, total, nil
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

// Delete removes an embedding by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}
	s.logger.Info("embedding deleted", zap.String("id", id.String()))
	return nil
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// Search performs semantic search over stored embeddings.
//
// Flow:
//  1. Embed the query text using the configured embedding client
//  2. Convert the vector to pgvector format
//  3. Call repository.CosineSearch for vector similarity
//  4. Apply post-query filters (similarity threshold)
//
// Returns results ordered by similarity (highest first).
func (s *Service) Search(ctx context.Context, query string, filter SearchFilter) ([]SearchResult, string, error) {
	if query == "" {
		return nil, "", ErrQueryRequired
	}

	// Embed the query text
	queryVec, err := s.embeddingClient.Embed(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("search embed query: %w", err)
	}

	// Convert to pgvector string format
	vecStr := pgvector.FormatVector(queryVec)

	// Normalize filter defaults
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 10
	}

	// Execute semantic search
	results, err := s.repo.SemanticSearch(ctx, vecStr, filter)
	if err != nil {
		return nil, "", fmt.Errorf("search: %w", err)
	}

	s.logger.Debug("semantic search complete",
		zap.String("query", query),
		zap.Int("results", len(results)),
		zap.String("model", s.embeddingClient.ModelName()),
	)

	return results, s.embeddingClient.ModelName(), nil
}
