// Package config provides configuration management for the kubeanalysis framework.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the kubeanalysis framework.
type Config struct {
	// Kubernetes configuration
	Kubernetes KubernetesConfig `yaml:"kubernetes"`

	// Temporal configuration
	Temporal TemporalConfig `yaml:"temporal"`

	// LLM configuration
	LLM LLMConfig `yaml:"llm"`

	// Embedder configuration
	Embedder EmbedderConfig `yaml:"embedder"`

	// RAG configuration
	RAG RAGConfig `yaml:"rag"`

	// Analysis configuration
	Analysis AnalysisConfig `yaml:"analysis"`

	// Alerting configuration
	Alerting AlertingConfig `yaml:"alerting"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`
}

// KubernetesConfig holds Kubernetes client configuration.
type KubernetesConfig struct {
	// Kubeconfig path (empty for in-cluster)
	Kubeconfig string `yaml:"kubeconfig"`

	// Context to use (empty for current context)
	Context string `yaml:"context"`

	// Namespaces to analyze (empty for all)
	Namespaces []string `yaml:"namespaces"`

	// ExcludeNamespaces to skip during analysis
	ExcludeNamespaces []string `yaml:"exclude_namespaces"`

	// QPS for Kubernetes client
	QPS float32 `yaml:"qps"`

	// Burst for Kubernetes client
	Burst int `yaml:"burst"`
}

// TemporalConfig holds Temporal workflow configuration.
type TemporalConfig struct {
	// Host is the Temporal server address
	Host string `yaml:"host"`

	// Namespace is the Temporal namespace
	Namespace string `yaml:"namespace"`

	// TaskQueue is the task queue name
	TaskQueue string `yaml:"task_queue"`

	// WorkerCount is the number of workflow workers
	WorkerCount int `yaml:"worker_count"`

	// MaxConcurrentActivities limits concurrent activities
	MaxConcurrentActivities int `yaml:"max_concurrent_activities"`

	// MaxConcurrentWorkflows limits concurrent workflows
	MaxConcurrentWorkflows int `yaml:"max_concurrent_workflows"`
}

// LLMConfig holds LLM client configuration.
type LLMConfig struct {
	// Provider is the LLM provider (ollama, openai, anthropic)
	Provider string `yaml:"provider"`

	// BaseURL is the API base URL
	BaseURL string `yaml:"base_url"`

	// APIKey is the API key for authentication
	APIKey string `yaml:"api_key"`

	// Model is the model to use
	Model string `yaml:"model"`

	// MaxTokens is the maximum tokens per request
	MaxTokens int `yaml:"max_tokens"`

	// Temperature for generation
	Temperature float64 `yaml:"temperature"`

	// Timeout for requests
	Timeout time.Duration `yaml:"timeout"`
}

// EmbedderConfig holds embedding configuration.
type EmbedderConfig struct {
	// Provider is the embedding provider (ollama, openai, local)
	Provider string `yaml:"provider"`

	// BaseURL is the API base URL
	BaseURL string `yaml:"base_url"`

	// APIKey is the API key
	APIKey string `yaml:"api_key"`

	// Model is the embedding model
	Model string `yaml:"model"`

	// Dimension is the embedding dimension
	Dimension int `yaml:"dimension"`

	// BatchSize for embedding requests
	BatchSize int `yaml:"batch_size"`
}

// RAGConfig holds RAG engine configuration.
type RAGConfig struct {
	// TopK is the default number of results
	TopK int `yaml:"top_k"`

	// MinScore is the minimum similarity score
	MinScore float64 `yaml:"min_score"`

	// MaxContextLength is the maximum context length
	MaxContextLength int `yaml:"max_context_length"`

	// ChunkSize for document splitting
	ChunkSize int `yaml:"chunk_size"`

	// ChunkOverlap for document splitting
	ChunkOverlap int `yaml:"chunk_overlap"`

	// KnowledgePath is the path to knowledge files
	KnowledgePath string `yaml:"knowledge_path"`
}

// AnalysisConfig holds analysis configuration.
type AnalysisConfig struct {
	// Interval for continuous monitoring
	Interval time.Duration `yaml:"interval"`

	// Types to run (security, performance, prediction)
	Types []string `yaml:"types"`

	// Severity threshold for alerts
	SeverityThreshold string `yaml:"severity_threshold"`

	// EnablePredictions enables predictive analysis
	EnablePredictions bool `yaml:"enable_predictions"`
}

// AlertingConfig holds alerting configuration.
type AlertingConfig struct {
	// Enabled enables alerting
	Enabled bool `yaml:"enabled"`

	// Slack webhook configuration
	Slack SlackConfig `yaml:"slack"`

	// PagerDuty configuration
	PagerDuty PagerDutyConfig `yaml:"pagerduty"`

	// Webhook for custom notifications
	Webhook WebhookConfig `yaml:"webhook"`
}

// SlackConfig holds Slack alerting configuration.
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

// PagerDutyConfig holds PagerDuty configuration.
type PagerDutyConfig struct {
	Enabled     bool   `yaml:"enabled"`
	RoutingKey  string `yaml:"routing_key"`
	ServiceName string `yaml:"service_name"`
}

// WebhookConfig holds custom webhook configuration.
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `yaml:"level"`

	// Format is the log format (json, text)
	Format string `yaml:"format"`

	// Output is the log output (stdout, stderr, file path)
	Output string `yaml:"output"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig:        "",
			Context:           "",
			Namespaces:        []string{},
			ExcludeNamespaces: []string{"kube-system", "kube-public", "kube-node-lease"},
			QPS:               50,
			Burst:             100,
		},
		Temporal: TemporalConfig{
			Host:                    "localhost:7233",
			Namespace:               "default",
			TaskQueue:               "k8s-analysis-queue",
			WorkerCount:             4,
			MaxConcurrentActivities: 10,
			MaxConcurrentWorkflows:  10,
		},
		LLM: LLMConfig{
			Provider:    "ollama",
			BaseURL:     "http://localhost:11434",
			Model:       "qwen2.5:32b",
			MaxTokens:   4096,
			Temperature: 0.1,
			Timeout:     120 * time.Second,
		},
		Embedder: EmbedderConfig{
			Provider:  "ollama",
			BaseURL:   "http://localhost:11434",
			Model:     "nomic-embed-text",
			Dimension: 768,
			BatchSize: 32,
		},
		RAG: RAGConfig{
			TopK:             10,
			MinScore:         0.5,
			MaxContextLength: 8000,
			ChunkSize:        1000,
			ChunkOverlap:     200,
			KnowledgePath:    "./knowledge",
		},
		Analysis: AnalysisConfig{
			Interval:          15 * time.Minute,
			Types:             []string{"security", "performance", "prediction"},
			SeverityThreshold: "HIGH",
			EnablePredictions: true,
		},
		Alerting: AlertingConfig{
			Enabled: false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

// Load loads configuration from a YAML file.
func Load(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	config.applyEnvOverrides()

	return config, nil
}

// LoadOrDefault loads configuration from file or returns default.
func LoadOrDefault(path string) *Config {
	if path == "" {
		return DefaultConfig()
	}

	config, err := Load(path)
	if err != nil {
		return DefaultConfig()
	}

	return config
}

// Save saves configuration to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides.
func (c *Config) applyEnvOverrides() {
	// Kubernetes
	if v := os.Getenv("KUBECONFIG"); v != "" {
		c.Kubernetes.Kubeconfig = v
	}

	// Temporal
	if v := os.Getenv("TEMPORAL_HOST"); v != "" {
		c.Temporal.Host = v
	}
	if v := os.Getenv("TEMPORAL_NAMESPACE"); v != "" {
		c.Temporal.Namespace = v
	}

	// LLM
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		c.LLM.BaseURL = v
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		c.LLM.Model = v
	}

	// Embedder
	if v := os.Getenv("EMBEDDER_PROVIDER"); v != "" {
		c.Embedder.Provider = v
	}
	if v := os.Getenv("EMBEDDER_BASE_URL"); v != "" {
		c.Embedder.BaseURL = v
	}
	if v := os.Getenv("EMBEDDER_API_KEY"); v != "" {
		c.Embedder.APIKey = v
	}

	// Alerting
	if v := os.Getenv("SLACK_WEBHOOK_URL"); v != "" {
		c.Alerting.Slack.Enabled = true
		c.Alerting.Slack.WebhookURL = v
	}
	if v := os.Getenv("PAGERDUTY_ROUTING_KEY"); v != "" {
		c.Alerting.PagerDuty.Enabled = true
		c.Alerting.PagerDuty.RoutingKey = v
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Temporal.Host == "" {
		return fmt.Errorf("temporal.host is required")
	}

	if c.LLM.Provider != "" && c.LLM.BaseURL == "" {
		return fmt.Errorf("llm.base_url is required when provider is set")
	}

	if c.RAG.TopK <= 0 {
		return fmt.Errorf("rag.top_k must be positive")
	}

	if c.RAG.ChunkSize <= 0 {
		return fmt.Errorf("rag.chunk_size must be positive")
	}

	return nil
}
