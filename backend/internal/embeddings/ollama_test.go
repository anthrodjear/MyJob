package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"backend/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// ---------------------------------------------------------------------------
// NoopEmbeddingClient
// ---------------------------------------------------------------------------

func TestNoopEmbeddingClient_Embed(t *testing.T) {
	c := NewNoopEmbeddingClient(newTestLogger())
	vec, err := c.Embed(context.Background(), "hello")
	assert.Nil(t, vec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestNoopEmbeddingClient_EmbedBatch(t *testing.T) {
	c := NewNoopEmbeddingClient(newTestLogger())
	vecs, err := c.EmbedBatch(context.Background(), []string{"a", "b"})
	assert.Nil(t, vecs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestNoopEmbeddingClient_ModelName(t *testing.T) {
	c := NewNoopEmbeddingClient(newTestLogger())
	assert.Equal(t, "noop", c.ModelName())
}

// ---------------------------------------------------------------------------
// NewEmbeddingClientFromConfig
// ---------------------------------------------------------------------------

func TestNewEmbeddingClientFromConfig_Ollama(t *testing.T) {
	cfg := config.LLMConfig{
		Embeddings: config.LLMProvider{
			Provider: "ollama",
			BaseURL:  "http://localhost:11434",
			Model:    "nomic-embed-text",
		},
	}
	c := NewEmbeddingClientFromConfig(newTestLogger(), cfg)
	_, ok := c.(*OllamaEmbeddingClient)
	assert.True(t, ok, "expected OllamaEmbeddingClient")
}

func TestNewEmbeddingClientFromConfig_Noop(t *testing.T) {
	cfg := config.LLMConfig{
		Embeddings: config.LLMProvider{
			Provider: "none",
		},
	}
	c := NewEmbeddingClientFromConfig(newTestLogger(), cfg)
	_, ok := c.(*NoopEmbeddingClient)
	assert.True(t, ok, "expected NoopEmbeddingClient")
}

func TestNewEmbeddingClientFromConfig_OllamaMissingBaseURL(t *testing.T) {
	cfg := config.LLMConfig{
		Embeddings: config.LLMProvider{
			Provider: "ollama",
			BaseURL:  "", // missing
			Model:    "nomic-embed-text",
		},
	}
	c := NewEmbeddingClientFromConfig(newTestLogger(), cfg)
	_, ok := c.(*NoopEmbeddingClient)
	assert.True(t, ok, "expected NoopEmbeddingClient when baseURL is empty")
}

// ---------------------------------------------------------------------------
// OllamaEmbeddingClient — HTTP tests
// ---------------------------------------------------------------------------

func TestOllamaEmbeddingClient_ModelName(t *testing.T) {
	c := NewOllamaEmbeddingClient(newTestLogger(), "http://localhost:11434", "nomic-embed-text")
	assert.Equal(t, "nomic-embed-text", c.ModelName())
}

func TestOllamaEmbeddingClient_Embed_Success(t *testing.T) {
	expected := []float32{0.1, 0.2, 0.3}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/embeddings", r.URL.Path)

		var req ollamaEmbedRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, "hello world", req.Prompt)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: expected})
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vec, err := c.Embed(context.Background(), "hello world")
	require.NoError(t, err)
	assert.Equal(t, expected, vec)
}

func TestOllamaEmbeddingClient_Embed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vec, err := c.Embed(context.Background(), "hello")
	assert.Nil(t, vec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOllamaEmbeddingClient_Embed_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{}})
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vec, err := c.Embed(context.Background(), "hello")
	assert.Nil(t, vec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding")
}

func TestOllamaEmbeddingClient_Embed_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json")
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vec, err := c.Embed(context.Background(), "hello")
	assert.Nil(t, vec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestOllamaEmbeddingClient_Embed_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := NewOllamaEmbeddingClient(newTestLogger(), "http://localhost:1", "test-model")
	vec, err := c.Embed(ctx, "hello")
	assert.Nil(t, vec)
	assert.Error(t, err)
}

func TestOllamaEmbeddingClient_EmbedBatch_Success(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		var req ollamaEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Return different vectors per prompt
		var vec []float32
		switch req.Prompt {
		case "text1":
			vec = []float32{1.0, 2.0}
		case "text2":
			vec = []float32{3.0, 4.0}
		case "text3":
			vec = []float32{5.0, 6.0}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: vec})
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vecs, err := c.EmbedBatch(context.Background(), []string{"text1", "text2", "text3"})
	require.NoError(t, err)
	assert.Len(t, vecs, 3)
	assert.Equal(t, []float32{1.0, 2.0}, vecs[0])
	assert.Equal(t, []float32{3.0, 4.0}, vecs[1])
	assert.Equal(t, []float32{5.0, 6.0}, vecs[2])
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestOllamaEmbeddingClient_EmbedBatch_Empty(t *testing.T) {
	c := NewOllamaEmbeddingClient(newTestLogger(), "http://localhost:1", "test-model")
	vecs, err := c.EmbedBatch(context.Background(), []string{})
	assert.Nil(t, vecs)
	assert.NoError(t, err)
}

func TestOllamaEmbeddingClient_EmbedBatch_PartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Prompt == "bad" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{1.0}})
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	vecs, err := c.EmbedBatch(context.Background(), []string{"good", "bad", "good2"})
	assert.Nil(t, vecs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embed batch")
}

func TestOllamaEmbeddingClient_Embed_LargeResponseTruncation(t *testing.T) {
	// Test that responses > 500 bytes are truncated in error messages
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		longMsg := make([]byte, 1000)
		for i := range longMsg {
			longMsg[i] = 'x'
		}
		w.Write(longMsg)
	}))
	defer srv.Close()

	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL, "test-model")
	_, err := c.Embed(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "truncated")
}

func TestOllamaEmbeddingClient_Embed_URLTrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/embeddings", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{1.0}})
	}))
	defer srv.Close()

	// URL with trailing slash should be handled correctly
	c := NewOllamaEmbeddingClient(newTestLogger(), srv.URL+"/", "test-model")
	vec, err := c.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, []float32{1.0}, vec)
}
