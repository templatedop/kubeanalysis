package rag

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// SplitterConfig holds configuration for the text splitter.
type SplitterConfig struct {
	// ChunkSize is the maximum size of each chunk in characters.
	ChunkSize int `json:"chunk_size"`

	// ChunkOverlap is the number of overlapping characters between chunks.
	ChunkOverlap int `json:"chunk_overlap"`

	// Separators are the characters to split on, in order of priority.
	Separators []string `json:"separators"`

	// KeepSeparator indicates whether to keep the separator in chunks.
	KeepSeparator bool `json:"keep_separator"`

	// StripWhitespace indicates whether to strip leading/trailing whitespace.
	StripWhitespace bool `json:"strip_whitespace"`
}

// DefaultSplitterConfig returns a default splitter configuration.
func DefaultSplitterConfig() SplitterConfig {
	return SplitterConfig{
		ChunkSize:       1000,
		ChunkOverlap:    200,
		Separators:      []string{"\n\n", "\n", ". ", " ", ""},
		KeepSeparator:   true,
		StripWhitespace: true,
	}
}

// RecursiveTextSplitter implements TextSplitter using recursive character splitting.
type RecursiveTextSplitter struct {
	config SplitterConfig
}

// NewRecursiveTextSplitter creates a new recursive text splitter.
func NewRecursiveTextSplitter(config SplitterConfig) *RecursiveTextSplitter {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	if config.ChunkOverlap < 0 {
		config.ChunkOverlap = 0
	}
	if config.ChunkOverlap >= config.ChunkSize {
		config.ChunkOverlap = config.ChunkSize / 4
	}
	if len(config.Separators) == 0 {
		config.Separators = []string{"\n\n", "\n", ". ", " ", ""}
	}

	return &RecursiveTextSplitter{config: config}
}

// Split divides a document into chunks.
func (s *RecursiveTextSplitter) Split(doc Document) ([]Chunk, error) {
	texts, err := s.SplitText(doc.Content)
	if err != nil {
		return nil, err
	}

	chunks := make([]Chunk, len(texts))
	offset := 0

	for i, text := range texts {
		startIdx := strings.Index(doc.Content[offset:], text)
		if startIdx >= 0 {
			startIdx += offset
		} else {
			startIdx = offset
		}
		endIdx := startIdx + len(text)

		chunks[i] = Chunk{
			ID:         uuid.New().String(),
			DocumentID: doc.ID,
			Content:    text,
			Metadata:   copyMetadata(doc.Metadata),
			StartIndex: startIdx,
			EndIndex:   endIdx,
		}

		offset = endIdx - s.config.ChunkOverlap
		if offset < 0 {
			offset = 0
		}
	}

	return chunks, nil
}

// SplitText divides text content into chunks.
func (s *RecursiveTextSplitter) SplitText(text string) ([]string, error) {
	return s.splitTextRecursive(text, s.config.Separators), nil
}

func (s *RecursiveTextSplitter) splitTextRecursive(text string, separators []string) []string {
	var finalChunks []string

	// Find the best separator
	separator := separators[len(separators)-1]
	newSeparators := []string{}

	for i, sep := range separators {
		if sep == "" || strings.Contains(text, sep) {
			separator = sep
			newSeparators = separators[i+1:]
			break
		}
	}

	// Split by the separator
	var splits []string
	if separator == "" {
		splits = splitByChar(text)
	} else {
		splits = strings.Split(text, separator)
	}

	// Process splits
	goodSplits := []string{}
	separatorToUse := separator
	if !s.config.KeepSeparator {
		separatorToUse = ""
	}

	for _, split := range splits {
		if s.config.StripWhitespace {
			split = strings.TrimSpace(split)
		}
		if split == "" {
			continue
		}

		if len(split) < s.config.ChunkSize {
			goodSplits = append(goodSplits, split)
		} else if len(goodSplits) > 0 {
			merged := s.mergeSplits(goodSplits, separatorToUse)
			finalChunks = append(finalChunks, merged...)
			goodSplits = []string{}

			// Recursively split the large chunk
			if len(newSeparators) > 0 {
				otherChunks := s.splitTextRecursive(split, newSeparators)
				finalChunks = append(finalChunks, otherChunks...)
			} else {
				finalChunks = append(finalChunks, split)
			}
		} else {
			// Recursively split
			if len(newSeparators) > 0 {
				otherChunks := s.splitTextRecursive(split, newSeparators)
				finalChunks = append(finalChunks, otherChunks...)
			} else {
				finalChunks = append(finalChunks, split)
			}
		}
	}

	// Handle remaining good splits
	if len(goodSplits) > 0 {
		merged := s.mergeSplits(goodSplits, separatorToUse)
		finalChunks = append(finalChunks, merged...)
	}

	return finalChunks
}

func (s *RecursiveTextSplitter) mergeSplits(splits []string, separator string) []string {
	var docs []string
	var currentDoc strings.Builder
	var currentLength int

	for _, split := range splits {
		splitLen := len(split)

		if currentLength+splitLen+(len(separator)*boolToInt(currentLength > 0)) > s.config.ChunkSize {
			if currentLength > 0 {
				doc := currentDoc.String()
				if s.config.StripWhitespace {
					doc = strings.TrimSpace(doc)
				}
				if doc != "" {
					docs = append(docs, doc)
				}

				// Handle overlap
				if s.config.ChunkOverlap > 0 {
					currentDoc.Reset()
					currentLength = 0
					// Add overlap from previous content
					overlapStart := len(doc) - s.config.ChunkOverlap
					if overlapStart > 0 {
						currentDoc.WriteString(doc[overlapStart:])
						currentLength = s.config.ChunkOverlap
					}
				} else {
					currentDoc.Reset()
					currentLength = 0
				}
			}
		}

		if currentLength > 0 {
			currentDoc.WriteString(separator)
			currentLength += len(separator)
		}
		currentDoc.WriteString(split)
		currentLength += splitLen
	}

	if currentLength > 0 {
		doc := currentDoc.String()
		if s.config.StripWhitespace {
			doc = strings.TrimSpace(doc)
		}
		if doc != "" {
			docs = append(docs, doc)
		}
	}

	return docs
}

func splitByChar(text string) []string {
	var result []string
	for _, r := range text {
		result = append(result, string(r))
	}
	return result
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func copyMetadata(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// SentenceSplitter splits text by sentences.
type SentenceSplitter struct {
	config SplitterConfig
}

// NewSentenceSplitter creates a new sentence-based splitter.
func NewSentenceSplitter(config SplitterConfig) *SentenceSplitter {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	return &SentenceSplitter{config: config}
}

// Split divides a document into chunks.
func (s *SentenceSplitter) Split(doc Document) ([]Chunk, error) {
	texts, err := s.SplitText(doc.Content)
	if err != nil {
		return nil, err
	}

	chunks := make([]Chunk, len(texts))
	offset := 0

	for i, text := range texts {
		chunks[i] = Chunk{
			ID:         uuid.New().String(),
			DocumentID: doc.ID,
			Content:    text,
			Metadata:   copyMetadata(doc.Metadata),
			StartIndex: offset,
			EndIndex:   offset + len(text),
		}
		offset += len(text)
	}

	return chunks, nil
}

// SplitText divides text into sentence-based chunks.
func (s *SentenceSplitter) SplitText(text string) ([]string, error) {
	sentences := splitSentences(text)

	var chunks []string
	var currentChunk strings.Builder

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		if currentChunk.Len()+len(sentence)+1 > s.config.ChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks, nil
}

func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	runes := []rune(text)

	for i, r := range runes {
		current.WriteRune(r)

		// Check for sentence end
		if r == '.' || r == '!' || r == '?' {
			// Look ahead for space or end
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
				sentences = append(sentences, current.String())
				current.Reset()
			}
		}
	}

	// Don't forget remaining text
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}
