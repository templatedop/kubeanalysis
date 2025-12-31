package rag

import (
	"context"
	"testing"
	"time"
)

func TestNewDefaultEngine(t *testing.T) {
	engine := NewDefaultEngine()
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	if engine.GetEmbedder() == nil {
		t.Error("expected non-nil embedder")
	}
	if engine.GetVectorStore() == nil {
		t.Error("expected non-nil vector store")
	}
	if engine.GetSplitter() == nil {
		t.Error("expected non-nil splitter")
	}
}

func TestEngineIngestDocument(t *testing.T) {
	engine := NewDefaultEngine()
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc-1",
		Content: "This is a test document about Kubernetes security best practices.",
		Metadata: map[string]string{
			"category": "security",
			"source":   "test",
		},
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := engine.IngestDocument(ctx, doc)
	if err != nil {
		t.Fatalf("failed to ingest document: %v", err)
	}

	// Verify document was stored
	count, err := engine.Count(ctx, "")
	if err != nil {
		t.Fatalf("failed to count documents: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one chunk to be stored")
	}
}

func TestEngineQuery(t *testing.T) {
	engine := NewDefaultEngine()
	ctx := context.Background()

	// Ingest test documents
	docs := []Document{
		{
			ID:      "doc-1",
			Content: "Kubernetes pods should not run as root user. Use runAsNonRoot security context.",
			Metadata: map[string]string{
				"category": "security",
				"source":   "test",
			},
		},
		{
			ID:      "doc-2",
			Content: "Resource limits should be set for all containers to prevent resource exhaustion.",
			Metadata: map[string]string{
				"category": "performance",
				"source":   "test",
			},
		},
		{
			ID:      "doc-3",
			Content: "Network policies isolate pods and control traffic flow between them.",
			Metadata: map[string]string{
				"category": "security",
				"source":   "test",
			},
		},
	}

	for _, doc := range docs {
		if err := engine.IngestDocument(ctx, doc); err != nil {
			t.Fatalf("failed to ingest document: %v", err)
		}
	}

	// Query for security-related content
	result, err := engine.Query(ctx, "kubernetes security root user", 3)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Query != "kubernetes security root user" {
		t.Errorf("expected query to be preserved, got %s", result.Query)
	}
	if result.SearchTimeMs < 0 {
		t.Error("expected non-negative search time")
	}
}

func TestEngineQueryWithFilters(t *testing.T) {
	config := DefaultEngineConfig()
	config.DefaultMinScore = 0.0 // Accept all results for testing
	engine := NewEngine(config, nil, nil, nil)
	ctx := context.Background()

	// Ingest test documents with different namespaces
	docs := []Document{
		{
			ID:      "security-doc",
			Content: "Security best practices for Kubernetes clusters.",
			Metadata: map[string]string{
				"namespace": "security",
			},
		},
		{
			ID:      "performance-doc",
			Content: "Performance optimization for Kubernetes clusters.",
			Metadata: map[string]string{
				"namespace": "performance",
			},
		},
	}

	for _, doc := range docs {
		if err := engine.IngestDocument(ctx, doc); err != nil {
			t.Fatalf("failed to ingest document: %v", err)
		}
	}

	// Query with namespace filter
	filters := map[string]string{"namespace": "security"}
	result, err := engine.QueryWithFilters(ctx, "kubernetes best practices", filters, 5)
	if err != nil {
		t.Fatalf("failed to query with filters: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify results are filtered
	for _, r := range result.Results {
		if r.Chunk.Metadata["namespace"] != "security" {
			t.Errorf("expected namespace to be 'security', got %s", r.Chunk.Metadata["namespace"])
		}
	}
}

func TestEngineBuildContext(t *testing.T) {
	engine := NewDefaultEngine()

	results := []SearchResult{
		{
			Chunk: Chunk{
				Content: "First chunk content",
				Metadata: map[string]string{
					"source": "doc1",
				},
			},
			Score: 0.95,
		},
		{
			Chunk: Chunk{
				Content: "Second chunk content",
				Metadata: map[string]string{
					"source": "doc2",
				},
			},
			Score: 0.85,
		},
	}

	context := engine.BuildContext(results)
	if context == "" {
		t.Error("expected non-empty context")
	}

	// Verify context contains chunk content
	if len(context) < len("First chunk content") {
		t.Error("expected context to contain chunk content")
	}
}

func TestEngineRegisterSource(t *testing.T) {
	engine := NewDefaultEngine()

	source := &KnowledgeSource{
		ID:          "source-1",
		Name:        "Test Source",
		Description: "A test knowledge source",
		Category:    CategorySecurity,
		Type:        "file",
	}

	engine.RegisterSource(source)

	retrieved := engine.GetSource("source-1")
	if retrieved == nil {
		t.Fatal("expected to retrieve registered source")
	}
	if retrieved.Name != "Test Source" {
		t.Errorf("expected name 'Test Source', got %s", retrieved.Name)
	}

	sources := engine.ListSources()
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(sources))
	}
}

func TestEngineClear(t *testing.T) {
	config := DefaultEngineConfig()
	config.DefaultMinScore = 0.0
	engine := NewEngine(config, nil, nil, nil)
	ctx := context.Background()

	// Ingest documents
	doc := Document{
		ID:      "test-doc",
		Content: "Test content",
		Metadata: map[string]string{
			"namespace": "test-ns",
		},
	}
	if err := engine.IngestDocument(ctx, doc); err != nil {
		t.Fatalf("failed to ingest document: %v", err)
	}

	// Clear namespace
	if err := engine.Clear(ctx, "test-ns"); err != nil {
		t.Fatalf("failed to clear namespace: %v", err)
	}

	// Verify cleared
	count, _ := engine.Count(ctx, "test-ns")
	if count != 0 {
		t.Errorf("expected count 0 after clear, got %d", count)
	}
}
