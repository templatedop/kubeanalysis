package k8s

import (
	"testing"
)

func TestClientConfig(t *testing.T) {
	cfg := ClientConfig{
		KubeconfigPath: "/path/to/kubeconfig",
		Context:        "test-context",
		InCluster:      false,
	}

	if cfg.KubeconfigPath != "/path/to/kubeconfig" {
		t.Errorf("expected kubeconfig path '/path/to/kubeconfig', got %s", cfg.KubeconfigPath)
	}

	if cfg.Context != "test-context" {
		t.Errorf("expected context 'test-context', got %s", cfg.Context)
	}

	if cfg.InCluster {
		t.Error("expected InCluster to be false")
	}
}

func TestClientConfigInCluster(t *testing.T) {
	cfg := ClientConfig{
		InCluster: true,
	}

	if !cfg.InCluster {
		t.Error("expected InCluster to be true")
	}
}

func TestNewClientWithInvalidKubeconfig(t *testing.T) {
	cfg := ClientConfig{
		KubeconfigPath: "/nonexistent/kubeconfig",
		InCluster:      false,
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}
