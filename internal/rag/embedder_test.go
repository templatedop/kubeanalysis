package rag

import (
	"context"
	"math"
	"testing"
)

func TestLocalEmbedderEmbed(t *testing.T) {
	embedder := NewLocalEmbedder(768)
	ctx := context.Background()

	texts := []string{
		"Hello world",
		"Kubernetes security",
		"Another text",
	}

	embeddings, err := embedder.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("failed to embed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, embedding := range embeddings {
		if len(embedding) != 768 {
			t.Errorf("embedding %d has wrong dimension: %d", i, len(embedding))
		}
	}
}

func TestLocalEmbedderEmbedQuery(t *testing.T) {
	embedder := NewLocalEmbedder(512)
	ctx := context.Background()

	embedding, err := embedder.EmbedQuery(ctx, "test query")
	if err != nil {
		t.Fatalf("failed to embed query: %v", err)
	}

	if len(embedding) != 512 {
		t.Errorf("expected dimension 512, got %d", len(embedding))
	}
}

func TestLocalEmbedderDimension(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{768, 768},
		{512, 512},
		{0, 768},   // Default
		{-1, 768},  // Default
	}

	for _, tt := range tests {
		embedder := NewLocalEmbedder(tt.input)
		if embedder.Dimension() != tt.expected {
			t.Errorf("input %d: expected dimension %d, got %d", tt.input, tt.expected, embedder.Dimension())
		}
	}
}

func TestLocalEmbedderConsistency(t *testing.T) {
	embedder := NewLocalEmbedder(256)
	ctx := context.Background()

	text := "consistent text"

	// Embed the same text twice
	emb1, _ := embedder.EmbedQuery(ctx, text)
	emb2, _ := embedder.EmbedQuery(ctx, text)

	// Should produce identical embeddings
	for i := range emb1 {
		if emb1[i] != emb2[i] {
			t.Errorf("embedding not consistent at index %d", i)
			break
		}
	}
}

func TestLocalEmbedderNormalization(t *testing.T) {
	embedder := NewLocalEmbedder(128)
	ctx := context.Background()

	embedding, _ := embedder.EmbedQuery(ctx, "test normalization")

	// Calculate L2 norm
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	// Normalized vector should have norm close to 1
	if norm < 0.99 || norm > 1.01 {
		t.Errorf("expected normalized embedding with norm ~1, got %f", norm)
	}
}

func TestLocalEmbedderEmptyTexts(t *testing.T) {
	embedder := NewLocalEmbedder(256)
	ctx := context.Background()

	embeddings, err := embedder.Embed(ctx, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embeddings) != 0 {
		t.Errorf("expected 0 embeddings for empty input, got %d", len(embeddings))
	}
}

func TestHTTPEmbedderConfig(t *testing.T) {
	config := DefaultEmbedderConfig()

	if config.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %s", config.Provider)
	}
	if config.EmbeddingDimension != 768 {
		t.Errorf("expected dimension 768, got %d", config.EmbeddingDimension)
	}
	if config.BatchSize != 32 {
		t.Errorf("expected batch size 32, got %d", config.BatchSize)
	}
}

func TestHTTPEmbedderDimension(t *testing.T) {
	config := EmbedderConfig{
		EmbeddingDimension: 1024,
	}
	embedder := NewHTTPEmbedder(config)

	if embedder.Dimension() != 1024 {
		t.Errorf("expected dimension 1024, got %d", embedder.Dimension())
	}
}

func TestHTTPEmbedderConfigDefaults(t *testing.T) {
	config := EmbedderConfig{
		// Leave all fields empty
	}
	embedder := NewHTTPEmbedder(config)

	if embedder.config.BatchSize != 32 {
		t.Errorf("expected default batch size 32, got %d", embedder.config.BatchSize)
	}
	if embedder.config.EmbeddingDimension != 768 {
		t.Errorf("expected default dimension 768, got %d", embedder.config.EmbeddingDimension)
	}
}

func TestLocalEmbedderSimilarTexts(t *testing.T) {
	embedder := NewLocalEmbedder(256)
	ctx := context.Background()

	// Similar texts should have similar embeddings
	texts := []string{
		"kubernetes security",
		"k8s security",
		"database management", // Different topic
	}

	embeddings, _ := embedder.Embed(ctx, texts)

	// Calculate similarity between first two (similar topic)
	sim12 := cosineSimilarity(embeddings[0], embeddings[1])

	// Calculate similarity between first and third (different topic)
	sim13 := cosineSimilarity(embeddings[0], embeddings[2])

	// Note: For the simple hash-based embedder, we can't guarantee
	// semantic similarity, but we can at least test the function works
	t.Logf("Similarity between 'kubernetes security' and 'k8s security': %f", sim12)
	t.Logf("Similarity between 'kubernetes security' and 'database management': %f", sim13)
}
