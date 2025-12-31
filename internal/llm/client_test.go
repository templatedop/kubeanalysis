package llm

import (
	"context"
	"testing"
)

func TestMockClient(t *testing.T) {
	response := "This is a mock response"
	client := NewMockClient(response)
	ctx := context.Background()

	req := CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != response {
		t.Errorf("expected content '%s', got '%s'", response, resp.Content)
	}
	if resp.ID != "mock-response" {
		t.Errorf("expected ID 'mock-response', got '%s'", resp.ID)
	}
}

func TestMockClientChat(t *testing.T) {
	response := "Chat response"
	client := NewMockClient(response)
	ctx := context.Background()

	messages := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
	}

	resp, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != response {
		t.Errorf("expected content '%s', got '%s'", response, resp.Content)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %s", config.Provider)
	}
	if config.BaseURL != "http://localhost:11434" {
		t.Errorf("expected base URL 'http://localhost:11434', got %s", config.BaseURL)
	}
	if config.MaxTokens != 4096 {
		t.Errorf("expected max tokens 4096, got %d", config.MaxTokens)
	}
	if config.Temperature != 0.1 {
		t.Errorf("expected temperature 0.1, got %f", config.Temperature)
	}
}

func TestHTTPClientConfigDefaults(t *testing.T) {
	config := ClientConfig{
		// Empty config
	}
	client := NewHTTPClient(config)

	if client.config.MaxTokens != 4096 {
		t.Errorf("expected default max tokens 4096, got %d", client.config.MaxTokens)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "generic code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "with surrounding text",
			input:    "Here is the response: {\"key\": \"value\"} and more text",
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
