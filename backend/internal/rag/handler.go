// Handler handles HTTP requests for the RAG domain.
//
// API surface:
//   - POST   /rag/search           → Semantic search (vector similarity)
//   - GET    /rag/embeddings       → List embeddings with filters
//   - GET    /rag/embeddings/:id   → Get single embedding
//   - DELETE /rag/embeddings/:id   → Delete embedding
//
// This file contains NO business logic. It binds HTTP requests to
// service calls and maps domain errors to HTTP responses.
//
// The search endpoint embeds the query text via the configured embedding
// client, then performs cosine similarity against stored vectors.
package rag

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler holds the RAG HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new RAG handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("rag.handler"),
	}
}

// RegisterRoutes registers RAG routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rag := rg.Group("/rag")
	{
		rag.POST("/search", h.Search)
		rag.GET("/embeddings", h.ListEmbeddings)
		rag.GET("/embeddings/:id", h.GetEmbedding)
		rag.DELETE("/embeddings/:id", h.DeleteEmbedding)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseEmbeddingID extracts and validates the UUID from the :id path parameter.
func parseEmbeddingID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid embedding ID")
		return uuid.Nil, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// Search handles POST /rag/search.
// Embeds the query text and performs cosine similarity search.
func (h *Handler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body")
		return
	}

	// Parse filter
	var filter SearchFilter
	if req.Filter.SourceType != "" {
		if !IsValidSourceType(SourceType(req.Filter.SourceType)) {
			httpresp.BadRequest(c, "INVALID_SOURCE_TYPE", "invalid source_type filter")
			return
		}
		filter.SourceType = SourceType(req.Filter.SourceType)
	}
	if req.Filter.Limit > 0 {
		filter.Limit = req.Filter.Limit
	}
	if req.Filter.Similarity > 0 {
		filter.Similarity = req.Filter.Similarity
	}
	if req.Filter.ExcludeSource != nil {
		excludeID, err := uuid.Parse(req.Filter.ExcludeSource.SourceID)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_EXCLUDE_ID", "invalid exclude_source.source_id")
			return
		}
		if !IsValidSourceType(SourceType(req.Filter.ExcludeSource.SourceType)) {
			httpresp.BadRequest(c, "INVALID_EXCLUDE_SOURCE_TYPE", "invalid exclude_source.source_type")
			return
		}
		filter.ExcludeSource = &SourceFilter{
			SourceType: SourceType(req.Filter.ExcludeSource.SourceType),
			SourceID:   excludeID,
		}
	}

	results, model, err := h.svc.Search(c.Request.Context(), req.Query, filter)
	if err != nil {
		if errors.Is(err, ErrQueryRequired) {
			httpresp.BadRequest(c, "INVALID_INPUT", err.Error())
			return
		}
		h.logger.Error("search", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	// Convert to response DTOs
	respResults := make([]SearchResultResponse, len(results))
	for i := range results {
		respResults[i] = ToSearchResultResponse(&results[i])
	}

	httpresp.OK(c, SearchResponse{
		Results: respResults,
		Total:   len(respResults),
		Query:   req.Query,
		Model:   model,
	})
}

// ListEmbeddings handles GET /rag/embeddings.
// Query params: source_type, limit, offset.
func (h *Handler) ListEmbeddings(c *gin.Context) {
	var filter ListFilter

	if sourceType := c.Query("source_type"); sourceType != "" {
		if !IsValidSourceType(SourceType(sourceType)) {
			httpresp.BadRequest(c, "INVALID_SOURCE_TYPE", "invalid source_type filter")
			return
		}
		filter.SourceType = SourceType(sourceType)
	}

	// Pagination defaults
	filter.Limit = 50
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			httpresp.BadRequest(c, "INVALID_LIMIT", "invalid limit parameter")
			return
		}
		if limit > 50 {
			limit = 50
		}
		filter.Limit = limit
	}
	filter.Offset = 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			httpresp.BadRequest(c, "INVALID_OFFSET", "invalid offset parameter")
			return
		}
		filter.Offset = offset
	}

	embeddings, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list embeddings", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	resp := make([]EmbeddingResponse, len(embeddings))
	for i := range embeddings {
		resp[i] = ToEmbeddingResponse(&embeddings[i])
	}

	httpresp.OK(c, EmbeddingListResponse{
		Embeddings: resp,
		Total:      total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	})
}

// GetEmbedding handles GET /rag/embeddings/:id.
func (h *Handler) GetEmbedding(c *gin.Context) {
	id, ok := parseEmbeddingID(c)
	if !ok {
		return
	}

	e, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "NOT_FOUND", "embedding not found")
			return
		}
		h.logger.Error("get embedding", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ToEmbeddingResponse(e))
}

// DeleteEmbedding handles DELETE /rag/embeddings/:id.
func (h *Handler) DeleteEmbedding(c *gin.Context) {
	id, ok := parseEmbeddingID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "NOT_FOUND", "embedding not found")
			return
		}
		h.logger.Error("delete embedding", zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, gin.H{"status": "deleted"})
}
