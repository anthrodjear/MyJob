package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"backend/internal/config"
)

// EmbeddingClient defines the interface for generating embeddings.
type EmbeddingClient interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
	// ModelName returns the identifier of the embedding model.
	ModelName() string
}

// OllamaEmbeddingClient calls a local Ollama model for embedding generation.
type OllamaEmbeddingClient struct {
	logger   *zap.Logger
	baseURL  string
	model    string
	client   *http.Client
}

// NewOllamaEmbeddingClient creates a new Ollama-based embedding client.
func NewOllamaEmbeddingClient(logger *zap.Logger, baseURL, model string) *OllamaEmbeddingClient {
	return &OllamaEmbeddingClient{
		logger:  logger.Named("embeddings.ollama"),
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// ModelName returns the Ollama embedding model identifier.
func (o *OllamaEmbeddingClient) ModelName() string {
	return o.model
}

// ollamaEmbedRequest is the payload for POST /api/embeddings.
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the response from POST /api/embeddings.
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed calls the Ollama /api/embeddings endpoint and returns the embedding vector.
func (o *OllamaEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(ollamaEmbedRequest{
		Model:  o.model,
		Prompt: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(o.baseURL, "/") + "/api/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama embeddings: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit for embeddings response
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("ollama embeddings returned %d: %s", resp.StatusCode, msg)
	}

	var ollamaResp ollamaEmbedResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("unmarshal ollama embeddings response: %w", err)
	}

	if len(ollamaResp.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding")
	}

	return ollamaResp.Embedding, nil
}

// NewEmbeddingClientFromConfig creates an EmbeddingClient based on configuration.
func NewEmbeddingClientFromConfig(logger *zap.Logger, cfg config.LLMConfig) EmbeddingClient {
	if cfg.Embeddings.Provider == "ollama" && cfg.Embeddings.BaseURL != "" {
		return NewOllamaEmbeddingClient(logger, cfg.Embeddings.BaseURL, cfg.Embeddings.Model)
	}
	return NewNoopEmbeddingClient(logger)
}

// NoopEmbeddingClient is a fallback when embeddings are disabled.
type NoopEmbeddingClient struct {
	logger *zap.Logger
}

// NewNoopEmbeddingClient creates a no-op embedding client.
func NewNoopEmbeddingClient(logger *zap.Logger) *NoopEmbeddingClient {
	return &NoopEmbeddingClient{logger: logger.Named("embeddings.noop")}
}

// Embed returns nil and an error — callers must check the error before using the slice.
// Returning nil (not empty slice) prevents downstream code from treating a failed
// embedding as a valid zero-dimensional vector and crashing on pgvector inserts or
// similarity calculations.
func (n *NoopEmbeddingClient) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("embedding generation disabled")
}

// ModelName returns the model identifier.
func (n *NoopEmbeddingClient) ModelName() string {
	return "noop"
}