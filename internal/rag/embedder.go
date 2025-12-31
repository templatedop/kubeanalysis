package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// EmbedderConfig holds configuration for the embedder.
type EmbedderConfig struct {
	// Provider specifies the embedding provider (e.g., "openai", "ollama", "local").
	Provider string `json:"provider"`

	// BaseURL is the base URL for the embedding API.
	BaseURL string `json:"base_url"`

	// APIKey is the API key for authentication.
	APIKey string `json:"api_key"`

	// Model specifies the embedding model to use.
	Model string `json:"model"`

	// Dimension specifies the embedding dimension.
	EmbeddingDimension int `json:"embedding_dimension"`

	// BatchSize specifies the batch size for embedding requests.
	BatchSize int `json:"batch_size"`

	// Timeout specifies the request timeout.
	Timeout time.Duration `json:"timeout"`
}

// DefaultEmbedderConfig returns a default embedder configuration.
func DefaultEmbedderConfig() EmbedderConfig {
	return EmbedderConfig{
		Provider:           "ollama",
		BaseURL:            "http://localhost:11434",
		Model:              "nomic-embed-text",
		EmbeddingDimension: 768,
		BatchSize:          32,
		Timeout:            30 * time.Second,
	}
}

// HTTPEmbedder implements Embedder using HTTP API calls.
type HTTPEmbedder struct {
	config     EmbedderConfig
	client     *http.Client
	cache      sync.Map
	cacheSize  int
	maxCache   int
}

// NewHTTPEmbedder creates a new HTTP-based embedder.
func NewHTTPEmbedder(config EmbedderConfig) *HTTPEmbedder {
	if config.BatchSize <= 0 {
		config.BatchSize = 32
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.EmbeddingDimension <= 0 {
		config.EmbeddingDimension = 768
	}

	return &HTTPEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		maxCache: 10000,
	}
}

// Embed generates embeddings for the given texts.
func (e *HTTPEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float64, len(texts))

	// Check cache first
	uncachedIndices := make([]int, 0)
	uncachedTexts := make([]string, 0)

	for i, text := range texts {
		if cached, ok := e.cache.Load(text); ok {
			results[i] = cached.([]float64)
		} else {
			uncachedIndices = append(uncachedIndices, i)
			uncachedTexts = append(uncachedTexts, text)
		}
	}

	if len(uncachedTexts) == 0 {
		return results, nil
	}

	// Process in batches
	for start := 0; start < len(uncachedTexts); start += e.config.BatchSize {
		end := start + e.config.BatchSize
		if end > len(uncachedTexts) {
			end = len(uncachedTexts)
		}

		batch := uncachedTexts[start:end]
		embeddings, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch: %w", err)
		}

		for i, embedding := range embeddings {
			idx := uncachedIndices[start+i]
			results[idx] = embedding
			e.cacheEmbedding(batch[i], embedding)
		}
	}

	return results, nil
}

// EmbedQuery generates an embedding optimized for queries.
func (e *HTTPEmbedder) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	embeddings, err := e.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}
	return embeddings[0], nil
}

// Dimension returns the embedding dimension.
func (e *HTTPEmbedder) Dimension() int {
	return e.config.EmbeddingDimension
}

func (e *HTTPEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	switch e.config.Provider {
	case "ollama":
		return e.embedOllama(ctx, texts)
	case "openai":
		return e.embedOpenAI(ctx, texts)
	default:
		return e.embedOllama(ctx, texts)
	}
}

func (e *HTTPEmbedder) embedOllama(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		payload := map[string]interface{}{
			"model":  e.config.Model,
			"prompt": text,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST",
			e.config.BaseURL+"/api/embeddings", strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("embedding request failed with status %d: %s",
				resp.StatusCode, string(bodyBytes))
		}

		var result struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		embeddings[i] = result.Embedding
	}

	return embeddings, nil
}

func (e *HTTPEmbedder) embedOpenAI(ctx context.Context, texts []string) ([][]float64, error) {
	payload := map[string]interface{}{
		"model": e.config.Model,
		"input": texts,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		e.config.BaseURL+"/v1/embeddings", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.config.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding request failed with status %d: %s",
			resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float64, len(texts))
	for _, data := range result.Data {
		embeddings[data.Index] = data.Embedding
	}

	return embeddings, nil
}

func (e *HTTPEmbedder) cacheEmbedding(text string, embedding []float64) {
	if e.cacheSize >= e.maxCache {
		return
	}
	e.cache.Store(text, embedding)
	e.cacheSize++
}

// LocalEmbedder provides a simple local embedding implementation for testing.
type LocalEmbedder struct {
	dimension int
}

// NewLocalEmbedder creates a new local embedder for testing.
func NewLocalEmbedder(dimension int) *LocalEmbedder {
	if dimension <= 0 {
		dimension = 768
	}
	return &LocalEmbedder{dimension: dimension}
}

// Embed generates simple hash-based embeddings for testing.
func (e *LocalEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = e.hashEmbed(text)
	}
	return embeddings, nil
}

// EmbedQuery generates an embedding for a query.
func (e *LocalEmbedder) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	return e.hashEmbed(query), nil
}

// Dimension returns the embedding dimension.
func (e *LocalEmbedder) Dimension() int {
	return e.dimension
}

// hashEmbed creates a simple hash-based embedding for testing purposes.
func (e *LocalEmbedder) hashEmbed(text string) []float64 {
	embedding := make([]float64, e.dimension)
	text = strings.ToLower(text)

	// Simple character-based hashing
	for i, char := range text {
		idx := (i * int(char)) % e.dimension
		embedding[idx] += float64(char) / 256.0
	}

	// Normalize the embedding
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}
