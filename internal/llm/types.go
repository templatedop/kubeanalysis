// Package llm provides LLM integration for Kubernetes analysis.
package llm

import (
	"context"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// CompletionRequest represents a completion request.
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// CompletionResponse represents a completion response.
type CompletionResponse struct {
	ID        string   `json:"id"`
	Model     string   `json:"model"`
	Content   string   `json:"content"`
	Usage     Usage    `json:"usage"`
	CreatedAt time.Time `json:"created_at"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMClient defines the interface for LLM providers.
type LLMClient interface {
	// Complete generates a completion for the given request.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Chat sends a chat message and returns the response.
	Chat(ctx context.Context, messages []Message) (*CompletionResponse, error)
}

// ClientConfig holds configuration for the LLM client.
type ClientConfig struct {
	// Provider specifies the LLM provider (e.g., "ollama", "openai", "anthropic").
	Provider string `json:"provider"`

	// BaseURL is the base URL for the LLM API.
	BaseURL string `json:"base_url"`

	// APIKey is the API key for authentication.
	APIKey string `json:"api_key"`

	// Model specifies the model to use.
	Model string `json:"model"`

	// MaxTokens specifies the default max tokens.
	MaxTokens int `json:"max_tokens"`

	// Temperature specifies the default temperature.
	Temperature float64 `json:"temperature"`

	// Timeout specifies the request timeout.
	Timeout time.Duration `json:"timeout"`
}

// DefaultClientConfig returns a default LLM client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Provider:    "ollama",
		BaseURL:     "http://localhost:11434",
		Model:       "qwen2.5:32b",
		MaxTokens:   4096,
		Temperature: 0.1,
		Timeout:     120 * time.Second,
	}
}

// AnalysisRequest represents a request for K8s analysis.
type AnalysisRequest struct {
	Type        string `json:"type"` // security, performance, prediction, incident
	Context     string `json:"context"`
	Findings    string `json:"findings"`
	RAGContext  string `json:"rag_context"`
	ClusterInfo string `json:"cluster_info"`
}

// AnalysisResponse represents the LLM analysis response.
type AnalysisResponse struct {
	ExecutiveSummary   string              `json:"executive_summary"`
	RiskLevel          string              `json:"risk_level"`
	KeyRisks           []string            `json:"key_risks,omitempty"`
	PrioritizedFindings []PrioritizedFinding `json:"prioritized_findings,omitempty"`
	RemediationRoadmap *RemediationRoadmap `json:"remediation_roadmap,omitempty"`
	Compliance         *ComplianceAnalysis `json:"compliance,omitempty"`
	Confidence         float64             `json:"confidence"`
}

// PrioritizedFinding represents a finding with additional LLM analysis.
type PrioritizedFinding struct {
	Rank           int      `json:"rank"`
	FindingID      string   `json:"finding_id"`
	Exploitability string   `json:"exploitability"`
	Impact         string   `json:"impact"`
	AttackChains   []string `json:"attack_chains,omitempty"`
	BlastRadius    string   `json:"blast_radius,omitempty"`
}

// RemediationRoadmap provides structured remediation guidance.
type RemediationRoadmap struct {
	Immediate []RemediationAction `json:"immediate,omitempty"`
	ShortTerm []RemediationAction `json:"short_term,omitempty"`
	LongTerm  []RemediationAction `json:"long_term,omitempty"`
}

// RemediationAction represents a specific remediation action.
type RemediationAction struct {
	Action     string   `json:"action"`
	FindingIDs []string `json:"finding_ids,omitempty"`
	Effort     string   `json:"effort,omitempty"`
	Impact     string   `json:"impact,omitempty"`
}

// ComplianceAnalysis provides compliance status against frameworks.
type ComplianceAnalysis struct {
	NSACISA *FrameworkStatus `json:"nsa_cisa,omitempty"`
	CIS     *FrameworkStatus `json:"cis,omitempty"`
	MITRE   *FrameworkStatus `json:"mitre,omitempty"`
}

// FrameworkStatus represents compliance status for a specific framework.
type FrameworkStatus struct {
	Passed     int      `json:"passed"`
	Failed     int      `json:"failed"`
	Total      int      `json:"total"`
	Score      float64  `json:"score"`
	Violations []string `json:"violations,omitempty"`
}
