package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EngineConfig holds configuration for the RAG engine.
type EngineConfig struct {
	// DefaultTopK is the default number of results to return.
	DefaultTopK int `json:"default_top_k"`

	// DefaultMinScore is the minimum similarity score for results.
	DefaultMinScore float64 `json:"default_min_score"`

	// MaxContextLength is the maximum length of context to return.
	MaxContextLength int `json:"max_context_length"`

	// EmbedderConfig holds embedder configuration.
	EmbedderConfig EmbedderConfig `json:"embedder_config"`

	// SplitterConfig holds splitter configuration.
	SplitterConfig SplitterConfig `json:"splitter_config"`
}

// DefaultEngineConfig returns a default engine configuration.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		DefaultTopK:      10,
		DefaultMinScore:  0.5,
		MaxContextLength: 8000,
		EmbedderConfig:   DefaultEmbedderConfig(),
		SplitterConfig:   DefaultSplitterConfig(),
	}
}

// Engine is the main RAG engine that orchestrates retrieval and context generation.
type Engine struct {
	config      EngineConfig
	embedder    Embedder
	vectorStore VectorStore
	splitter    TextSplitter
	sources     map[string]*KnowledgeSource
}

// NewEngine creates a new RAG engine.
func NewEngine(config EngineConfig, embedder Embedder, vectorStore VectorStore, splitter TextSplitter) *Engine {
	if embedder == nil {
		embedder = NewLocalEmbedder(config.EmbedderConfig.EmbeddingDimension)
	}
	if vectorStore == nil {
		vectorStore = NewInMemoryVectorStore()
	}
	if splitter == nil {
		splitter = NewRecursiveTextSplitter(config.SplitterConfig)
	}

	return &Engine{
		config:      config,
		embedder:    embedder,
		vectorStore: vectorStore,
		splitter:    splitter,
		sources:     make(map[string]*KnowledgeSource),
	}
}

// NewDefaultEngine creates a new RAG engine with default configuration.
func NewDefaultEngine() *Engine {
	config := DefaultEngineConfig()
	return NewEngine(config, nil, nil, nil)
}

// IngestDocument ingests a single document into the knowledge base.
func (e *Engine) IngestDocument(ctx context.Context, doc Document) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = time.Now()

	// Split document into chunks
	chunks, err := e.splitter.Split(doc)
	if err != nil {
		return fmt.Errorf("failed to split document: %w", err)
	}

	if len(chunks) == 0 {
		return nil
	}

	// Generate embeddings for chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := e.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Attach embeddings to chunks
	for i := range chunks {
		chunks[i].Embedding = embeddings[i]
	}

	// Store chunks in vector store
	if err := e.vectorStore.Upsert(ctx, chunks); err != nil {
		return fmt.Errorf("failed to store chunks: %w", err)
	}

	return nil
}

// IngestDocuments ingests multiple documents into the knowledge base.
func (e *Engine) IngestDocuments(ctx context.Context, docs []Document) error {
	for _, doc := range docs {
		if err := e.IngestDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to ingest document %s: %w", doc.ID, err)
		}
	}
	return nil
}

// Query retrieves relevant context for a query.
func (e *Engine) Query(ctx context.Context, query string, topK int) (*RAGContext, error) {
	startTime := time.Now()

	if topK <= 0 {
		topK = e.config.DefaultTopK
	}

	// Generate query embedding
	queryEmbedding, err := e.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search vector store
	searchQuery := SearchQuery{
		Query:    query,
		TopK:     topK,
		MinScore: e.config.DefaultMinScore,
	}

	results, err := e.vectorStore.Search(ctx, queryEmbedding, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	return &RAGContext{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// QueryWithFilters retrieves relevant context with filters.
func (e *Engine) QueryWithFilters(ctx context.Context, query string, filters map[string]string, topK int) (*RAGContext, error) {
	startTime := time.Now()

	if topK <= 0 {
		topK = e.config.DefaultTopK
	}

	// Generate query embedding
	queryEmbedding, err := e.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search vector store
	searchQuery := SearchQuery{
		Query:    query,
		TopK:     topK,
		MinScore: e.config.DefaultMinScore,
		Filters:  filters,
	}

	results, err := e.vectorStore.Search(ctx, queryEmbedding, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	return &RAGContext{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// BuildContext builds a context string from search results.
func (e *Engine) BuildContext(results []SearchResult) string {
	var builder strings.Builder
	totalLength := 0

	for i, result := range results {
		content := result.Chunk.Content
		contentLength := len(content)

		if totalLength+contentLength > e.config.MaxContextLength {
			// Truncate if necessary
			remaining := e.config.MaxContextLength - totalLength
			if remaining > 100 {
				content = content[:remaining] + "..."
				builder.WriteString(fmt.Sprintf("\n[%d] (score: %.3f, source: %s)\n%s\n",
					i+1, result.Score, result.Chunk.Metadata["source"], content))
			}
			break
		}

		builder.WriteString(fmt.Sprintf("\n[%d] (score: %.3f, source: %s)\n%s\n",
			i+1, result.Score, result.Chunk.Metadata["source"], content))
		totalLength += contentLength
	}

	return builder.String()
}

// RegisterSource registers a knowledge source.
func (e *Engine) RegisterSource(source *KnowledgeSource) {
	e.sources[source.ID] = source
}

// GetSource retrieves a knowledge source by ID.
func (e *Engine) GetSource(id string) *KnowledgeSource {
	return e.sources[id]
}

// ListSources returns all registered knowledge sources.
func (e *Engine) ListSources() []*KnowledgeSource {
	sources := make([]*KnowledgeSource, 0, len(e.sources))
	for _, source := range e.sources {
		sources = append(sources, source)
	}
	return sources
}

// Delete removes a document and its chunks from the knowledge base.
func (e *Engine) Delete(ctx context.Context, documentID string) error {
	// For now, we don't track document-to-chunk mapping
	// In a production system, this would delete all chunks for the document
	return e.vectorStore.Delete(ctx, []string{documentID})
}

// Clear removes all documents from a namespace.
func (e *Engine) Clear(ctx context.Context, namespace string) error {
	return e.vectorStore.Clear(ctx, namespace)
}

// Count returns the number of chunks in the knowledge base.
func (e *Engine) Count(ctx context.Context, namespace string) (int64, error) {
	return e.vectorStore.Count(ctx, namespace)
}

// GetEmbedder returns the embedder.
func (e *Engine) GetEmbedder() Embedder {
	return e.embedder
}

// GetVectorStore returns the vector store.
func (e *Engine) GetVectorStore() VectorStore {
	return e.vectorStore
}

// GetSplitter returns the text splitter.
func (e *Engine) GetSplitter() TextSplitter {
	return e.splitter
}
