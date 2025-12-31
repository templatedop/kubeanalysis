package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
)

// InMemoryVectorStore provides an in-memory vector store implementation.
type InMemoryVectorStore struct {
	mu         sync.RWMutex
	chunks     map[string]Chunk
	namespaces map[string]map[string]struct{}
}

// NewInMemoryVectorStore creates a new in-memory vector store.
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		chunks:     make(map[string]Chunk),
		namespaces: make(map[string]map[string]struct{}),
	}
}

// Upsert adds or updates chunks in the store.
func (s *InMemoryVectorStore) Upsert(ctx context.Context, chunks []Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chunk := range chunks {
		if chunk.ID == "" {
			return fmt.Errorf("chunk ID cannot be empty")
		}
		if len(chunk.Embedding) == 0 {
			return fmt.Errorf("chunk embedding cannot be empty for chunk %s", chunk.ID)
		}

		s.chunks[chunk.ID] = chunk

		// Track namespace
		namespace := chunk.Metadata["namespace"]
		if namespace == "" {
			namespace = "default"
		}
		if s.namespaces[namespace] == nil {
			s.namespaces[namespace] = make(map[string]struct{})
		}
		s.namespaces[namespace][chunk.ID] = struct{}{}
	}

	return nil
}

// Search finds similar chunks based on the query embedding.
func (s *InMemoryVectorStore) Search(ctx context.Context, embedding []float64, query SearchQuery) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(embedding) == 0 {
		return nil, fmt.Errorf("query embedding cannot be empty")
	}

	topK := query.TopK
	if topK <= 0 {
		topK = 10
	}

	var results []SearchResult

	for _, chunk := range s.chunks {
		// Apply namespace filter
		if query.Namespace != "" {
			if ns, ok := chunk.Metadata["namespace"]; ok && ns != query.Namespace {
				continue
			}
		}

		// Apply metadata filters
		if len(query.Filters) > 0 {
			match := true
			for key, value := range query.Filters {
				if chunk.Metadata[key] != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		// Calculate cosine similarity
		score := cosineSimilarity(embedding, chunk.Embedding)
		distance := 1 - score

		if score >= query.MinScore {
			results = append(results, SearchResult{
				Chunk:    chunk,
				Score:    score,
				Distance: distance,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Delete removes chunks by their IDs.
func (s *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		if chunk, ok := s.chunks[id]; ok {
			namespace := chunk.Metadata["namespace"]
			if namespace == "" {
				namespace = "default"
			}
			delete(s.namespaces[namespace], id)
			delete(s.chunks, id)
		}
	}

	return nil
}

// Clear removes all chunks from the specified namespace.
func (s *InMemoryVectorStore) Clear(ctx context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	if ids, ok := s.namespaces[namespace]; ok {
		for id := range ids {
			delete(s.chunks, id)
		}
		delete(s.namespaces, namespace)
	}

	return nil
}

// Count returns the number of chunks in the store.
func (s *InMemoryVectorStore) Count(ctx context.Context, namespace string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if namespace == "" {
		return int64(len(s.chunks)), nil
	}

	if ids, ok := s.namespaces[namespace]; ok {
		return int64(len(ids)), nil
	}

	return 0, nil
}

// GetChunk retrieves a chunk by ID.
func (s *InMemoryVectorStore) GetChunk(ctx context.Context, id string) (*Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if chunk, ok := s.chunks[id]; ok {
		return &chunk, nil
	}
	return nil, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// euclideanDistance calculates the Euclidean distance between two vectors.
func euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}
