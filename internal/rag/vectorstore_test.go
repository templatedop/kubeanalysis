package rag

import (
	"context"
	"testing"
)

func TestInMemoryVectorStoreUpsert(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []Chunk{
		{
			ID:        "chunk-1",
			DocumentID: "doc-1",
			Content:   "Test content 1",
			Embedding: []float64{0.1, 0.2, 0.3, 0.4},
			Metadata: map[string]string{
				"namespace": "default",
			},
		},
		{
			ID:        "chunk-2",
			DocumentID: "doc-1",
			Content:   "Test content 2",
			Embedding: []float64{0.5, 0.6, 0.7, 0.8},
			Metadata: map[string]string{
				"namespace": "default",
			},
		},
	}

	err := store.Upsert(ctx, chunks)
	if err != nil {
		t.Fatalf("failed to upsert chunks: %v", err)
	}

	count, err := store.Count(ctx, "")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestInMemoryVectorStoreUpsertValidation(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Test empty ID
	chunks := []Chunk{
		{
			ID:        "",
			Content:   "Test content",
			Embedding: []float64{0.1, 0.2},
		},
	}
	err := store.Upsert(ctx, chunks)
	if err == nil {
		t.Error("expected error for empty chunk ID")
	}

	// Test empty embedding
	chunks = []Chunk{
		{
			ID:        "chunk-1",
			Content:   "Test content",
			Embedding: []float64{},
		},
	}
	err = store.Upsert(ctx, chunks)
	if err == nil {
		t.Error("expected error for empty embedding")
	}
}

func TestInMemoryVectorStoreSearch(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Insert test chunks
	chunks := []Chunk{
		{
			ID:        "chunk-1",
			Content:   "Kubernetes security best practices",
			Embedding: []float64{0.9, 0.1, 0.0, 0.0},
			Metadata: map[string]string{
				"source": "doc1",
			},
		},
		{
			ID:        "chunk-2",
			Content:   "Performance optimization guide",
			Embedding: []float64{0.0, 0.0, 0.9, 0.1},
			Metadata: map[string]string{
				"source": "doc2",
			},
		},
		{
			ID:        "chunk-3",
			Content:   "Security guidelines for pods",
			Embedding: []float64{0.8, 0.2, 0.0, 0.0},
			Metadata: map[string]string{
				"source": "doc3",
			},
		},
	}

	err := store.Upsert(ctx, chunks)
	if err != nil {
		t.Fatalf("failed to upsert: %v", err)
	}

	// Search with query embedding similar to security chunks
	queryEmbedding := []float64{0.85, 0.15, 0.0, 0.0}
	query := SearchQuery{
		TopK:     2,
		MinScore: 0.0,
	}

	results, err := store.Search(ctx, queryEmbedding, query)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// First result should be most similar
	if results[0].Chunk.ID != "chunk-1" && results[0].Chunk.ID != "chunk-3" {
		t.Errorf("expected security-related chunk as first result, got %s", results[0].Chunk.ID)
	}

	// Results should be sorted by score descending
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Error("results should be sorted by score descending")
		}
	}
}

func TestInMemoryVectorStoreSearchWithFilters(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []Chunk{
		{
			ID:        "chunk-1",
			Content:   "Security content",
			Embedding: []float64{0.9, 0.1},
			Metadata: map[string]string{
				"category": "security",
			},
		},
		{
			ID:        "chunk-2",
			Content:   "Performance content",
			Embedding: []float64{0.8, 0.2},
			Metadata: map[string]string{
				"category": "performance",
			},
		},
	}

	store.Upsert(ctx, chunks)

	// Search with filter
	queryEmbedding := []float64{0.85, 0.15}
	query := SearchQuery{
		TopK:     10,
		MinScore: 0.0,
		Filters: map[string]string{
			"category": "security",
		},
	}

	results, err := store.Search(ctx, queryEmbedding, query)
	if err != nil {
		t.Fatalf("failed to search with filters: %v", err)
	}

	// Should only return security chunks
	for _, r := range results {
		if r.Chunk.Metadata["category"] != "security" {
			t.Errorf("expected category 'security', got %s", r.Chunk.Metadata["category"])
		}
	}
}

func TestInMemoryVectorStoreSearchEmptyEmbedding(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	query := SearchQuery{TopK: 10}
	_, err := store.Search(ctx, []float64{}, query)
	if err == nil {
		t.Error("expected error for empty embedding")
	}
}

func TestInMemoryVectorStoreDelete(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []Chunk{
		{
			ID:        "chunk-1",
			Content:   "Content 1",
			Embedding: []float64{0.1, 0.2},
			Metadata:  map[string]string{},
		},
		{
			ID:        "chunk-2",
			Content:   "Content 2",
			Embedding: []float64{0.3, 0.4},
			Metadata:  map[string]string{},
		},
	}

	store.Upsert(ctx, chunks)

	// Delete one chunk
	err := store.Delete(ctx, []string{"chunk-1"})
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	count, _ := store.Count(ctx, "")
	if count != 1 {
		t.Errorf("expected count 1 after delete, got %d", count)
	}
}

func TestInMemoryVectorStoreClear(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []Chunk{
		{
			ID:        "chunk-1",
			Content:   "Content 1",
			Embedding: []float64{0.1, 0.2},
			Metadata: map[string]string{
				"namespace": "ns1",
			},
		},
		{
			ID:        "chunk-2",
			Content:   "Content 2",
			Embedding: []float64{0.3, 0.4},
			Metadata: map[string]string{
				"namespace": "ns2",
			},
		},
	}

	store.Upsert(ctx, chunks)

	// Clear one namespace
	err := store.Clear(ctx, "ns1")
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	count, _ := store.Count(ctx, "ns1")
	if count != 0 {
		t.Errorf("expected count 0 for ns1 after clear, got %d", count)
	}

	count, _ = store.Count(ctx, "ns2")
	if count != 1 {
		t.Errorf("expected count 1 for ns2 after clear, got %d", count)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 0},
			b:        []float64{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float64{0, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			// Use approximate comparison for floating point
			if diff := result - tt.expected; diff > 0.0001 || diff < -0.0001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
