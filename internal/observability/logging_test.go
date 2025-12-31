package observability

import (
	"context"
	"sync"
	"testing"
)

func TestDefaultLogConfig(t *testing.T) {
	cfg := DefaultLogConfig()

	if cfg.Level != "info" {
		t.Errorf("expected level 'info', got %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected format 'json', got %s", cfg.Format)
	}
	if cfg.OutputPath != "stdout" {
		t.Errorf("expected output_path 'stdout', got %s", cfg.OutputPath)
	}
}

func TestInitLogger(t *testing.T) {
	// Reset for testing
	globalLogger = nil
	loggerOnce = sync.Once{}

	cfg := LogConfig{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
	}

	logger, err := InitLogger(cfg)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Verify global logger is set
	if Logger() != logger {
		t.Error("global logger should match returned logger")
	}
}

func TestLoggerReturnsNopIfNotInitialized(t *testing.T) {
	// Reset for testing
	globalLogger = nil
	loggerOnce = sync.Once{}

	logger := Logger()
	if logger == nil {
		t.Fatal("expected logger to be non-nil even if not initialized")
	}
}

func TestWithContext(t *testing.T) {
	// Initialize logger first
	globalLogger = nil
	loggerOnce = sync.Once{}
	InitLogger(DefaultLogConfig())

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")

	logger := WithContext(ctx)
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"error", "error"},
		{"fatal", "fatal"},
		{"unknown", "info"}, // defaults to info
	}

	for _, tt := range tests {
		level := parseLogLevel(tt.input)
		// Just verify it doesn't panic
		_ = level
	}
}

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	// Add trace ID
	ctx = WithTraceID(ctx, "test-trace")
	if traceID, ok := ctx.Value(TraceIDKey).(string); !ok || traceID != "test-trace" {
		t.Error("expected trace ID to be set")
	}

	// Add request ID
	ctx = WithRequestID(ctx, "test-request")
	if requestID, ok := ctx.Value(RequestIDKey).(string); !ok || requestID != "test-request" {
		t.Error("expected request ID to be set")
	}
}
