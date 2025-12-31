package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient implements LLMClient using HTTP API calls.
type HTTPClient struct {
	config ClientConfig
	client *http.Client
}

// NewHTTPClient creates a new HTTP-based LLM client.
func NewHTTPClient(config ClientConfig) *HTTPClient {
	if config.Timeout <= 0 {
		config.Timeout = 120 * time.Second
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 4096
	}

	return &HTTPClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Complete generates a completion for the given request.
func (c *HTTPClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if req.Model == "" {
		req.Model = c.config.Model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = c.config.MaxTokens
	}
	if req.Temperature == 0 {
		req.Temperature = c.config.Temperature
	}

	switch c.config.Provider {
	case "ollama":
		return c.completeOllama(ctx, req)
	case "openai":
		return c.completeOpenAI(ctx, req)
	default:
		return c.completeOllama(ctx, req)
	}
}

// Chat sends a chat message and returns the response.
func (c *HTTPClient) Chat(ctx context.Context, messages []Message) (*CompletionResponse, error) {
	req := CompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	}
	return c.Complete(ctx, req)
}

func (c *HTTPClient) completeOllama(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Build the prompt from messages
	prompt := ""
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			prompt += fmt.Sprintf("[SYSTEM] %s\n\n", msg.Content)
		case "user":
			prompt += fmt.Sprintf("[USER] %s\n\n", msg.Content)
		case "assistant":
			prompt += fmt.Sprintf("[ASSISTANT] %s\n\n", msg.Content)
		}
	}

	payload := map[string]interface{}{
		"model":  req.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": req.Temperature,
			"num_predict": req.MaxTokens,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.config.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &CompletionResponse{
		Model:     req.Model,
		Content:   result.Response,
		CreatedAt: time.Now(),
	}, nil
}

func (c *HTTPClient) completeOpenAI(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	payload := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.config.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &CompletionResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   content,
		Usage:     result.Usage,
		CreatedAt: time.Now(),
	}, nil
}

// MockClient provides a mock LLM client for testing.
type MockClient struct {
	response string
}

// NewMockClient creates a new mock LLM client.
func NewMockClient(response string) *MockClient {
	return &MockClient{response: response}
}

// Complete returns a mock completion.
func (c *MockClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{
		ID:        "mock-response",
		Model:     req.Model,
		Content:   c.response,
		CreatedAt: time.Now(),
	}, nil
}

// Chat returns a mock chat response.
func (c *MockClient) Chat(ctx context.Context, messages []Message) (*CompletionResponse, error) {
	return &CompletionResponse{
		ID:        "mock-chat-response",
		Model:     "mock",
		Content:   c.response,
		CreatedAt: time.Now(),
	}, nil
}
