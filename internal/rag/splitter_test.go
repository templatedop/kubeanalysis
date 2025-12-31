package rag

import (
	"strings"
	"testing"
)

func TestRecursiveTextSplitterSplitText(t *testing.T) {
	config := SplitterConfig{
		ChunkSize:       100,
		ChunkOverlap:    20,
		Separators:      []string{"\n\n", "\n", " "},
		StripWhitespace: true,
	}
	splitter := NewRecursiveTextSplitter(config)

	text := "This is a test paragraph.\n\nThis is another paragraph that should be split."

	chunks, err := splitter.SplitText(text)
	if err != nil {
		t.Fatalf("failed to split text: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify no chunk exceeds max size (with some tolerance)
	for i, chunk := range chunks {
		if len(chunk) > config.ChunkSize+50 { // Allow some tolerance
			t.Errorf("chunk %d exceeds max size: %d > %d", i, len(chunk), config.ChunkSize)
		}
	}
}

func TestRecursiveTextSplitterSplit(t *testing.T) {
	config := DefaultSplitterConfig()
	config.ChunkSize = 200
	config.ChunkOverlap = 50
	splitter := NewRecursiveTextSplitter(config)

	doc := Document{
		ID:      "test-doc",
		Content: strings.Repeat("This is a test sentence. ", 50),
		Metadata: map[string]string{
			"source": "test",
		},
	}

	chunks, err := splitter.Split(doc)
	if err != nil {
		t.Fatalf("failed to split document: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	for i, chunk := range chunks {
		// Verify chunk has ID
		if chunk.ID == "" {
			t.Errorf("chunk %d has empty ID", i)
		}

		// Verify chunk has document ID
		if chunk.DocumentID != doc.ID {
			t.Errorf("chunk %d has wrong document ID: %s", i, chunk.DocumentID)
		}

		// Verify metadata is copied
		if chunk.Metadata["source"] != "test" {
			t.Errorf("chunk %d metadata not copied correctly", i)
		}

		// Verify content is not empty
		if chunk.Content == "" {
			t.Errorf("chunk %d has empty content", i)
		}
	}
}

func TestRecursiveTextSplitterEmptyText(t *testing.T) {
	splitter := NewRecursiveTextSplitter(DefaultSplitterConfig())

	chunks, err := splitter.SplitText("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestRecursiveTextSplitterSmallText(t *testing.T) {
	config := DefaultSplitterConfig()
	config.ChunkSize = 1000
	splitter := NewRecursiveTextSplitter(config)

	text := "Small text"
	chunks, err := splitter.SplitText(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small text, got %d", len(chunks))
	}

	if chunks[0] != text {
		t.Errorf("expected chunk content to be '%s', got '%s'", text, chunks[0])
	}
}

func TestRecursiveTextSplitterConfigDefaults(t *testing.T) {
	// Test with invalid config values
	config := SplitterConfig{
		ChunkSize:    0,  // Invalid
		ChunkOverlap: -1, // Invalid
		Separators:   nil,
	}

	splitter := NewRecursiveTextSplitter(config)

	// Should use defaults
	if splitter.config.ChunkSize != 1000 {
		t.Errorf("expected default chunk size 1000, got %d", splitter.config.ChunkSize)
	}
	if splitter.config.ChunkOverlap != 0 {
		t.Errorf("expected chunk overlap 0, got %d", splitter.config.ChunkOverlap)
	}
	if len(splitter.config.Separators) == 0 {
		t.Error("expected default separators")
	}
}

func TestRecursiveTextSplitterOverlapTooLarge(t *testing.T) {
	config := SplitterConfig{
		ChunkSize:    100,
		ChunkOverlap: 100, // Same as chunk size
	}

	splitter := NewRecursiveTextSplitter(config)

	// Overlap should be reduced
	if splitter.config.ChunkOverlap >= splitter.config.ChunkSize {
		t.Error("overlap should be less than chunk size")
	}
}

func TestSentenceSplitter(t *testing.T) {
	config := SplitterConfig{
		ChunkSize: 100,
	}
	splitter := NewSentenceSplitter(config)

	doc := Document{
		ID:      "test-doc",
		Content: "First sentence. Second sentence. Third sentence.",
	}

	chunks, err := splitter.Split(doc)
	if err != nil {
		t.Fatalf("failed to split: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	for i, chunk := range chunks {
		if chunk.ID == "" {
			t.Errorf("chunk %d has empty ID", i)
		}
		if chunk.DocumentID != doc.ID {
			t.Errorf("chunk %d has wrong document ID", i)
		}
	}
}

func TestSentenceSplitterText(t *testing.T) {
	splitter := NewSentenceSplitter(SplitterConfig{ChunkSize: 50})

	text := "Hello world. This is a test. Another sentence here."
	chunks, err := splitter.SplitText(text)
	if err != nil {
		t.Fatalf("failed to split: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Each chunk should not exceed chunk size
	for i, chunk := range chunks {
		if len(chunk) > 60 { // Allow some tolerance for sentence boundaries
			t.Errorf("chunk %d exceeds expected size: %d", i, len(chunk))
		}
	}
}

func TestCopyMetadata(t *testing.T) {
	original := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	copied := copyMetadata(original)

	// Verify copy is equal
	if len(copied) != len(original) {
		t.Error("copied metadata has different length")
	}
	for k, v := range original {
		if copied[k] != v {
			t.Errorf("copied value mismatch for key %s", k)
		}
	}

	// Verify modifying copy doesn't affect original
	copied["key1"] = "modified"
	if original["key1"] == "modified" {
		t.Error("modifying copy affected original")
	}
}

func TestCopyMetadataNil(t *testing.T) {
	copied := copyMetadata(nil)
	if copied == nil {
		t.Error("expected non-nil map for nil input")
	}
	if len(copied) != 0 {
		t.Error("expected empty map for nil input")
	}
}
