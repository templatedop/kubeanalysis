// Package rag provides Retrieval-Augmented Generation functionality
// for enhancing LLM responses with relevant context from knowledge bases.
package rag

import (
	"context"
	"time"
)

// Document represents a document in the knowledge base.
type Document struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	Embedding []float64         `json:"embedding,omitempty"`
	Source    string            `json:"source"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Chunk represents a chunk of a document after splitting.
type Chunk struct {
	ID         string            `json:"id"`
	DocumentID string            `json:"document_id"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	Embedding  []float64         `json:"embedding,omitempty"`
	StartIndex int               `json:"start_index"`
	EndIndex   int               `json:"end_index"`
}

// SearchResult represents a search result from the vector store.
type SearchResult struct {
	Chunk      Chunk   `json:"chunk"`
	Score      float64 `json:"score"`
	Distance   float64 `json:"distance"`
	Highlights []string `json:"highlights,omitempty"`
}

// SearchQuery represents a search query configuration.
type SearchQuery struct {
	Query      string            `json:"query"`
	TopK       int               `json:"top_k"`
	MinScore   float64           `json:"min_score"`
	Filters    map[string]string `json:"filters,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
}

// RAGContext represents the context retrieved for augmenting LLM prompts.
type RAGContext struct {
	Query        string         `json:"query"`
	Results      []SearchResult `json:"results"`
	TotalResults int            `json:"total_results"`
	SearchTimeMs int64          `json:"search_time_ms"`
}

// Embedder provides embedding generation functionality.
type Embedder interface {
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float64, error)

	// EmbedQuery generates an embedding optimized for queries.
	EmbedQuery(ctx context.Context, query string) ([]float64, error)

	// Dimension returns the embedding dimension.
	Dimension() int
}

// VectorStore provides vector storage and retrieval functionality.
type VectorStore interface {
	// Upsert adds or updates chunks in the store.
	Upsert(ctx context.Context, chunks []Chunk) error

	// Search finds similar chunks based on the query embedding.
	Search(ctx context.Context, embedding []float64, query SearchQuery) ([]SearchResult, error)

	// Delete removes chunks by their IDs.
	Delete(ctx context.Context, ids []string) error

	// Clear removes all chunks from the specified namespace.
	Clear(ctx context.Context, namespace string) error

	// Count returns the number of chunks in the store.
	Count(ctx context.Context, namespace string) (int64, error)
}

// TextSplitter provides text splitting functionality for chunking documents.
type TextSplitter interface {
	// Split divides a document into chunks.
	Split(doc Document) ([]Chunk, error)

	// SplitText divides text content into chunks.
	SplitText(text string) ([]string, error)
}

// DocumentLoader provides document loading functionality.
type DocumentLoader interface {
	// Load loads documents from the source.
	Load(ctx context.Context, source string) ([]Document, error)

	// LoadBatch loads multiple documents from sources.
	LoadBatch(ctx context.Context, sources []string) ([]Document, error)
}

// KnowledgeCategory represents a category of knowledge for Kubernetes analysis.
type KnowledgeCategory string

const (
	CategorySecurity     KnowledgeCategory = "security"
	CategoryPerformance  KnowledgeCategory = "performance"
	CategoryBestPractice KnowledgeCategory = "best_practice"
	CategoryIncident     KnowledgeCategory = "incident"
	CategoryRunbook      KnowledgeCategory = "runbook"
)

// KnowledgeSource represents a source of knowledge for the RAG system.
type KnowledgeSource struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    KnowledgeCategory `json:"category"`
	URL         string            `json:"url,omitempty"`
	Type        string            `json:"type"` // "file", "url", "api"
	LastSync    time.Time         `json:"last_sync"`
	Documents   int               `json:"documents"`
}
