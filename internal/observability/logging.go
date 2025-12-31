// Package observability provides logging, metrics, and health monitoring.
package observability

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *zap.Logger
	loggerOnce   sync.Once
)

// LogConfig holds logging configuration.
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"` // json or console
	OutputPath string `yaml:"output_path" json:"output_path"`
}

// DefaultLogConfig returns default logging configuration.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
	}
}

// InitLogger initializes the global logger.
func InitLogger(cfg LogConfig) (*zap.Logger, error) {
	var err error
	loggerOnce.Do(func() {
		globalLogger, err = newLogger(cfg)
	})
	return globalLogger, err
}

// Logger returns the global logger.
func Logger() *zap.Logger {
	if globalLogger == nil {
		// Return a no-op logger if not initialized
		return zap.NewNop()
	}
	return globalLogger
}

// L is a shorthand for Logger().
func L() *zap.Logger {
	return Logger()
}

// WithContext returns a logger with context fields.
func WithContext(ctx context.Context) *zap.Logger {
	logger := Logger()

	// Add trace ID if present
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		logger = logger.With(zap.String("trace_id", traceID))
	}

	// Add request ID if present
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		logger = logger.With(zap.String("request_id", requestID))
	}

	return logger
}

func newLogger(cfg LogConfig) (*zap.Logger, error) {
	level := parseLogLevel(cfg.Level)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var output zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" || cfg.OutputPath == "" {
		output = zapcore.AddSync(os.Stdout)
	} else if cfg.OutputPath == "stderr" {
		output = zapcore.AddSync(os.Stderr)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, output, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Context keys for logging.
type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	RequestIDKey contextKey = "request_id"
)

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
